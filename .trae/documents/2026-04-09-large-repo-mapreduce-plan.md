# Large Repo Map-Reduce Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 优化后端对特大型开源项目（如 `golang/go`）的解析流程，通过降低并发、减小分块以及多级递归 Map-Reduce 解决内存溢出和 API 限流问题。

**Architecture:** 修改 `parser.GitFetcher` 的文件过滤与分块大小；修改 `service.DecompositionService` 的 Worker 并发上限，并引入递归结构的 Tree Reduce 算法处理大批量局部摘要。

**Tech Stack:** Go 1.21+, Goroutine, Semaphore, DeepSeek API

---

### Task 1: 优化 GitFetcher 文件过滤与分块上限

**Files:**
- Modify: `backend/internal/parser/git_fetcher.go`

- [ ] **Step 1: 增加严格的忽略目录**

修改 `isIgnoredPath` 函数中的 `ignoredDirs` 切片，增加 `test`, `tests`, `e2e`, `benchmark`, `mocks`, `Godeps`, `third_party`。

```go
ignoredDirs := []string{
"node_modules", "vendor", "dist", "build", "out", "target", "bin",
".git", ".svn", ".idea", ".vscode", "__pycache__", "testdata", "docs", "examples", "scripts", "assets",
"test", "tests", "e2e", "benchmark", "mocks", "Godeps", "third_party",
}
```

- [ ] **Step 2: 减小最大分块大小**

将 `maxChunkChars` 常量修改为 150000。

```go
const maxChunkChars = 150000
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/parser/git_fetcher.go
git commit -m "refactor(parser): enhance git fetcher ignore rules and reduce chunk size"
```

### Task 2: 严格限制 Map 阶段并发度

**Files:**
- Modify: `backend/internal/service/decomposition.go`

- [ ] **Step 1: 修改 maxWorkers 上限**

在 `mapReduceAnalyze` 方法中，将最大的 `maxWorkers` 限制为 3。

```go
numCPU := runtime.NumCPU()
maxWorkers := numCPU
if maxWorkers < 2 {
maxWorkers = 2
}
if maxWorkers > 3 {
maxWorkers = 3
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/service/decomposition.go
git commit -m "fix(service): strict concurrency limit to prevent LLM 429 errors"
```

### Task 3: 实现多级递归 Tree Reduce

**Files:**
- Modify: `backend/internal/service/decomposition.go`

- [ ] **Step 1: 新增 recursiveTreeReduce 方法**

```go
func (s *DecompositionService) recursiveTreeReduce(ctx context.Context, summaries []string, sendProgress func(int, string, interface{}), round int) []string {
if len(summaries) <= 15 {
return summaries
}

sendProgress(2, fmt.Sprintf("局部摘要数量较多 (%d)，正在进行第 %d 轮 Tree Reduce 深度汇总...", len(summaries), round), nil)

var intermediateSummaries []string
batchSize := 10

for i := 0; i < len(summaries); i += batchSize {
end := i + batchSize
if end > len(summaries) {
end = len(summaries)
}
batchContent := strings.Join(summaries[i:end], "\n\n")

prompt := fmt.Sprintf(`你是一个高级架构师。以下是一个大型项目部分模块的局部摘要集合。
请将这些局部摘要融合成一个中级摘要，提炼出这些模块共同负责的核心功能、数据流和架构逻辑。
忽略过于细节的代码实现，重点关注模块间的关系和整体职责。字数控制在 800 字以内。

模块摘要如下：
%s`, batchContent)

req := []llm.Message{
{Role: "system", Content: "你是一个专业的架构师，擅长将零散的模块信息归纳为系统化的高层架构描述。"},
{Role: "user", Content: prompt},
}

modelStr := "deepseek-chat"
if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
modelStr = envModel
}

ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Minute)
interSummary, err := s.llmClient.Generate(ctxTimeout, modelStr, req)
cancel()
if err != nil {
intermediateSummaries = append(intermediateSummaries, summaries[i:end]...)
} else {
intermediateSummaries = append(intermediateSummaries, interSummary)
}
}

return s.recursiveTreeReduce(ctx, intermediateSummaries, sendProgress, round+1)
}
```

- [ ] **Step 2: 替换 AnalyzeStream 中的原有单层 Reduce 逻辑**

将 `AnalyzeStream` 中判断 `if len(summaries) > 20 { ... }` 的大段代码替换为调用递归方法：

```go
// Tree Reduce: 如果局部摘要数量过多，进行多级递归汇总
summaries = s.recursiveTreeReduce(ctx, summaries, sendProgress, 1)

finalContent.WriteString(treeContent)
finalContent.WriteString("\n=== Local Summaries ===\n")
for _, summary := range summaries {
finalContent.WriteString(summary)
finalContent.WriteString("\n\n")
}
```

- [ ] **Step 3: 运行并验证**

可以启动后端服务器，尝试发送一两个解析请求确保语法无误且逻辑跑通。
```bash
cd backend && go build -o bin/server cmd/server/main.go
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/decomposition.go
git commit -m "feat(service): implement recursive tree reduce for huge repositories"
```


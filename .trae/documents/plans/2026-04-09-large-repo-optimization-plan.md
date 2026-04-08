# InkWords Large Repo Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement smart filtering, exponential backoff, and Tree Reduce logic to robustly handle massive Git repositories like `golang/go`.

**Architecture:** 
1. `git_fetcher.go`: Exclude test files, docs, and third-party vendor directories during repo traversal.
2. `decomposition.go`: Implement exponential backoff inside `mapReduceAnalyze` to gracefully handle 429 rate limits.
3. `decomposition.go`: Add a Tree Reduce fallback when the number of chunks is large (>20) to prevent 128k context token limit errors.

**Tech Stack:** Go, Gin

---

### Task 1: Smart Filtering for Git Fetcher

**Files:**
- Modify: `backend/internal/parser/git_fetcher.go`

- [ ] **Step 1: Add helper function to check ignored paths**

Add this helper function to filter out unwanted directories and files:

```go
import "strings"

func isIgnoredPath(path string, info os.FileInfo) bool {
	// 忽略常见依赖和构建产物目录
	ignoredDirs := []string{
		"node_modules", "vendor", "dist", "build", "out", "target", "bin",
		".git", ".svn", ".idea", ".vscode", "__pycache__", "testdata", "docs", "examples", "scripts", "assets",
	}

	for _, dir := range ignoredDirs {
		if strings.Contains(path, "/"+dir+"/") || strings.HasPrefix(path, dir+"/") {
			return true
		}
	}

	if !info.IsDir() {
		name := strings.ToLower(info.Name())
		// 忽略测试文件
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".test.js") || strings.HasSuffix(name, ".spec.js") || strings.HasSuffix(name, ".test.ts") || strings.HasSuffix(name, ".spec.ts") {
			return true
		}
		// 忽略非代码的静态资源与文档
		ignoredExts := []string{
			".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg", ".mp4", ".mp3", ".wav", ".zip", ".tar", ".gz", ".rar", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".ttf", ".woff", ".woff2", ".eot",
		}
		for _, ext := range ignoredExts {
			if strings.HasSuffix(name, ext) {
				return true
			}
		}
	}
	return false
}
```

- [ ] **Step 2: Apply the filter in `extractGitRepo`**

In `extractGitRepo`, replace the current basic `node_modules` and `.git` check with the new helper:

```go
		if isIgnoredPath(relPath, info) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
```

- [ ] **Step 3: Run the code to verify compilation**

Run: `cd backend && go build -o server ./cmd/server`
Expected: PASS

---

### Task 2: Exponential Backoff in MapReduce

**Files:**
- Modify: `backend/internal/service/decomposition.go`

- [ ] **Step 1: Implement Exponential Backoff helper**

Add a simple math/rand helper for exponential backoff:

```go
import (
	"math/rand"
	"time"
)

// exponentialBackoff 返回退避时间： 2^retryCount 秒 + 随机抖动
func exponentialBackoff(retryCount int) time.Duration {
	base := float64(2) // 基础等待2秒
	for i := 0; i < retryCount; i++ {
		base *= 2
	}
	// 加上 0~1000 毫秒的随机抖动，防止惊群效应
	jitter := rand.Intn(1000)
	return time.Duration(base)*time.Second + time.Duration(jitter)*time.Millisecond
}
```

- [ ] **Step 2: Apply backoff in Map worker**

In `mapReduceAnalyze` inside the retry loop:

```go
					if err != nil {
						lastErr = err
						if attempt < maxRetries-1 {
							// 通知重试
							progressChan <- map[string]interface{}{
								"type": "chunk_analyzing",
								"data": map[string]interface{}{
									"id":       workerID,
									"chunk_id": i + 1,
									"path":     chunk.Directory,
									"retry":    attempt + 1,
								},
							}
							time.Sleep(exponentialBackoff(attempt))
							continue
						}
					}
```

- [ ] **Step 3: Verify compilation**

Run: `cd backend && go build -o server ./cmd/server`
Expected: PASS

---

### Task 3: Tree Reduce Implementation

**Files:**
- Modify: `backend/internal/service/decomposition.go`

- [ ] **Step 1: Implement intermediate reduce step**

Before the final global summary in `mapReduceAnalyze`, add logic to check if `len(summaries) > 20`. If so, group them and generate intermediate summaries.

```go
	// Tree Reduce: 如果局部摘要数量过多，先进行中间层汇总
	if len(summaries) > 20 {
		var intermediateSummaries []string
		batchSize := 10
		for i := 0; i < len(summaries); i += batchSize {
			end := i + batchSize
			if end > len(summaries) {
				end = len(summaries)
			}
			batchContent := strings.Join(summaries[i:end], "\n\n")
			
			// 生成中间层摘要
			prompt := fmt.Sprintf(`你是一个高级架构师。以下是一个大型项目部分模块的局部摘要集合。
请将这些局部摘要融合成一个中级摘要，提炼出这些模块共同负责的核心功能、数据流和架构逻辑。
忽略过于细节的代码实现，重点关注模块间的关系和整体职责。字数控制在 800 字以内。

模块摘要如下：
%s`, batchContent)

			req := &llm.GenerateRequest{
				SystemPrompt: "你是一个专业的架构师，擅长将零散的模块信息归纳为系统化的高层架构描述。",
				Prompt:       prompt,
			}
			
			ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Minute)
			interSummary, err := s.generatorService.llmClient.GenerateStream(ctxTimeout, req, nil)
			cancel()
			if err != nil {
				// 容错处理：如果中间层汇总失败，保留原文
				intermediateSummaries = append(intermediateSummaries, summaries[i:end]...)
			} else {
				intermediateSummaries = append(intermediateSummaries, interSummary)
			}
		}
		summaries = intermediateSummaries
	}
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend && go build -o server ./cmd/server`
Expected: PASS

---
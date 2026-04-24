# 大型仓库解析优化与优雅降级方案设计 (修订版 v2)

## 1. 背景与目标
针对像 `golang/go` 这样的大型项目，用户往往只关心某个核心模块（如 `src/net/http`）。现有全量解析方案不仅拉取极慢，还会导致 Token 超限与并发失败。
本方案的核心目标是提供**子目录精准拉取**能力，并在极端情况下实现**基于字符阈值的优雅截断降级**。同时，针对系统现有的并发机制进行深度优化，提升分析速度并降低 429 (Too Many Requests) 的触发概率。

## 2. 核心架构设计

### 2.1 Git 稀疏克隆与子目录支持 (Sparse Checkout)
- **后端 API 扩展**：`/api/v1/project/analyze` 和 `/api/v1/stream/analyze` 请求体新增 `sub_dir` (可选，字符串)。
- **GitFetcher 改造**：放弃传统的浅克隆。如果用户提供了 `sub_dir`，则使用 Git 2.25+ 支持的 **Sparse Checkout** 机制，仅下载指定的子目录：
  1. `git clone --filter=blob:none --no-checkout <url> <tempDir>`
  2. `git sparse-checkout set <sub_dir>`
  3. `git checkout`
  这能极大节省磁盘 I/O 和网络带宽，使得拉取 `golang/go` 中单个模块的速度从几分钟缩短到几秒。

### 2.2 优雅截断降级机制 (Graceful Degradation)
- **按 Chunk 数量阈值截断**：
  - 在 `filepath.Walk` 遍历提取文本时，统计生成的 Chunk 总数。
  - 当产生的 Chunk 数量达到 `maxTotalChunks = 15`（每个 300k 字符，总约 450w 字符）时，立即**停止遍历**并跳出。
- **AI 融合截断提示**：
  - 截断发生后，在拼接给 LLM 的 `treeContent` 结尾，或者给大纲生成的 System Prompt 中注入一段特殊的背景说明：
    `"【系统提示】由于该项目体量极其庞大，系统已执行优雅降级，自动截断了后续文件（仅保留了前15个核心模块的分块）。请你在生成的博客引言或开头中，自然地向读者说明：由于项目过于庞大，本文仅抽取分析了其核心的若干模块代码，并未包含全量内容。"`

### 2.3 前端交互优化
- **UI 组件**：在 Dashboard 输入 Git URL 的输入框下方，新增一个**“高级选项 (Advanced Options)”折叠面板 (Accordion)**。
- **子目录输入**：折叠面板展开后，提供一个“指定解析子目录”的输入框，提示语如：“针对大型仓库推荐使用，如 src/net/http”。

### 2.4 并发机制深度优化 (Concurrency Enhancements)
1. **解绑 CPU 核心限制**：
   - 移除 `runtime.NumCPU()` 硬编码。
   - 引入环境变量 `MAX_CONCURRENT_WORKERS` 控制全局大模型并发数（默认为 3，最大不超过 20），以适应不同 API 的额度。
2. **全局 RPM 令牌桶限流 (Token Bucket)**：
   - 引入 `golang.org/x/time/rate.Limiter`，基于环境变量 `LLM_API_RPM_LIMIT` (默认 60) 创建全局限流器。
   - 在每次调用 `llmClient.Generate` 之前，必须调用 `limiter.Wait(ctx)` 消费一个 Token，主动防范由于瞬间爆发的大量小 Chunk 请求导致的 API HTTP 429 封禁。
3. **Tree Reduce 并发化**：
   - 将原先串行执行的 `Tree Reduce` 中间层合并过程，改造为受 `semaphore` 限制的并发执行。使用 `sync.WaitGroup` 等待中间层合并完成，大幅缩短碎片极多时的合并耗时。

## 3. 错误处理与容错
- 若仓库地址无效或 Sparse Checkout 失败（如子目录拼写错误），向前端抛出明确的提示：“Git 拉取失败：请检查仓库地址或子目录路径是否正确”。
- 如果 `limiter.Wait` 等待超时，仍依赖底层的 Exponential Backoff 机制进行二次重试。

## 4. 实施计划 (Implementation Plan)
1. **并发机制改造**：在 `decomposition.go` 和 `llmClient` 中引入环境变量、RPM 限流器与并发 Tree Reduce。
2. **GitFetcher 改造**：引入 Sparse Checkout 逻辑，添加 15 个 Chunk 的阈值截断逻辑。
3. **前端 UI 更新**：使用 Shadcn UI 的 Accordion 编写高级选项面板。
4. **全链路测试**：测试完整的 `golang/go` 仓库配合 `src/net/http` 子目录，验证拉取速度、并发限流有效性与截断保护效果。

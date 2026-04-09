# 大型项目（如 golang/go）全量分层 Map-Reduce 解析方案设计

## 1. 背景与目标
在分析大型开源项目（如 `https://github.com/golang/go`）时，由于文件数量极多、嵌套极深，现有的全量解析方案容易出现：
1. **内存溢出 (OOM) 与拉取缓慢**：全量读取文件到内存。
2. **并发限流 (429 Too Many Requests)**：`maxWorkers` 过高导致大模型 API 拒绝请求。
3. **上下文超限**：Reduce 阶段生成的中间摘要总长度仍然过大，导致最终生成大纲时触发 `invalid_request_error` 或截断关键信息。

**目标**：保留全量读取的细节，但通过严格的并发控制、多级递归 Map-Reduce 汇总，确保大型项目分析过程稳定、可靠。

## 2. 核心架构设计

### 2.1 降低单次并发与分块大小
- **Worker 并发限制**：在 `mapReduceAnalyze` 中，将 `maxWorkers` 强制封顶为 **3**（原为 8）。牺牲一定的速度，换取请求的稳定，避免触发 API 限流。
- **Chunk 大小调整**：在 `GitFetcher` 中，将单块的最大字符数 `maxChunkChars` 从 `300,000` 降低到 `150,000`。缩小每个 Chunk 的体积，让单次 LLM 请求的 Context 变小，响应更快且更不容易出错。

### 2.2 多级递归 Tree Reduce (Recursive Map-Reduce)
- 废弃原有的单次“中间层”合并逻辑。
- 引入 `recursiveTreeReduce` 函数：
  - 如果摘要片段数量 `> 15` 个，则将它们按每批 `10` 个进行分组。
  - 并发（受限）地调用 LLM 生成这 `10` 个摘要的“上级摘要”。
  - 将生成的上级摘要收集起来。如果数量仍然 `> 15`，则继续递归调用 `recursiveTreeReduce`。
  - 直到摘要总数 `<= 15` 时，将它们拼接到最终的 Context 中送入大纲生成环节。
- **进度保活**：递归过程中，通过 `progressChan` 实时下发状态信息（如 `正在进行第 N 轮深度汇总，剩余 X 个模块...`），防止长耗时导致前端或 Nginx/Vite Proxy 判定超时。

### 2.3 更严格的文件过滤机制
- 在 `parser/git_fetcher.go` 中，增强 `isIgnoredPath` 的规则，专门针对大型开源库进行过滤：
  - 增加忽略目录：`test`, `tests`, `e2e`, `benchmark`, `mocks`, `Godeps`, `third_party`。
  - 这是为了过滤掉对核心架构理解无太大帮助但占用极大篇幅的文件，大幅缩减 Chunk 数量。

## 3. 错误处理与容错
- 在每一层 Reduce 汇总中，若某一组汇总彻底失败（超过重试次数），则降级保留该组的原始摘要拼接，确保最终大纲不会丢失该部分的“痕迹”。

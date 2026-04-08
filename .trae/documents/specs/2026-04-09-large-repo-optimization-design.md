# InkWords 大型仓库并发分析健壮性增强设计 (2026-04-09)

## 1. 目标与背景
用户在分析类似于 `golang/go` 这样的超大型 Git 仓库时，由于文件总数庞大，经常会切分出数百个 Chunk（如截图所示达到 700+）。这就导致：
- **API 并发轰炸与限流**：数百个请求并发涌向大模型，极易触发 `429 Too Many Requests`，简单的重试机制会导致频繁重试甚至账号封禁。
- **Context 容量溢出**：即使所有局部摘要生成完毕，在最后的 Reduce 阶段把数百份摘要一次性合并发给大模型，总 Token 极容易突破 128k 上限，导致 `unexpected end of JSON input` 或内容截断。
- **无效计算**：大型项目中包含海量的测试代码（`*_test.go`）、第三方依赖（`vendor/`）以及文档/样例，这些对生成核心架构大纲帮助不大，但占据了大量计算资源。

## 2. 详细设计方案

### 2.1 智能过滤与拉取减负 (Smart Filtering)
- **目标文件**: `backend/internal/parser/git_fetcher.go`
- **逻辑增强**:
  - 在 `extractGitRepo` 遍历文件树时，不仅要排除原本的 `node_modules` 等，还要增加以下黑名单：
    - 目录名包含：`vendor`, `testdata`, `docs`, `examples`, `scripts`, `assets` 等。
    - 文件名包含：`_test.go`, `.test.js/ts`, `.spec.js/ts` 等测试文件。
    - 排除二进制与常见静态资源文件（`.png`, `.jpg`, `.pdf`, `.mp4` 等）。
  - 这能大幅削减提取出的纯文本体积，从而成倍减少拆分出的 Chunk 数量。

### 2.2 指数退避重试机制 (Exponential Backoff)
- **目标文件**: `backend/internal/service/decomposition.go` 的 `GenerateSeries` / `mapReduceAnalyze` 中
- **逻辑增强**:
  - 针对大模型的流式或非流式请求（尤其在收到网络错误或 Token 耗尽/限流时），引入指数退避。
  - 第一重试等待 `2秒`，第二次 `4秒`，第三次 `8秒`。并加入随机的抖动（Jitter），防止多个协程在同一时刻集体苏醒并重试，造成 API 侧的“惊群效应”。

### 2.3 多级树状汇总 (Tree Reduce)
- **目标文件**: `backend/internal/service/decomposition.go` 中的 `mapReduceAnalyze` (Reduce 阶段)
- **逻辑增强**:
  - 在收集完所有 Map 阶段的局部摘要 `[]string` 后，如果摘要总数 `N > 20` 或汇总后的纯文本长度超过 150,000 字符，就**不要**直接将其塞入大模型的单次请求中。
  - **Tree Reduce 算法**:
    - 将长摘要数组分批，比如每批 10-20 个摘要为一组。
    - 针对每一组，调用一次非流式的中间层 Prompt：“请将以下项目的局部摘要合并为一个中级摘要，提炼核心架构与功能”。
    - 将所有组返回的“中级摘要”再次合并，作为最终全局汇总的输入。
  - 这不仅能完全规避 Token 上限，还能让大模型在逐级归纳中保持更高的逻辑连贯性，防止注意力分散。

## 3. 影响范围与测试
- **测试重点**：
  - 测试超大仓库（如 `https://github.com/golang/go`），验证 Chunk 数量是否显著下降（比如从 700 降到 200）。
  - 验证指数退避在网络断开或人为模拟 429 报错时，能否正确执行并延长等待时间。
  - 验证当触发 Tree Reduce 阈值时，程序能否正确切分、执行中层合并并最终输出完整的全局大纲。
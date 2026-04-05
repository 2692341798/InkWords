# 任务分解与执行计划

## 1. 拆分逻辑 (GitFetcher)
- [ ] 修改 `GitFetcher.Fetch` 返回类型，支持返回项目结构树和 `[]Chunk`。
- [ ] 实现基于目录的聚合逻辑：遍历文件时，按目录累加内容，达到阈值 (`300,000` 字符) 则分割为一个 Chunk。

## 2. 并发调用与 Map 阶段 (DecompositionService)
- [ ] 在 `DecompositionService` 中增加 `MapReduceAnalyze` 方法。
- [ ] 实现 Goroutine 并发池，最大并发数设为 5。
- [ ] 编写局部摘要生成的 Prompt。
- [ ] 实现单任务 3 次重试逻辑。若失败，返回空摘要或跳过。
- [ ] 添加 SSE 进度事件，向前端发送各个 Chunk 的状态。

## 3. 汇总与 Reduce 阶段
- [ ] 收集所有成功的局部摘要，拼接为最终的上下文。
- [ ] 将上下文传入 `GenerateOutline`，获取最终的项目大纲。

## 4. 前端对接与 UI 适配
- [ ] 在 `streamStore.ts` 增加 Map-Reduce 细粒度状态字段 (如分析文件总数、完成数等)。
- [ ] 在 `useBlogStream.ts` 监听新的细粒度事件并更新状态。
- [ ] 在 `Generator.tsx` 展现 Map 阶段的进度（例如：进度条或滚动列表）。

## 5. 验证与测试
- [ ] 运行 `playwright_e2e_test.py` 或手工验证大型 GitHub 仓库的解析流程，确保不断流且并发不报错。
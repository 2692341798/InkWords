# Tasks
- [x] Task 1: 实现 GitFetcher (`backend/internal/parser/git_fetcher.go`)
  - [x] SubTask 1.1: 编写克隆仓库代码至临时目录逻辑。
  - [x] SubTask 1.2: 编写文件遍历与过滤逻辑（忽略 `.git`, `node_modules`, `dist` 等二进制或编译产物）。
  - [x] SubTask 1.3: 提取纯文本内容，并在解析结束后执行 `defer os.RemoveAll` 实现阅后即焚。
  - [x] SubTask 1.4: 编写 `git_fetcher_test.go` 进行单元测试（可 Mock 一个小型本地仓库）。

- [x] Task 2: 实现大项目评估与大纲规划 API
  - [x] SubTask 2.1: 在 `backend/internal/service/decomposition.go` 中编写大纲生成 Prompt 与逻辑。
  - [x] SubTask 2.2: 在 `backend/internal/api/` 新增 `POST /api/v1/project/analyze` 路由，接收仓库 URL 或超长文本，返回大纲 JSON。

- [x] Task 3: 实现并发调度生成机制
  - [x] SubTask 3.1: 在 `decomposition.go` 中实现 `GenerateSeries` 方法，使用 Goroutine 池与 `sync.WaitGroup` 按大纲并发生成章节。
  - [x] SubTask 3.2: 将生成的各个章节与对应的 `ParentID` (主记录) 和 `ChapterSort` (排序) 落库至 PostgreSQL (`blogs` 表)。

- [x] Task 4: 调整 SSE 推流适配系列生成
  - [x] SubTask 4.1: 更新 `POST /api/v1/stream/generate`，支持接收已确认的大纲进行生成，并在生成各章节时，推送当前章节进度或状态给前端。

# Task Dependencies
- [Task 2] depends on [Task 1]
- [Task 3] depends on [Task 2]
- [Task 4] depends on [Task 3]

# Core / Blog / Task 边界治理 Runbook

本文档用于固化 `Task 4` 的“服务写入归属矩阵”与当前边界治理规则，避免在共享数据库阶段继续把“同库”误当成“无边界”。

## 1. 目标

- 明确哪些表由哪个服务事实拥有。
- 记录当前允许的跨服务写入例外。
- 列出已确认的待收口写点，给后续 Task 8 拆库和 Task 9 CI/Runbook 固化提供基线。

## 2. 当前事实归属矩阵

| 表 / 资源 | 事实归属服务 | 当前允许写入方 | 备注 |
| --- | --- | --- | --- |
| `users` | `core-api` | `core-api` | 注册、登录失败计数、GitHub 绑定、Token 记账 |
| `oauth_tokens` | `core-api` | `core-api` | 第三方平台授权信息；当前模型已存在 |
| `user_prompt_settings` | `core-api` | `core-api` | 用户写作模板覆盖值 |
| `blogs` | `core-api` | `core-api` | 博客、系列父子结构、润色与续写落库 |
| `job_tasks` | `core-api` | `core-api`、`llm-stream`、`parser-service`、`export-service` | 统一任务控制面 |
| `job_task_events` | `core-api` | `core-api`、`llm-stream`、`parser-service`、`export-service` | 统一事件回放面 |
| `review_sessions` | `review-service` | `review-service` | 已拆到 `inkwords_review_db` |
| `review_turns` | `review-service` | `review-service` | 已拆到 `inkwords_review_db` |

## 3. 当前允许的跨服务例外

### 3.1 允许项
- `llm-stream`、`parser-service`、`export-service` 可以写 `job_tasks / job_task_events`。

### 3.2 允许条件
- 只能通过 `internal/domain/task` 中已经定义的显式 repository / service 接口写入。
- 只能写任务状态、任务结果与可回放事件，不得顺带扩展为任意业务表写入。
- 若未来需要新增跨服务写入白名单，必须先补文档、再补测试、最后补实现。

### 3.3 明确禁止
- 非 `core-api` 服务直接写 `blogs`、`users`、`user_prompt_settings`、`oauth_tokens`
- 非 `review-service` 服务直接写 `review_sessions`、`review_turns`
- 在业务 service 中直接拿全局 `db.DB` 写不属于本服务的表

## 4. 扫描结果与当前技术债

### 4.1 已确认的直接全局 `db.DB` 写点
- `backend/internal/service/decomposition_generate.go`
  - `skip` 章节标题/排序更新已收口到 `SeriesPersistence`
- `backend/internal/service/decomposition_generate_intro.go`
  - 导读成功/失败落库已收口到 `SeriesPersistence`
- `backend/internal/service/decomposition_generate_continue.go`
  - 续写正文读取与最终更新已收口到 `ContinuePersistence`
- `backend/internal/service/decomposition_generate_persistence.go`
  - 系列父稿创建、章节草稿预建、父稿来源更新、系列章节成功/失败落库均已收口到 `SeriesPersistence`
- `backend/internal/service/decomposition_generate_prompt_helpers.go`
  - 旧章节正文读取已收口到 `SeriesPersistence`

### 4.2 当前判断
- 以上写点都仍属于 `core-api` 自有业务边界，没有跨服务越权。
- `GeneratorService` 已完成显式 `GeneratedBlogPersistence` 收口；`DecompositionService` 也已通过 `SeriesPersistence / ContinuePersistence` 把系列前置草稿准备、导读、章节成功/失败、旧正文读取、`skip` 元信息以及 `continue` 正文读写全部从 service 主逻辑里抽离出来。
- 当前这条深拆主线的剩余技术债已不再是“业务逻辑里还有散落的直连写库”，而是“默认 GORM persistence 适配器后续是否要继续并入 `domain/blog` 或服务私有 repository”。
- 默认 `SeriesPersistence / ContinuePersistence / GeneratedBlogPersistence` 的缺省装配点现已统一收紧到 service 构造器；业务方法内的隐式 `nil -> GORM` fallback 已删除，后续迁移适配器归属时不必再逐个方法清理兜底逻辑。
- 当前生产装配已进一步下沉：`GeneratedBlogPersistence`、`ContinuePersistence` 与 `SeriesPersistence` 的默认 GORM 适配器均已由 `internal/domain/blog` 提供，并在 `llm-stream`、`core-api` 与 `cmd/server` 中通过 bootstrap 显式注入；service 层当前主要保留接口定义与测试替身。
- 当前中立契约也已开始抽离：`internal/domain/blog/contracts` 先承接共享错误与 persistence 输入/接口定义，`domain/blog` 已不再反向 import `internal/service`；service 层当前更多扮演兼容别名与构造器桥接层。
- `backend/internal/domain/stream/service.go` 现已直接依赖 `internal/domain/blog/contracts.Chapter`；当前非 `service` 包对 `GeneratedBlogPersistence / ContinuePersistence / SeriesPersistence / SeriesDraftPreflightInput / SeriesChapterPersistenceInput / Chapter` 等兼容别名的显式引用已清零，为后续评估删除 service 层桥接类型创造了条件。
- `backend/internal/service/generator_persistence.go`、`backend/internal/service/decomposition_continue_persistence.go` 与 `backend/internal/service/decomposition_series_persistence.go` 已删除；`GeneratorService` 与 `DecompositionService` 当前分别直接依赖 `blogcontracts.GeneratedBlogPersistence`、`blogcontracts.ContinuePersistence`、`blogcontracts.SeriesPersistence` 以及 `blogdomain` 默认适配器。同时 `Chapter` 本地兼容别名也已删除，service 包内部相关代码统一直接依赖 `blogcontracts.Chapter`；至此 blog contracts 在 service 层的兼容桥接已清零。

## 5. 收口优先级建议

### 第一优先级
- 评估是否把默认 `SeriesPersistence / ContinuePersistence / GeneratedBlogPersistence` GORM 适配器继续并入 `domain/blog` 或服务私有 repository，减少 service 层对 legacy model/ORM 的感知。
- Why: 当前 service 主逻辑的边界已经基本清晰，而且三类默认 blog 写入适配器与中立契约都已开始迁入 blog-domain；后续优化重点转向是否继续删除 service 侧兼容别名，并把更细粒度仓储能力也一起归并，而不是回头处理已清理的方法级 fallback。

### 第二优先级
- 为 `SeriesPersistence` 增加更细粒度的边界测试或仓储级测试，覆盖父稿存在/不存在、旧子稿清理、草稿预建失败回滚等事务场景。
- Why: 现在 preflight 逻辑已经被抽到显式接口，最有价值的下一步是巩固行为契约，而不是再重复做接口外壳。

### 第三优先级
- 保持 `task` 领域继续作为唯一允许的跨服务共享写入控制面，不新增第二套“谁都能写”的共享表模式。

## 6. 进入 Task 8 前的门槛

只有同时满足以下条件，才建议评估独立实例拆分：

1. `blogs / job_tasks / job_task_events` 的表归属已文档化并被团队接受
2. 核心写点已经从全局 `db.DB` 收口到显式 repository / service 接口
3. 迁移与回滚 Runbook 已存在且可演练
4. Docker Compose 冒烟检查已能覆盖核心任务创建、SSE 与基础读写链路

## 7. 最小检查清单

在每次涉及服务边界的改动后，至少复查以下项目：

- 是否新增了跨服务直接写业务表的代码
- 是否复用了 `internal/domain/task` 之外的共享写入捷径
- `README.md`、`.trae/documents/InkWords_Architecture.md`、`.trae/documents/InkWords_Database.md` 是否同步更新
- 是否需要补充到 `docs/runbooks/microservices-smoke-check.md`

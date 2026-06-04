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
- `backend/internal/service/generator.go`
  - 已收口为显式 `GeneratedBlogPersistence` 接口
  - 默认 GORM 适配器仍在事务内写 `blogs` 并更新 `users.tokens_used`
- `backend/internal/service/decomposition_generate.go`
  - 直接更新章节标题、排序、失败状态
- `backend/internal/service/decomposition_generate_intro.go`
  - 直接更新系列父博客导读内容与失败状态
- `backend/internal/service/decomposition_generate_continue.go`
  - 直接更新续写后的博客正文
- `backend/internal/service/decomposition_generate_persistence.go`
  - 直接创建系列父博客、章节草稿，并更新父博客来源

### 4.2 当前判断
- 以上写点都仍属于 `core-api` 自有业务边界，没有跨服务越权。
- `GeneratorService` 已完成第一步边界收口，但默认 persistence 适配器仍是 legacy GORM 实现；其余写点仍绕过 `domain/blog` 的显式仓储边界，属于共享数据库阶段最需要继续收口的技术债。
- 在这些写点没有被接口化之前，不适合推进 `blogs` 或 `job_tasks` 相关表的真正独立实例拆分。

## 5. 收口优先级建议

### 第一优先级
- 继续沿着 `GeneratorService -> GeneratedBlogPersistence` 模板，把默认 GORM 适配器替换为更贴近 `core-api` 归属边界的 persistence / repository 实现。
- Why: 单篇生成主链路已经接口化，下一步重点是让具体实现也逐步摆脱 legacy service 直连数据库的形态。

### 第二优先级
- 把 `DecompositionService` 对系列父博客、章节草稿、失败状态和续写正文的写入从全局 `db.DB` 收口到 `domain/blog` 或专用 persistence interface。
- Why: 当前系列链路写点分散，是后续拆分 `blogs` 相关边界的主要阻力。

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

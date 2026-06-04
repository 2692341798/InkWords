# 墨言知识训练平台 (InkWords Trainer) - 数据库设计文档

## 0. 变更记录
- 2026-06-04：新增 `FRONTEND_PORT` 仅影响 Docker Compose 的前端宿主机端口映射，不涉及 PostgreSQL 表结构、字段、索引、迁移或任何数据库写入语义变更；本次仅用于在宿主机 `:80` 被占用时，将前端入口临时切到如 `http://localhost:8088` 完成运行验证。
- 2026-06-04：Generation Task-Only Task 4 为 `generate_series` 任务增强 `job_tasks.result_json` 语义并打通最终业务写入。系列成功结果现在保存 `parent_blog` 与 `chapters[]`：父博客包含系列标题与导读正文，章节包含 `blog_id / chapter_sort / title / content / word_count / tech_stacks / status / error_message`；`core-api` 会基于该结果事务性更新 `blogs` 中的父子记录，并继续使用 `usage.estimated_tokens` 累计 `users.tokens_used`。本次不新增表、字段、索引或迁移。
- 2026-06-04：Generation Task-Only Task 3 为 `continue` 任务增强 `job_tasks.result_json` 语义并打通最终业务写入。续写成功结果现在保存 `blog_id / appended_content / final_content`，`core-api` 会基于 `payload.final_content` 更新 `blogs.content`，并继续使用 `usage.estimated_tokens` 累计 `users.tokens_used`。本次不新增表、字段、索引或迁移。
- 2026-06-04：Generation Task-Only Task 2 让 `core-api` 开始消费单篇 `generate_single` 的结构化 `job_tasks.result_json` 并完成最终业务写入：当任务成功时，会把 `payload.title/content/source_type/word_count/tech_stacks` 写回 `blogs`，并基于 `usage.estimated_tokens` 累计 `users.tokens_used`。本次不新增表、字段、索引或迁移，但 `job_tasks.result_json -> blogs/users` 的单篇持久化闭环已从设计进入实现。
- 2026-06-04：Generation Task-Only Task 1 为 `generate_single` 任务增强 `job_tasks.result_json` 语义。在 `INKWORDS_TASK_PERSISTENCE_MODE=task_only` 下，单篇生成成功结果不再固定为 `{"done":true}`，而是保存结构化 JSON：外层含 `result_version / task_type / task_subtype / persistence_mode / final_status / usage`，`payload` 含单篇标题、正文、来源类型、字数与技术栈；本次不新增表、字段、索引或迁移。
- 2026-06-04：Task 3 将 `GeneratorService` 的最终落库改为通过显式 `GeneratedBlogPersistence` 接口完成，默认 GORM 适配器仍在同一事务中创建 `blogs` 记录并累计 `users.tokens_used`。本次不新增表、字段、索引或迁移，但数据库写入边界从“service 直接拿全局 `db.DB`”收紧为“service 产出业务事实，再交给 persistence 适配器落库”。
- 2026-06-04：Phase 2 执行 `core-api / llm-stream` 深层拆分第一轮。数据库表结构、索引与迁移保持不变，但服务写入边界进一步收紧：`core-api` 新增 `ResultPersister` 抽象，作为后续把任务结果落回 `blogs` 与累计 `users.tokens_used` 的服务自有承载点；同时新增 `INKWORDS_TASK_PERSISTENCE_MODE=task_only` 运行开关，让 legacy `generator / decomposition` 在显式开启时停止直接写 `blogs / users`，为最终把业务事实回收至 `core-api` 做过渡。
- 2026-06-04：Task 4 将 `export-service` 的导出适配器、consumer、artifact store 与启动装配迁入 `backend/services/export-service/` 服务自有目录；本次不新增数据库表、字段、索引或迁移，`job_tasks / job_task_events` 的跨服务受控写入边界保持不变。
- 2026-06-03：Task 4 补齐数据库层面的“服务写入归属矩阵”。明确 `core-api` 事实拥有 `users / oauth_tokens / user_prompt_settings / blogs / job_tasks / job_task_events`，`review-service` 事实拥有 `review_sessions / review_turns`；同时记录当前允许的跨服务例外仅限 worker 通过 `task` 领域接口写 `job_tasks / job_task_events`。其中 `GeneratorService` 已进一步收口到显式 `GeneratedBlogPersistence` 接口，`DecompositionService` 仍是待收口技术债。
- 2026-06-03：生成链路进入 RabbitMQ 任务式 SSE Phase B。核心库新增 `job_tasks` 与 `job_task_events` 两张表，分别存储任务主状态与可回放事件流；RabbitMQ 仅负责跨服务投递，不承载最终状态，任务真实状态仍以 PostgreSQL 为准。
- 2026-06-03：Docker 微服务化 Phase 2（已落地到代码与编排）。同一 Postgres 实例新增 review 独立 database：`inkwords_review_db`；`review-service` 使用 `REVIEW_DATABASE_URL` 连接；review 数据迁移与回滚按 Runbook 执行：[review-db-migration.md](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/docs/runbooks/review-db-migration.md)。本次不新增/不修改任何表字段表格（仅补充说明与引用）。
- 2026-06-03：稳定性与工程化优化（Task 1-5）。本次不新增数据库表、字段或索引；主要变更为：系列生成链路补齐“前置草稿创建/清理 + 章节完成落库 + `users.tokens_used` 累加”的事务边界与可观测错误，避免章节正文写入成功但 Token 记账静默失败的状态不一致。
- 2026-06-02：系列生成失败原因可视化与 SSE 稳定性修复仅调整前端状态管理和后端流式写出策略，不涉及 PostgreSQL 表结构、字段、索引、迁移或写入时机变更；`blogs`、`review_sessions`、`review_turns` 等既有表保持不变。
- 2026-06-01：知识漫游复习升级为“文章驱动提问 + 结构化命中/遗漏反馈”；本次复习增强仅复用 `review_sessions.metadata_snapshot` 与 `key_points_snapshot` 存储会话快照，不新增数据库表、字段、索引或迁移脚本。
- 2026-06-01：知识漫游复习推荐卡新增 `提问开始` 入口，并修复恢复会话/首页继续任务判定；本次仅调整前端状态与展示逻辑，不涉及数据库表结构、字段、索引、迁移或写入时机变更。
- 2026-05-29：工程化结构拆分 Phase 1：仅进行 review 领域服务、Sidebar 和生成链路辅助逻辑的代码拆分与测试补强，不涉及数据库表结构、字段、索引、迁移或写入时机变更。
- 2026-05-29：知识漫游复习入口从“三入口”收敛为“随机抽题 + 手动选文”双入口；仅影响前端入口与 review 随机选题逻辑，不涉及数据库表结构变更，`review_sessions.entry_type` 仍保留 `today / manual_random / manual_select` 以兼容历史记录与既有服务端枚举。
- 2026-05-29：生成器前端改为“三步 + 步内进度”工作流，文件上传链路拆分为“先 parse、后 analyze”；本次仅调整前端状态机、交互时机与进度展示，不涉及数据库表结构、字段、索引、迁移或写入时机变更。
- 2026-05-28：工程规范收尾与提交前同步：未新增任何数据库表、字段、索引或迁移，但“流式生成完成后落库”链路调整为事务化写入，要求 `blogs` 新纪录创建与 `users.tokens_used` 累加在同一事务内成功或同时回滚，避免只写入博客正文却漏记 Token 消耗的状态不一致问题。Docker Compose 部署基线同步调整为仅前端默认暴露宿主机端口；PostgreSQL 与 Redis 默认仅在 `inkwords-network` 内部网络中互通。
- 2026-05-27：项目定位升级为“墨言知识训练平台（InkWords Trainer）”，口号“把资料变成知识，把知识变成能力”；本次仅同步文档命名口径，不涉及表结构、索引或迁移变更。
- 2026-05-27：前端新增流程型入口页 `HomeEntry`、共享 `StepStrip` 步骤条，并将生成器/知识复习改为“当前步骤单屏聚焦”交互；本次不涉及数据库表结构、索引、迁移或写入时机变更。
- 2026-05-27：为“知识漫游复习”新增 `review_sessions`、`review_turns` 两张表，并接入后端启动时的 GORM `AutoMigrate`，用于持久化复习会话主记录与轮次记录。
- 2026-05-25：修复 AI 思考/对话式前言混入正文的问题；通过后端流式正文清洗和前端正文应用前兜底清洗，确保写入 `blogs.content` 的 Markdown 不再包含 `<think>` 标签或“收到你的需求 / 作为高级全栈架构师”等开头套话（无表结构变更）。
- 2026-05-25：将本地文档上传上限提升至 888MB；仅调整网关、前端与应用层上传阈值，不涉及数据库表结构变更。
- 2026-05-25：修复前端 `scenario_mode` 的场景锁定与展示一致性问题；仅调整前端状态管理与交互约束，不涉及数据库表结构、字段或写入时机变更。
- 2026-05-25：修复系列生成失败时 `blogs` 父子结构丢失的问题；不新增字段，但调整 `blogs` 表写入时机，系列章节会在正式流式生成前先插入 `parent_id = 系列父ID` 的草稿记录，生成成功后更新为 `status = 1` 与正式正文，失败时保留记录并标记错误状态，确保历史树可以稳定返回 `children`。
- 2026-05-21：新增 ZIP 课件包解析能力（仅文件解析与前端展示调整，无数据库表结构变更）。
- 2026-05-21：新增用户写作要求模板覆盖表 `user_prompt_settings`，用于持久化不同文章类型的自定义写作要求。
- 2026-05-21：修复本地文档上传后流式分析的来源判定兼容性问题（仅前后端请求/判定逻辑调整，无数据库表结构变更）。
- 2026-04-29：代码模块化整理（不涉及数据库表结构变更）。
- 2026-04-29：新增“手写草稿”能力（复用 `blogs` 表，使用 `source_type=manual`，不新增表结构）。
- 2026-05-08：写博客编辑器新增“语音输入”（纯前端能力，不涉及数据库表结构变更）。
- 2026-05-08：写博客编辑器新增“润色”（SSE 生成润色草稿，不新增表结构、不落库）。
- 2026-05-08：工程化整理（无数据库表结构变更，主要为仓库文件治理与文档同步）。
- 2026-05-08：目录结构工程化调整落地（无数据库表结构变更）。
- 2026-05-08：后端 Blog Domain 垂直切片迁移（无数据库表结构变更）。
- 2026-05-08：后端 User Domain 垂直切片迁移（无数据库表结构变更）。
- 2026-05-08：后端 Auth Domain 垂直切片迁移（无数据库表结构变更）。
- 2026-05-08：后端 Stream/Project Domain 垂直切片迁移（无数据库表结构变更）。
- 2026-05-10：修复导出到 Obsidian 初始化 scaffold 时的目录列表解析失败（无数据库表结构变更）。
## 1. 数据库选型
- **类型**: 关系型数据库 (RDBMS)
- **引擎**: PostgreSQL 14+
- **ORM**: GORM (Go)
- **连接字符串**: `postgres://inkwords:inkwords_password@db:5432/inkwords_db?sslmode=disable`
- **挂载卷**: Docker volume `pgdata` 持久化至 `/var/lib/postgresql/data`。

### 1.2 服务写入归属矩阵（Task 4）
| 表 / 资源 | 事实归属服务 | 当前允许写入方 | 当前状态 |
| --- | --- | --- | --- |
| `users` | `core-api` | `core-api` | 已归属；登录、注册、Token 记账都在 core 侧 |
| `oauth_tokens` | `core-api` | `core-api` | 已归属；模型存在，业务写入链路待后续恢复时继续沿用 core 归属 |
| `user_prompt_settings` | `core-api` | `core-api` | 已归属；用户模板覆盖值不应被其它服务写入 |
| `blogs` | `core-api` | `core-api` | 已归属；但仍有 `internal/service` 直写全局 `db.DB` 的技术债 |
| `job_tasks` | `core-api` | `core-api`、`llm-stream`、`parser-service`、`export-service` | 允许受控跨服务写；仅限任务状态/结果 |
| `job_task_events` | `core-api` | `core-api`、`llm-stream`、`parser-service`、`export-service` | 允许受控跨服务写；仅限可回放事件追加 |
| `review_sessions` | `review-service` | `review-service` | 已拆到 `inkwords_review_db` |
| `review_turns` | `review-service` | `review-service` | 已拆到 `inkwords_review_db` |

### 1.3 当前允许的跨服务写入例外
- `llm-stream`、`parser-service`、`export-service` 允许写 `job_tasks / job_task_events`，原因是统一任务中心目前承担跨服务控制面职责。
- 上述例外的前提是：只能通过 `internal/domain/task` 的显式仓储接口写入，禁止在其它 service 中直接拿全局 `db.DB` 写任务表。
- 除任务控制面外，当前不存在“非归属服务直接写业务表”的允许清单；`blogs`、`users`、`review_*` 仍按服务归属严格约束。

### 1.4 Phase 2：review 拆库（同实例不同 database）
- **core db**：`inkwords_db`（博客、用户、导出相关结构化数据等）
- **review db**：`inkwords_review_db`（仅 review 相关数据）
- **迁移与回滚**：按 Runbook 执行：[review-db-migration.md](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/docs/runbooks/review-db-migration.md)

## 2. 表结构化设计

### 2.1 用户表 (`users`)
存储用户的基本信息与第三方授权状态。

| 字段名 | 数据类型 | 约束/索引 | 描述 |
| ------ | -------- | --------- | ---- |
| `id` | UUID | Primary Key | 用户唯一标识 |
| `created_at` | TIMESTAMP | | 创建时间 |
| `updated_at` | TIMESTAMP | | 更新时间 |
| `deleted_at` | TIMESTAMP | Index | 软删除标识 |
| `email` | VARCHAR(255) | Unique | 用户邮箱 (普通注册) |
| `password` | VARCHAR(255) | | Bcrypt 哈希密码 |
| `github_id` | VARCHAR(255) | Unique | GitHub OAuth ID |
| `wechat_openid` | VARCHAR(255) | Unique | 第三方授权微信OpenID |
| `avatar_url` | VARCHAR(255) | | 头像地址 |
| `name` | VARCHAR(255) | | 用户昵称/显示名 |
| `subscription_tier` | SMALLINT | Default 0 | 订阅等级 (0=Free, 1=Pro) |
| `tokens_used` | INTEGER | Default 0 | 当前已消耗的 Token 数量 |
| `token_limit` | INTEGER | Default 1000000000 | 用户的最大 Token 额度 |
| `failed_login_attempts` | INTEGER | Default 0 | 连续登录失败次数 |
| `locked_until` | TIMESTAMP | Nullable | 账号锁定到期时间 |

### 2.2 博客表 (`blogs`)
存储通过解析文档或 Git 仓库生成的博客数据。支持树形结构（父节点 - 子章节）。

| 字段名 | 数据类型 | 约束/索引 | 描述 |
| ------ | -------- | --------- | ---- |
| `id` | UUID | Primary Key | 博客唯一标识 |
| `created_at` | TIMESTAMP | | 创建时间 |
| `updated_at` | TIMESTAMP | | 更新时间 |
| `deleted_at` | TIMESTAMP | Index | 软删除标识 |
| `user_id` | UUID | Foreign Key | 关联 `users.id` |
| `parent_id` | UUID | Foreign Key, Nullable | 关联 `blogs.id`，指向系列父节点 |
| `title` | VARCHAR(255) | | 博客/章节标题 |
| `content` | TEXT | | Markdown 格式正文内容 |
| `source_type` | VARCHAR(50) | | 来源类型 (`file`, `git`) |
| `source_url` | VARCHAR(512)| | 来源URL (`git` 类型对应的仓库地址) |
| `status` | INTEGER | Default 0 | 状态 (0:生成中, 1:已完成, -1:失败) |
| `word_count` | INTEGER | Default 0 | 生成博客的总字数统计 |
| `tech_stacks` | JSONB | Nullable | 自动提取的涉及技术栈列表 |
| `chapter_sort`| INTEGER | Default 1 | 在系列博客中的排序序号 |

### 2.3 生成任务表 (`job_tasks`)
存储任务式生成链路中的主任务状态，作为跨 `core-api` 与 `llm-stream` 的事实来源。

| 字段名 | 数据类型 | 约束/索引 | 描述 |
| ------ | -------- | --------- | ---- |
| `id` | UUID | Primary Key | 任务唯一标识 |
| `task_type` | VARCHAR(32) | Index | 任务大类，第一阶段固定为生成类任务 |
| `task_subtype` | VARCHAR(64) | Index | 任务子类，如 `generate_single` / `generate_series` |
| `status` | VARCHAR(16) | Index | 任务状态：`pending / queued / running / streaming / succeeded / failed / cancelled` |
| `requested_by` | UUID | Index | 发起任务的用户 ID |
| `idempotency_key` | VARCHAR(255) | Index | 幂等键；用于复用同一用户的重复提交 |
| `payload_json` | JSONB | Not Null | 原始任务载荷 |
| `result_json` | JSONB | Nullable | 任务成功后的结果摘要 |
| `error_message` | TEXT | Nullable | 任务失败或取消原因 |
| `retry_count` | INTEGER | Default 0 | 当前重试次数 |
| `started_at` | TIMESTAMP | Nullable | 开始执行时间 |
| `finished_at` | TIMESTAMP | Nullable | 结束时间 |
| `created_at` | TIMESTAMP | | 创建时间 |
| `updated_at` | TIMESTAMP | | 更新时间 |

### 2.4 生成任务事件表 (`job_task_events`)
存储可回放的任务事件流，供 `core-api` 轮询并向前端输出 SSE。

| 字段名 | 数据类型 | 约束/索引 | 描述 |
| ------ | -------- | --------- | ---- |
| `id` | BIGSERIAL | Primary Key | 自增事件 ID，用于游标式拉取 |
| `task_id` | UUID | Index | 关联 `job_tasks.id` |
| `event_type` | VARCHAR(32) | Index | 事件类型，如 `chunk / error / done` |
| `status` | VARCHAR(16) | Index | 事件产生时对应的任务状态 |
| `payload` | JSONB | Not Null | 事件载荷，直接作为 SSE 数据源 |
| `created_at` | TIMESTAMP | | 创建时间 |

### 2.5 用户写作模板表 (`user_prompt_settings`)
存储用户针对不同文章类型（如通用技术博客、小白手把手、备考复习）配置的“写作要求”覆盖值。

| 字段名 | 数据类型 | 约束/索引 | 描述 |
| ------ | -------- | --------- | ---- |
| `user_id` | UUID | Primary Key, Foreign Key | 关联 `users.id` |
| `overrides` | JSONB | Not Null, Default `{}` | 用户按文章类型覆盖的写作要求映射 |
| `created_at` | TIMESTAMP | | 创建时间 |
| `updated_at` | TIMESTAMP | | 更新时间 |

### 2.6 复习会话表 (`review_sessions`)
存储一次“知识漫游复习”训练的主记录与最终反馈快照。

| 字段名 | 数据类型 | 约束/索引 | 描述 |
| ------ | -------- | --------- | ---- |
| `id` | UUID | Primary Key | 复习会话唯一标识 |
| `user_id` | UUID | Index | 关联 `users.id` |
| `note_path` | TEXT | Not Null, Index | 对应 Obsidian `concept` 笔记路径 |
| `note_title` | VARCHAR(255) | Not Null | 复习笔记标题 |
| `source_title` | VARCHAR(255) | Nullable | 所属来源或系列标题 |
| `entry_type` | VARCHAR(32) | Not Null | 进入方式 (`today` / `manual_random` / `manual_select`) |
| `mode` | VARCHAR(32) | Not Null | 训练模式 (`light_recall` / `detailed_qa`) |
| `status` | VARCHAR(32) | Not Null, Index | 会话状态 (`created` / `in_progress` / `completed` / `abandoned`) |
| `review_reason` | TEXT | Nullable | 推荐理由 |
| `estimated_minutes` | INTEGER | Default 0 | 预估训练耗时（分钟） |
| `content_digest` | TEXT | Nullable | 训练快照摘要或正文摘要 |
| `summary_snapshot` | TEXT | Nullable | 会话创建时保存的摘要快照 |
| `key_points_snapshot` | JSONB | Not Null, Default `[]` | 关键点快照 |
| `metadata_snapshot` | JSONB | Not Null, Default `{}` | 其他结构化元数据 |
| `hint_used_count` | INTEGER | Not Null, Default 0 | 已使用提示次数 |
| `max_hint_count` | INTEGER | Not Null, Default 2 | 最多可用提示次数 |
| `turn_count` | INTEGER | Not Null, Default 0 | 已产生轮次数 |
| `final_summary` | TEXT | Nullable | 最终复习总结 |
| `strengths` | JSONB | Not Null, Default `[]` | 已掌握点 |
| `gaps` | JSONB | Not Null, Default `[]` | 薄弱点 |
| `next_focus` | JSONB | Not Null, Default `[]` | 下次优先补强点 |
| `feedback_tags` | JSONB | Not Null, Default `[]` | 结构化反馈标签 |
| `started_at` | TIMESTAMP | Not Null | 会话开始时间 |
| `completed_at` | TIMESTAMP | Nullable | 完成时间 |
| `abandoned_at` | TIMESTAMP | Nullable | 放弃时间 |
| `created_at` | TIMESTAMP | | 创建时间 |
| `updated_at` | TIMESTAMP | | 更新时间 |
| `deleted_at` | TIMESTAMP | Index | 软删除标识 |

### 2.7 复习轮次表 (`review_turns`)
存储一次复习训练中的系统提问、提示、阶段反馈与用户回答。

| 字段名 | 数据类型 | 约束/索引 | 描述 |
| ------ | -------- | --------- | ---- |
| `id` | UUID | Primary Key | 轮次唯一标识 |
| `session_id` | UUID | Not Null, Unique Composite Index | 关联 `review_sessions.id` |
| `turn_index` | INTEGER | Not Null, Unique Composite Index | 在一次 session 内的轮次序号 |
| `role` | VARCHAR(16) | Not Null | 发言角色 (`system` / `user`) |
| `turn_type` | VARCHAR(32) | Not Null | 轮次类型（开场、提问、回答、提示、反馈等） |
| `content` | TEXT | Not Null | 轮次正文 |
| `evaluation_tags` | JSONB | Not Null, Default `[]` | 该轮结构化评价标签 |
| `extra_payload` | JSONB | Not Null, Default `{}` | 该轮附加结构化数据 |
| `created_at` | TIMESTAMP | | 创建时间 |
| `updated_at` | TIMESTAMP | | 更新时间 |

## 3. 关联关系 (Associations)
- **User (1) <-> (N) Blog**: 一个用户可以拥有多篇博客历史记录。
- **Blog (1) <-> (N) Blog**: 自引用（Self-Referencing）。一个父级 Blog（代表系列入口，例如 "Hydrogen语言系列"）可以拥有多个子级 Blog（代表具体章节内容，例如 "第 1 篇：架构概览"）。通过 `parent_id` 建立一对多父子关系。
- **User (1) <-> (N) JobTask**: 一个用户可以发起多个任务式生成请求；通过 `job_tasks.requested_by` 关联用户维度的任务历史。
- **JobTask (1) <-> (N) JobTaskEvent**: 一个生成任务会产生多条可回放事件；通过 `job_task_events.task_id` 与自增 `id` 形成顺序事件流。
- **User (1) <-> (N) ReviewSession**: 一个用户可以发起多次“知识漫游复习”训练；通过 `review_sessions.user_id` 关联用户维度的复习历史。
- **ReviewSession (1) <-> (N) ReviewTurn**: 一次复习会话包含多轮系统追问、提示与用户回答；通过 `review_turns.session_id` + `turn_index` 保证同一 session 内轮次有序且唯一。

## 4. 迁移策略 (Migration)
- 系统在启动时，会通过 GORM 的 `AutoMigrate` 功能自动根据 Go 模型结构 (`internal/model`) 同步创建或更新数据库表。
- `job_tasks` 与 `job_task_events` 已纳入 core db 的统一迁移列表；当前阶段不依赖单独迁移脚本，随 `core-api` / `llm-stream` 启动自动对齐。
- `review_sessions` 与 `review_turns` 已纳入 `backend/internal/infra/db/db.go` 的统一迁移列表，避免主程序启动与测试路径使用不同的迁移集合。
- 敏感数据如密码，在存入数据库之前必须经过 `golang.org/x/crypto/bcrypt` 进行哈希加密。

### 4.1 当前待收口写点（Task 4 扫描结论）
- `backend/internal/service/generator.go`：已不再直接操作全局 `db.DB`，而是通过 `GeneratedBlogPersistence` 显式接口提交 `blogs` 与 `users.tokens_used` 的持久化请求；默认 GORM 适配器位于 `backend/internal/service/generator_persistence.go`。
- `backend/internal/service/decomposition_generate.go`、`decomposition_generate_intro.go`、`decomposition_generate_continue.go`、`decomposition_generate_persistence.go`：仍直接通过全局 `db.DB` 写系列父博客、章节草稿、续写正文和失败状态。
- `GeneratorService` 已完成第一步边界收口；其余写点虽然没有跨服务越权，但仍绕过了 `domain/blog` 的仓储边界；在推进独立实例拆分前，需要继续收口到显式 repository / service 接口。

## 5. 外部数据持久化 (External Persistence)
- **Obsidian 本地知识库导出**: 除关系型数据库外，系统支持将 `blogs` 表中的结构化数据导出为纯文本的 Markdown 文件，并在文件头部自动生成兼容 Karpathy LLM Wiki Pattern 的 YAML Frontmatter。导出支持两种形态：
  - **单篇导出**：写入一篇带 `type: concept` 的笔记。
  - **系列批量 Ingest**：系列父节点写入 `wiki/sources/`，子章节写入 `wiki/concepts/`，并通过大模型抽取关键实体写入 `wiki/entities/`，自动编织双向链接网络。同时会初始化 `.raw/` 目录，并生成 `sources/_index.md`、`concepts/_index.md`、`entities/_index.md`、`domains/_index.md` 等索引页，避免 Obsidian 地图出现空入口；并更新 `wiki/index.md`、`wiki/log.md` 与 `wiki/hot.md`。
  导出写入通过 Obsidian Local REST API（HTTPS + API Key）完成；容器通过 sidecar `obsidian-bridge`（27125）转发访问宿主机插件端口（27124），从而实现与用户本地个人知识管理（PKM）系统的直通与同步。

- **系列合并 PDF 导出**：PDF 导出不新增数据库表，仅复用 `blogs` 表的系列父子结构数据；后端会将系列内容渲染为 HTML 并通过容器内 Chromium 生成 PDF 后直接以文件附件形式返回给前端下载。

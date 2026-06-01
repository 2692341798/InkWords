# 墨言知识训练平台 (InkWords Trainer) - 数据库设计文档

## 0. 变更记录
- 2026-06-01：系列章节质量流水线继续落地 Task 5。本次仅扩展前端 `streamStore`、SSE 事件消费与进度卡展示，让章节质量阶段和 DeepSeek Prompt Cache 命中摘要对用户可见；不新增表、字段、索引、迁移脚本，也不改变任何落库时机。
- 2026-06-01：系列章节质量流水线继续落地 Task 4。本次新增 DeepSeek usage 与 Prompt Cache 命中 telemetry，但全部属于运行时流式事件与内存态统计；不新增表、字段、索引、迁移脚本，也不改变 `blogs` / `users` 的持久化结构。章节 usage 只通过 SSE 透传给前端观测，不落库。
- 2026-06-01：系列章节质量流水线继续落地 Task 3。系列章节主链路已切换为“理解 -> 草稿 -> 审稿 -> 终稿补强后再落库”，但仍复用既有 `blogs` 表字段；本次不新增表、字段、索引或迁移，仅将章节正文的最终写入时机收敛为“只持久化终稿补强结果”，避免草稿/审稿中间态落库。
- 2026-06-01：系列章节质量流水线继续落地 Task 2。本次新增稳定前缀 builder、章节理解阶段解析器与读者画像 helper，全部属于后端运行时 Prompt 组装与内存态结构化校验能力；不涉及数据库表结构、字段、索引、迁移脚本或写入时机变更。
- 2026-06-01：系列章节质量流水线开始落地 Task 1。本次仅在后端内存态新增章节质量结构体与门禁校验函数，用于拦截不完整的章节理解/草稿/审稿结果；不涉及数据库表结构、字段、索引、迁移脚本或写入时机变更。
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

### 2.3 用户写作模板表 (`user_prompt_settings`)
存储用户针对不同文章类型（如通用技术博客、小白手把手、备考复习）配置的“写作要求”覆盖值。

| 字段名 | 数据类型 | 约束/索引 | 描述 |
| ------ | -------- | --------- | ---- |
| `user_id` | UUID | Primary Key, Foreign Key | 关联 `users.id` |
| `overrides` | JSONB | Not Null, Default `{}` | 用户按文章类型覆盖的写作要求映射 |
| `created_at` | TIMESTAMP | | 创建时间 |
| `updated_at` | TIMESTAMP | | 更新时间 |

### 2.4 复习会话表 (`review_sessions`)
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

### 2.5 复习轮次表 (`review_turns`)
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
- **User (1) <-> (N) ReviewSession**: 一个用户可以发起多次“知识漫游复习”训练；通过 `review_sessions.user_id` 关联用户维度的复习历史。
- **ReviewSession (1) <-> (N) ReviewTurn**: 一次复习会话包含多轮系统追问、提示与用户回答；通过 `review_turns.session_id` + `turn_index` 保证同一 session 内轮次有序且唯一。

## 4. 迁移策略 (Migration)
- 系统在启动时，会通过 GORM 的 `AutoMigrate` 功能自动根据 Go 模型结构 (`internal/model`) 同步创建或更新数据库表。
- `review_sessions` 与 `review_turns` 已纳入 `backend/internal/infra/db/db.go` 的统一迁移列表，避免主程序启动与测试路径使用不同的迁移集合。
- 敏感数据如密码，在存入数据库之前必须经过 `golang.org/x/crypto/bcrypt` 进行哈希加密。

## 5. 外部数据持久化 (External Persistence)
- **Obsidian 本地知识库导出**: 除关系型数据库外，系统支持将 `blogs` 表中的结构化数据导出为纯文本的 Markdown 文件，并在文件头部自动生成兼容 Karpathy LLM Wiki Pattern 的 YAML Frontmatter。导出支持两种形态：
  - **单篇导出**：写入一篇带 `type: concept` 的笔记。
  - **系列批量 Ingest**：系列父节点写入 `wiki/sources/`，子章节写入 `wiki/concepts/`，并通过大模型抽取关键实体写入 `wiki/entities/`，自动编织双向链接网络。同时会初始化 `.raw/` 目录，并生成 `sources/_index.md`、`concepts/_index.md`、`entities/_index.md`、`domains/_index.md` 等索引页，避免 Obsidian 地图出现空入口；并更新 `wiki/index.md`、`wiki/log.md` 与 `wiki/hot.md`。
  导出写入通过 Obsidian Local REST API（HTTPS + API Key）完成；容器通过 sidecar `obsidian-bridge`（27125）转发访问宿主机插件端口（27124），从而实现与用户本地个人知识管理（PKM）系统的直通与同步。

- **系列合并 PDF 导出**：PDF 导出不新增数据库表，仅复用 `blogs` 表的系列父子结构数据；后端会将系列内容渲染为 HTML 并通过容器内 Chromium 生成 PDF 后直接以文件附件形式返回给前端下载。

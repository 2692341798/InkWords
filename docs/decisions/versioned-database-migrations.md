# ADR：版本化数据库迁移与本地工作区边界

- 状态：接受；PR-02 实现中
- 日期：2026-09-02
- 决策人：InkWords 本地单用户产品
- 关联：`.trae/documents/InkWords_PRD.md`、`docs/superpowers/specs/2026-09-02-textbook-platform-development-design.md`、`docs/superpowers/plans/2026-09-02-textbook-platform-implementation.md` PR-02

## 背景

当前运行时由 `backend/internal/infra/db/db.go` 的 GORM `AutoMigrate` 创建 core 与 review 表；`backend/shared/platform/postgres/core.go` 将它包装为 `InitCore` / `InitReview`。这使既有本地功能可以快速启动，但无法审计新教材数据表的精确 DDL、升级顺序和回滚边界。

运行入口还支持两种拓扑：聚合 `backend/cmd/server/main.go` 可以把 review 与 core 指向同一 DSN，也可以由 `REVIEW_DATABASE_URL` 指向独立数据库。迁移方案必须同时支持这两种情况，且不能将 core 表误建到独立 review 数据库，也不能将 review 表误建到仅运行 core 的数据库。

产品 V1 是单用户、本地浏览器应用，但不是“信任客户端传来的 owner”。教材新表需要稳定的 `workspace_id`；旧 Blog、Task、Review 仍带 `user_id`，必须在过渡期保持可读可写。

## 决策

### 1. 新教材 schema 使用嵌入式、版本化 SQL migration

后续依赖 PR 固定引入 `github.com/pressly/goose/v3 v3.27.2`（MIT），使用随二进制打包的 SQL 文件，而不是在生产启动时由 GORM 推断新表结构。该版本最低要求 Go 1.25.7，兼容本仓库当前的 Go 1.25.13。

- migration 文件只追加，使用单调递增版本号，例如 `00001_local_workspace.sql`、`00002_textbook_core.sql`。
- 每个 migration 必须声明目标数据库角色：`core`、`review` 或 `shared`。启动器仅对与当前 DSN 角色匹配的 migration 集合执行 Up。
- 若 core 与 review 使用同一 DSN，启动器在同一进程中按确定顺序执行对应集合，并以独立的 migration tracking table 记录各角色版本。
- 若 core 与 review 使用不同 DSN，核心启动器只执行 core/shared 集合；review 启动器只执行 review/shared 集合。不得为了复用代码而对第二个 DSN 执行全部迁移。
- migration runner 只在服务启动时执行一次；worker 处理任务、HTTP 请求和导出操作不得触发迁移。
- `embed.go` 仅暴露迁移资源和受测试的 runner 边界；业务包不直接依赖 goose 的全局配置。若所选 API 需要全局文件系统设置，必须在启动临界区串行、恢复状态，并由测试证明 core/review 不相互污染；优先采用可隔离的 provider 实例。

初始引入不重写既有表的历史。过渡期启动顺序为“打开连接 → legacy `AutoMigrate` 创建/维护已有表 → goose 执行新教材 migration → 构建路由”；这让首份映射 migration 能以外键引用已有 `users(id)`，同时不让 GORM 推断新教材表。production 日志必须输出一次明确迁移期警告。待旧表具备完整的版本化基线、升级和恢复演练后，再另行 PR 移除 production `AutoMigrate`。

PR-14 的 `00001_mastery_review.sql` 是首个 review 角色 migration：它创建 workspace-owned 的 `mastery_objectives`、append-only 的 `mastery_attempts` 和可重建的 `mastery_schedules`。`InitReview` 执行 `UpReview`；`NewGormStore` 不再调用 `AutoMigrate`。core 与 review provider 分别加载嵌入资源中的对应角色文件，使用 `inkwords_core_schema_migrations` / `inkwords_review_schema_migrations` 及不同 PostgreSQL advisory lock，避免共库拓扑把 review 表写入 core migration 历史。

PR-12 追加 core 迁移 `00010_textbook_runtime_evidence.sql`：`textbook_code_artifacts`、`textbook_runtime_evidence` 与 `textbook_manuscript_assets` 分别保存可执行教学工件、一次运行/观察的原始证据，以及母稿稳定引用的视觉资产。`verified` 证据在数据库约束层必须带命令清单、运行器镜像、工具链、原始证据引用和采集时间；任何已有运行证据或资产都会使 Down 明确拒绝，要求从备份恢复。

PR-16 的 `00034_textbook_task_chapter_identity.sql` 将样章任务的章节身份从不可变
`payload_json` 投影到受外键与 CHECK 约束的 `job_tasks.textbook_chapter_id`，供章节工作区
确定性恢复最新任务。局部索引只覆盖 `textbook_sample_generate`，目标查询按
`workspace_id, textbook_chapter_id, created_at DESC, id DESC` 读取；它缩小恢复读取范围，
代价是每个样章任务增加一个列写入和一个索引项。该列不成为任务输入的新事实来源；Down
删除投影和索引时不会删除不可变载荷、任务结果或候选稿，但会失去浏览器状态丢失后的按章
发现能力。

### 2. `local_workspaces` 是新教材数据的稳定身份根

`00001_local_workspace.sql` 将创建以下最小 core 结构：

- `local_workspaces`：稳定 UUID 主键、不可由 HTTP 参数选择的本地工作区名称、创建时间；
- `local_workspace_legacy_owner`：唯一 workspace 到旧 `users.id` 的兼容映射，以便旧 Blog/Task/Review 继续通过既有 owner 查询；
- 满足映射读取的唯一约束和索引；新增索引前在 PostgreSQL 中保存目标查询和 `EXPLAIN` 证据，并记录写入/存储成本。

建表 migration **不会**删除、重写或猜测任何 `users`、Blog、Task、Review 数据。首次教材路由请求在事务内执行“读取或创建唯一 workspace + 读取或创建 legacy owner 映射”；如果现有数据库没有能安全映射的 owner，则该教材请求返回可操作诊断，不能静默创建会令旧数据不可见的假身份，也不得阻断既有路由。

`LocalWorkspaceContext` middleware 从受控的本地配置/持久化记录取得这个唯一 `workspace_id` 并写入 Gin context。它不接受查询参数、Header、Cookie 或 body 中的 workspace/owner 值。PR-02 只把它应用到新增教材路由；旧路由继续使用当前认证桥梁，避免一次性删掉 JWT、OAuth 或 `UserID`。

### 3. 迁移、回滚与恢复按状态机执行

```text
备份已验证 → migration preflight → Up 已记录 → 服务健康检查
                         │                 │
                         └──失败/不一致─────┴→ 停止启动 + 保留原数据 + 恢复/人工处置
```

每次本地升级执行前：

1. 停止写入路径或关闭应用；对目标 PostgreSQL 数据卷执行可恢复备份，并记录版本与时间。
2. runner 检查已应用版本、目标角色、校验和和 pending migration；发现版本漂移、缺失历史或校验和不一致即失败，不尝试修复。
3. 在事务能力允许的 DDL 范围内执行 Up，并把版本记录与 schema 变更作为同一受控步骤；失败后进程启动失败，不继续提供半迁移的教材接口。
4. 重新启动后运行 `ready`、核心读写 smoke 和 migration integration 测试指定的健康查询。

仅“安全且不丢数据”的 migration 提供 Down。`00001_local_workspace.sql` 的 Down 在没有教材表引用、没有映射/工作区业务数据时才允许删除新表；一旦有数据或后续 migration 依赖它，Down 必须拒绝并要求从备份恢复。任何删除列、删除表、重写历史数据或不可逆转换，都需要单独 ADR、备份/恢复演练和用户明确批准，不能由应用自动执行。

## 测试与验收

在授权引入 goose 与 `testcontainers-go` 的同一原子 PR 中，必须先实现真实 PostgreSQL 集成测试，SQLite 仅用于不涉及 DDL 语义的快速单测：

- 空数据库：core/shared 与 review/shared 分别 Up，断言只出现各自应有的表和 tracking record；
- 共库：两组 migration 依次执行，断言版本记录和表集合正确；
- 重复执行：第二次 Up 无 schema/data 副作用；
- 既有数据库：legacy `AutoMigrate` 表保留，workspace/mapping 创建不改写 legacy 数据；
- 失败：错误 SQL、角色不匹配、校验和漂移会使启动失败且不暴露教材路由；
- 安全 Down：空数据可回退；有引用或数据时明确拒绝；
- middleware：忽略客户端注入的 `workspace_id`，每个教材请求得到同一受控 UUID。

这些集成测试要求本机 Docker daemon 已启动并可由当前用户访问；`docker compose config` 只能验证 Compose 语法，不能证明 PostgreSQL 容器、migration 或恢复流程已经运行。Docker 不可用时，CI/本机执行必须明确标为未执行，不得以静态 migration asset 测试替代。

验收命令在依赖落地后为：

```bash
cd backend
GOCACHE=/tmp/inkwords-go-build go test ./shared/platform/postgres/... ./shared/kernel/httpx/... ./services/core-api/... -run 'Migration|Workspace' -v
OBSIDIAN_VAULT_PATH=/tmp/obsidian-vault docker compose config
```

## 取舍与后续

- 版本化 SQL 增加了 DDL 编写和测试成本，但换来可审计、可复现的本地升级路径，符合教材稿件和学习记录不可静默丢失的约束。
- `local_workspace_legacy_owner` 是临时兼容层，不是多用户模型。后续逐域将 legacy `user_id` 数据迁到 `workspace_id` 后，才可删除认证和映射桥梁。
- 不在此 ADR 或 PR-02 中新增远程服务、后台同步、用户账号、在线部署或移动端行为。
- 测试依赖固定为 `github.com/testcontainers/testcontainers-go v0.43.0` 及其 `modules/postgres` 同版本（MIT）；它只在 `_test.go` 使用，生产镜像和运行时业务代码不得引入 Docker client。
- 两个依赖按固定版本进入独立依赖审查；安全升级须复查 Goose 的 migration provider API、Testcontainers 的 Docker 兼容性、许可证和 `go.mod`/`go.sum` 最小 diff，不能与无关依赖升级混在同一提交。

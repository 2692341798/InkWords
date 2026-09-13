# Legacy identity 与 ProjectCourse 清理盘点

日期：2026-09-05

本文件最初是 PR-16 最终清理的只读前置盘点；2026-09-05 12:36 已在获授权的
真实备份/恢复演练后执行 v24/v25。下文保留迁移前证据，当前状态以“清理结果”一节为准。
统计命令排除 `node_modules`、
`dist` 和 `.git`：

```bash
for spec in \
  'project_course::ProjectCourse|project_course' \
  'auth_login_captcha::(?i)(?<![[:alnum:]_])auth(?![[:alnum:]_])|auth(?:middleware|token|service|handler|domain|group|header|bypass|config|state)|/auth|login|captcha|authorization|authentication' \
  'oauth::(?i)oauth' \
  'jwt::(?i)jwt' \
  'user_identity::UserID|user_id|CurrentUserID'; do
  name=${spec%%::*}
  pattern=${spec#*::}
  files=$(rg -l -P --glob '!node_modules/**' --glob '!dist/**' \
    --glob '!coverage/**' --glob '!docs/**' --glob '!*.sum' \
    --glob '!package-lock.json' "$pattern" backend frontend | sort || true)
  total=$(printf '%s\n' "$files" | sed '/^$/d' | wc -l | tr -d ' ')
  tests=$(printf '%s\n' "$files" | sed '/^$/d' | \
    rg '(_test\.go$|\.test\.[cm]?[jt]sx?$|/e2e/|/testdata/|^backend/scripts/test_)' | \
    wc -l | tr -d ' ')
  printf '%s total=%s production_or_config=%s tests_or_fixtures=%s\n' \
    "$name" "$total" "$((total-tests))" "$tests"
done
```

“生产/配置”统计排除 `_test.go`、`*.test.ts(x)`、`e2e/`、`testdata/`
和 `backend/scripts/test_*`；测试仍单列，因为它们也是兼容合同仍被维护的证据。
不同类别会匹配同一文件，不能把各行相加当作唯一文件总数。
本表在教材 MQ envelope 去除 `user_id` 并增加消费者拒绝门禁后重新执行；
门禁本身会命中关键词，因此调用清理必须结合具体文件语义，不能只看总数下降。

## 当前结果

### 清理结果

- core migration v24 已删除 `users`、`local_workspace_legacy_owner`、
  `o_auth_tokens`、`project_courses`、`user_prompt_settings`、`blogs.user_id`
  与 `job_tasks.requested_by`。
- review migration v25 已删除 `review_sessions.user_id` 并把 `workspace_id`
  固定为 NOT NULL。
- 真实迁移保留 59 篇博客、26 个任务、4 个复习会话和唯一 installation workspace；
  幂等重启没有重建旧表/列。
- 源码中的剩余关键词来自历史 migration、清理前置脚本、条件化空库 bootstrap
  或防回流测试，不是运行时账户边界。

### 迁移前盘点（历史）

| 范围 | 全部匹配文件 | 生产/配置 | 测试/夹具 | 结论 |
| --- | ---: | ---: | ---: | --- |
| `ProjectCourse` / `project_course` | 3 | 1 | 2 | 可执行生产调用已归零；唯一生产分类匹配是不可改写的历史 migration，两个测试匹配分别是退役架构门禁和教材任务拒绝旧 subtype 的回归。空表仍需备份/恢复与破坏性 migration 授权后才能删除。 |
| auth / login / captcha / authorization | 36 | 13 | 23 | 前端登录与 token helper、Nginx Authorization 转发、core-api 公开 auth 路由和 auth 领域均已删除；legacy owner middleware/resolver、`DEV_AUTH_USER_ID` 注入和孤立 OAuth 模型也已删除。剩余生产匹配主要来自第三方请求 header 与历史模型，不能按关键词批量删除。 |
| OAuth | 3 | 1 | 2 | GitHub 登录 OAuth 与孤立的后端 `OAuthToken`/AutoMigrate 投影均已删除；真实 `o_auth_tokens` 表为空并保留。唯一“生产/配置”分类匹配是用户工作树中的 `frontend/playwright.config 2.ts` 兼容副本，不是可执行后端调用。 |
| JWT | 1 | 0 | 1 | JWT 包、依赖、配置和生产调用已归零；唯一匹配是防止服务重新引入 JWT middleware 的架构测试。 |
| `UserID` / `user_id` / `CurrentUserID` | 20 | 8 | 12 | 博客、review 和全部通用/教材任务运行时均已迁到 installation workspace；legacy owner HTTP/resolver 与孤立 OAuth 用户模型已删除，新安装不再创建桥接用户、映射或 OAuth 表。历史博客/review `user_id` 与任务 `requested_by` 只保留为数据库审计数据。 |

## 阻断删除的生产边界

`ProjectCourse` 的 UI、公开 HTTP、core-api 领域/结果协调、llm-stream 生成 worker、
course-runner 实验验证消费者、export-service 课程包构建器、共享 kernel 与旧验证
RabbitMQ 消息均已退役。删除前的真实数据库只读证据为 `project_courses=0`、
ProjectCourse task `0`、对应 task event `0`；删除后架构门禁禁止上述包、符号和
routing key 恢复。历史 `00002_textbook_core.sql` 仍保留建表记录，真实空表也未删除。

通用任务也已通过 `00023_task_workspace.sql` 迁到 installation workspace：所有历史
任务只按显式 legacy-owner 映射回填，无法映射即停止迁移；新任务、HTTP 读取/取消/
重试/下载、幂等查询、结果落库和 generation/parse/export MQ 都只使用 workspace。
`requested_by`、博客/review 的旧 `user_id`、`users` 与映射表仍作为历史数据存在，
第三方博客发布的 OAuth 模型也仍有关联字段，因此 schema 清理尚未完成，但这些字段
已不再承担任务运行时归属。

为防止清理期间倒退，`backend/services/architecture_test.go` 现在约束
core-api、llm-stream、course-runner 与共享平台中的教材专属包不得导入旧
`auth`、`user`、`projectcourse` 或 `pkg/jwt` 包。该门禁只证明新教材域
保持本地 workspace 边界；新增门禁还禁止各服务 bootstrap 和 monolith 入口重新
调用 `httpx.AuthMiddleware()`。这些检查不证明旧数据字段已经迁移。

core-api 的 HTTP 装配也已拆成 `RegisterBlogRoutes`、`RegisterProjectRoutes`、
`RegisterTaskRoutes` 与 `RegisterTextbookRoutes`；四类注册器都只接收
`LocalWorkspaceContext`，不再装配兼容 owner middleware。
路由合同测试证明教材表面可以在
没有注册 `/auth/login` 的 router 中独立工作，同时完整组合仍保留
任务 retry、book-build 与候选稿 reject。
公开 register/login/captcha/GitHub OAuth 已从完整路由、bootstrap 和 monolith 中删除；
auth 领域与 JWT 包也已删除。随后 `/api/v1/user/profile`、avatar、stats 和
prompt-settings 六个公开 HTTP 操作及 core-api user 领域、前端“个人中心”页面和
相关 API service 一并退役，项目扫描/分析/解析也不再读取用户额度。随后 llm-stream
与 parser-service 的按用户额度检查、`user_prompt_settings` 覆盖读取以及
`users.tokens_used` 累计写入也已删除；每个任务/候选自身的 Token 遥测继续保留。
`users`、`user_prompt_settings`、固定 legacy owner 和历史关系字段仍作为未迁移数据
保留，所以这不等于旧用户表或所有身份字段已经可以删除。启动路径已不再
AutoMigrate 旧 `User` 资料/订阅/额度模型和 `UserPromptSettings`；全新安装只创建
支撑 legacy `user_id` 外键的最小 `LegacyOwner` 表投影，而 GORM 的非破坏性迁移
不会从现有本地库删除历史列或 prompt 设置表。

2026-09-04 本机重建 `core-api` 和前端后，全部 Compose 服务继续运行且定义了
healthcheck 的服务均为 healthy。`/api/v1/auth/captcha` 与 `/api/v1/auth/login`
均返回 HTTP 404；无 Authorization 的博客、review、mastery 请求返回 HTTP 200，
携带伪造 Authorization 的博客请求也返回同一本地工作区结果，证明服务不会恢复
请求身份。该 smoke 未创建、修改或删除任何业务记录。

前端 `ProjectCourse` 迁移已独立闭合：HomeEntry 不再显示“项目精通课程”，
`App`/`BlogStore` 不再接受 `project-course` view，专用 page、component、store、
service、类型与 API 路由常量已经删除，教材项目卡片只进入
`textbook-projects`。删除前的失败回归证明旧入口仍可达；随后删除前端登录 gate、
登录页、auth/OAuth service、bypass、request/SSE/export token helper 与 Nginx
Authorization 转发后，全量 61 个测试文件、224 个测试、coverage、
lint、Knip、TypeScript/Vite build 和 bundle budget 均通过，
构建产物不再包含 `ProjectCourse` chunk。随后路由失败回归证明 8 个公开 URL 仍可达，
再删除 routes、bootstrap handler 装配及旧 handler/service，并以架构门禁禁止恢复；
数据库只读证明确认表、任务和事件均为空后，又删除后台任务生产/结果回写、生成、
实验验证和课程打包兼容链。真实空表与历史 migration 保持不变，没有执行数据删除。

本机真实网关与浏览器已验证固定 owner 兼容读取：无 Authorization 的
`/api/v1/blogs`、`/api/v1/review/history`、`/api/v1/review/today`、
`/api/v1/review/notes` 和 `/api/v1/mastery/due` 均返回 HTTP 200。前端默认直接进入
教材项目，工作入口显示现有博客/复习记录，知识复习能抽取真实知识卡，浏览器控制台
为 0 错误、0 警告。复习题卡改从 `/app/obsidian` 只读挂载读取，使用受限文件根拒绝
路径与符号链接逃逸，因此不再要求 Obsidian REST 插件保持在线；写入型 Obsidian
适配器没有改变。该 smoke 没有创建复习 session、任务或 Provider 调用。

教材导出与 mastery 也已做无认证真实网关检查：不存在的教材章节 Markdown 与
冻结 BookBuild Markdown 分别返回领域错误 `TEXTBOOK_CHAPTER_NOT_FOUND` 和
`TEXTBOOK_BOOK_BUILD_NOT_FOUND`（HTTP 404），而不是认证 401；`/api/v1/mastery/due`
返回 HTTP 200 和空任务列表。路由回归进一步强制博客与教材导出都只走 workspace
middleware。该检查全部为 GET，没有创建学习或导出记录。

教材任务状态与重试也已从旧 `/api/v1/tasks/:id` 迁到
`/api/v1/textbook-projects/tasks/:taskID`。服务端只从本地 workspace 解析
`job_tasks.workspace_id`，并再次限定 task subtype 必须是样章生成、资料导入、
官网导入或教学工件验证；即使旧任务属于同一个 legacy owner，ProjectCourse/博客
任务也会被拒绝。`00019_textbook_task_workspace_identity.sql` 仅通过显式
`local_workspace_legacy_owner` 映射回填历史教材任务，任何无法映射的教材任务都会
使迁移 fail-closed；新教材任务的幂等查询也改用
`(workspace_id, task_type, idempotency_key)` 唯一索引。前端教材 service/store
已不再引用旧 task URL，旧任务路由继续供未迁移的博客和导出调用方
使用。

该切片还没有删除 `requested_by`，但 `00020_textbook_task_optional_legacy_owner.sql`
已把“字段兼容”和“新教材任务依赖 bridge user”拆开：历史教材任务保留原 owner
审计值，新教材生成、资料导入、官网导入和教学工件验证只写 `workspace_id`，
`requested_by` 留空；非教材任务仍受数据库 check constraint 保护，必须写入旧 owner。
`NewLocalWorkspaceResolver` 也只创建/读取 installation workspace，不再创建用户或
`local_workspace_legacy_owner` 映射。生成/解析仍与旧任务共享 routing key 和 worker
队列，但生产者改用无 `user_id` 的教材专用 envelope；现有消费者可向后兼容解码，
并在读取用户字段前按 subtype 进入 workspace 分支；若教材消息仍夹带非零 `user_id`，
消费者会在模型或解析前标记失败。独立教材验证 envelope 同样没有 `user_id`。三类
教材 worker 的归属都只认 `workspace_id`；兼容 generation/parse/export envelope
继续携带旧用户 ID 作为任务审计字段，同时新任务也携带 workspace，博客持久化与导出
不会再用该用户字段决定归属。随后修复了
core-api RabbitMQ publisher 重组 envelope 时漏传 `workspace_id` 的问题；llm-stream、
parser-service 与 course-runner 会在模型、解析或 sandbox 执行前，用任务表核对
`task_id + workspace_id + task_subtype`，缺失或不匹配即失败。教材 worker 因此也不再
把 MQ `user_id` 当作任务归属依据。后续只有在所有非教材 worker 调用迁移完成且旧调用
归零后，才可以讨论删除剩余的 `requested_by`、旧 generation/parse MQ `user_id` 和
bridge 映射表。

重建 `core-api` 与前端后，全部 Compose 服务继续运行，带 healthcheck 的服务
全部为 healthy。经新教材任务路由只读查询此前已成功的真实 DeepSeek 样章任务，
返回 `task_type=generation`、`task_subtype=textbook_sample_generate`、
`status=succeeded`；本次仅读取历史任务，没有触发 retry、模型调用或费用。

`00019` 上线前的真实库预检得到教材任务 `4`、可映射 `4`、孤立 `0`；重建
core-api 后启动日志显示恰应用 `1` 个新 migration。真实表已具备
`workspace_id` 外键、`ck_job_tasks_textbook_workspace` 与
`ux_job_tasks_workspace_type_idempotency`，聚合结果为 workspace-owned `4`、
旧任务 `22`，且没有非教材任务被错误赋予 workspace。新教材路由读取历史真实
DeepSeek 任务返回 HTTP 200；用同一路由读取一条真实旧任务返回 HTTP 403。
该验收没有执行 retry、创建任务或调用 Provider。

`00020` 上线前再次确认真实库 schema 为 `19`、任务总数 `26`、
`requested_by IS NULL` 为 `0`、教材任务 `4` 且 workspace 映射 `4`。迁移只放宽
`requested_by` 的列约束并增加 `ck_job_tasks_legacy_owner_or_textbook_workspace`，没有
改写历史行；重建 core-api 后 schema 为 `20`、列为 nullable、保护约束存在，任务计数
仍为 `26`。PostgreSQL 回归另外证明 ownerless workspace 教材任务可写，而 ownerless
旧任务被拒绝。该运行态验收没有创建真实任务、投递 MQ 或调用 Provider。

完成 MQ workspace 校验后，core-api、llm-stream、parser-service 与 course-runner
镜像已顺序重建并恢复 healthy；其余本机服务继续运行。RabbitMQ 已存在的
generation/parse/export 队列均为 ready `0`、unacknowledged `0`，数据库任务总数
仍为 `26`（workspace-owned `4`、旧任务 `22`），历史样章任务仍是
`succeeded`/`retry_count=0`。因此该运行态检查没有暗中投递任务或消耗 Provider。

完成 ProjectCourse 后台链退役后，重新盘点由 `28/16/12` 降至 `3/1/2`；唯一
“生产/配置”匹配来自历史 migration，不是可执行调用。重建 core-api、llm-stream、
export-service 与 course-runner 后四者均 healthy，8 个旧 URL 全部为 404，博客与教材
项目列表为 200；任务计数仍为 `26|4|22`，ProjectCourse 表/任务/事件均为 0，
RabbitMQ generation/parse/export 队列仍为 `0/0`。本轮没有创建任务、调用 Provider、
修改业务记录或执行 schema 删除。

完成公开用户账户表面退役后，六个 `/api/v1/user/*` 操作由路由合同固定为 404，
前端 Dashboard、个人中心导航和 user service 已删除；只被该页面使用的 `recharts`
与顶层 `react-is` 依赖也已清理。全量前端 60 个测试文件、222 个测试、coverage、
lint、Knip、生产构建和 bundle budget 通过；全量 Go 与 4 项集成合同通过。按本文件
命令重新盘点为 auth `43/18/25`、OAuth `5/3/2`、JWT `1/0/1`、user identity
`67/47/20`。真实运行态 smoke 与服务重建结果见同日完成审计；本轮仍未删除用户、
prompt 设置、legacy owner 或任何 schema 数据。

额度兼容链继续完成无数据删除退役：llm-stream 和 parser-service 不再查询
`users.tokens_used/token_limit`，旧生成提示词只使用版本化场景、风格与 profile 默认
合同，不再读取 `user_prompt_settings`；旧博客直写和 task-only 结果落库也不再累计
用户 Token 余额。任务结果中的 prompt/completion/cache/estimated Token 遥测保持不变。
架构门禁禁止上述两个 quota adapter、个性化 prompt 表查询和 per-user token balance
写入恢复。重新盘点为 user identity `64/45/19`，其他四类计数不变。
顺序重建 core-api、llm-stream 与 parser-service 后三者均 healthy，其余本机服务
继续正常运行；真实数据库只读快照为用户 `1`、历史 Token 余额合计 `493195`、
prompt 设置 `0`、博客 `59`、任务 `26|4|22`，三个 RabbitMQ 队列仍为 `0/0`，
博客与教材项目列表仍为 HTTP 200。本轮没有触发生成、解析、Provider 或业务写入，
历史余额只保留、不再被运行时代码读写。

Provider 隐私边界也已收口：`ChatRequest`/`ChatOptions` 不再定义 `UserID`，DeepSeek
普通、轻量和流式调用都不发送 `user_id`，旧博客 UUID、章节流水线标识和固定教材
字符串也不再写入 Provider options。任务归属仍由本地 workspace/legacy owner 合同
校验，请求 ID 与 Token/cache/latency/retry 遥测保持不变。架构门禁和序列化测试共同
禁止身份字段恢复。重新盘点为 user identity `62/43/19`，其他类别计数不变。
顺序重建 core-api 与 llm-stream 后两者恢复 healthy，全部已定义 healthcheck 的本机
服务均为 healthy；数据库任务仍为 `26|4|22`、历史 Token 余额仍为 `493195`，
三个 RabbitMQ 队列仍为 `0/0`，博客与教材项目列表仍为 200。该验证没有创建请求
任务或调用 Provider，因此证明的是新镜像启动和只读兼容性，不冒充真实模型验收。

生成质量流水线随后删除了 `seriesQualityPipelineInput.UserID` 和理解、草稿、审稿、
修复阶段的五层 `userID string` 透传。这些值只曾用于现已退役的 Provider request
标识，从未承担博客归属或权限校验；`sharedblog` 持久化输入中的真实 owner UUID
继续保留。架构门禁禁止伪身份字段恢复，重新盘点为 user identity `60/41/19`。
llm-stream 重建后恢复 healthy，全部已定义 healthcheck 的本机服务继续 healthy；
数据库仍为任务 `26|4|22`、历史 Token 余额 `493195`，三个 RabbitMQ 队列仍为
`0/0`。本轮没有创建任务或调用 Provider。

启动迁移投影继续收缩：删除旧 `User` / `UserPromptSettings` Go 模型，
`autoMigrateCore` 改为只创建最小 `LegacyOwner` 兼容行；这一阶段尚未处理的
`OAuthToken` 模型，已在后续确认无消费者且真实表为空后退役。SQLite 回归同时
证明新库不会创建资料/订阅/额度列和 prompt 设置表，已有库则保留它们；PostgreSQL
全新初始化回归继续通过。重建
五个 core 数据库服务后，全部 healthcheck 为 healthy；真实库仍为用户 `1`、
prompt 设置 `0`、博客 `59`、任务 `26|4|22`、历史 Token 余额 `493195`，八个旧用户列和
prompt 设置表均仍存在，三个 RabbitMQ 队列仍为 `0/0`。按关键词重新盘点为
auth `44/18/26`、OAuth `5/3/2`、JWT `1/0/1`、user identity `61/41/20`；增量均来自新的
防恢复测试，不是生产身份依赖回流。本轮没有调用 Provider、创建任务或删除 schema/数据。

review session 归属随后完成 workspace 迁移：`00021_review_session_workspace.sql`
增量增加 `workspace_id`、放宽但不删除旧 `user_id`，并用数据库 check 保证两种身份
至少存在其一；回滚在任何 workspace 数据存在时 fail-closed。review-service 启动先解析
唯一 installation workspace，再在事务中幂等回填历史会话；发现其他 workspace 时拒绝
启动，不猜测或覆盖。真实库迁移前为 schema `15`、会话 `4`、旧 owner 非空 `4`；
重建后为 schema `21`、会话 `4`、workspace 非空 `4`、旧 owner 非空仍为 `4`，且
workspace 与 core 唯一记录一致。再次重启后计数不变，无认证的 history/today/notes/
mastery due 均为 200，三个任务队列仍为 `0/0`。重新盘点为 auth `43/17/26`、
OAuth `5/3/2`、JWT `1/0/1`、user identity `56/38/18`。本轮没有创建 session/task、
调用 Provider 或删除旧数据。

博客运行时归属随后完成 workspace 迁移：`00022_blog_workspace.sql` 为 59 条历史博客按
显式 `local_workspace_legacy_owner` 映射回填 `workspace_id`，任何未映射行都会使迁移
fail-closed；旧 `user_id` 列和值完整保留且新 workspace-only 数据存在时 Down 会拒绝。
core-api 博客 CRUD、llm-stream 直连持久化、任务结果新建博客、export-service HTTP/MQ
读取都按 workspace 过滤；历史兼容任务缺 workspace 时只允许通过显式 owner 映射解析，
消息声明其他 workspace 时失败。真实库从 core schema `20` 迁到 `22`，结果为博客
`59|workspace 59|legacy owner 59|workspace NULL 0`、活动博客 `11|workspace 11`；任务仍为
`26|workspace 4`，三个队列仍为 `0/0`。ZIP GET 返回 200，幂等重启无重复迁移，五个
受影响服务均 healthy。最新盘点为 auth `42/17/25`、OAuth `5/3/2`、JWT `1/0/1`、
user identity `34/19/15`。本轮没有创建任务、调用 Provider 或删除旧数据。

项目准备 HTTP 入口也已移除无实际用途的 legacy owner 装配：core-api 的 scan/analyze
使用独立 `RegisterProjectRoutes` 和 workspace middleware，parser-service 的 parse/crawl
同样只解析 installation workspace，llm-stream 分析参数语义统一为 workspace。路由回归
与架构门禁禁止 parser bootstrap 恢复 legacy owner。重建 core-api、parser-service、
llm-stream 后三者 healthy；对四个入口发送空载荷均返回 handler 的 HTTP 400，而非身份
401/503，任务仍为 `26|workspace 4`、博客仍为 `59|workspace 59`、队列仍为 `0/0`。
空载荷在任何抓取、解析或模型调用前失败，因此本轮没有 Provider/网络执行或业务写入。

通用任务运行时归属随后完成 workspace 迁移：`00023_task_workspace.sql` 在真实升级前
确认 schema `22`、任务 `26`、workspace 非空 `4`、待回填 `22`、无法映射 `0`，且三条
队列均为 `0/0`。迁移后 schema 为 `23`，26/26 任务都有 workspace，26/26 历史
`requested_by` 值完整保留，状态仍为成功 21、失败 3、取消 2。旧
`generate_series` 任务经新 `/api/v1/tasks/:id` workspace 路由返回 HTTP 200；core-api
再次重启后计数不变。generation/parse/export 出站 envelope 不再包含 `user_id`，旧队列
载荷中的多余字段仍可兼容解码但不会再次传播；任务结果持久化也不再用旧 owner 反查。
五个相关服务全部 healthy，队列仍为空。最新盘点为 auth `42/17/25`、OAuth `5/3/2`、
JWT `1/0/1`、user identity `24/11/13`。本轮没有创建任务、重试任务或调用 Provider。

随后删除已经没有生产调用的 `LocalLegacyOwnerContext`、
`NewLocalLegacyOwnerResolver`、自动选择/创建 bridge user 和 `DEV_AUTH_USER_ID` Compose
注入；`EnsureLocalWorkspace` 现在只幂等创建 installation workspace。PostgreSQL 新库回归
明确验证 workspace 为 1、bridge user 为 0、owner mapping 为 0；已有真实数据库中的
用户、映射及历史 owner 数据没有删除或改写。架构门禁禁止这些运行时桥梁恢复，盘点
进一步更新为 auth `38/15/23`、OAuth `5/3/2`、JWT `1/0/1`、user identity
`21/9/12`。随后确认后端没有任何 OAuth 读写消费者、真实 `o_auth_tokens` 为 0，因而
删除孤立 `OAuthToken` 模型并停止 AutoMigrate 在新库创建该表；旧库表不执行 DROP。
最新盘点为 auth `36/13/23`、OAuth `3/1/2`、JWT `1/0/1`、user identity
`20/8/12`。

BuildKit 因配置的 `docker.1panel.live` 代理拒绝连接一度无法解析基础镜像；改用本机
兼容构建路径后，core-api、llm-stream、parser-service、export-service、course-runner
和 review-service 六个新版镜像均已串行构建并切换成功。容器镜像 ID 与本地新 tag
逐项一致，全部 healthcheck 为 healthy，`DEV_AUTH_USER_ID` 6/6 未注入；core/review
数据计数、空 OAuth 表、任务 GET 200 和队列 `0/0` 均保持不变。

## 下一步门槛

在任何删除、`00009_remove_legacy_identity.sql` 或不可逆 schema 迁移之前，
必须按域完成下面的可验证步骤。ProjectCourse 已满足第 1、2 项，但仍未满足第 3、4 项：

1. 为每个旧 HTTP/MQ/export 入口确定教材等价路径，或明确产品下线决定。
2. 删除一个域前运行 `rg`、对应单元/路由测试和至少一次真实兼容流量检查，
   证明没有生产消费者。
3. 在实际目标数据库生成带时间戳的备份并演练恢复；该操作会写入外部状态，
   需要单独用户授权和一个由用户确认、位于仓库外的绝对空目录。
4. 只有前三项成功后，才新增并执行删除 migration；migration 必须注明回滚
   不可用时的恢复来源与步骤。

`backend/scripts/backup_local.sh` 已实现上述备份与恢复演练的 fail-closed 工具链，
`backend/scripts/check_legacy_cleanup_ready.sh` 进一步只读校验 v2 备份哈希、恢复标记、
写服务已停止，以及任务/教材项目/博客/掌握目标/复习会话与轮次的行数和更新时间
水位漂移、workspace 归属及 OAuth/ProjectCourse
空表条件。两者的静态回归和
真实数据库只读计数 SQL 已通过；由于本轮没有用户确认的目标目录，
尚未创建 dump 或临时恢复库。工具就绪不满足第 3 项本身。因此本轮仍只完成
“统计并确认调用归零”的前置条件；公开 auth/JWT 与 ProjectCourse 可执行领域已完成
无数据删除的退役，但没有删除任何旧表、用户数据、历史 migration 或兼容 owner 字段。

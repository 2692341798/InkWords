# 墨言知识训练平台 (InkWords Trainer) - API 接口文档

## 0. 变更记录
- 2026-06-08：执行一次文档与仓库治理同步：刷新 `README.md`、补充 `docs/superpowers/*` 设计/计划文档，并移除仓库内旧的 `CODE_OF_CONDUCT.md` 与 `skills/llm-wiki-ingest/*` 文档资产；本次不新增、不修改任何对外 API 路由、请求字段或响应字段。
- 2026-06-05：继续推进 blog-domain 内部边界修复。本次将 `SeriesPersistence.SaveSeriesIntro()` 与 `SeriesPersistence.MarkSeriesIntroFailed()` 收紧为必须同时校验 `user_id + parent_id`，并让 service 导读生成调用链显式透传当前用户；不新增也不修改任何对外 API 路由、请求字段或响应字段。
- 2026-06-05：继续推进 blog-domain 内部边界修复。本次将 `SeriesPersistence.LoadSeriesOldContent()` 收紧为必须同时按 `user_id + blog_id` 读取旧正文，并让 service 调用链显式透传当前用户；不新增也不修改任何对外 API 路由、请求字段或响应字段。
- 2026-06-05：继续推进 blog-domain 内部边界修复。本次仅修正 `SeriesPersistence.EnsureSeriesParentAndDrafts()` 的父稿归属校验：如果 `parent_id` 指向其它用户的系列父稿，将返回错误并拒绝继续预建章节草稿；不新增也不修改任何对外 API 路由、请求字段或响应字段。
- 2026-06-05：继续推进 service 内部 Chapter bridge 收口。本次删除 `decomposition_service.go` 中最后的 `Chapter` 本地别名，并让 service 包内部相关生成、提示词、质量门禁与测试代码统一直接依赖 `domain/blog/contracts.Chapter`；不新增也不修改任何对外 API 路由、请求字段或响应字段。
- 2026-06-05：继续推进 service 内部 series bridge 收口。本次删除 `decomposition_series_persistence.go`，并让 `DecompositionService` 与相关持久化辅助逻辑直接依赖 `domain/blog/contracts` 的 `SeriesPersistence / SeriesDraftPreflightInput / SeriesChapterPersistenceInput` 以及 `domain/blog` 默认适配器；不新增也不修改任何对外 API 路由、请求字段或响应字段。
- 2026-06-05：继续推进 service 内部 blog bridge 收口。本次删除 `generator` 与 `continue` 两层仅用于内部装配的桥接文件，改为 service 直接依赖 `domain/blog/contracts` 与 `domain/blog` 默认适配器；不新增也不修改任何对外 API 路由、请求字段或响应字段。
- 2026-06-05：继续推进 blog contracts 收口。本次仅把 `backend/internal/domain/stream/service.go` 的内部依赖从 `internal/service.Chapter` 切到 `internal/domain/blog/contracts.Chapter`，不新增也不修改任何对外 API 路由、请求字段或响应字段。
- 2026-06-04：新增 `FRONTEND_PORT` 作为 Docker Compose 前端宿主机端口覆盖变量。本次不新增也不修改任何 API 路由、请求字段或响应字段；仅补充运行约定：默认仍通过 `http://localhost` 访问，当宿主机 `:80` 被占用时，可临时用 `FRONTEND_PORT=8088` 将前端入口切到 `http://localhost:8088`，其 `/api/*` 代理语义保持不变。
- 2026-06-04：Generation Task-Only Task 4 继续扩展 generation 任务成功路径的内部语义，不新增也不修改任何对外 API 路由、请求字段或响应字段；当 `generate_series` 任务成功时，系统会把结构化 `result_json` 中的 `parent_blog` 与 `chapters[]` 交给 `core-api` 消费，并由 `core-api` 完成系列父博客与章节草稿的最终持久化。
- 2026-06-04：Generation Task-Only Task 3 继续扩展 generation 任务成功路径的内部语义，不新增也不修改任何对外 API 路由、请求字段或响应字段；当 `continue` 任务成功时，系统会把结构化 `result_json` 中的 `blog_id / appended_content / final_content` 交给 `core-api` 消费，并以 `final_content` 完成正文更新。
- 2026-06-04：Generation Task-Only Task 2 继续收紧 generation 任务成功路径的内部语义，不新增也不修改任何对外 API 路由、请求字段或响应字段；当 `generate_single` 任务成功且 `job_tasks.result_json` 为结构化结果时，`core-api` 现会在任务成功路径中消费该结果并完成最终博客正文与 token 记账的业务落库。
- 2026-06-04：Generation Task-Only Task 1 仅加强生成任务成功结果的内部契约，不新增也不修改任何对外 API 路由、请求字段或响应字段；`generate_single` 任务在 `INKWORDS_TASK_PERSISTENCE_MODE=task_only` 下写入 `job_tasks.result_json` 时，不再固定保存 `{"done":true}`，而是保存带 `result_version / task_type / task_subtype / persistence_mode / final_status / usage / payload` 的结构化结果，供后续 `core-api` 持久化闭环消费。
- 2026-06-04：Phase 2 执行 `core-api / llm-stream` 深层拆分第一轮。对外 API 路由、请求/响应字段与 `http://localhost` 单入口保持不变，但 `core-api` 与 `llm-stream` 的主 HTTP 装配已分别迁入 `backend/services/core-api/` 与 `backend/services/llm-stream/` 的服务自有 `bootstrap/routes/cmd`；共享 `internal/transport/http/v1/routes.go` 与 `internal/transport/http/v1/api/stream_api.go` 仅保留为过渡兼容层。另新增 `INKWORDS_TASK_PERSISTENCE_MODE=task_only` 运行开关，用于在拆分阶段控制 legacy 生成链路不再由 `llm-stream` 直接把最终结果写入 `blogs / users`。
- 2026-06-04：Task 4 将 `export-service` 的启动装配、私有路由、RabbitMQ consumer 与导出产物 store 收口到 `backend/services/export-service/` 服务自有目录；本次仅调整代码归属与服务入口组织，对外导出 API、任务接口、下载协议、请求/响应字段与数据库结构均保持不变。
- 2026-06-03：Task 6 继续推进 `parser-service` 异步化。`POST /api/v1/tasks/parse` 仍由 `core-api` 创建 `parse_file / parse_archive` 任务并发布到 RabbitMQ `parse.requested`；`parser-service` 作为 parse worker 消费任务并把结果回写到 `job_tasks`。前端当前对 `.zip` 课件包和 `50MB` 以上普通单文件默认走任务式解析，`50MB` 及以下普通单文件仍保留 `/api/v1/project/parse` 作为同步兼容路径。
- 2026-06-03：Task 3 为 `core-api / llm-stream / parser-service / export-service / review-service` 补齐统一运行契约。各服务新增 `GET /health`（进程存活）与 `GET /ready`（依赖就绪）端点，并继续保留 `GET /api/v1/ping` 兼容检查；请求链路统一注入/透传 `X-Request-ID`，服务端访问日志统一输出 `service / request_id / path / method / status / latency_ms` 结构化字段。Docker Compose 同步为五个后端服务与前端增加 healthcheck，前端依赖改为等待各后端 `healthy` 后再启动。
- 2026-06-03：Task 2 将任务式 SSE 收口为默认生成主链路。前端默认通过 `POST /api/v1/tasks/generation` 创建 `generate_single / generate_series / continue / polish` 四类任务，再订阅 `GET /api/v1/tasks/:id/stream`；旧 `/api/v1/stream/*`、`/api/v1/blogs/:id/continue`、`/api/v1/blogs/:id/polish` 继续保留，仅作为回滚兼容入口。
- 2026-06-03：生成链路进入 RabbitMQ 任务式 SSE Phase B。新增任务接口族：`POST /api/v1/tasks/generation`、`GET /api/v1/tasks/:id`、`GET /api/v1/tasks/:id/stream`、`POST /api/v1/tasks/:id/cancel`；前端公开入口仍保持 `http://localhost` 与 `/api/*` 不变，`core-api` 负责创建/查询/取消任务与基于数据库事件表输出 SSE，`llm-stream` 负责消费 RabbitMQ 中的生成任务。
- 2026-06-03：Docker 微服务化 Phase 2（已落地到代码与编排）。对外 API 路由与请求/响应字段保持不变；前端 Nginx 继续作为单一公开入口并按路径分流到 `core-api/llm-stream/parser-service/export-service/review-service`：`/api/v1/project/parse` → `parser-service`，`/api/v1/blogs/:id/export*` → `export-service`，`/api/v1/review/*` → `review-service`；review 拆库后的数据迁移需按 Runbook 执行：[review-db-migration.md](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/docs/runbooks/review-db-migration.md)。
- 2026-06-03：Docker 微服务化 Phase 1。对外 API 路由与请求/响应字段保持不变；Docker Compose 中后端拆分为 `core-api` 与 `llm-stream`，由前端 Nginx 按路径分流（`/api/v1/stream/*` 与 `/api/v1/blogs/:id/(continue|polish)` → `llm-stream`，其余 `/api/*` → `core-api`），支持仅扩容 `llm-stream`。
- 2026-06-03：稳定性与工程化优化（Task 1-5）。本次不新增、不删除任何后端 API 路由或请求/响应字段；主要变更为：后端启动链路补齐显式 `http.Server` 与优雅停机，`/api/v1/stream/scan`、`/api/v1/stream/analyze`、`/api/v1/blogs/:id/continue` 等流式主链路恢复遵守请求取消语义（客户端断开后默认停止后台任务）；系列生成链路补齐“前置草稿创建/清理 + 章节完成落库 + Token 记账”的事务边界与可观测错误；前端对 SSE 401 统一收口为清 token 并返回登录页，不再强制 `location.reload()`。
- 2026-06-02：系列生成失败原因可视化与 SSE 稳定性修复。本次不新增、不删除任何后端 API 路由、请求字段或响应字段；前端开始消费并展示既有系列生成 SSE 事件中的 `status=error` 与 `message`，后端 `stream` handler 统一为生成/分析类流增加缓冲并在每次写事件后主动 `flush`，降低慢客户端导致的流式背压与误超时风险。
- 2026-06-01：文件来源 Analyze 链路新增“动态提示词 profile”锁定机制。`POST /api/v1/stream/analyze` 在完成大纲分析后会额外返回 `resolved_prompt_profile`（含 `key`、`display_name`、`document_kind`、`reason`）；`POST /api/v1/stream/generate` 请求新增 `prompt_profile_key`、`document_kind`，用于让单篇/系列生成沿用同一次 Analyze 已锁定的内容类型提示词。
- 2026-06-01：知识漫游复习会话升级为“文章驱动提问 + 结构化反馈”；`POST /api/v1/review/sessions` 与 `GET /api/v1/review/sessions/:id` 新增 `session_outline`、`current_round_goal`，`POST /api/v1/review/sessions/:id/respond` 新增 `review_feedback` 与 `current_round_goal`，用于明确返回本轮目标、命中点、遗漏点与下一步建议。
- 2026-05-29：工程化结构拆分 Phase 1：review 领域与 Sidebar/export 逻辑完成模块化拆分，生成链路辅助逻辑拆分为更小文件；本次不新增、不删除、不修改任何对外 API 路由或请求结构。
- 2026-05-29：知识漫游复习入口调整为“随机抽题 + 手动选文”双入口；`POST /api/v1/review/pick` 的后端实现改为从候选集中真正随机选题，不再固定返回首个符合条件的笔记。`GET /api/v1/review/today` 路由保留以兼容既有客户端，但当前前端主入口不再展示“今日推荐”卡片。
- 2026-05-29：生成器前端工作流改为 `选择来源 -> 配置解析 -> 确认大纲` 三步模型；解析/分析进度内嵌在“配置解析”，写作进度内嵌在“确认大纲”。本次不新增、不修改任何后端 API 路由或请求结构，但文件上传前端交互调整为“先完成 `/api/v1/project/parse`，再由用户在配置页显式触发 `/api/v1/stream/analyze` 生成大纲”，避免上传 ZIP/课件后跳过场景选择。
- 2026-05-28：工程规范收尾与提交前同步：本次未新增或删除后端 API 路由，但统一收紧了部分既有接口的外部错误输出约定。`/api/v1/blogs` 相关接口不再直接透出内部错误详情，`/api/v1/blogs/:id` 在目标不存在时明确返回 `404 blog not found`；流式接口 `/api/v1/stream/generate`、`/api/v1/blogs/:id/continue`、`/api/v1/blogs/:id/polish`、`/api/v1/stream/analyze`、`/api/v1/stream/scan` 的 SSE `error` 事件统一返回稳定错误文案，避免泄漏底层数据库/系统异常文本。
- 2026-05-27：项目定位升级为“墨言知识训练平台（InkWords Trainer）”，口号“把资料变成知识，把知识变成能力”；本次仅同步文档命名口径，不新增、不修改任何后端 API 路由或请求结构。
- 2026-05-27：前端新增 `HomeEntry` 引导入口、共享 `StepStrip` 步骤条，以及“同一时间只显示当前主步骤”的流程式工作台；本次仅调整前端编排与交互，不新增、不修改任何后端 API 路由或请求结构。
- 2026-05-27：新增“知识漫游复习”接口族 `/api/v1/review/*`，覆盖今日推荐、随机抽题、手动选文、会话创建、追问、提示、结束训练与最近记录查询；所有接口均要求 JWT Bearer Token。
- 2026-05-25：将 `/api/v1/project/parse` 的文件上传上限从 100MB 提升到 888MB，并同步更新前端文件选择校验与 Nginx `client_max_body_size`，避免网关层和应用层限制不一致。
- 2026-05-25：修复 AI 思考/对话式前言混入正文的问题；流式正文输出链路新增统一清洗，默认剥离 `<think>...</think>` 与“好的，收到你的需求 / 作为高级全栈架构师”等开头套话，`/api/v1/stream/generate`、`/api/v1/blogs/:id/continue`、`/api/v1/blogs/:id/polish` 均受此约束。
- 2026-05-25：修复前端“创作场景”在文件上传与大纲生成过程中的交互歧义；保持 `/api/v1/stream/analyze` 与 `/api/v1/stream/generate` 的 `scenario_mode` 请求结构不变，但前端在上传分析时改为读取最新场景值，并在大纲生成后锁定该场景（仅 UI/请求时机修复，无 API 路由变更）。
- 2026-05-25：修复系列生成异常时历史博客只剩父级导读的问题；`/api/v1/stream/generate` 的后端实现改为先为每个章节创建子博客草稿，再在流式成功后回填正文、失败时标记错误状态。API 路由与请求结构不变，但 `/api/v1/blogs` 返回的系列 `children` 在章节失败场景下也会保留占位子节点。
- 2026-05-24：`/api/v1/stream/analyze` 与 `/api/v1/stream/generate` 新增 `scenario_mode` 请求字段，支持 `ebook_interpretation`、`open_book_exam_review`、`beginner_walkthrough` 三种创作场景；后端缺省按来源兜底（`git -> beginner_walkthrough`，其它来源 -> `ebook_interpretation`）。
- 2026-05-21：`/api/v1/project/parse` 新增 ZIP 课件包解析能力；支持返回 `data.archive_summary`，用于展示压缩包扫描、保留、去重、忽略与失败统计。
- 2026-05-21：新增用户写作模板接口 `/api/v1/user/prompt-settings`（GET/PUT），并为 `/api/v1/stream/generate` 增加 `article_style` 请求字段，用于控制文章类型/写作要求模板。
- 2026-05-21：修复本地 PDF/Word/Markdown 上传后触发 `git_url is required for git source type` 弹窗的问题；前端在 `/api/v1/stream/analyze` 显式发送 `source_type=file`，后端增加基于 `source_content` 的文件来源兼容推断（无 API 路由变更）。
- 2026-04-29：新增“写博客”入口配套接口 `/api/v1/blogs/draft`（创建手写草稿）。
- 2026-05-08：写博客编辑器新增“语音输入”（纯前端能力，无 API 变更）。
- 2026-05-08：新增“博客润色”流式接口 `/api/v1/blogs/:id/polish`（SSE 输出润色草稿，不落库）。
- 2026-05-08：工程化整理（无 API 路由变更，主要为仓库文件治理与文档同步）。
- 2026-05-08：目录结构工程化调整落地（无 API 路由变更）。
- 2026-05-08：后端 Blog Domain 垂直切片迁移（无 API 路由变更）。
- 2026-05-08：后端 User Domain 垂直切片迁移（无 API 路由变更）。
- 2026-05-08：后端 Auth Domain 垂直切片迁移（无 API 路由变更）。
- 2026-05-08：后端 Stream/Project Domain 垂直切片迁移（无 API 路由变更）。
- 2026-05-10：修复导出到 Obsidian 时“初始化知识库目录失败”（兼容 Obsidian Local REST API 目录列表 `{ "files": [...] }` 返回格式；无 API 路由变更）。

## 0.1 运行契约补充
- 所有 HTTP 请求都会透传或生成 `X-Request-ID` 响应头；若上游已携带该请求头，后端优先沿用，便于跨服务排障。
- 以下运维健康检查端点由各后端服务直接暴露给容器内探针使用：

| 接口地址 | 请求方法 | 功能描述 | 说明 |
| -------- | -------- | -------- | ---- |
| `/api/v1/ping` | GET | 兼容历史探针 | 返回既有 `{ code, message, data }` 结构 |
| `/health` | GET | 进程存活检查 | 不检查外部依赖，只表示服务进程已启动 |
| `/ready` | GET | 依赖就绪检查 | 默认至少检查数据库；`llm-stream` 额外检查 RabbitMQ 配置是否可用 |

## 1. 认证模块 (AuthAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/auth/captcha` | GET | 获取图形验证码 | 无 -> 返回 `{ captcha_id, image }` |
| `/api/v1/auth/register` | POST | 邮箱密码注册 | `{ username, email, password, captcha_id, captcha_value }` |
| `/api/v1/auth/login` | POST | 邮箱密码登录 | `{ email, password, captcha_id, captcha_value, remember_me }` -> 返回 `token` |
| `/api/v1/auth/oauth/:provider` | GET | 第三方授权跳转 (如 `github`) | 无 |
| `/api/v1/auth/callback/:provider` | GET | OAuth回调 | `code`, `state` -> 重定向至前端 (带 `token` 或 `bind_required` 等参数) |
| `/api/v1/auth/bind-github` | POST | GitHub 登录发现邮箱冲突时绑定本地账号 | `{ email, password, github_id, username, avatar_url }` -> 返回 `token` |

## 2. 用户模块 (UserAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/user/profile` | GET | 获取当前登录用户信息 | JWT Bearer Token |
| `/api/v1/user/profile` | PUT | 更新当前登录用户名 | `{ username }` |
| `/api/v1/user/avatar` | POST | 上传用户头像图片 | `multipart/form-data` -> `avatar` |
| `/api/v1/user/stats` | GET | 获取用户仪表盘统计数据 (Token, 费用, 字数, 技术栈) | JWT Bearer Token |
| `/api/v1/user/prompt-settings` | GET | 获取文章类型默认模板与当前用户自定义覆盖 | JWT Bearer Token |
| `/api/v1/user/prompt-settings` | PUT | 更新当前用户的写作要求模板覆盖（空字符串表示恢复默认） | `{ overrides: { [styleKey]: string } }` |

## 3. 项目解析模块 (ProjectAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/project/analyze` | POST | 解析 Git 仓库生成大纲 (Legacy) | `{ git_url, sub_dir }` |
| `/api/v1/project/parse` | POST | 解析本地文件或 ZIP 课件包并提取 `source_content` | `multipart/form-data` -> `file` (最大支持 888MB；支持 `.pdf/.docx/.md/.markdown/.txt/.zip`) |

### 3.1 `/api/v1/project/parse` 返回说明
- 普通文件上传时，成功响应保持兼容：`data.source_content`
- ZIP 课件包上传时，成功响应会额外返回：
  - `data.archive_summary.total_files`
  - `data.archive_summary.supported_files`
  - `data.archive_summary.kept_files`
  - `data.archive_summary.duplicate_files`
  - `data.archive_summary.ignored_files`
  - `data.archive_summary.failed_files`
  - `data.archive_summary.kept_paths`
- ZIP 解析会自动完成白名单筛选、内容去重、顺序聚合，并在“无有效文本文件”或“存在非法压缩路径”时返回错误。

## 4. 流式生成模块 (StreamAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/stream/scan` | POST | 快速扫描 Git 仓库一级目录并通过 README 智能提取描述 | `{ git_url }` -> SSE Stream |
| `/api/v1/stream/analyze` | POST | 实时流式拉取 Git 或解析长文本文件生成大纲 | `{ git_url, selected_modules, source_type, source_content, scenario_mode }` -> SSE Stream；当请求仅包含 `source_content` 且未传 `git_url` 时，后端会兼容判定为 `file` 来源；文件来源完成时会在结果中返回 `resolved_prompt_profile` |
| `/api/v1/stream/generate` | POST | 根据大纲或内容流式生成博客章节 (Legacy/Rollback) | `{ source_content, source_type, git_url, outline, series_title, parent_id, article_style, scenario_mode, prompt_profile_key, document_kind }` -> SSE Stream |
| `/api/v1/blogs/:id/continue` | POST | 继续生成被截断的单篇博客 (Legacy/Rollback) | 无 -> SSE Stream |
| `/api/v1/blogs/:id/polish` | POST | 对当前草稿全文润色并返回“润色草稿” (Legacy/Rollback) | `{ title, content }` -> SSE Stream |

### 4.1 `scenario_mode` 场景字段说明
- 支持枚举：
  - `ebook_interpretation`：电子书解读
  - `open_book_exam_review`：开卷复习
  - `beginner_walkthrough`：小白教程
- 设计边界：
  - `scenario_mode` 决定“这次要产出什么任务形态”。
  - `article_style` 继续决定“内容以什么写法呈现”。

### 4.2 `/api/v1/stream/analyze` 请求补充说明
- 新增字段：`scenario_mode`
- 缺省兜底：
  - `git -> beginner_walkthrough`
  - `file` 及其它来源 -> `ebook_interpretation`
- 作用：
  - 控制大纲拆解偏向“章节解读 / 考点速查 / 学习路径”中的哪一种结构。
- 文件来源补充能力：
  - 当 `source_type=file` 时，后端会在大纲生成前先做一次轻量内容分类，为当前文件锁定最匹配的动态提示词 profile。
  - Analyze 完成事件中的 `content` 会额外包含 `resolved_prompt_profile`，结构如下：

```json
{
  "series_title": "《非暴力沟通》解读",
  "chapters": [],
  "resolved_prompt_profile": {
    "key": "psychology_communication_book",
    "display_name": "心理学经典解读",
    "document_kind": "psychology_communication",
    "reason": "已根据文件内容自动匹配提示词。"
  }
}
```

- 回退策略：
  - 若分类器不可用、内容为空、返回非法 key 或 JSON 解析失败，后端会按 `scenario_mode` 回退到默认 profile，并在 `reason` 中明确标记“已回退到默认提示词”。
- 前端交互约束：
  - 用户可在发起 Analyze 前手动切换 `scenario_mode`。
  - 大纲返回后，前端会锁定本次 Analyze 使用的 `scenario_mode`，隐藏选择器并以只读标签展示当前场景，避免“大纲按 A 分析、正文按 B 生成”的歧义。
  - 当返回 `resolved_prompt_profile` 时，前端会在大纲区额外显示“当前提示词类型”只读标签，并将该 profile 锁定到后续 Generate 请求。

### 4.3 `/api/v1/stream/generate` 请求补充说明
- 新增字段：`scenario_mode`
- 新增字段：`prompt_profile_key`、`document_kind`
- 作用范围：
  - 单篇生成
  - 系列章节生成
  - 系列导读生成
- 字段作用：
  - `prompt_profile_key`：指定当前生成链路沿用的动态提示词 profile key。
  - `document_kind`：记录当前文件被识别出的文档类别，便于前后端保持一致语义。
- 兼容策略：
  - 旧前端不传 `scenario_mode` 仍可调用，后端按 `source_type` 自动回填默认值。
  - 旧前端不传 `prompt_profile_key` 或传非法值时，后端会按 `scenario_mode` 自动回退到默认 profile，保证旧链路可继续工作。
- 前端约束：
  - 当本次任务已经生成大纲时，Generate 会沿用该次 Analyze 已锁定的 `scenario_mode`，不再允许用户在大纲生成后修改。
  - 当本次任务来自文件 Analyze，Generate 会同时沿用 Analyze 返回的 `resolved_prompt_profile.key` 与 `resolved_prompt_profile.document_kind`，避免“大纲像心理学解读、正文又回退成通用技术博客”的漂移。

### 4.4 流式正文清洗约束
- 适用范围：
  - `/api/v1/stream/generate`
  - `/api/v1/blogs/:id/continue`
  - `/api/v1/blogs/:id/polish`
- 清洗目标：
  - 剥离 `<think>...</think>` 思考标签块
  - 跳过 `reasoning_content`
  - 去除开头的对话式前言/角色自述，例如“好的，收到你的需求”“作为高级全栈架构师……”“你是一位文本解读专家……”
- 设计目标：
  - 用户最终看到和落库的正文应只包含 Markdown 正文内容，不应混入模型思考过程或对话式套话

### 4.5 `/api/v1/stream/generate` 系列章节阶段事件
- 适用范围：
  - 仅系列章节生成链路；单篇生成与系列导读仍沿用既有事件语义。
- 新增状态：
  - `understanding`：章节理解阶段开始
  - `drafting`：章节草稿生成阶段开始
  - `reviewing`：章节技术审稿阶段开始
  - `revising`：终稿补强准备阶段开始
  - `streaming`：仅终稿补强阶段持续输出正文 chunk
  - `usage`：终稿补强完成后返回本章节的 DeepSeek usage 与 Prompt Cache 命中统计
- `usage` 事件载荷：
  - `prompt_tokens`
  - `completion_tokens`
  - `prompt_cache_hit_tokens`
  - `prompt_cache_miss_tokens`
- 典型 `event: chunk` 载荷示例：

```json
{
  "status": "understanding",
  "chapter_sort": 1,
  "title": "Gin 路由"
}
```

```json
{
  "status": "streaming",
  "chapter_sort": 1,
  "title": "Gin 路由",
  "content": "### 1. 请求先进入 Engine\\n"
}
```

```json
{
  "status": "usage",
  "chapter_sort": 1,
  "prompt_tokens": 1200,
  "completion_tokens": 500,
  "prompt_cache_hit_tokens": 900,
  "prompt_cache_miss_tokens": 300
}
```
- 兼容说明：
  - 路由、请求体、`completed/error` 终态事件不变。
  - 旧前端即使暂未消费新增阶段，也仍可通过 `streaming/completed/error` 维持基本链路。

## 4.6 任务式生成模块 (TaskAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/tasks/generation` | POST | 创建一个生成任务并返回任务 ID 与 SSE 订阅地址 | `{ kind, payload, idempotency_key? }` |
| `/api/v1/tasks/parse` | POST | 创建一个解析任务并返回任务 ID 与 SSE 订阅地址 | `{ kind, payload, idempotency_key? }` |
| `/api/v1/tasks/export` | POST | 创建一个 PDF 导出任务并返回任务 ID 与 SSE 订阅地址 | `{ kind: "export_pdf", payload: { blog_id }, idempotency_key? }` |
| `/api/v1/tasks/:id` | GET | 查询任务当前状态、结果摘要与错误信息 | 路径参数 `id` |
| `/api/v1/tasks/:id/stream` | GET | 订阅任务事件流（DB 事件表轮询转 SSE） | 路径参数 `id` -> SSE Stream |
| `/api/v1/tasks/:id/cancel` | POST | 请求取消一个排队中或执行中的任务 | 路径参数 `id` |
| `/api/v1/tasks/:id/download` | GET | 下载已完成导出任务的 PDF 产物（成功后删除文件） | 路径参数 `id` |

### 4.6.1 创建生成任务
- 请求头：`Authorization: Bearer <token>`
- 请求体最小结构：

```json
{
  "kind": "generate_series",
  "payload": {
    "source_type": "file",
    "source_content": "..."
  },
  "idempotency_key": "series:abc"
}
```

- 成功响应（`202 Accepted`）：

```json
{
  "task_id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "queued",
  "stream_url": "/api/v1/tasks/123e4567-e89b-12d3-a456-426614174000/stream"
}
```

- 行为约束：
  - `kind` 用于区分生成子类型，例如 `generate_single`、`generate_series`。
  - `idempotency_key` 可选；当同一用户重复提交相同 key 时，服务端可直接复用既有未完成任务，避免前端重复点击造成重复生成。
  - 当前阶段由 `core-api` 创建任务并把消息投递到 RabbitMQ，实际生成仍由 `llm-stream` worker 异步执行。

### 4.6.2 创建解析任务
- 请求头：`Authorization: Bearer <token>`
- 请求体最小结构：

```json
{
  "kind": "parse_archive",
  "payload": {
    "filename": "courseware.zip",
    "content_base64": "<base64>"
  },
  "idempotency_key": "parse:courseware.zip:12345:1717400000"
}
```

- 成功响应（`202 Accepted`）：

```json
{
  "task_id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "queued",
  "stream_url": "/api/v1/tasks/123e4567-e89b-12d3-a456-426614174000/stream"
}
```

- 行为约束：
  - `kind` 当前支持 `parse_file`、`parse_archive` 两类。
  - `payload.filename` 必填；`payload.content_base64` 保存文件内容的 Base64 文本，用于让 `core-api -> RabbitMQ -> parser-service` 之间保持纯 JSON 任务载荷。
  - `parser-service` 消费 `parse.requested` 后，会把解析结果写回任务 `result`，其中至少包含 `source_content`，ZIP 课件包额外包含 `archive_summary`。
  - 当前前端默认让 `.zip` 课件包和 `50MB` 以上的普通单文件走该异步路径；`50MB` 及以下普通单文件解析继续保留 `/api/v1/project/parse` 同步接口作为兼容与回滚路径。

### 4.6.3 创建 PDF 导出任务
- 请求头：`Authorization: Bearer <token>`
- 请求体最小结构：

```json
{
  "kind": "export_pdf",
  "payload": {
    "blog_id": "123e4567-e89b-12d3-a456-426614174000"
  },
  "idempotency_key": "export-pdf:123e4567-e89b-12d3-a456-426614174000"
}
```

- 成功响应（`202 Accepted`）：

```json
{
  "task_id": "223e4567-e89b-12d3-a456-426614174000",
  "status": "queued",
  "stream_url": "/api/v1/tasks/223e4567-e89b-12d3-a456-426614174000/stream"
}
```

- 行为约束：
  - 当前 `kind` 仅支持 `export_pdf`。
  - `export-service` 消费 `export.requested` 后，会复用既有 Chromium PDF 导出能力生成文件，再把受控下载元数据写回任务 `result_json`。
  - `GET /api/v1/tasks/:id/download` 只接受 `status=succeeded` 的 `export` 任务；文件下载成功后会从 `EXPORT_ARTIFACTS_DIR` 共享目录中删除，避免产物长期堆积。

### 4.6.4 查询与取消任务
- `GET /api/v1/tasks/:id` 返回任务状态快照，典型字段包括：
  - `id`
  - `status`：`pending / queued / running / streaming / succeeded / failed / cancelled`
  - `task_type`、`task_subtype`
  - `result`
  - `error_message`
- `POST /api/v1/tasks/:id/cancel` 用于请求取消任务；第一阶段的取消语义为：
  - 队列中任务会被标记为 `cancelled`
  - 运行中任务依赖 worker 周期性检查取消状态后尽快停止

### 4.6.5 任务 SSE 订阅语义
- `GET /api/v1/tasks/:id/stream` 由 `core-api` 轮询 `job_task_events` 表并向前端输出标准 SSE。
- 当前阶段的典型事件：
  - `chunk`：正文或进度片段
  - `error`：任务流执行失败
  - `done`：任务完成，数据为 `[DONE]`
- 设计边界：
  - 对外仍是标准 `text/event-stream`
  - 对内不再要求前端直接命中 `llm-stream` 长连接；任务创建与状态查询统一经 `core-api`
  - 旧 `/api/v1/stream/*` 链路仍保留，作为任务式前端稳定前的兼容路径

## 5. 知识漫游复习模块 (ReviewAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/review/today` | GET | 获取今日推荐复习题卡 | JWT Bearer Token |
| `/api/v1/review/pick` | POST | 手动随机抽一篇可复习文章 | JWT Bearer Token |
| `/api/v1/review/notes` | GET | 获取可手动选择复习的文章列表 | `query`, `series_title`, `page`, `page_size` |
| `/api/v1/review/history` | GET | 获取最近复习记录摘要 | `limit` |
| `/api/v1/review/sessions` | POST | 创建一次复习会话 | `{ note_path, mode, entry_type }` |
| `/api/v1/review/sessions/:id` | GET | 获取复习会话当前状态与轮次 | 路径参数 `id` |
| `/api/v1/review/sessions/:id/respond` | POST | 提交一轮回答并推进会话 | `{ answer }` |
| `/api/v1/review/sessions/:id/hint` | POST | 请求一条提示 | `{}` |
| `/api/v1/review/sessions/:id/finish` | POST | 主动结束当前训练 | `{}` |

### 5.1 题卡与候选列表字段
- `GET /api/v1/review/today` 与 `POST /api/v1/review/pick` 返回：
  - `note_path`: Obsidian 笔记路径
  - `title`: 题卡标题
  - `source_title`: 所属来源或系列标题
  - `review_reason`: 推荐原因
  - `estimated_minutes`: 预估耗时
  - `available_modes`: 可选训练模式数组（`light_recall` / `detailed_qa`）
- `GET /api/v1/review/notes` 返回：
  - `items[].note_path`
  - `items[].title`
  - `items[].source_title`
  - `items[].last_reviewed_at`
  - `items[].preferred_mode`
  - `total`, `page`, `page_size`

### 5.2 会话与反馈字段
- `POST /api/v1/review/sessions`、`GET /api/v1/review/sessions/:id` 返回：
  - `session_id`, `status`, `mode`, `title`
  - `opening_prompt`: 开场提问
  - `initial_hints`: 初始提示列表
  - `session_outline.summary`: 当前文章的复习摘要
  - `session_outline.main_question`: 当前文章主问题
  - `session_outline.core_concepts / process_steps / application_cases / checkpoints`: 会话提炼出的文章关键点
  - `current_round_goal`: 当前这一轮最应该完成的回答目标
  - `latest_review_feedback`: 最近一轮回答的结构化判定（仅会话详情接口在有回答后返回）
  - `next_question`: 下一轮问题（可选）
  - `turn_index`: 当前轮次
  - `turns[]`: 已落库的轮次记录（仅会话详情接口返回）
- `POST /api/v1/review/sessions/:id/respond` 返回：
  - `session_id`, `session_status`, `turn_index`
  - `stage_feedback`: 当前阶段反馈（可选）
  - `current_round_goal`: 下一轮或当前轮的目标提示
  - `review_feedback.judgement`: 当前回答的判定（如 `答对较多 / 部分答对 / 偏题`）
  - `review_feedback.hit_points`: 当前回答已命中的文章关键点
  - `review_feedback.missed_points`: 当前回答尚未覆盖的关键点
  - `review_feedback.suggestion`: 下一步补充建议
  - `next_question`: 下一轮问题（可选）
  - `completed`: 是否已结束
  - `final_feedback.summary / strengths / gaps / next_focus`
- `POST /api/v1/review/sessions/:id/hint` 返回：
  - `session_id`, `hint_text`, `remaining_hint_count`
- `POST /api/v1/review/sessions/:id/finish` 返回：
  - `session_id`, `session_status`
  - `final_feedback.summary / strengths / gaps / next_focus`

### 5.3 复习枚举约束
- `entry_type`：
  - `today`：今日推荐入口
  - `manual_random`：手动随机抽题入口
  - `manual_select`：手动选文入口
- `mode`：
  - `light_recall`：轻提示复述
  - `detailed_qa`：细致问答
- `status`：
  - `created`
  - `in_progress`
  - `completed`
  - `abandoned`

## 6. 博客管理模块 (BlogAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/blogs` | GET | 获取用户的博客历史列表 (含系列结构) | 无 |
| `/api/v1/blogs/draft` | POST | 创建一篇手写草稿（顶级单篇，便于进入编辑器手写写作） | 无 |
| `/api/v1/blogs/:id` | PUT | 更新博客内容 (标题、内容等) | `{ title, content }` |
| `/api/v1/blogs` | DELETE | 批量删除博客 | `{ blog_ids: [] }` |
| `/api/v1/blogs/:id/export` | GET | 将系列博客或单篇博客导出为 Markdown Zip 包 | 无 -> application/zip |
| `/api/v1/blogs/:id/export/pdf` | GET | 将系列博客导出为合并 PDF（封面 + 目录 + 正文，无页码） | 无 -> application/pdf |
| `/api/v1/blogs/:id/export/obsidian` | POST | 将单篇博客导出到本地 Obsidian Vault（通过 Obsidian Local REST API） | 无 -> JSON `{ code: 200, message: "success" }` |
| `/api/v1/blogs/:id/export/obsidian/series` | POST | 批量同步系列到 Obsidian（遵循 Karpathy LLM Wiki Pattern：生成 sources/concepts/entities 并更新 index/log/hot；通过 Obsidian Local REST API） | 无 -> JSON `{ code: 200, message: "success" }` |

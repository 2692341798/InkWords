# 墨言知识训练平台 (InkWords Trainer) - 架构设计与工程规范

## 0. 变更记录
- 2026-06-08：执行一次文档与仓库治理同步：刷新 `README.md`、补充 `docs/superpowers/*` 设计/计划文档，并移除仓库内旧的 `CODE_OF_CONDUCT.md` 与 `skills/llm-wiki-ingest/*` 文档资产；本次不改变 Docker Compose 多服务形态、Nginx 单入口、服务拆分边界或任何运行时架构约束。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第十六轮。`internal/domain/blog/series_persistence.go` 现在在 `SaveSeriesIntro()` 与 `MarkSeriesIntroFailed()` 中显式按 `user_id + parent_id` 更新系列父稿；`DecompositionService.generateSeriesIntro()` 也同步透传当前用户，避免跨用户改写他人的系列导读正文或失败状态。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第十五轮。`internal/domain/blog/series_persistence.go` 现在在 `LoadSeriesOldContent()` 中显式按 `user_id + blog_id` 读取旧正文；`DecompositionService` 的 regenerate 路径也同步透传当前用户，避免跨用户读取他人的历史章节内容。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第十四轮。`internal/domain/blog/series_persistence.go` 现在在 `EnsureSeriesParentAndDrafts()` 中显式校验父稿归属用户；若传入的 `parent_id` 指向其它用户的系列父稿，将直接返回错误而不是继续在其下创建当前用户的章节草稿，避免跨用户系列树挂接。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第十三轮。`internal/service` 已删除最后的 `Chapter` 本地兼容别名，service 包内部相关生成、提示词、质量门禁与测试代码均改为直接依赖 `blogcontracts.Chapter`；至此 blog contracts 的 `GeneratedBlogPersistence / ContinuePersistence / SeriesPersistence / SeriesDraftPreflightInput / SeriesChapterPersistenceInput / Chapter` 在 service 层的兼容桥接已全部清空。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第十二轮。`internal/service` 已删除 `decomposition_series_persistence.go`；`DecompositionService` 与其持久化辅助逻辑现在直接依赖 `blogcontracts.SeriesPersistence`、`blogcontracts.SeriesDraftPreflightInput`、`blogcontracts.SeriesChapterPersistenceInput` 以及 `blogdomain.NewSeriesPersistence(db.DB)`。service 层剩余主要兼容层已进一步收缩为 `Chapter` 本地别名。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第十一轮。`internal/service` 已删除 `generator_persistence.go` 与 `decomposition_continue_persistence.go` 两层最薄的 blog bridge；`GeneratorService` 与 `DecompositionService` 现分别直接依赖 `blogcontracts.GeneratedBlogPersistence`、`blogcontracts.ContinuePersistence` 和 `blogdomain` 默认适配器，service 层剩余兼容桥接进一步收缩为 `series` 与 `Chapter` 相关类型。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第十轮。`backend/internal/domain/stream/service.go` 已不再依赖 `internal/service.Chapter` 兼容别名，改为直接组装 `internal/domain/blog/contracts.Chapter`；当前非 `service` 包对 `GeneratedBlogPersistence / ContinuePersistence / SeriesPersistence / Chapter` 等 blog contracts 兼容别名的显式引用已清零，service 层进一步退化为内部兼容桥接层。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第九轮。新增中立契约包 `internal/domain/blog/contracts`，先承接 `ErrSeriesNotFound` 与 blog persistence 的输入/接口定义；`domain/blog` 适配器现已直接依赖契约包而不再 import `internal/service`，service 层退化为类型别名与兼容构造器。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第八轮。`SeriesPersistence` 的默认生产适配器也已迁入 `internal/domain/blog`，并在 `llm-stream`、`core-api` 与聚合调试入口通过 bootstrap 显式注入；至此 `GeneratedBlogPersistence / ContinuePersistence / SeriesPersistence` 三类默认 blog 写入适配器已统一由 blog-domain 提供，service 层主要保留接口定义与测试替身。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第七轮。默认生产适配器开始真正下沉到 `internal/domain/blog`：`GeneratedBlogPersistence` 与 `ContinuePersistence` 新增 blog-domain GORM 适配器，并在 `llm-stream`、`core-api` 与聚合调试入口通过 bootstrap 显式注入；当前仍留在 `internal/service` 的主要默认适配器只剩 `SeriesPersistence`。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第六轮。`DecompositionService` 与 `GeneratorService` 的默认 GORM persistence 现在统一只在构造阶段补齐，业务方法内的隐式 `nil -> GORM` fallback 已删除；当前剩余技术债进一步收缩为“这些默认适配器是否继续并入 `domain/blog` 或服务私有 repository”，而不是继续在 service 方法里兜底装配。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第五轮。`DecompositionService` 已把系列父稿创建、章节草稿预建与父稿来源更新也收口到 `SeriesPersistence`，至此该 service 主流程中的主要博客持久化读写都已改为经由显式 persistence 边界完成；当前生产代码中保留的 `db.DB` 只剩默认 GORM fallback 构造路径，不再承担业务主逻辑直写角色。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第四轮。`DecompositionService` 进一步把系列重写中的旧正文读取与 `skip` 章节标题/排序更新收口到 `SeriesPersistence`；当前剩余待收口点进一步缩小为系列父稿创建、章节草稿预建与父稿来源更新。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第三轮。`DecompositionService` 新增显式 `ContinuePersistence` 边界，把 `continue` 链路中的旧正文读取与最终正文更新从主 service 逻辑里抽离到默认 GORM 适配器；当前剩余待收口点收缩为系列父稿/章节草稿前置准备、`skip` 元信息更新与旧内容读取。
- 2026-06-05：继续推进 `core-api / llm-stream` 深拆分第二轮。`DecompositionService` 新增显式 `SeriesPersistence` 边界，先把“系列章节完成/失败 + 系列导读完成/失败”这批最终业务事实写入从主 service 逻辑里抽离到默认 GORM 适配器；现阶段仍保留前置草稿准备、旧内容读取、`skip` 章节元信息与 `continue` 正文读写的直连数据库路径，作为下一轮收口对象。
- 2026-06-04：为本地环境端口冲突增加 `FRONTEND_PORT` 端口覆盖能力。`docker-compose.yml` 中 `frontend` 端口映射由固定 `80:80` 调整为 `${FRONTEND_PORT:-80}:80`，默认生产/标准本地入口仍是 `http://localhost`；当宿主机 `:80` 被其它进程占用时，可临时以 `FRONTEND_PORT=8088` 方式在 `http://localhost:8088` 验证前端 Nginx 与 `/api/v1/ping`。
- 2026-06-04：Generation Task-Only Task 4 打通 `generate_series` 系列链路的结果收集、任务结果交接与父子博客持久化。`llm-stream` 现会在系列生成完成后把父博客导读、章节成功/失败状态、章节正文与技术栈汇总为结构化 `job_tasks.result_json`；`core-api` 在 generation 成功路径中消费该结果，并以事务方式更新系列父博客与章节草稿。
- 2026-06-04：Generation Task-Only Task 3 打通 `continue` 续写链路的结果交接与最终持久化。`llm-stream` 现会在续写成功后输出带 `blog_id / appended_content / final_content` 的结构化 `job_tasks.result_json`；`core-api` 在 generation 成功路径中消费该结果，依据 `final_content` 更新目标博客正文，并统一累计 `users.tokens_used`。
- 2026-06-04：Generation Task-Only Task 2 打通 `core-api` 对单篇 `generate_single` 任务结果的最终持久化闭环。`core-api` 现通过服务自有 `generation_result_repository` 消费结构化 `job_tasks.result_json`，在 generation 任务成功路径中把单篇正文落回 `blogs` 并累计 `users.tokens_used`；`llm-stream` 仍继续只负责生成执行、事件流与任务结果产出。
- 2026-06-04：Generation Task-Only Task 1 落地单篇生成结果 contract。`llm-stream` 当前在 `generate_single + INKWORDS_TASK_PERSISTENCE_MODE=task_only` 下不再把任务成功固定写成 `{"done":true}`，而是输出带 `result_version / task_type / task_subtype / persistence_mode / final_status / usage / payload` 的结构化 `job_tasks.result_json`；本轮仅覆盖单篇生成结果交接，对外入口、SSE 路径与 `core-api` 最终业务落库职责暂未改变。
- 2026-06-04：Task 3 将 `GeneratorService` 对 `blogs / users` 的最终写入收口到显式 `GeneratedBlogPersistence` 接口，并新增 `generator_persistence.go` 作为默认 GORM 适配器。当前生成链路仍保持 legacy 落库行为与 `INKWORDS_TASK_PERSISTENCE_MODE=task_only` 开关不变，但 `GeneratorService -> GeneratedBlogPersistence -> GORM` 的依赖边界已可被测试替换，为后续继续把业务事实回收到 `core-api` 铺路。
- 2026-06-04：Phase 2 执行 `core-api / llm-stream` 深层拆分第一轮。`core-api` 与 `llm-stream` 均新增服务自有 `app/bootstrap`、`transport/http/v1` 与 `cmd` 入口，`backend/Dockerfile` 也切换为从 `backend/services/core-api/cmd` 与 `backend/services/llm-stream/cmd` 构建；共享 `internal/transport/http/v1/routes.go` 与 `internal/transport/http/v1/api/stream_api.go` 被标记为过渡兼容层。与此同时，为后续停止 `llm-stream` 直写 `blogs / users`，仓库新增 `services/llm-stream/domain/generation` 骨架、`services/core-api/domain/task/ResultPersister` 抽象，并通过 `INKWORDS_TASK_PERSISTENCE_MODE=task_only` 开关开始收紧 legacy 生成链路写库边界。
- 2026-06-04：Task 6 收尾同步文档并完成最终 Docker Compose 冒烟验证。确认 `parser-service`、`review-service`、`export-service` 的服务私有入口与装配均已分别收口到 `backend/services/<service>/`，对外入口仍保持 `http://localhost` 与既有 `/api/*` 路径不变；执行 `docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build` 后，`core-api / llm-stream / parser-service / export-service / review-service / frontend` 均恢复 `healthy`，`curl -I http://localhost` 与 `curl http://localhost/api/v1/ping` 通过。
- 2026-06-04：Task 4 完成 `export-service` 服务目录归属迁移。`export-service` 的 bootstrap、入口、导出适配器、私有路由、RabbitMQ consumer 与 artifact store 现统一归属到 `backend/services/export-service/`；本次不改变导出链路行为、对外 API、任务中心协议或数据库读写边界。
- 2026-06-03：Task 4 补齐“服务写入归属矩阵”。明确 `core-api` 事实拥有 `users / oauth_tokens / user_prompt_settings / blogs / job_tasks / job_task_events`，`review-service` 事实拥有 `review_sessions / review_turns`；同时把当前允许的跨服务写入例外文档化为“仅通过显式 repository / task service 写 `job_tasks / job_task_events`”。其中 `GeneratorService` 已进一步收口为显式 `GeneratedBlogPersistence` 边界，`DecompositionService` 仍保留直接全局 `db.DB` 写 `blogs / users` 的过渡性技术债。
- 2026-06-03：Task 6 继续推进 `parser-service` 异步化。`core-api` 新增 `POST /api/v1/tasks/parse`，用于创建 `parse_file / parse_archive` 任务并发布到 RabbitMQ `parse.requested`；`parser-service` 新增 parse worker consumer，消费后把解析结果写回 `job_tasks.result_json`。前端当前让 `.zip` 课件包与 `50MB` 以上普通单文件默认走任务式解析，`50MB` 及以下普通单文件仍保留同步 `/api/v1/project/parse` 作为兼容路径。
- 2026-06-03：Task 3 为 `core-api / llm-stream / parser-service / export-service / review-service` 补齐统一运行契约：各服务统一接入 `X-Request-ID` 中间件、结构化请求日志（`service / request_id / path / method / status / latency_ms`），并新增 `/health` 与 `/ready` 端点；Docker Compose 为 5 个后端服务与前端增加 healthcheck，前端启动依赖改为等待各后端 `healthy`，降低“容器已启动但接口未就绪”的误判。
- 2026-06-03：Task 2 将“任务式 SSE”收口为默认生成主链路。前端的单篇生成、系列生成、继续生成、润色统一改为“先创建 generation task，再订阅 `/api/v1/tasks/:id/stream`”；`llm-stream` 的 task consumer 也扩展支持 `continue / polish` 两类任务。旧 `/api/v1/stream/*` 与 `/api/v1/blogs/:id/(continue|polish)` 仍保留为兼容回滚路径。
- 2026-06-03：生成链路进入 RabbitMQ 事件驱动 Phase B。Compose 新增 `rabbitmq` 服务，`core-api` 与 `llm-stream` 通过 `RABBITMQ_URL / RABBITMQ_EXCHANGE / RABBITMQ_GENERATION_QUEUE` 接入任务队列；对外入口仍保持 `http://localhost` 与 `/api/*` 不变，`core-api` 负责创建任务、查询任务与基于 `job_task_events` 输出 SSE，`llm-stream` 负责消费生成任务并写回任务事件。
- 2026-06-03：澄清生产形态为 Docker Compose 多服务 + 前端 Nginx 单入口；`backend/Dockerfile` 不再构建/复制 `server` 二进制，镜像默认 CMD 调整为运行 `core-api`；`cmd/server` 明确为本地/集成调试聚合入口。
- 2026-06-03：Docker 微服务化 Phase 2（已落地到代码与编排）。后端在 Compose 中进一步拆分为 `core-api` / `llm-stream` / `parser-service` / `export-service` / `review-service`，对外入口仍为 `http://localhost`，由前端 Nginx 按路径分流；review 数据迁移与拆库（同 Postgres 实例、不同 database：`inkwords_review_db`）需按 Runbook 执行：[review-db-migration.md](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/docs/runbooks/review-db-migration.md)。
- 2026-06-03：Docker 微服务化 Phase 1。对外入口仍为 `http://localhost`，前端 Nginx 继续代理 `/api/*`；后端在 Docker Compose 中拆分为 `core-api` 与 `llm-stream` 两个服务，支持仅扩容 `llm-stream` 承接流式生成压力。
- 2026-06-03：收紧后端生命周期与请求取消边界。`backend/cmd/server/main.go` 不再使用裸 `r.Run`，改为显式 `http.Server`（`ReadTimeout=15s`、`ReadHeaderTimeout=10s`、`WriteTimeout=0`、`IdleTimeout=60s`）并接入 `SIGINT/SIGTERM` 优雅停机；`internal/domain/stream/handler.go` 收回 `Continue/Analyze/Scan` 主链路里默认的 `context.WithoutCancel`，改为直接透传 `c.Request.Context()`，让客户端断连/请求取消可以真实传递到分析、扫描和续写任务。
- 2026-06-02：优化后端 Docker 构建稳定性。`backend/Dockerfile` 不再强制把 Alpine 软件源切到阿里云镜像，改为直接使用默认 `dl-cdn.alpinelinux.org`，原因是当前运行环境下默认 CDN 首包明显更快；经 `docker compose --env-file backend/.env down && up -d --build` 实测，整套重建耗时约 48.47 秒，后端最重的 Chromium/字体/PDF 运行时依赖层约 16.1 秒完成，不再出现“像卡住”的长时间停顿。
- 2026-06-02：修复系列文章生成失败时前端只显示 `Error` 的可观测性缺陷。前端 `streamStore` 新增 `chapterErrors`，生成进度面板与侧边栏任务区可直接显示每章/系列导读的失败原因；后端 `stream` handler 为生成类 SSE 通道增加缓冲并在每次写事件后主动 `flush`，降低慢客户端导致的流式背压与误超时风险。
- 2026-06-01：文件来源 Analyze 新增“动态提示词 profile 锁定”链路。后端在 `stream/analyze(file)` 阶段先做轻量内容分类并返回 `resolved_prompt_profile`，前端在大纲阶段展示“当前提示词类型”只读标签；后续 `stream/generate`（单篇/系列/导读）统一透传并沿用同一个 profile，避免 Analyze 与 Generate 语义漂移。
- 2026-06-01：知识漫游复习从“固定模板问答”升级为“文章驱动的结构化追问”；后端 `review` 会在建 session 时提炼 `session_outline`，并在回答阶段返回结构化 `review_feedback`，前端 `ReviewSessionCard` 新增“本轮目标 / 你答到的点 / 你还漏掉的点 / 下一步建议”展示区。
- 2026-05-29：工程化结构拆分 Phase 1 落地：review 领域 service 按题卡/历史/会话职责拆成同包多文件，`Sidebar` 拆成 shell、批量操作条、选择 hook 与导出 service，生成链路的 decomposition 辅助逻辑拆为更小文件；本次不新增 API 路由或数据库结构。
- 2026-05-29：知识漫游复习入口最终收敛为“随机抽一篇 + 手动选文”两张卡片；前端主入口不再展示“今日推荐”卡片，并通过随机候选重试避免“再抽一篇”频繁返回同一文章。
- 2026-05-29：生成器工作流从“四步 + 独立处理页”收敛为“三步 + 步内进度”模型；`generatorViewState` 顶层阶段只保留 `source / configure / outline`，`GeneratorStatus` 退化为可复用的内嵌进度面板，分别挂载到 `GeneratorConfigureStage` 与 `GeneratorOutlineStage`。文件上传链路拆分为“先 parse、再在配置页显式触发 analyze”，避免 ZIP/课件上传后直接跳入独立进度页并倒序回到大纲。
- 2026-05-28：Docker Compose 部署基线加固：显式声明 `inkwords-network`，默认仅暴露前端 `http://localhost`，后端/Redis/PostgreSQL 改为容器内互通；`OBSIDIAN_VAULT_PATH` 不再回退到机器私有绝对路径，需由 `backend/.env` 或外部环境显式提供。
- 2026-05-27：项目定位升级为“墨言知识训练平台（InkWords Trainer）”，口号“把资料变成知识，把知识变成能力”；文档口径同步更新（不涉及架构与实现层面的强制改造）。
- 2026-05-27：前端入口改为 `HomeEntry` 引导式工作台；生成器与知识复习页统一采用“单次只显示当前主步骤”的流程编排，并抽取共享 `StepStrip` 组件（`preview` / `progress` 双变体）承载首页预览和页内进度条。
- 2026-05-27：新增“知识漫游复习”主链路；后端引入独立 `internal/domain/review` 垂直切片与 `review_sessions` / `review_turns` 持久化模型，前端新增独立主视图“知识漫游复习”，承接今日推荐、随机抽题、手动选文、会话追问、提示与最近记录。
- 2026-05-29：收敛“知识漫游复习”入口，前端移除与随机抽题职责重复的“今日推荐”卡片，保留“随机抽一篇 / 选择文章复习”两种入口；后端 `review` 随机选题改为真正随机而非固定返回首个候选。
- 2026-05-25：将本地文档上传链路上限从 100MB 提升到 888MB；前端上传页校验、前端 Nginx `client_max_body_size` 与后端 Gin `MaxMultipartMemory` 三层限制保持一致。
- 2026-05-25：在后端 LLM 公共流式出口新增“正文净化”层，并在前端润色正文应用前增加兜底清洗；统一剥离 `<think>` 思考标签、`reasoning_content` 和开头的对话式前言，防止 AI 思考/套话进入正文。
- 2026-05-25：修复前端“创作场景”在文件上传与大纲生成阶段的状态漂移；`streamStore.setSource` 不再因来源切换覆盖用户手动场景，文件 Analyze 请求统一从最新 store 读取 `scenario_mode`，生成器在 `outline` 出现后隐藏场景选择区，并在大纲头部显示只读“当前创作场景”标签。
- 2026-05-25：修复系列生成“父级成功、子级缺失”链路；后端 `GenerateSeries` 先创建章节草稿再异步流式回填，确保系列树即使在章节流式失败时也能保留子节点结构；前端 `Sidebar` 在自动选中系列父节点或子节点时同步展开历史树，避免用户误判为“只生成了导读”。
- 2026-05-24：新增 `scenario_mode` 场景切换能力；后端引入独立场景枚举与默认 Prompt 约束，分析与生成链路统一透传，前端生成器增加“创作场景”中文卡片入口。
- 2026-05-21：本地文件解析链路新增 ZIP 课件包聚合能力；后端 `project parse` 在保留单文件解析的同时，新增 ZIP 安全解压、白名单筛选、文本去重与 `archive_summary` 摘要返回，前端生成器支持 `.zip` 上传并展示解析摘要。
- 2026-05-21：修复 Docker 开发态 Obsidian 证书挂载兜底错误；移除将宿主机 `/etc/hosts` 作为证书文件的错误回退，改为默认通过 `OBSIDIAN_REST_API_INSECURE_SKIP_VERIFY=true` 访问本地 Obsidian Local REST API，并用测试锁定 compose 配置。
- 2026-05-21：修复文件上传解析链路的来源判定漂移；前端上传后统一从 `project/parse` 提取 `data.source_content`，并在调用 `stream/analyze` 时显式发送 `source_type=file`；后端 `stream` handler 增加基于 `source_content` 的兼容推断，避免旧静态资源或缓存请求被误判为 Git 分析。
- 2026-04-29：对后端 `parser`/`service` 与前端核心组件进行模块化拆分，消除超大文件（>500 行），以提升可维护性与复用性（无业务行为变更）。
- 2026-04-29：新增“手写博客入口”，支持创建草稿并直接进入现有双栏编辑器（新增后端草稿创建接口与前端侧边栏入口）。
- 2026-05-08：写博客编辑器新增“语音输入”（浏览器 SpeechRecognition 实时转写，插入正文光标处）。
- 2026-05-08：写博客编辑器新增“润色”（后端新增 `/api/v1/blogs/:id/polish` SSE；前端新增润色预览与一键应用），并优化 Markdown 预览标题/表格排版。
- 2026-05-08：工程化整理：移除仓库中的大二进制/调试产物追踪（避免泄漏 token），并将超大文件按职责拆分为同包多文件/子组件目录。
- 2026-05-08：目录结构工程化调整（目标态设计）：后端明确 `domain/transport/infra` 边界，前端明确 `pages/services` 边界（仅规划，不改代码）。
- 2026-05-10：修复导出到 Obsidian 初始化 scaffold 时的目录列表解析失败（兼容 Obsidian Local REST API `{ \"files\": [...] }` 返回格式）。
- 2026-05-10：提示词“写作要求”抽离为文章类型模板（内置默认 + 用户自定义覆盖），生成接口支持 `article_style`，编辑器新增“模板管理”入口。
## 1. 整体架构 (Monorepo)
项目采用前后端分离的 Monorepo 结构，根目录隔离：
- **`frontend/`**: 包含所有前端界面、状态管理和客户端逻辑。
- **`backend/`**: 包含所有的 RESTful API 服务、数据库交互、第三方登录与大模型通信。
- **`docker-compose.yml`**: 项目唯一的容器化编排入口。

## 1.1 目录结构目标态（规划）
目录结构的目标态与迁移路线图见：[2026-05-08-project-directory-structure-design.md](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/docs/superpowers/specs/2026-05-08-project-directory-structure-design.md)。

## 2. 核心技术栈
### 2.1 前端 (Frontend)
- **核心框架**: React + Vite
- **UI 库**: Tailwind CSS + Shadcn UI + Recharts
- **状态管理**: Zustand (含多 store：`blogStore`, `streamStore`, `authStore`, `reviewStore`)
- **流式通信**: `@microsoft/fetch-event-source` 维持 SSE 连接
- **Markdown 渲染**: `react-markdown` 配合 `rehype-highlight`、`remark-gfm` 和 `mermaid`。
- **场景锁定策略**：生成器在大纲生成前展示“创作场景”卡片；一旦 `outline` 存在，页面立即隐藏场景选择区，并在大纲头部展示只读场景标签，保证 Analyze 与后续 Generate 使用同一场景语义。
- **提示词类型锁定策略**：当文件 Analyze 返回 `resolved_prompt_profile` 后，前端 `streamStore` 会持久化 `resolvedPromptProfile/classificationStatus/classificationReason`，并在大纲区展示“当前提示词类型”只读标签；后续单篇/系列生成请求统一透传 `prompt_profile_key` 与 `document_kind`。
- **流程型工作台编排**：默认入口为 `HomeEntry`；`Generator`、`KnowledgeReview`、`HomeEntry` 三处共享 `StepStrip` 展示流程预览/进度，但业务状态仍保留在页面层，通过 `generatorViewState`、`knowledgeReviewViewState`、`homeEntryViewState` 做纯前端编排，避免把共享 UI 组件耦合成全局状态机。
- **生成器三步模型**：`Generator` 当前固定为 `选择来源 -> 配置解析 -> 确认大纲` 三步；解析/分析时仍停留在 `configure` 并内嵌展示 `GeneratorStatus`，正文生成时仍停留在 `outline` 并内嵌展示章节进度，`progress` 不再是顶层页面阶段。
- **系列章节质量阶段状态**：`streamStore` 额外维护 `chapterPhases` 与 `chapterUsage`，`useSeriesGenerator.handleSeriesChunkMessage()` 负责把后端 `understanding / drafting / reviewing / revising / streaming / usage / completed / error / retrying` 事件统一映射为前端运行时状态，`GeneratorStatus` 只消费 store 而不直接理解 SSE 协议细节。
- **知识漫游复习工作台**：新增 `KnowledgeReview` 主视图，入口位于侧边栏；同一页面内收敛“随机抽一篇 / 选择文章复习 / 当前会话 / 最近记录”四类状态，避免在多个页面间来回跳转。

### 2.2 后端 (Backend)
- **核心语言**: Go 1.25+
- **Web 框架**: Gin (`github.com/gin-gonic/gin`)
- **依赖注入**: 后端通过明确的构造函数（如 `NewAuthAPI(authService)`）进行依赖注入，降低 `api` 层和 `service` 层、全局变量之间的耦合，便于单元测试。
- **目录升级（渐进式垂直切片）**: 新增 `internal/domain/blog`、`internal/domain/user`、`internal/domain/auth`、`internal/domain/stream`、`internal/domain/project`、`internal/domain/review` 作为领域切片（repo/service/handler），并在 `cmd/server/main.go` 统一完成依赖组装（repo -> service -> handler -> api 适配）。
- **服务自有入口收口（Phase 2 当前态）**：
  - `core-api` 入口、bootstrap 与私有路由现位于 `backend/services/core-api/`。
  - `llm-stream` 入口、bootstrap 与私有路由现位于 `backend/services/llm-stream/`。
  - 共享 `internal/transport/http/v1/routes.go` 与 `internal/transport/http/v1/api/stream_api.go` 仅保留为过渡兼容层，不再承担这两个服务的主入口装配职责。
- **数据库 ORM**: GORM (`gorm.io/gorm` + `gorm.io/driver/postgres`)
- **认证与安全**: 
  - JWT Token (长短效签发) + GitHub OAuth (`golang.org/x/oauth2`)
  - 图形验证码防刷 (`github.com/mojocn/base64Captcha`)
  - 密码强度与连续登录失败防爆破锁定 (`LockedUntil`)
- **并发架构**: 引入了 Go 原生的 Goroutine 池与 `x/sync/semaphore` 信号量控制（动态范围 3~8），保障并发生成稳定且不超限。
- **提示词模板化**：新增 `internal/prompt` 作为提示词约束的单一来源；当前将 `scenario_mode`（任务场景）与 `article_style`（写作风格）解耦，生成链路通过 `PromptRequirementsService` 统一合并场景默认约束、风格默认约束与用户覆盖值，并注入到单篇/系列章节 prompt 中（system 注入与安全约束仍由系统固定）。
- **动态 Prompt Profile 解析与锁定**：
  - 新增 `internal/prompt/prompt_profile.go` 作为 profile 单一来源，定义 `PromptProfile`、`ResolvedPromptProfile`、可选 profile 常量及按 `scenario_mode` 的回退策略。
  - 新增 `PromptProfileResolver`（`internal/service/prompt_profile_resolver.go`），在文件 Analyze 阶段调用轻量分类模型识别内容类型；当分类失败、返回非法 key 或 JSON 解析失败时，统一按场景回退。
  - `OutlineResult` 新增 `resolved_prompt_profile`，由 Analyze 返回给前端并在后续 Generate 全链路复用。
  - `PromptRequirementsService.ResolveWithProfile` 在场景/风格要求前追加 profile 级要求，确保“内容类型语义优先”。
- **场景默认兜底**：后端在 `internal/domain/stream/handler.go` 统一规范化 `scenario_mode`；当请求缺失或非法时，按来源类型回填默认值（`git -> beginner_walkthrough`，其它来源 -> `ebook_interpretation`），保证旧前端兼容。
- **特大型项目保护 (Map-Reduce)**:
  - **Map 阶段**: 按目录分块(针对 Git 仓库)或按字数智能段落分块(针对大于 1,000,000 字符的长文本文件)并发提炼局部摘要，当遇到 LLM 限流时启用带随机抖动的**指数退避 (Exponential Backoff)**。
  - **文件长文优化**: 超长 PDF/课件文件会改用更细粒度的 `120,000` 字符分块，并严格按原始顺序回收局部摘要，避免后半部分章节因块过大或并发错序而在大纲阶段被稀释。
  - **Reduce 阶段**: Git 仓库在局部摘要过多（>20个）时仍会触发 **Tree Reduce** 多级树状汇总；长文文件分析则直接保留全部局部摘要进入大纲阶段，优先保证章节覆盖率与“篇数不设上限”的拆解目标，整体输入最高支持 15,000,000 字符上限。
- **深度剖析与博客再生 (Deep Generation & Regeneration)**:
  - **思考模式与 JSON 模式**：通过调用 DeepSeek 的 `Thinking` 模式加强逻辑推理，并在大纲生成、技术栈提取等场景强制启用 `json_object` 模式配合极低的 `Temperature` (如 0.1) 确保严格的结构化稳定输出。
  - **上下文注入**：在 `regenerate` 更新重写阶段，系统会从数据库提取旧版博客（截断至 50万 字符）并注入 Prompt，指导大模型基于最新源码进行“松散参考重写”，有效避免优秀沉淀丢失。
- **文档/课件解析管线**：
  - 普通文件继续使用 `DocParser` 处理 `.pdf/.docx/.md/.markdown/.txt`。
  - PDF 采用“双通道提取”策略：优先使用 `ledongthuc/pdf` 直接提取；若检测到严重乱码或异常控制字符，则自动回退到运行时 `pdftotext`（来自 `poppler-utils`）恢复文本，避免将失真内容继续送入后续分析链路。
  - ZIP 课件包通过 `ArchiveParser` 进入单独管线：临时落盘 -> 安全解压 -> 白名单筛选 -> 文本提取 -> 规范化去重 -> 按路径顺序聚合为统一 `source_content`。
  - ZIP 成功响应会附带 `archive_summary`，供前端展示保留、去重、忽略与失败统计；解析完成后临时 ZIP 与解压目录均立即清理，保持“阅后即焚”。
- **知识漫游复习链路**：
  - `review` 领域通过 `NoteSource` 读取 Obsidian `wiki` 中可复习的 `concept` 页面，保持“正文来自知识库、业务状态来自 PostgreSQL”的边界。
  - 创建 session 时固化笔记标题、摘要、关键点与提示预算快照，并进一步提炼 `session_outline`（主问题、核心概念、步骤/场景、checkpoints），避免后续追问退化成固定模板。
  - `Respond` 不再只返回一段泛化鼓励文案，而是同时返回 `current_round_goal + review_feedback`，让前端可以显式展示“命中点 / 遗漏点 / 下一步建议”。
  - `cmd/server/main.go` 启动时统一组装 `review` 的 repository、service、handler，并注册 `/api/v1/review/*` 路由。
- **数据推送**: 基于标准 HTTP `text/event-stream` 实现 SSE 推送机制。
- **RabbitMQ 任务队列化生成**：
  - `core-api` 负责把生成请求转换为任务记录，并把消息投递到 `generation.requested`。
  - `llm-stream` 通过 RabbitMQ consumer 异步消费任务，复用既有生成服务执行业务逻辑。
  - 任务进度与正文 chunk 先落 PostgreSQL 事件表，再由 `core-api` 统一向前端输出 SSE，避免第一阶段引入跨服务内存总线。
- **parser-service 异步解析（Task 6 起步态）**：
  - `core-api` 新增 `/api/v1/tasks/parse`，负责创建解析任务并把消息投递到 `parse.requested`。
  - `parser-service` 新增 parse worker consumer，复用既有 `fileparse.Service` 执行文件/ZIP 解析，不再要求前端直连同步长请求才能完成重解析。
  - 当前阶段让 `.zip` 课件包与 `50MB` 以上普通单文件默认走任务化，以最小改动验证“非生成类任务”模型；`50MB` 及以下普通文件解析继续保留同步接口，避免一次性把所有上传链路都改成 Base64 任务载荷。
- **export-service 异步导出（Task 7 起步态）**：
  - `core-api` 新增 `/api/v1/tasks/export` 与 `/api/v1/tasks/:id/download`，负责创建导出任务并在完成后提供受控下载入口。
  - `export-service` 新增 export worker consumer，订阅 `export.requested`，复用现有 Chromium PDF 导出逻辑生成文件，再把 `file_token / filename / content_type / expires_at` 写回任务 `result_json`。
  - `export-service` 当前已把启动入口、装配、私有路由、consumer 与 artifact store 收口到 `backend/services/export-service/`，与 `parser-service`、`review-service` 一样采用服务自有目录承载运行时边界。
  - `core-api` 与 `export-service` 通过共享卷 `EXPORT_ARTIFACTS_DIR` 交换导出产物；下载成功后由 `core-api` 删除文件，避免共享目录无限增长。
- **正文净化层**：`internal/infra/llm` 在流式输出进入业务层前统一做开头段落清洗，剥离 `<think>` 标签、跳过 `reasoning_content`，并删除“收到你的需求 / 作为高级全栈架构师”等非正文前言；前端润色应用正文前再做一次兜底提取，避免污染 `blogs.content`。

## 2.2.1 服务写入归属矩阵（Task 4）

### 核心原则
- 生产形态虽然仍共享同一 Postgres 实例，但“共享实例”不再等于“任意服务都能随意写任意表”。
- 表的事实归属先以服务边界为准，再决定是否推进“同实例不同库”或独立实例拆分。
- 非归属服务若必须写入共享表，只允许通过显式 repository / task service 收口，禁止在业务 service 中直接操作全局 `db.DB`。

### 当前归属矩阵
| 表 / 资源 | 事实归属服务 | 当前允许写入方 | 说明 |
| --- | --- | --- | --- |
| `users` | `core-api` | `core-api` | 用户注册、登录失败计数、GitHub 绑定、Token 记账均由核心应用链路维护 |
| `oauth_tokens` | `core-api` | `core-api` | 第三方平台授权信息；当前仓库已有模型与迁移，但业务写入链路尚未重新启用 |
| `user_prompt_settings` | `core-api` | `core-api` | 用户写作模板覆盖值，由核心应用持久化 |
| `blogs` | `core-api` | `core-api` | 博客正文、系列父子关系、润色/续写落库都属于核心内容域 |
| `job_tasks` | `core-api` | `core-api`、`llm-stream`、`parser-service`、`export-service` | 事实归属仍是 `core-api`，但 worker 允许通过任务仓储更新状态和结果 |
| `job_task_events` | `core-api` | `core-api`、`llm-stream`、`parser-service`、`export-service` | 统一任务控制面；跨服务写入仅限事件回放所需的追加写 |
| `review_sessions` | `review-service` | `review-service` | 已拆到 `inkwords_review_db`，非 review 服务不应再直接写入 |
| `review_turns` | `review-service` | `review-service` | 与 `review_sessions` 同归属，保持会话和轮次同域管理 |

### 当前允许的跨服务写入例外
- `llm-stream`、`parser-service`、`export-service` 可以写 `job_tasks / job_task_events`，但前提是：
  - 只能通过 `internal/domain/task` 提供的显式 repository / service 接口。
  - 写入目标仅限任务状态、任务结果、可回放事件，不得借机扩展为任意业务表写入。
- `review-service` 使用独立 `REVIEW_DATABASE_URL` 后，不再允许其它服务绕过 review repository 写 `review_sessions / review_turns`。

### 当前已知技术债
- `GeneratorService` 已通过 `GeneratedBlogPersistence` 完成单篇生成最终写入的显式接口化，但默认实现仍是 GORM 适配器，尚未进一步并入 `domain/blog` 仓储边界。
- `DecompositionService` 已通过 `SeriesPersistence / ContinuePersistence` 收口系列前置草稿准备、章节完成/失败、导读完成/失败、旧正文读取、`skip` 元信息更新以及 `continue` 正文读取/最终更新；当前 service 主逻辑层面已基本不再直接承担博客事实的数据库读写。
- 当前真正剩余的技术债从“主流程仍有直连数据库”切换为“默认 GORM persistence 适配器后续是否继续归并到 `domain/blog` 仓储边界”；同时，默认生产装配与中立契约都已开始迁入 blog-domain，下一步更偏向是否继续把 service 层遗留的兼容别名与更细粒度仓储能力也一起收口。

### 退化与拆分判断
- 在 `blogs / job_tasks / job_task_events` 仍存在大量全局 `db.DB` 直接写之前，不推进真正的独立实例拆分。
- 只有当“表归属明确 + 跨服务写入接口化 + 回滚 Runbook 可执行”三项同时满足，才进入 Task 8 的下一阶段。

### 2.3 基础设施 (Infrastructure)
- **数据库**: PostgreSQL 14 (Docker volume 挂载持久化)
- **消息队列**: RabbitMQ 3 Management（当前用于 `generation.requested / parse.requested / export.requested` 三类任务投递；管理界面端口保留在容器内网，不对宿主机暴露）
- **本地知识库导出**: 后端通过 Obsidian Local REST API（HTTPS + API Key）写入用户本地 Vault，并遵循 Karpathy LLM Wiki Pattern 将系列批量 Ingest 为 `sources/`、`concepts/`、`entities/` 并自动编织双向链接网络，同时自动生成 `sources/_index.md`、`concepts/_index.md`、`entities/_index.md`、`domains/_index.md` 等“地图索引页”以避免知识孤岛与空页面；容器通过 sidecar `obsidian-bridge`（27125）转发访问宿主机插件端口（27124）
- **本地知识库复习输入**: Review 模块默认从 `OBSIDIAN_WIKI_DIR` 指向的 `wiki/` 根目录读取候选笔记；若 Obsidian Store 初始化失败，服务仍可启动，但 review 入口会返回稳定错误而不是让整个服务崩溃。
- **系列 PDF 导出**: 后端将系列 Markdown 渲染为 HTML（封面 + 目录 + 正文），并使用容器内 Chromium Headless 打印为 PDF。当前默认链路已切到“创建 export task -> export-service worker 生成 PDF -> `/api/v1/tasks/:id/download` 受控下载”，原同步 `/api/v1/blogs/:id/export/pdf` 继续保留为回滚路径。为保证中文正常显示，后端运行时镜像需安装 `chromium` 与 `font-noto-cjk` 等字体依赖。
- **代理与网关**: Nginx (构建前端静态页面并反向代理后端 `/api/` 路径，配置 `client_max_body_size 888M` 以支持大文件解析)
- **大语言模型**: DeepSeek-V4-Flash API (支持 128k 输出及 1M Token 上下文)
- **基础运行契约**:
  - 所有后端服务统一暴露 `/api/v1/ping`、`/health`、`/ready` 三个探针端点；`/ready` 默认至少检查数据库，`llm-stream` 额外检查 RabbitMQ 配置是否已注入。
  - 所有 HTTP 请求统一透传或生成 `X-Request-ID`，并在结构化请求日志中输出 `service / request_id / path / method / status / latency_ms`，作为多服务形态下的最小排障上下文。

## 3. 并发生成架构
在处理项目到系列博客的生成时，后端采取如下架构：
1. **模块扫描与场景选择**：针对 Git 源码，使用无盘 `ls-tree` 与 GitHub API 极速提取一级核心代码目录；前端生成器同时暴露“电子书解读 / 开卷复习 / 小白教程”三张中文场景卡片，由用户显式确定本次创作目标。
2. **Map-Reduce 分析**：针对选中的模块和超大文件（>5万字）进行预切片处理，并行调用 LLM 抽取各个分块摘要；其中超长文件会采用更细粒度切块并保持摘要顺序稳定，不再对文件摘要做二次 Tree Reduce 压缩；大纲阶段会读取 `scenario_mode`，分别偏向“篇章解读”“考点速查”或“学习路径”三类拆解方式。
3. **多协程并发生成**：
   - 接收到前端下发的大纲后，启动多个 `goroutine` 为每个章节并行生成内容。
   - 使用 `semaphore.NewWeighted(3)` 将全局并发数严格限制为 3。
   - 每个 `goroutine` 均拥有独立的错误隔离环境，通过同一个 `progressChan` 向前端推送包含自身 `chapter_sort` ID 的 Chunk（数据切片）。
  - 单篇生成、系列章节生成和系列导读生成都会复用同一份 `scenario_mode + article_style + resolved prompt profile` 组合后的 Prompt 约束，避免“大纲像心理学解读、正文回到通用技术博客”的割裂。
  - 系列章节当前内部执行顺序已升级为 `understanding -> drafting -> reviewing -> revising(streaming)`；只有终稿补强阶段会把正文 chunk 推给前端，前置阶段只发送状态事件。
  - 终稿补强的流式收尾阶段还会额外发送 `usage` 事件，携带 `prompt_tokens / completion_tokens / prompt_cache_hit_tokens / prompt_cache_miss_tokens`，用于衡量稳定前缀是否真正提升了同系列多章节的 DeepSeek 原生前缀缓存命中率。
4. **系列导读生成**：
   - 所有单篇博客生成完毕后（`wg.Wait()` 返回），主流程自动触发一次 AI 调用，生成“系列导读”文章，将其作为整个系列的父节点，将各个单篇博客串联成专栏。
5. **前端批量更新防卡顿**：
   - 前端接收到密集交织的 SSE Chunk 时，使用 `pendingUpdates` 缓冲队列进行暂存。
   - 通过 `setTimeout(200ms)` 的节流（Throttle）机制，将缓冲区内的文本批量合并，定期只触发一次 Zustand 状态更新和 React DOM 重绘。
   - 极大缓解了多章节 Markdown 同时渲染导致的主线程卡死。

## 3.1 `scenario_mode` 场景层设计
- **场景枚举**：
  - `ebook_interpretation`：面向电子书、长文、经典著作解读
  - `open_book_exam_review`：面向考试资料、课件、实验步骤速查
  - `beginner_walkthrough`：面向源码仓库、项目教程、小白上手路径
- **链路位置**：
  - 前端：`frontend/src/lib/scenarioMode.ts` 定义选项与中文描述，`frontend/src/pages/Generator.tsx` 提供交互入口。
  - 前端展示状态：`frontend/src/pages/generatorViewState.ts` 统一控制“何时显示场景选择区”和“何时显示只读场景标签”。
  - HTTP：`stream/analyze` 与 `stream/generate` 请求体新增 `scenario_mode`。
  - 后端：`internal/prompt/scenario_mode.go` 与 `default_scenario_requirements.go` 提供场景枚举和默认约束，`PromptRequirementsService` 统一做 Prompt 组装。
- **与动态 profile 的协同关系**：
  - `scenario_mode` 定义“任务目标层”（电子书解读 / 开卷复习 / 小白教程）。
  - `prompt_profile_key/document_kind` 定义“内容类型层”（如心理学经典解读、技术资料讲解）。
  - Analyze 负责锁定 profile，Generate 负责严格沿用 profile；二者都缺失时按 `scenario_mode` 兜底，保证兼容性与一致性。
- **兼容性策略**：
  - `scenario_mode` 非必填，旧请求继续有效。
  - 后端统一做默认值与非法值兜底，避免前端静态资源版本漂移时出现链路回归。
- **一致性策略**：
  - `streamStore.setSource` 仅更新来源信息，不再在来源切换时覆盖用户手动选择的场景。
  - 文件 Analyze 请求在发起时通过 `useStreamStore.getState()` 读取最新 `scenario_mode`，避免旧渲染快照导致“UI 显示 A、请求发送 B”。

## 3.2 RabbitMQ 事件驱动 Phase B
- **目标**：把生成链路从“前端直连长 SSE 请求”收口为“创建任务 -> 队列消费 -> DB 事件回放”的后台任务模型，同时保持前端单入口与 `/api/*` 路径不变。
- **主链路**：
  1. 前端向 `core-api` 提交 `/api/v1/tasks/generation` 创建任务。
  2. `core-api` 写入 `job_tasks`，并把消息发布到 RabbitMQ exchange `inkwords.events`。
  3. `llm-stream` 订阅 `inkwords.generation` 队列，消费生成任务并执行既有生成逻辑。
  4. Worker 把状态变化与 chunk 追加到 `job_task_events`。
  5. 前端再通过 `GET /api/v1/tasks/:id/stream` 从 `core-api` 订阅 SSE；`core-api` 轮询事件表并对外回放。
- **当前边界**：
  - 第一阶段优先使用 PostgreSQL 事件表作为跨服务事实来源，不额外引入 Redis pubsub。
  - 旧 `/api/v1/stream/*` 路由仍保留，作为兼容回滚路径。
  - 取消任务由 `core-api` 标记任务状态，worker 通过轮询取消状态尽快停止。
- **Task 6 扩展**：
  - 同一套 `job_tasks / job_task_events / RabbitMQ` 基础设施已开始复用于解析链路：`generation.requested` 之外，再增加 `parse.requested`。
  - `parser-service` 和 `llm-stream` 一样消费 MQ 任务，但执行体复用 `fileparse.Service`，结果直接回写任务 `result_json`，不要求解析链路也像生成链路一样产生大量 chunk 事件。

## 4. 部署架构 (Docker-First)
- **前端镜像**: 采用多阶段构建（Node.js 安装依赖并构建，Nginx 轻量级运行并作为反向代理网关）。默认仅映射宿主机 `80` 端口，统一以 `http://localhost` 作为前端入口。
- **后端镜像**: 采用多阶段构建（Go 官方镜像编译，Alpine 运行）。Phase 2 后端拆分为 `core-api` / `llm-stream` / `parser-service` / `export-service` / `review-service` 五个服务，对外入口仍保持不变：统一由前端 Nginx 网关按路径分流转发到对应服务；后端各服务默认只在 Docker 内部网络暴露 `8080`，不直接对宿主机开放。
- **PDF 运行时依赖**: 后端 Alpine 运行时镜像除 `chromium`、`font-noto-cjk` 外，还需安装 `poppler-utils`，以便 `DocParser` 在中文 PDF 提取失真时回退到 `pdftotext`。
- **数据库 / Redis**: PostgreSQL 与 Redis Stack 默认仅在容器网络内暴露，避免开发态无意开放宿主机调试端口。Phase 2 引入 review 拆库：同一 Postgres 实例新增 `inkwords_review_db`，`review-service` 使用 `REVIEW_DATABASE_URL` 连接；数据迁移与回滚步骤见 Runbook：[review-db-migration.md](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/docs/runbooks/review-db-migration.md)。
- **容器互联**: 全部服务显式加入 `inkwords-network` 内部网络，后端通过服务名 `db:5432`、`redis:6379`、`rabbitmq:5672`、`obsidian-bridge:27125` 与依赖互通。
- **健康探针**: `core-api / llm-stream / parser-service / export-service / review-service / frontend` 均在 Compose 中声明 `healthcheck`；前端依赖五个后端服务的 `service_healthy` 状态，避免 Nginx 先起来但后端还未 ready 的短暂空窗。
- **环境装载约定**: Docker Compose 运行时统一建议通过 `docker compose --env-file backend/.env ...` 启动；`OBSIDIAN_VAULT_PATH` 必须显式提供，不再回退到某台开发机的绝对路径。
- **任务队列环境变量**: `core-api`、`llm-stream` 与 `parser-service` 统一读取 `RABBITMQ_URL`、`RABBITMQ_EXCHANGE`；其中生成链路使用 `RABBITMQ_GENERATION_QUEUE`，解析链路新增 `RABBITMQ_PARSE_QUEUE`。默认值分别指向 `amqp://guest:guest@rabbitmq:5672/`、`inkwords.events`、`inkwords.generation`、`inkwords.parse`。
- **Task 6 冒烟验证补充**: 请直接使用 `docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build`；这样 Compose 会显式加载 `backend/.env`，避免因 `OBSIDIAN_VAULT_PATH` 等变量缺失而在解析 bind mount 时直接失败。

## 4.1 仓库产物与敏感信息策略
- 禁止提交构建产物与大文件（例如后端二进制、PDF、批量截图等），统一通过 `.gitignore` 管理本地产物目录。
- `dogfood-output/` 可能包含浏览器 localStorage 等敏感信息（例如 token），必须保持为本地目录，严禁进入 Git 追踪。

## 5. 全局缓存机制 (Prompt Caching)
- **目标**：降低 DeepSeek Token 消耗，提高首字响应速度 (TTFT)。
- **原生支持**：全面拥抱 DeepSeek V4 API 级别的原生前缀缓存 (Prompt Caching)。
- **Prompt 结构重构**：将数百万字的巨量源码 `sourceContent` 提取至 `system` 消息并置于请求最前，将易变的“指令”置于 `user` 消息并置于请求尾部，以最大化原生缓存的命中率，将长文本输入成本降低 80% 以上。
- **观测补强**：自 2026-06-01 的系列章节质量流水线 Task 4 起，后端会从 DeepSeek 非流式响应体与流式尾块中统一解析 `prompt_cache_hit_tokens / prompt_cache_miss_tokens`，并在系列章节终稿完成后把 usage 事件推给前端，避免“只设计缓存友好 Prompt，却无法验证是否真正命中”的黑盒状态。

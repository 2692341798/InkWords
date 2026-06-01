# 墨言知识训练平台 (InkWords Trainer) - 架构设计与工程规范

## 0. 变更记录
- 2026-06-01：系列章节质量流水线继续落地 Task 2。后端新增 `internal/service/series_quality_pipeline.go`，先抽出 `buildSeriesSharedPromptPrefix()` 作为系列级稳定前缀 builder，再实现 `parseSeriesChapterUnderstanding()` 与 `generateSeriesChapterUnderstanding()`，用于把“系列级固定规则前置、章节级变量后置”的 Prompt 结构固化下来；`decomposition_generate_prompt_helpers.go` 同步新增 `buildSeriesReaderProfile()`，为后续质量流水线统一生成读者画像。当前仍未切换 `GenerateSeriesWithProfile` 主链路与前端 SSE 协议。
- 2026-06-01：系列章节质量流水线开始落地 Task 1。后端在 `internal/service` 新增 `series_quality_pipeline_types.go`，统一定义 `SeriesChapterUnderstanding / Draft / Review / Final / Usage` 结构体，并在结构化输出进入后续阶段前增加硬门禁校验，先从类型边界拦截“缺机制解释、缺案例、缺修订动作”的空心结果；本次尚未切换 `GenerateSeriesWithProfile` 主链路和前端 SSE 协议。
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
- **知识漫游复习工作台**：新增 `KnowledgeReview` 主视图，入口位于侧边栏；同一页面内收敛“随机抽一篇 / 选择文章复习 / 当前会话 / 最近记录”四类状态，避免在多个页面间来回跳转。

### 2.2 后端 (Backend)
- **核心语言**: Go 1.25+
- **Web 框架**: Gin (`github.com/gin-gonic/gin`)
- **依赖注入**: 后端通过明确的构造函数（如 `NewAuthAPI(authService)`）进行依赖注入，降低 `api` 层和 `service` 层、全局变量之间的耦合，便于单元测试。
- **目录升级（渐进式垂直切片）**: 新增 `internal/domain/blog`、`internal/domain/user`、`internal/domain/auth`、`internal/domain/stream`、`internal/domain/project`、`internal/domain/review` 作为领域切片（repo/service/handler），并在 `cmd/server/main.go` 统一完成依赖组装（repo -> service -> handler -> api 适配）。
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
- **正文净化层**：`internal/infra/llm` 在流式输出进入业务层前统一做开头段落清洗，剥离 `<think>` 标签、跳过 `reasoning_content`，并删除“收到你的需求 / 作为高级全栈架构师”等非正文前言；前端润色应用正文前再做一次兜底提取，避免污染 `blogs.content`。

### 2.3 基础设施 (Infrastructure)
- **数据库**: PostgreSQL 14 (Docker volume 挂载持久化)
- **本地知识库导出**: 后端通过 Obsidian Local REST API（HTTPS + API Key）写入用户本地 Vault，并遵循 Karpathy LLM Wiki Pattern 将系列批量 Ingest 为 `sources/`、`concepts/`、`entities/` 并自动编织双向链接网络，同时自动生成 `sources/_index.md`、`concepts/_index.md`、`entities/_index.md`、`domains/_index.md` 等“地图索引页”以避免知识孤岛与空页面；容器通过 sidecar `obsidian-bridge`（27125）转发访问宿主机插件端口（27124）
- **本地知识库复习输入**: Review 模块默认从 `OBSIDIAN_WIKI_DIR` 指向的 `wiki/` 根目录读取候选笔记；若 Obsidian Store 初始化失败，服务仍可启动，但 review 入口会返回稳定错误而不是让整个服务崩溃。
- **系列 PDF 导出**: 后端将系列 Markdown 渲染为 HTML（封面 + 目录 + 正文），并使用容器内 Chromium Headless 打印为 PDF（前端在侧边栏批量模式中逐个触发下载）。为保证中文正常显示，后端运行时镜像需安装 `chromium` 与 `font-noto-cjk` 等字体依赖。
- **代理与网关**: Nginx (构建前端静态页面并反向代理后端 `/api/` 路径，配置 `client_max_body_size 888M` 以支持大文件解析)
- **大语言模型**: DeepSeek-V4-Flash API (支持 128k 输出及 1M Token 上下文)

## 3. 并发生成架构
在处理项目到系列博客的生成时，后端采取如下架构：
1. **模块扫描与场景选择**：针对 Git 源码，使用无盘 `ls-tree` 与 GitHub API 极速提取一级核心代码目录；前端生成器同时暴露“电子书解读 / 开卷复习 / 小白教程”三张中文场景卡片，由用户显式确定本次创作目标。
2. **Map-Reduce 分析**：针对选中的模块和超大文件（>5万字）进行预切片处理，并行调用 LLM 抽取各个分块摘要；其中超长文件会采用更细粒度切块并保持摘要顺序稳定，不再对文件摘要做二次 Tree Reduce 压缩；大纲阶段会读取 `scenario_mode`，分别偏向“篇章解读”“考点速查”或“学习路径”三类拆解方式。
3. **多协程并发生成**：
   - 接收到前端下发的大纲后，启动多个 `goroutine` 为每个章节并行生成内容。
   - 使用 `semaphore.NewWeighted(3)` 将全局并发数严格限制为 3。
   - 每个 `goroutine` 均拥有独立的错误隔离环境，通过同一个 `progressChan` 向前端推送包含自身 `chapter_sort` ID 的 Chunk（数据切片）。
  - 单篇生成、系列章节生成和系列导读生成都会复用同一份 `scenario_mode + article_style + resolved prompt profile` 组合后的 Prompt 约束，避免“大纲像心理学解读、正文回到通用技术博客”的割裂。
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

## 4. 部署架构 (Docker-First)
- **前端镜像**: 采用多阶段构建（Node.js 安装依赖并构建，Nginx 轻量级运行并作为反向代理网关）。默认仅映射宿主机 `80` 端口，统一以 `http://localhost` 作为前端入口。
- **后端镜像**: 采用多阶段构建（Go 官方镜像编译，Alpine 运行）。使用 `FRONTEND_URL`、`DATABASE_URL`、`REDIS_URL` 等环境变量控制运行逻辑，但默认只在 Docker 内部网络暴露 `8080`，不直接对宿主机开放。
- **PDF 运行时依赖**: 后端 Alpine 运行时镜像除 `chromium`、`font-noto-cjk` 外，还需安装 `poppler-utils`，以便 `DocParser` 在中文 PDF 提取失真时回退到 `pdftotext`。
- **数据库 / Redis**: PostgreSQL 与 Redis Stack 默认仅在容器网络内暴露，避免开发态无意开放宿主机调试端口。
- **容器互联**: 全部服务显式加入 `inkwords-network` 内部网络，后端通过服务名 `db:5432`、`redis:6379`、`obsidian-bridge:27125` 与依赖互通。
- **环境装载约定**: Docker Compose 运行时统一建议通过 `docker compose --env-file backend/.env ...` 启动；`OBSIDIAN_VAULT_PATH` 必须显式提供，不再回退到某台开发机的绝对路径。

## 4.1 仓库产物与敏感信息策略
- 禁止提交构建产物与大文件（例如后端二进制、PDF、批量截图等），统一通过 `.gitignore` 管理本地产物目录。
- `dogfood-output/` 可能包含浏览器 localStorage 等敏感信息（例如 token），必须保持为本地目录，严禁进入 Git 追踪。

## 5. 全局缓存机制 (Prompt Caching)
- **目标**：降低 DeepSeek Token 消耗，提高首字响应速度 (TTFT)。
- **原生支持**：全面拥抱 DeepSeek V4 API 级别的原生前缀缓存 (Prompt Caching)。
- **Prompt 结构重构**：将数百万字的巨量源码 `sourceContent` 提取至 `system` 消息并置于请求最前，将易变的“指令”置于 `user` 消息并置于请求尾部，以最大化原生缓存的命中率，将长文本输入成本降低 80% 以上。

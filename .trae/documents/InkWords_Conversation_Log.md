# 墨言博客助手 (InkWords) - AI 对话与决策摘要 (Conversation Log)
> **目的**：记录在 Vibe Coding 过程中，每一次核心对话的上下文、用户指令意图以及关键架构决策。以便在长周期的开发中，不论更换 AI 会话窗口还是重新梳理思路，都能快速找回项目背景。

### 对话 21：项目文档更新与代码提交
- **用户需求**：要求更新项目的所有基准文档（PRD、API、架构、数据库、开发日志、对话日志），更新完成后提交代码至 GitHub，并在查阅现有标签后打上新标签。
- **AI 动作**：
  1. 查阅并更新了 `InkWords_API.md`，将分析接口 `/api/v1/project/analyze` 更新为流式接口 `/api/v1/stream/analyze`。
  2. 查阅并更新了 `InkWords_Architecture.md`，在“核心业务流转设计”中补充了分析阶段的 SSE 进度推送机制。
  3. 查阅并更新了 `InkWords_PRD.md`，在“大项目拆解流程”中补充了前端打字机效果及 SSE 进度条的需求描述。
  4. 查阅并更新了 `InkWords_Database.md`，确认数据库表结构无需变更，保持最新状态。
  5. 检查了 `InkWords_Development_Plan_and_Log.md` 与本文件，确认日志已是最新。
  6. 接下来将执行 `git add`, `git commit`, `git push` 以及打标签操作。
- **决策/变更**：
  - 确保“文档即代码”的一致性，将今日修复的“串行打字机”、“SSE进度流”、“解决隐藏挂起 Bug”等核心机制正式写入基准文档，为后续版本迭代提供准确参考。

### 对话 20：修复 SSE 流被浏览器后台标签页机制中断的问题 (ERR_ABORTED)
- **用户需求**：用户贴出了 `net::ERR_ABORTED ... at onVisibilityChange` 的报错日志，在分析/生成大项目期间，如果切换了浏览器标签页，SSE 流会被直接切断并报错。
- **AI 动作**：
  1. 识别到这是 `@microsoft/fetch-event-source` 库的默认行为：当标签页失去焦点（Visibility Change）时，它会主动中断并挂起连接。
  2. 修改 `frontend/src/hooks/useBlogStream.ts` 中所有的 `fetchEventSource` 调用（包括 `/analyze` 和两个 `/generate` 场景），统一加上 `openWhenHidden: true` 参数。
- **决策/变更**：
  - 由于大模型生成或 Git 仓库克隆的耗时较长，用户极大可能会在此期间切到别的标签页干别的事情。关闭隐藏时的断流保护能确保任务在后台继续执行，提高整体体验与成功率。

### 对话 19：优化 Git 仓库分析过程，引入真实 SSE 进度流
- **AI 动作**：
  1. 通过 `AskUserQuestion` 与用户确认了采用“真实 SSE 进度流 + 轻量化克隆”的优化策略。
  2. 修改 `backend/internal/api/stream.go`，新增了 `/api/v1/stream/analyze` 路由处理器，专门用于将原本 HTTP POST 的短连接改造为长连接的 SSE 流。
  3. 修改 `backend/internal/service/decomposition.go`，新增了 `AnalyzeStream` 方法。在克隆仓库、生成大纲的各个核心节点，通过通道主动向下发包含 `step` 和 `message` 的 JSON 数据；并在最后一步 `step: 3` 时将大纲和提取的源码 `data` 下发给前端。
  4. 修改 `frontend/src/hooks/useBlogStream.ts` 中的 `analyzeGit`，弃用原有的 `fetch` 改用 `fetchEventSource`。通过解析下发的事件流，动态更新 `analysisStep` 和 `analysisMessage` 状态。
  5. 优化了 `frontend/src/components/Generator.tsx` 中的进度展示，移除了原有的 `useEffect` 假定时器，现在分析进度文字会随着后端 SSE 的真实状态同步改变。
- **决策/变更**：
  - 将大项目分析阶段也从“黑盒等待”升级为了“白盒透明”，用户可以看到当前是在“克隆拉取”还是在“AI生成大纲”，缓解了因大项目耗时过长带来的焦虑感。
  - 保留了本地文件的假进度条（因为文件解析非常快），仅对耗时极长的大模型分析步骤开启 SSE 改造。
- **用户需求**：用户指出在输入 Git 仓库生成系列博客时：1. 没有进度条（生成过程不明显，卡在0字符很久然后突然完成）；2. 不希望同时生成多篇文章；3. 无法点击卡片，想知道如何查看真实文章。
- **AI 动作**：
  1. 通过 `AskUserQuestion` 与用户确认了采用“修复打字机 + 恢复完成状态点击”的解决方案。
  2. 修改 `backend/internal/service/decomposition.go`：修复了极其隐蔽的 Bug —— 之前在 Go 中使用 `fmt.Sprintf` 拼接 SSE 的 JSON 返回数据，大模型输出的文本中带有回车、引号等特殊字符导致前端 `JSON.parse` 报错被静默吞掉，从而表现为“假死，0 字符，很久后突然完成”。现已改用标准的 `json.Marshal(map[string]interface{})`。
  3. 修改 `backend/internal/service/decomposition.go` 中针对每一篇文章生成期间的 SSE 事件发送逻辑：将 LLM 返回的每个 chunk 打包为 `{"status":"streaming", "content":"..."}` 事件推送给前端。
  4. 修改 `frontend/src/hooks/useBlogStream.ts`，接收到 `streaming` 事件时实时更新 Zustand 的 `generatedContent`；当某章 `completed` 时主动调用 `fetchBlogs()` 刷新历史列表。
  5. 修改 `frontend/src/components/Generator.tsx`，合并了 `file` 和 `git` 的生成中状态，在系列生成时统一使用 `MarkdownEngine` 实时渲染当前正在生成的章节内容。
  6. 修改 `frontend/src/components/Sidebar.tsx`，恢复了卡片的 `onClick` 事件，但仅限于该章节状态变为 `completed`（变绿）时触发。点击后不再渲染“草稿假数据”，而是使用递归算法从最新的 `blogs` 树中查找对应的真实历史文章，并自动跳转至编辑器面板打开它。
- **决策/变更**：
  - 取消了系列文章的并发生成（为了提供沉浸式的打字机反馈体验并防止内容过短）。
  - 将单文件生成与系列文章生成的“生成中”UI体验完全对齐，利用串行化和精确的 JSON 序列化彻底解决了状态展示混乱、打字机罢工的问题。
  - 通过匹配历史文章的方式恢复卡片点击，解决了之前“草稿数据”被当做最终结果的误导问题。

### 对话 17：新增“新建博客 / 返回首页”功能
- **用户需求**：用户反馈在点击进入生成的博客详情（编辑器）后，无法退出当前状态并返回到首页进行下一篇博客的创作，要求修复并通过提问方式明确交互功能。
- **AI 动作**：
  1. 通过 `AskUserQuestion` 提供了三个交互位置选项（侧边栏顶部按钮、Logo可点击、编辑器顶部按钮），用户选择了“侧边栏顶部按钮 (推荐)”。
  2. 修改 `frontend/src/components/Sidebar.tsx`，在左侧边栏顶部的 Logo 下方增加了一个醒目的“新建博客 / 返回首页”按钮。
  3. 绑定点击事件 `selectBlog(null)`，清空当前选中的博客状态，从而触发 `App.tsx` 重新渲染 `<Generator />` 首页组件。
  4. 同步更新了开发日志和对话记录，保持“文档即代码”的一致性。
- **决策/变更**：
  - 采用显式的全局常驻按钮来处理“退出阅读/编辑态”的交互，比单纯的 Logo 点击更符合直觉，提升了用户在不同博客间切换及新建任务的体验。

### 对话 15：修复 GitHub 仓库解析内容少与单文件上传进度条体验
- **用户需求**：用户指出解析 GitHub 仓库时生成的内容过少且未按要求拆分系列博客；另外上传文件生成博客时，前端缺少流式生成的进度条提示。
- **AI 动作**：
  1. 通过提问明确了用户的具体痛点（GitHub源码读取不完整/内容太简略/未拆分系列，上传文件需要全链路综合进度提示）。
  2. 修改 `backend/internal/parser/git_fetcher.go`，在读取源码时优先将目录结构（Tree）拼接到最上方，保证即便后续因过长截断，大模型也能了解仓库整体结构。
  3. 修改 `backend/internal/service/decomposition.go`，在 `GenerateOutline` 提示词中**强制要求**大型项目至少拆分为3个章节；并在 `GenerateSeries` 的提示词中要求大模型必须深入分析、引用核心代码、提供复现步骤，甚至要求它根据目录结构合理推演缺失部分。
  4. 修改 `frontend/src/store/streamStore.ts` 增加 `generatedContent` 状态来保存单篇博客生成的流式文本。
  5. 修改 `frontend/src/hooks/useBlogStream.ts` 在 `generateSingle` 接收到 `chunk` 事件时，拼接实时内容。
  6. 修改 `frontend/src/components/Generator.tsx`，在单文件生成过程中使用 `MarkdownEngine` 实时渲染生成的 Markdown 文本流，实现打字机效果。
- **决策/变更**：
  - GitHub 仓库过大截断是不可避免的物理限制，通过“注入目录结构”加“强化 Prompt 深度”来对抗截断导致的内容空洞。
  - 单文件生成的体验从“Loading Spinner 盲等”升级为“实时 Markdown 渲染”，提升了反馈感知。

### 对话 16：修复 Mermaid 图表渲染与 PDF 解析报错
- **用户需求**：用户贴出了控制台中出现的 `UnknownDiagramError: No diagram type detected matching given configuration for text: undefined` 以及 PDF 解析时出现的 `failed to create PDF reader: not a PDF file: missing %%EOF` 报错日志。
- **AI 动作**：
  1. 修复了 `frontend/src/components/MarkdownEngine.tsx`，在传递给 Mermaid 渲染引擎之前增加了非空校验和对 `undefined` 文本流的过滤防护，防止因为打字机流式渲染期间不完整的图表语法导致崩溃。
  2. 修复了 `backend/internal/parser/doc_parser.go` 中 PDF 的解析流程。在临时文件拷贝完成后，主动调用 `tempFile.Sync()` 强制刷盘，并执行 `Seek(0, 0)`，防止第三方 PDF 库因为读取指针位置错误或文件未完全写入导致的 `missing %%EOF`（文件不完整）错误。
  3. 为 `backend/internal/api/project.go` 增加了对空文件的阻断拦截（`header.Size == 0`）。
  4. 为 `frontend/src/hooks/useBlogStream.ts` 补充了针对文件解析报错的用户友好提示框（`alert`），防止前端页面只在控制台打错而没有任何反馈。
- **决策/变更**：
  - 流式生成的 Markdown 渲染组件必须具备极强的容错性，因为中间态的代码块极大概率是不完整的语法。
  - Go 语言中利用 `io.Copy` 处理文件时必须严格控制文件指针和缓冲区刷盘，尤其是在第三方依赖直接使用该文件句柄时。

### 对话 14：修复 SSE 连接与 AbortError 日志报错
- **用户需求**：用户贴出了控制台中出现的 `AbortError: BodyStreamBuffer was aborted` 报错日志，要求进行分析与修复。并且要求修复完毕后同步文档并推送到 GitHub。
- **AI 动作**：
  1. 分析并定位到问题原因是 SSE 网络流由于服务端在返回 `[DONE]` 后迅速关闭或组件卸载时的残余流导致的。
  2. 修改 `useBlogStream.ts`，引入 `useRef` 保存 `AbortController` 实例，在组件卸载 (`useEffect` 清理函数) 和每次发起新生成前主动调用 `abort()`。
  3. 在 `fetchEventSource` 的 `onerror` 和外层 `catch` 中增加了对 `AbortError` 的静默过滤，将其转化为受控的 `StopStreamError` 以阻止默认报错和重连。
  4. 顺带修复了 `Generator.tsx` 和 `useBlogStream.ts` 中的 Lint 报错（`react-hooks/set-state-in-effect` 及 `any` 类型使用），目前项目 `npm run lint` 已零错误。
  5. 补充并更新了本文件和开发日志。
- **决策/变更**：
  - 主动接管 `fetch-event-source` 的中止信号生命周期，明确区分正常结束/组件卸载导致的流切断与网络真实断开的错误。

### 对话 13：添加用户注销（退出登录）功能
- **用户需求**：要求为项目添加支持用户注销的功能，完成后更新项目文档、提交 GitHub 并打上对应标签。
- **AI 动作**：
  1. 修改前端 `frontend/src/components/Sidebar.tsx` 组件，在底部添加了带有注销图标的「退出登录」按钮。
  2. 实现了点击按钮后清除 `localStorage` 中的 `token` 并重定向到根路径强制刷新页面，显示回登录组件。
  3. 更新了开发日志和对话记录。
  4. 使用 Git 提交代码并打上对应功能的 Tag。
- **决策/变更**：
  - 确认了系统使用的是无状态的 JWT 方案，因此注销逻辑完全由前端承担（清理本地凭证），无需后端 API 参与。

### 对话 12：修复 GitHub 登录重定向与前端列表加载 Bug
- **用户需求**：要求修复 GitHub 授权登录后卡在后端 JSON 响应页面无法返回前端的问题，以及前端页面加载时 `fetchBlogs` 抛出 JSON 解析错误的问题。
- **AI 动作**：
  1. 修改后端 `internal/api/auth.go` 的 `OAuthCallback`，将 JSON 响应替换为 `c.Redirect` 重定向回前端（携带 `?token=` 或 `?error=`）。
  2. 修改前端 `Login.tsx`，使用 `useEffect` 解析并提示 URL 中的 `error` 参数，随后清理 URL。
  3. 修改前端 `blogStore.ts`，将 `fetchBlogs` 中的直接 `res.json()` 改为先获取 `res.text()`，增加判空和 `try-catch` 保护，防止空响应导致报错。
  4. 更新了 PRD、API、开发日志等相关基准文档。
- **决策/变更**：
  - 明确第三方 OAuth 回调的闭环必须通过 HTTP 重定向交还控制权给前端，由前端提取状态（Token 或 Error）并处理后续路由和用户提示。

### 对话 11：常规登录实现与依赖/图标修复
- **用户需求**：要求更新开发日志和对话记录，补充今日的 Bug 修复与新特性开发：移除 `rehype-mermaid`、修复 `lucide-react` 的 Github 图标以及后端常规登录与注册功能的实现。
- **AI 动作**：
  1. 在 `InkWords_Development_Plan_and_Log.md` 中追加了对应的 Bugfix & Feature 记录。
  2. 在 `InkWords_Conversation_Log.md` 中补充了本次对话与决策的摘要。
- **决策/变更**：
  - **架构微调**：为了严格执行“纯净无样式”的图表渲染约束，移除了 `rehype-mermaid` 依赖，避免了样式冲突。
  - **功能完善**：在已有的第三方 OAuth 授权之外，补充了传统的账号密码登录与注册机制，完善了用户鉴权闭环。

### 对话 10：修复编辑器输入覆盖 Bug
- **用户需求**：要求修复 `Editor.tsx` 中由于自动保存导致 `selectedBlog` 更新，从而触发 `useEffect` 覆盖用户正在输入的内容的 Bug。同时要求**不更新**个人任务文件 `tasks.md` 或 `checklist.md`。
- **AI 动作**：
  1. 将 `Editor.tsx` 中同步博客内容的 `useEffect` 依赖项从 `selectedBlog` 修改为 `selectedBlog?.id`。
  2. 禁用了相关的 `eslint-disable-next-line react-hooks/exhaustive-deps` 检查以消除 linter 警告。
  3. 顺手修复了 `MarkdownEngine.tsx` 中的 `node` 未使用报错。
  4. 运行 `npm run lint` 验证所有问题均已修复。
  5. 更新了项目开发日志 `InkWords_Development_Plan_and_Log.md` 以及本对话日志。
- **决策/变更**：
  - 核心决策是**只在切换文章时同步状态**，而不再每次保存后覆盖本地组件状态，以保障用户的输入流畅性。

---

## 2026-04-04 (项目规划与脚手架搭建)

### 对话 1：项目初步规划 (Plan Mode)
- **用户需求**：要求开发一个基于 DeepSeek API 的博客写作助手，前端使用 React 18，后端使用 Go，支持 Word/PDF/MD 及 Git 仓库解析，大项目自动拆解，Mermaid 图表无样式，并要求配置严格的工程 rules。
- **AI 动作**：在 Plan 模式下起草了初步的《开发计划与架构设计》。
- **决策/变更**：用户多次反馈修改意向，要求强调“小白友好、可复现、枯燥概念加代码”的核心生成逻辑。

### 对话 2：文档共创流程引入 (Doc Co-Authoring)
- **用户需求**：用户拒绝将所有计划写在一个文件里，要求采用“结构化共创流程”，逐个拆分出产品需求(PRD)、数据库、架构文档。
- **AI 动作**：启动 `doc-coauthoring` 技能，通过多轮提问，与用户依次完成了：
  1. **PRD**：确立 5000 字拆分限制、50MB 体积限制、阅后即焚策略。
  2. **Architecture**：确立 Monorepo、SSE 流推送、并发调度策略及 JWT+OAuth 体系。
  3. **Database**：确立了父子系列博客关联 (`parent_id`) 及雪花 ID 策略。
- **决策/变更**：
  - 架构设计过程中，用户发现缺少“账号体系”，临时决定引入 GitHub/Wechat 第三方 OAuth 一键登录。
  - 数据库选型阶段，用户要求**将 MySQL 替换为 PostgreSQL 14+**，并采用 `TEXT` 和 `UUID` 类型。

### 对话 3：补充核心基准文档与重构 Rules
- **用户需求**：发现缺少前后端 API 定义，要求补全。随后要求将 `.cursorrules` 按规则类型拆分，并强调自己极度偏好在 Vibe Coding 中使用**共创模式**。
- **AI 动作**：
  - 补充了第 4 份文档《API 接口规范文档》。
  - 将 `.cursorrules` 删除，拆分为 `.trae/rules/` 下的三个 Markdown 文件（架构规范、业务约束、Vibe协作规范）。
  - 在规则中确立了**“强制阅读上下文 (Context First)”**的第一铁律。

### 对话 4：开发计划与里程碑
- **用户需求**：基于已有文档，要求给出开发计划和每日开发日志。
- **AI 动作**：使用共创模式编写了《开发计划与日志》，划分了 MVP、Alpha、Beta、V1.0 四个里程碑，并对各模块进行了耗时预估和强单测约束。

### 对话 5：创建对话日志文件 
- **用户需求**：在 `.trae/documents/` 目录下添加一个文件，专门用于简单记录每次的对话。
- **AI 动作**：创建了本文件 (`InkWords_Conversation_Log.md`)，并回溯记录了今日从项目启动到脚手架搭建的完整对话脉络与关键决策点。

### 对话 6：完成 MVP 骨架与基建开发
- **用户需求**：要求继续按照开发计划执行。
- **AI 动作**：
  1. 使用 Spec 模式执行并完成了“基础架构与数据库搭建”(Task 1, 2)
  2. 使用 Spec 模式执行并完成了“第三方 OAuth2.0 与 JWT 签发”(Task 3)
  3. 使用 Spec 模式执行并完成了“基础 PDF/MD 解析器”(Task 4)
- **决策/变更**：
  - 遇到 PDF 流解析问题，决定采用生成临时文件然后 `defer os.Remove` 的策略，确保兼顾 `io.ReaderAt` 接口需求与“阅后即焚”的安全需求。

### 对话 7：封装 DeepSeek 与 SSE 流式推流通道 (当前)
- **用户需求**：继续开发“封装 DeepSeek 客户端，建立前后端 SSE 实时推流渲染通道”功能。
- **AI 动作**：
  1. 编写了前后端 `spec.md` 及相关任务，并执行开发。
  2. 封装了 `DeepSeekClient` 以及 Gin 的 SSE 推流接口 `/api/v1/stream/generate`。
  3. 前端集成了 Zustand、`fetch-event-source`，以及基于 `react-markdown` 的视图组件。
  4. 利用自建 Remark 插件移除了 Mermaid 图表代码块内的样式注入。
  5. 自动更新了 `InkWords_API.md`、`InkWords_Architecture.md` 和 `InkWords_PRD.md`，随后又将开发记录同步至 `InkWords_Development_Plan_and_Log.md` 和本文件。
- **决策/变更**：
  - 因为原生 EventSource 不支持 POST 请求，且基于 Ticket GET 状态拉取长文本设计过于复杂，**更改架构决策**为前端携带提取出的纯文本（`source_content`），直接向后端发起 `POST` 请求获取 SSE 数据流（使用 `@microsoft/fetch-event-source`）。

### 对话 8：阶段二 Alpha (大项目智能拆解) 开发完成
- **用户需求**：要求根据开发计划进行阶段二的开发（大项目智能拆解），并在完成后更新所有架构与设计文档。
- **AI 动作**：
  1. 查阅了 `doc-coauthoring` 和 `find-skills` 技能说明。
  2. 使用 Spec 模式编写了 `05-project-decomposition` 的规范、任务与检查单。
  3. 实现了 `GitFetcher`，支持 Git 仓库的拉取（`git clone --depth 1`）、过滤与文本提取，并确保阅后即焚。
  4. 实现了 `DecompositionService` 和 `POST /api/v1/project/analyze` 路由，支持长文本的非流式大纲生成。
  5. 实现了基于 Goroutine 池的并发调度生成机制，确保同一系列文章能够并行生成并正确落库（绑定相同的 `ParentID`）。
  6. 更新了 SSE 推流接口，支持接收大纲并下发各个章节的进度事件。
  7. 批量更新了所有 `.trae/documents` 下的基础文档并提交至 GitHub。
- **决策/变更**：
  - 采用 Goroutine 池并结合信号量 (Semaphore) 控制并发量，防止大量并发请求导致大模型 API 频率超限或内存激增。
  - SSE 推流在系列生成模式下，不再单纯推送 chunk，而是推送每个章节的整体生成状态，以便前端展示全局进度。

### 对话 9：前端工作台 (Workspace) UI 搭建与联调
- **用户需求**：要求启动项目并解决初始化的 React 空白页面问题，构建实际的前端工作台。
- **AI 动作**：
  1. 诊断并修复了因缺失数据库导致的后端启动失败问题（创建 `inkwords` 数据库并初始化 `.env`）。
  2. 开发了前端双栏布局界面，左侧边栏用于展示大项目大纲与实时进度（Loading / 完成图标），右侧用于输入 Git URL 和触发分析与生成。
  3. 完善了 Zustand 状态库 `streamStore.ts` 以及自定义 Hooks `useBlogStream.ts`，打通了前后端的 `Analyze` 与 `Generate` 接口。
  4. 修复了 Vite 预构建缓存导致 `lucide-react` 找不到的 404 错误（使用 `npm run dev -- --force` 重启）。
  5. 修复了 `MarkdownEngine` 中的 ESLint 报错，并清理了冗余代码。
- **决策/变更**：
  - 在 `vite.config.ts` 中配置了跨域代理，将前端 `/api` 转发到后端 `8080` 端口，避免了开发环境的 CORS 问题。

---

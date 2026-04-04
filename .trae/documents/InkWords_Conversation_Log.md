# 墨言博客助手 (InkWords) - AI 对话与决策摘要 (Conversation Log)
> **目的**：记录在 Vibe Coding 过程中，每一次核心对话的上下文、用户指令意图以及关键架构决策。以便在长周期的开发中，不论更换 AI 会话窗口还是重新梳理思路，都能快速找回项目背景。

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

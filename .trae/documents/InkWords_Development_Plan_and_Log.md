# 墨言博客助手 (InkWords) - 开发计划与日志
> **目标**：跟踪项目的核心开发模块、里程碑进度以及每日开发记录。

## 1. 里程碑划分 (Milestones)

### 阶段 1: MVP (核心单篇生成)
**目标**：跑通前后端最小核心闭环，完成单篇轻量级文档的智能转换。
- [x] 完成 Go + Gin + PostgreSQL 基础架构搭建与依赖注入。
- [x] 搭建前端 React 18 + Zustand + Tailwind 极简阅读风骨架。
- [x] 实现第三方（GitHub/WeChat）OAuth2 登录与 JWT 签发。
- [x] 实现基础 PDF/MD 文本解析器 (阅后即焚)。
- [x] 封装 DeepSeek 客户端，建立前后端 SSE 实时推流渲染通道。

### 阶段 2: Alpha (大项目智能拆解)
**目标**：支持超长代码库的解析与系列博客的生成。
- [x] 接入 Git 仓库拉取，实现代码文件过滤与提取规则 (`GitFetcher`)。
- [x] 实现大项目评估逻辑，开发“大纲规划 -> 并发调度生成 -> 拼接”的复杂调度机制。
- [x] 前端深度定制 `react-markdown`，引入并严格控制 `rehype-mermaid` 实现无样式图表渲染。

### 阶段 3: Beta (历史记录与编辑)
**目标**：完善数据库持久化与用户创作体验。
- [x] 完成 `blogs` 表的自引用查询与系列博客展示。
- [x] 前端开发类似 Notion 的双栏 Markdown 二次编辑器。
- [x] 支持自动保存、覆盖更新及文章导出 (MD/PDF)。

### 阶段 4: V1.0 (商业化与多端分发)
**目标**：打通流量分发与用户额度体系。
- 接入掘金、CSDN OpenAPI，实现一键授权发文。
- 上线用户用量统计（Tokens 消耗）及 `subscription_tier`（订阅会员）防刷限流系统。

## 2. 核心模块拆解与时间预估

| 模块类别 | 核心功能点 | 难度/风险 | 预计开发时间 |
| --- | --- | --- | --- |
| **Backend (Go)** | OAuth2 第三方授权与 JWT 签发 | 中 | 1.5 天 |
| | DocParser / GitFetcher 源码提取与过滤算法 | 高 (边缘格式多) | 2 天 |
| | DeepSeek API 封装与流式 SSE 转发管道 | 中 | 1.5 天 |
| | 大项目并发调度引擎 (Goroutine Pool) | 极高 (死锁风险) | 3 天 |
| | PostgreSQL GORM 模型映射与组合查询优化 | 低 | 1 天 |
| **Frontend (React)** | Shadcn UI 基础布局与 Zustand 状态挂载 | 低 | 1 天 |
| | 拖拽上传与仓库 URL 解析组件 | 中 | 1 天 |
| | Markdown 实时打字机渲染与 Mermaid 图表接管 | 高 (样式覆盖难) | 2.5 天 |
| | 类似 Notion 的二次编辑器与状态同步 | 中 | 2 天 |
| **Integration** | 第三方发文 OpenAPI 对接与 Token 加密 | 高 (各平台不一) | 3 天 |

## 3. 测试与联调计划
遵循 Vibe Coding **“小步迭代与强制验证”** 的铁律，在实际开发过程中，严禁越过测试环节强行合并代码。

### 3.1 后端单元测试 (Unit Testing)
- **重点目标**：`internal/parser` (Git与文档的提取清洗)、`internal/llm` (Prompt构建) 及 `internal/service` (大项目拆解算法与 Goroutine 调度)。
- **约束**：使用 Go 的内置 `testing` 框架和 `testify` 库。所有核心 Service 必须包含 Mock（如 `gomock`），特别是对 DeepSeek API 的 Mock 测试，确保在断网下仍能测试内部状态机的流转。

### 3.2 前端组件测试 (Component Testing)
- **重点目标**：Markdown 渲染器（验证是否准确剥离了 Mermaid 的样式 `style`）以及 Zustand 的状态变化。
- **约束**：引入 `Jest` 和 `React Testing Library`，为 `MermaidViewer` 和核心的 `useBlogStream` Hooks 编写隔离测试。

### 3.3 端到端联调测试 (E2E Integration)
- **重点目标**：打通上传 -> SSE 流接收 -> 渲染 -> 落库保存的完整闭环。
- **约束**：后端在每次 API 开发完成后，使用 Postman 导出或在代码中编写 `httptest` 进行路由联调验证；前后端联调阶段关注 50MB 超大文件的上传稳定性及流媒体推送时的断线重连（Resume）。

### 3.4 真实场景/效果验证 (Reader Testing)
- **目标**：验证 AI 的 Prompt 策略是否真的达到了“小白友好、可独立复现、图文并茂”的要求。
- **测试用例**：
  1. 上传一个极简的小脚本（如 100 行 Go 代码），验证生成的单篇内容是否丰富详实。
  2. 导入一个著名长篇开源项目或官方教程（如 React 官方教程），测试系统的“复杂度评估器”是否能准确切分为多个结构连贯的 5000 字篇章。

## 4. 每日开发日志 (Dev Log)
> 该区域将由 Vibe Coding 工程师（AI 助手）在每天/每次开发周期结束时，如实记录当天的完成事项、遇到的技术坑点及架构小规模调整。

### 2026-04-04 (规划与脚手架阶段)
- **开发模块**: [全栈架构与基准文档设计]
- **完成事项**:
  1. 完成产品需求文档、数据库、架构设计、API 接口这四份基准 Markdown。
  2. 确立了遵循“文档即代码”与“共创模式优先”的 `.trae/rules` 规范。
  3. 创建了 README.md 项目入口指引。
  4. 初始化了 `backend/` 目录下的 Go+Gin 项目骨架，跑通了 `/api/v1/ping` 健康检查。
- **踩坑记录 / 架构调整**: 
  - 架构设计阶段将数据库选型由 MySQL 修改为 PostgreSQL 14+，以利用其原生的 `UUIDv4` 和更优秀的 `TEXT` 存储及部分索引特性。
  - 在 PRD 与 API 阶段，发现缺少了用户注册体系，及时补充了 GitHub/Wechat 第三方 OAuth2.0 一键登录机制，并完善了数据库的 `subscription_tier` 和 `tokens_used` 字段。
- **遗留问题 (TODO)**: 
  - ~~尚未初始化 `frontend/` 目录的 React 骨架。~~ (已在后续开发中完成)
  - ~~后端的 `gorm` 数据库模型 (Entity) 暂未建立。~~ (已在后续开发中完成)

### 2026-04-04 (MVP - 基础骨架与鉴权模块)
- **开发模块**: [前端骨架初始化, 后端数据库模型, OAuth2 与 JWT 鉴权]
- **完成事项**:
  1. **前端**：在 `frontend/` 目录下初始化了基于 Vite 的 React 18 (TS) 项目，并成功集成了最新版的 Tailwind CSS v4、Shadcn UI 以及 Zustand 状态管理。
  2. **后端模型**：在 `backend/internal/model/` 目录下完成了基于 `GORM` 的 `User`, `Blog`, `OAuthToken` 实体模型定义，支持了 UUIDv4 主键生成、软删除，并在 `blogs` 实现了多字段组合索引。
  3. **后端鉴权**：集成了 `golang-jwt/jwt/v5` 与 `golang.org/x/oauth2`，编写了 `auth` 的 Middleware 拦截器。打通了 GitHub OAuth2 授权回调获取用户信息并 UPSERT 数据库记录的完整闭环流程。
  4. **版本控制**：项目成功梳理了根目录及子目录的 `.gitignore` 过滤规则，初始化 Git 仓库，并推送至 GitHub。
- **踩坑记录 / 架构调整**: 
  - Shadcn UI 最新版本（v4.x）默认使用并推荐了 Tailwind CSS v4（弃用旧版 tailwind.config.js），因此在前端安装时做了最新技术的适配，使用 `@tailwindcss/vite` 插件替代了旧版的 PostCSS 插件方案。
- **遗留问题 (TODO)**: 
  - ~~尚未实现核心的文件解析器（DocParser/GitFetcher）及与 DeepSeek API 的 SSE 流式交互管道。~~ (已部分实现基础 DocParser，待实现 GitFetcher 及 SSE 流式交互管道)

### 2026-04-04 (MVP - 基础解析器开发)
- **开发模块**: [后端文件解析器]
- **完成事项**:
  1. **文档解析器**: 在 `backend/internal/parser/` 中实现了 `DocParser`，支持解析 PDF 与 MD/TXT 纯文本格式。
  2. **安全策略 (阅后即焚)**: 实现了严格的临时文件处理机制，使用 `defer os.Remove(tempFile.Name())` 确保无论是上传的文件流还是内存数据，解析完成后立即物理销毁，不产生任何持久化实体。
  3. **单元测试**: 引入了 `github.com/jung-kurt/gofpdf` 生成测试 PDF，并完成了 `doc_parser_test.go`，覆盖了纯文本提取、PDF 解析与不支持格式的错误拦截。
- **踩坑记录 / 架构调整**: 
  - PDF 文本提取通常需要实现 `io.ReaderAt`，而用户上传的文件或网络流默认仅为 `io.Reader`。为解决此冲突并兼顾“阅后即焚”的原则，统一将流转存到 `os.CreateTemp` 的临时文件中再行解析，这样既获得了 `io.ReaderAt` 的能力，又能在函数结束时精准清空。
- **遗留问题 (TODO)**: 
  - ~~尚未封装 DeepSeek 客户端。~~ (已在后续开发中完成)
  - ~~需要建立前后端 SSE 实时推流渲染通道。~~ (已在后续开发中完成)

### 2026-04-04 (MVP - DeepSeek 客户端与流式渲染)
- **开发模块**: [封装 DeepSeek 客户端与 SSE 实时推流]
- **完成事项**:
  1. **后端 Client 与 Service**：在 `backend/internal/llm/` 封装了 `DeepSeekClient`，支持 `stream=true` 并使用 `bufio.Reader` 逐行解析 Delta 数据。在 `generator.go` 中内置了系统 Prompt (约束“小白友好”与 Mermaid 格式)，接管大模型流式输出与入库保存策略。
  2. **后端 SSE 接口**：在 Gin 路由层 (`/api/v1/stream/generate`) 利用 `c.Stream` 将接收到的文字通道数据封装为标准 `event: chunk` 以及结束事件 `event: done`。
  3. **前端状态层**：建立 `streamStore.ts` 管理文本拼接及流状态。使用 `@microsoft/fetch-event-source` 替换原生 `EventSource`，以支持 `POST` 方法下发文本 Payload (`source_content`)。
  4. **前端渲染层**：结合 `react-markdown`、`remark-gfm` 与 `react-syntax-highlighter` 建立代码高亮阅读视图。通过自定义 remark 插件 `remarkStripMermaidStyles` 拦截 AST 树，剔除 `style` 节点，完美实现极简无样式的图表渲染 (`rehype-mermaid`)。
- **踩坑记录 / 架构调整**: 
  - 架构上调整了数据传递链路，因为原生 `EventSource` 仅支持 `GET` 方法，而将长达数千字的纯文本通过 `task_id` 获取状态不仅复杂，且限制较多。因此果断切换到前端 POST `fetch-event-source` 携带文本直连后端的模式。
- **遗留问题 (TODO)**: 
  - 核心模块 MVP 单篇生成能力已跑通，需进一步开发“工作台界面”或直接开展阶段 2 (大项目自动拆解) 的相关任务。

### 2026-04-04 (Alpha - 大项目智能拆解)
- **开发模块**: [Git 解析器, 大纲生成 API, 并发调度引擎, SSE 系列推流]
- **完成事项**:
  1. **GitFetcher**: 在 `backend/internal/parser/` 实现了基于 `git clone --depth 1` 的仓库浅克隆，严格过滤无用目录，提取文本后使用 `defer os.RemoveAll` 实现阅后即焚。
  2. **大项目评估 API**: 实现了 `POST /api/v1/project/analyze` 路由，通过非流式调用 DeepSeek 接口，生成结构化的 JSON 大纲（包含章节标题、摘要、排序）。
  3. **并发调度生成**: 在 `decomposition.go` 中实现 `GenerateSeries`，利用 Goroutine 池和信号量（限制并发为 3）并发生成各章节，并统一 `ParentID` 和 `ChapterSort` 落库。
  4. **SSE 系列推流**: 更新了 `/api/v1/stream/generate`，支持前端携带大纲请求。在系列生成期间，通过通道向前端下发 `event: progress` 事件，实时同步各章节的生成状态。
- **踩坑记录 / 架构调整**: 
  - 并发请求大模型时，若不加限制极易触发并发超限或引发死锁。通过引入 `golang.org/x/sync/semaphore` 进行严格的并发控制，既保障了生成速度，又维持了稳定性。
  - 在系列生成时，原有的 SSE 接口主要用于单篇打字机效果。为此，专门增加了一种 `progress` 事件类型，用于向前端播报“某章节开始生成”和“某章节生成完毕”的状态。
- **遗留问题 (TODO)**: 
  - 后端功能已齐备，接下来需要前端开发“工作台界面”，打通 `Analyze` 与 `Generate` 两步走的 UI 交互（阶段 3 任务准备）。

### 2026-04-04 (Frontend - 工作台 UI 搭建与联调)
- **开发模块**: [React 工作台布局, Zustand 状态集成, API Hooks]
- **完成事项**:
  1. **工作台布局**: 基于 Tailwind CSS 和 Shadcn UI 构建了双栏布局，左侧展示系列博客的大纲和状态，右侧提供仓库 URL 输入和动作按钮。
  2. **状态与 Hooks**: 实现 `streamStore.ts` 用于管理 `isAnalyzing`, `isGenerating` 以及各章节的生成状态；在 `useBlogStream.ts` 中封装了 fetch 与 fetchEventSource 请求。
  3. **环境代理**: 在 `vite.config.ts` 中配置了 HTTP 代理，将前端发往 `/api` 的请求自动转发至后端 `:8080`，解决了开发阶段的 CORS 问题。
  4. **环境修复**: 修复了后端缺少 `inkwords` 数据库的启动报错问题，以及前端 Vite `lucide-react` 的预构建缓存 404 错误。
- **踩坑记录 / 架构调整**: 
  - 前端使用 `lucide-react` 时，Vite 预构建缓存未及时更新导致页面报 `ERR_ABORTED` 错，通过 `vite --force` 成功解决。
  - TypeScript 对第三方库的 AST AST类型检查较严，在 `MarkdownEngine` 中补充了 `eslint-disable-next-line` 以屏蔽 `any` 类型的警告。
- **遗留问题 (TODO)**: 
  - 当前已经能够解析并自动入库生成博客内容，下一步需完成阶段 3 (Beta 历史记录与编辑)：在前端侧边栏增加历史记录的拉取，并实现右侧区域的 Markdown 实时渲染和二次编辑能力。

### 2026-04-04 (Beta - 历史记录与编辑功能)
- **开发模块**: [后端历史记录与更新 API, 前端侧边栏, Markdown 编辑器, 自动保存与导出]
- **完成事项**:
  1. **后端 API**: 完成了 `blogs` 表的自引用查询与系列博客展示 API，实现了博客历史记录的拉取和内容的覆盖更新。
  2. **前端侧边栏**: 开发了前端侧边栏，支持历史记录列表的拉取、展示与切换。
  3. **Markdown 编辑器**: 实现了类似 Notion 的双栏 Markdown 二次编辑器，提供了沉浸式的创作体验。
  4. **自动保存与导出**: 支持编辑器内容的自动保存，并成功集成了文章导出为 MD 与 PDF 格式的功能。
- **踩坑记录 / 架构调整**: 
  - 前后端联调自动保存时，优化了防抖机制以减少后端写入频率。导出 PDF 时处理了中文字体兼容性问题。
### 2026-04-04 (Bugfix - 编辑器状态同步)
- **开发模块**: [前端 Markdown 编辑器]
- **完成事项**:
  1. **修复编辑器输入覆盖问题**: 修复了 `Editor.tsx` 中由于自动保存导致 `selectedBlog` 状态更新，进而触发 `useEffect` 重新赋值 `title` 和 `content` 的问题，确保只有在切换不同博客 (`selectedBlog?.id` 改变) 时才同步初始内容，保护了用户打字期间的输入内容。
  2. **代码质量优化**: 清理了 `MarkdownEngine.tsx` 中的无用变量并解决了 ESLint 警告。
- **遗留问题 (TODO)**: 
  - 准备进入阶段 4: V1.0 (商业化与多端分发)，接入掘金、CSDN OpenAPI 等第三方平台。

### 2026-04-04 (Bugfix & Feature - 依赖移除与常规账号体系)
- **开发模块**: [前端 Markdown 渲染, 前端 UI 图标, 后端鉴权模块]
- **完成事项**:
  1. **移除 rehype-mermaid**: 移除了 `rehype-mermaid` 依赖，优化了 Markdown 中 Mermaid 图表的渲染机制，避免样式污染与不必要的复杂度。
  2. **修复 Github 图标**: 修复了 `lucide-react` 中 Github 图标组件的导入/渲染问题。
  3. **常规登录/注册实现**: 在后端实现了基础的账号密码登录与注册体系，作为第三方 OAuth 登录的重要补充，完善了用户鉴权闭环。
- **踩坑记录 / 架构调整**: 
  - `rehype-mermaid` 插件的内置行为与项目要求的“纯净无样式”图表渲染存在冲突，移除后能更好地控制渲染输出。
  - 常规账号体系的加入丰富了 JWT 签发场景，提高了平台对不同类型用户的兼容性。

### 2026-04-04 (Bugfix - GitHub 登录重定向与前端容错修复)
- **开发模块**: [后端鉴权模块, 前端状态管理与登录]
- **完成事项**:
  1. **GitHub 登录回调修复**: 修复了 `OAuthCallback` 接口在成功/失败后直接返回 JSON 的问题，改为读取 `FRONTEND_URL` 环境变量并重定向至前端页面，通过 URL Query 参数传递 `token` 或 `error`。
  2. **前端错误捕获**: 在 `Login.tsx` 组件中增加了对 URL `error` 参数的监听与提示，并使用 `history.replaceState` 阅后即焚清理 URL。
  3. **前端 JSON 解析容错**: 修复了未登录时 `fetchBlogs` 接口返回空内容导致 `Unexpected end of JSON input` 的崩溃问题，增加了文本判空和 `try-catch` 保护。
- **踩坑记录 / 架构调整**: 
  - 第三方登录的常规流程中，回调接口 (Callback) 必须充当后端与前端的桥梁，直接返回 JSON 会导致浏览器停留在这个 API 路径上。改为重定向并利用 URL 传参是标准的解决方式。
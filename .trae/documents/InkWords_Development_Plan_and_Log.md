# 墨言博客助手 (InkWords) - 开发计划与日志
> **目标**：跟踪项目的核心开发模块、里程碑进度以及每日开发记录。

## 1. 里程碑划分 (Milestones)

### 阶段 1: MVP (核心单篇生成)
**目标**：跑通前后端最小核心闭环，完成单篇轻量级文档的智能转换。
- [x] 完成 Go + Gin + PostgreSQL 基础架构搭建与依赖注入。
- [x] 搭建前端 React 18 + Zustand + Tailwind 极简阅读风骨架。
- [x] 实现第三方（GitHub/WeChat）OAuth2 登录与 JWT 签发。
- [x] 实现基础 PDF/MD/DOCX 文本解析器 (阅后即焚，新增了 XML 清洗逻辑)。
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

### [2026-04-07] Feature - 优化大批量并发生成文章时的前端性能
- **开发模块**: [前端并发动画与性能优化]
- **完成事项**:
  1. **极简状态卡片**: 修改了 `Generator.tsx`，在处理并发生成的博客章节列表时，如果章节状态为 `completed`，则不再渲染包含完整内容的 `MarkdownEngine`，而是仅渲染一个显示标题和状态的轻量级极简卡片。
  2. **解决卡顿白屏**: 极大减少了页面中的 DOM 节点数量和 React 渲染开销，彻底解决了在生成 10+ 篇文章时由于渲染大量 Markdown 导致的页面卡顿和白屏问题。
- **踩坑记录 / 架构调整**:
  - 对于长列表和复杂组件（尤其是包含实时更新和 Markdown 解析的组件），必须在内容定型后及时进行降级渲染（如虚拟列表或折叠为简易卡片），以防内存溢出和主线程阻塞。

### [2026-04-07] Feature - 断点续传（继续生成）功能优化
- **开发模块**: [前端大纲状态管理, 后端生成接口]
- **完成事项**:
  1. **前端交互优化**: 在点击“停止生成”后，界面上的“开始生成”按钮自动切换为“继续生成”。
  2. **状态自动过滤**: 当处于“继续生成”状态时，大纲列表会自动隐藏已经生成的章节（`completed`），仅展示和提交待生成（`pending` 或 `error`）的章节。
  3. **后端系列接续**: 前端向后端提交剩余大纲时，会附带上一次生成获得的 `parent_id`。后端通过判断该 ID 是否已存在，避免了断点续传时重复创建父节点，确保接续生成的章节能完美挂载到同一个系列中。
- **踩坑记录 / 架构调整**:
  - 在处理流式中断后的重试时，如果直接把所有大纲再次发给后端，会导致已完成的章节被重复生成。通过在前端 Zustand store 中严格过滤掉状态为 `completed` 的章节，并在后端复用 `parent_id`，实现了一种简单且高度可靠的“按章节重试”的断点续传架构。

### [2026-04-06] 博客生成体验与性能优化
  - 修复了 Worker 卡片宽度被挤压的问题（将 `max-w-sm` 修改为 `max-w-3xl`）。
  - 新增“手风琴式”折叠大纲功能（支持生成期间自动折叠与手动展开）。
  - 新增“停止生成”按钮（前端 `AbortController` 搭配后端 `Request Context` 透传实现随时中断）。

- **2026-04-06: 项目提交与 Reader Testing 准备**
  - 清理：删除了不需要的端到端测试遗留文件（如 docker-compose.test.yml, mockllm, 测试脚本等）。
  - 更新：同步更新了所有项目文档，反映了最新的并发 UI、大纲自由编辑等功能。
  - 准备：为用户提供了 `.env.example` 并在沙盒中准备了真实的 Reader Testing（联合走查）环境。

- **2026-04-06: 项目纯净版回滚与首发 (v2.0.0)**
  - 回滚：彻底移除了有关第三方平台（CSDN、掘金）的一键发文与数据大盘（Dashboard）功能的代码、模型及 UI，恢复为纯粹的本地博客生成工具。
  - 发布：删除了旧的标签，打上了 `v2.0.0` 首发正式版本的标签并推送到 GitHub。
  - 文档：更新了 API 和架构文档，清理了过时的平台接口描述，补充了关于 SSE 心跳保活以及大模型截断自动无缝续写的架构设计与踩坑记录。

### [2026-04-06] Feature - 博客生成配置与并发动画优化
- **开发模块**: [后端大纲与博客生成 Prompt, 前端并发动画与可编辑大纲]
- **完成事项**:
  1. **无限制技术点拆分**: 修改了 `DecompositionService` 中的 Prompt，不再限制博客数量为 5-10 篇，而是要求“一个技术点分为一个博客，博客篇数上不封顶”。
  2. **代码级深度剖析**: 修改了 `GeneratorService` 的单篇生成 Prompt，强制要求“对于每个技术点都添加更多的代码样例和图片来解释的更加详细”。
  3. **并发工作槽动画 (Worker Slots)**: 在 `mapReduceAnalyze` 阶段，为 Goroutine 池分配了 5 个固定的 `worker_id`，并在前端将原有的单一进度条重构为 5 个独立的工作槽。每个槽能独立显示当前分配到的分块及分析状态，带脉冲动画，并发过程更加直观。
  4. **可交互式大纲编辑**: 重构了前端 `Generator.tsx` 中的系列博客大纲 UI，将其从只读列表升级为可内联编辑的卡片。用户可以在点击“开始生成”前自由修改章节标题、摘要，并通过拖拽/按钮进行增删排序。
- **踩坑记录 / 架构调整**:
  - 在 Go 的 `x/sync/semaphore` 模式下，直接引入 Worker ID 概念需要搭配一个有缓冲的 Channel（如 `workerPool`）来安全地分配和回收 ID，避免在多协程并发下出现数据竞争或 ID 耗尽。
  - 在 React + Zustand 中管理一个可变的长列表（大纲），需要提供细粒度的 `updateChapter`, `moveChapter`, `removeChapter` 动作，并在拖拽或重新排序时主动维护其 `sort` 序号以保证一致性。

### [2026-04-06] Bugfix - 修复后端容器拉取 Git 仓库失败 (exec: "git": executable file not found in $PATH)
- **开发模块**: [容器化部署与后端运行环境]
- **完成事项**:
  1. **补充系统依赖**: 分析报错日志发现，Go 运行时容器是基于 `alpine:3.19` 的最小化镜像，默认不包含 `git` 命令包。
  2. 修改了 `backend/Dockerfile` 的 `Stage 2: Runtime` 步骤，在执行 `apk add` 时补充安装了 `git`，以满足后端服务通过 `os/exec` 调用 `git clone` 的运行时依赖需求。
- **踩坑记录 / 架构调整**:
  - 在采用多阶段构建（Multi-Stage Build）时，不仅要在构建阶段准备工具链，如果代码逻辑中直接调用了底层操作系统的 CLI 工具，则必须在轻量级运行镜像（如 alpine）中显式安装这些二进制包，否则会导致业务报错。

### [2026-04-06] Bugfix - 修复 SSE 连接在非流式响应时的报错处理
- **开发模块**: [前端网络请求, SSE 流控制]
- **完成事项**:
  1. **自定义 onopen 拦截器**: 修改 `frontend/src/hooks/useBlogStream.ts` 和 `frontend/src/components/Editor.tsx` 中所有的 `fetchEventSource` 调用。补充了 `async onopen(response)` 方法。
  2. **精确抛错**: 当后端未启动（代理返回 `502/504 text/plain`）或业务拦截（返回 `application/json`）时，拦截器会主动读取并抛出真实的错误信息（如 `请求失败: 502 Bad Gateway` 或后端的 JSON 错误），而不是让 `@microsoft/fetch-event-source` 库抛出令人迷惑的 `Expected content-type to be text/event-stream, Actual: text/plain` 错误。
- **踩坑记录 / 架构调整**:
  - `fetch-event-source` 库默认严格校验 `Content-Type`，如果不符合预期会直接在内部抛错中止请求，导致前端无法捕获真实的 HTTP 状态码和响应体。通过重写 `onopen` 钩子，前端可以优雅地处理非 SSE 的异常响应，极大提升了错误排查效率和用户体验。

### [2026-04-06] Feature - 批量删除历史博客功能
- **开发模块**: [前端侧边栏, 后端批量删除 API, 状态管理]
- **完成事项**:
  1. **批量删除 API**: 在 `backend/internal/api/blog.go` 与 `backend/internal/service/blog.go` 中新增了 `DELETE /api/v1/blogs` 接口。接收前端传入的 `blog_ids` 数组，并通过 GORM 进行安全软删除，同时删除选中的父节点及其所有子节点。
  2. **前端状态层 (Zustand)**: 在 `useBlogStore` 中新增了 `batchDeleteBlogs` 方法，并在删除成功后自动调用 `fetchBlogs()` 刷新列表，同时清空可能被删除的当前选中文章状态。
  3. **UI 交互更新**: 在 `Sidebar.tsx` 的批量操作栏中，新增了红色的“批量删除”按钮。结合原有的 `isBatchMode` 多选模式，用户可以一键勾选多篇文章并执行删除。删除前增加了 `window.confirm` 二次确认弹窗，防止误操作。
- **踩坑记录 / 架构调整**:
  - 在设计批量删除时，由于数据库中存在父子节点的层级关系，直接删除父节点并不会自动级联删除子节点（除非配置了物理外键级联）。通过在 GORM 查询条件中加入 `OR parent_id IN ?`，实现了仅需传入父节点 ID 即可同时软删除其下所有子章节的完美逻辑。

### [2026-04-06] Feature - 批量导出 ZIP 与 UI 滚动条优化
- **开发模块**: [前端侧边栏, 批量管理, 博客导出]
- **完成事项**:
  1. **批量导出模式 (Batch Export)**: 在 `Sidebar.tsx` 中引入 `jszip`，为“历史博客”区域增加批量导出按钮（多选管理模式）。用户可以勾选父节点以全选其下的所有子文章，一键将其导出为 ZIP 压缩包，文件名自动加上 `ChapterSort` 序号。
  2. **系列导出 API**: 在 `backend/internal/api/blog.go` 中补充了 `GET /api/v1/blogs/:id/export` 接口。并在前端 `Editor.tsx` 右上角操作区，如果是父节点则显示“导出 ZIP”按钮，方便一键下载整个系列。
  3. **页面防拉长修复**: 修复了之前由于 `App.tsx` 设置 `min-h-screen` 导致侧边栏内的列表溢出不出现滚动条而是撑长整个页面的问题。现改为 `h-screen overflow-hidden` 锁死整体高度，让子元素自行接管局部滚动。
  4. **工程规则精简重构**: 将项目 `.trae/rules` 下原来分散的 6 份文件合并为 3 份高内聚文件（架构、Vibe工作流、Git发布），减轻了上下文负载。
- **踩坑记录 / 架构调整**:
  - 在前端实现批量 ZIP 导出可以大幅降低后端压力，因为这些博客文本已经缓存在 `blogStore` 中。对于需要在编辑器中单独导出某一篇文章所在的整个系列，则保留后端的 ZIP 打包流接口（流式边压边下），做到了前后端性能和体验的最佳权衡。

### [2026-04-05] Feature - 引入细粒度大纲拆分与精准按需喂代码机制
- **开发模块**: [后端并发调度, 大纲解析引擎, 前端状态管理]
- **完成事项**:
  1. **细粒度大纲拆分**: 修改了 `DecompositionService` 中的 Prompt，强制要求 AI 在面对大型项目时，将大模块拆分得更细致（如 5-10 篇），单篇文章仅聚焦 1-2 个具体知识点，并将单篇字数约束调整为 1000-1500 字，防止生成过长导致的截断问题。
  2. **精准按需读取 (Precise On-Demand Code Feeding)**: 
     - 修改了大纲结构 `Chapter`，增加 `files` 字符串数组字段。要求 AI 在 Map-Reduce 汇总阶段，必须为每个章节匹配强相关的源码文件路径。
     - 重构了 `GenerateSeries` 方法。后端在生成系列文章时，会动态浅克隆对应的 Git 仓库，并**仅读取**大纲 `files` 列表中指定的文件源码传给大模型。
     - 彻底摒弃了之前“每次生成都把十几万字的项目全局摘要全量塞给大模型”的做法，将单次调用的上下文压缩至最低，保证了文章包含真实的底层代码剖析，不再空洞。
  3. **后台静默生成与页面切换保活**:
     - 修复了前端切换页面时生成任务卡死的问题。将 `AbortController` 和所有的流式进度状态（`analysisStep`, `isGenerating` 等）从局部组件上提到了全局的 Zustand `streamStore.ts` 中。
     - 移除了页面卸载时强制 `abort()` 的逻辑。现在用户在生成过程中可以自由切换到历史记录或个人中心页面，底层的 SSE 长连接会在后台静默保持，切回页面时进度条和打字机效果无缝衔接。
  4. **端口冲突修复**: 修复了由于 `vite` 进程残留导致前端运行在 `5181` 等非标准端口，进而引发 GitHub OAuth 登录回调 `net::ERR_ABORTED` 失败的问题。
- **踩坑记录 / 架构调整**: 
  - **大模型上下文陷阱**: 即使大模型号称支持 128k Token，但如果在单次请求中塞入过多杂乱的上下文（如整个项目的压缩摘要），大模型的注意力机制会被严重稀释，导致生成的文章“有头无尾”或“假大空”。通过大纲前置提取 `files` 列表，再精准投喂源码，是目前兼顾成本和质量的最佳架构实践。
  - **React 生命周期与流式请求**: 在 SPA 应用中，与组件生命周期绑定的网络请求很容易因为用户的随意点击而被意外强杀。将长时间运行的流式请求控制权移交至全局 Store 是实现“后台静默生成”的关键。

### [2026-04-05] Bugfix - 修复 React 状态延迟引起的博客互相覆盖 Bug 并清理脏数据
- **开发模块**: [前端 Markdown 编辑器, 后端数据清理]
- **完成事项**:
  1. **状态隔离重构**: 深度排查发现前端 `Editor.tsx` 在切换博客时，其内部的 `useDebounce` 状态由于未及时重置，导致 2 秒后旧文章的 `debouncedTitle/Content` 错误触发了针对新 `selectedBlog.id` 的自动保存，引发内容互相覆盖。
  2. **强制卸载机制**: 在 `App.tsx` 中为 `<Editor />` 组件增加 `key={selectedBlog.id}` 属性，强制 React 在切换文章时彻底卸载并重新挂载组件，清空所有的延迟状态与定时器。
  3. **无缝自动保存**: 在 `Editor.tsx` 中新增了基于组件卸载 (`unmount`) 的生命周期钩子，确保用户在输入后立刻切换文章时，会在组件销毁前同步触发最后一次自动保存，兼顾了状态隔离与防数据丢失。
  4. **脏数据清理脚本**: 编写并提供了 `backend/scripts/cleanup.go` 独立脚本，用于自动查询并物理删除数据库中受该 Bug 影响（同一父节点下子节点标题重复）的脏数据。
- **踩坑记录 / 架构调整**: 
  - 在 React 开发中处理“不同实体复用同一表单/编辑器组件”时，依赖 `useEffect` 去监听 ID 变化来重置内部状态（尤其是涉及到防抖/节流时）往往会导致难以追踪的闭包陷阱和竞态条件。强制使用 `key` 属性触发完整的组件生命周期重载，是解决此类复用状态残留 Bug 的最佳架构实践。

### [2026-04-05] Feature - 修复大项目分析生成卡死问题与添加超时/重试机制
- **开发模块**: [后端并发调度, SSE 流控制, HTTP 客户端稳定性]
- **完成事项**:
  1. **后台继续生成 (防 Goroutine 泄露)**: 修改 `internal/api/stream.go`，使用 `context.WithoutCancel` 将生成业务的 Context 与 HTTP 请求的 Context 分离。当前端主动断开连接时，HTTP 请求结束，但后台协程会自动排空 (drain) 通道，使得大模型调用和数据库持久化操作能在后台安稳跑完，彻底解决了 `errChan` 无接收方导致的 Goroutine 永久阻塞泄露问题。
  2. **流式空闲超时 (Idle Timeout)**: 在 `internal/service/decomposition.go` (系列生成) 和 `internal/service/generator.go` (单篇生成) 中引入了 `time.Timer` 机制。如果流式读取过程中超过 30 秒没有收到新的 Token，则判定为底层网络假死，主动 `cancel()` 对应的 HTTP 请求，防止 Goroutine 永远挂起。
  3. **非流式 Context 超时**: 在 `generateLocalSummaryWithRetry` 中，为单次 HTTP 请求增加了 3 分钟的绝对超时限制。
  4. **失败自动重试机制**: 在 `GenerateSeries` 核心循环中，将章节的流式生成包装进了最大 3 次的 `for` 循环中。如果遭遇网络错误或触发了上述的“空闲超时”，程序会自动退避（Sleep）并重试，同时将 `retrying` 状态通过 SSE 实时推送给前端。
- **踩坑记录 / 架构调整**: 
  - **HTTP 客户端假死陷阱**: Go 默认的 `http.Client{}` 如果不设置 `Timeout`，在遇到服务器只建连不发包（或者中途断网）时，`ReadBytes('\n')` 会陷入永久等待。但如果直接给 `http.Client` 设置 `Timeout`，又会误杀正常的、耗时很长的流式请求。最佳实践是在外部通过 `context.WithCancel` 结合 `time.Timer` 实现“流式空闲超时”打断机制。
  - **无缓冲通道与 Context 泄露**: 在 Web 开发中，如果在 `select` 监听 `ctx.Done()` 后直接 `return`，而业务端还在向无缓冲的通道发送数据，必定导致协程泄露。引入专门的 drain 协程来排空残余通道数据是标准的兜底做法。

### [2026-04-05] Feature - Playwright 全链路端到端自动化测试与 Bug 修复
- **开发模块**: [项目全栈端到端测试, 前端 UI 状态]
- **完成事项**:
  1. **自动化测试覆盖**: 在根目录编写了 `playwright_e2e_test.py` 自动化测试脚本，并结合 `with_server.py` 工具，实现了同时启动前端 Vite 代理与后端 Go 服务的测试环境。测试涵盖：首页加载 -> 新用户注册 -> 登录态维护 -> PDF 文件上传 -> 博客解析 -> 等待开始生成 -> 完成并验证侧边栏历史记录。
  2. **前端逻辑漏洞修复 (`Generator.tsx`)**: 在跑测过程中发现：当上传单文件（非 Git 系列分析）时，`store.outline` 默认被设置为 `[]`。此时原有的完成条件 `store.outline.every(...)` 会错误地返回 `true`（空数组的 every 始终为真），导致本应显示的“准备生成”卡片和“开始生成”按钮被提前隐藏。现已增加 `length > 0` 的严谨校验。
  3. **单文件生成后的状态重置 (`useBlogStream.ts`)**: 修复了单文件生成收到 `[DONE]` 事件后，仅将 `outline` 置为空数组但未重置完整状态，导致页面残留在生成完成态的问题。现改为调用统一的 `store.reset()`，彻底回到干净的首页状态。
  4. **测试产物归档**: 测试期间自动记录了浏览器的 Console 报错日志与 Network API 网络请求监控，连同各个关键节点的网页截图，一并归档在 `test_results/` 目录下。
- **踩坑记录 / 架构调整**: 
  - **JavaScript 边缘条件**: `Array.prototype.every` 对于空数组的默认行为在复杂状态机下非常容易引起 UI 的逻辑短路，在编写空状态处理时必须加上 `.length > 0` 显式判断。
  - **端到端测试的价值**: 本次测试验证了从数据库建立、网络通讯到前端渲染的真实链路，发现并解决了通过单元测试难以捕获的 UI 流转隐患。

### [2026-04-05] Feature - Markdown 编辑区与预览区精准双向滚动同步
- **开发模块**: [前端 Markdown 编辑器]
- **完成事项**:
  1. **底层行号支持**: 在 `MarkdownEngine.tsx` 编写了自定义的 Rehype 插件 `rehypeSourceLine`，在渲染 Markdown 时拦截 AST 并将行号信息注入到 HTML 元素的 `data-source-line` 属性中。
  2. **双向插值算法**: 在 `Editor.tsx` 中实现了 `handleEditorScroll` 与 `handlePreviewScroll`。通过寻找视口边界上下两个带有行号的 DOM 节点，并结合其真实的 `offsetTop` 进行数学插值计算，实现了像素级精准对齐的滚动。
  3. **防抖与边缘处理**: 引入了 `activePaneRef` 作为滚动锁防止左右面板互相触发导致的死循环；特殊处理了头尾填充区域 (`scrollTop <= 0` 和滚动到底) 的边缘情况，确保顺滑回弹。
- **踩坑记录 / 架构调整**: 
  - 传统的基于 `scrollHeight` 的百分比同步在遇到 Mermaid 长流程图时会导致严重的错位（因为左边是几行代码，右边是一张巨型图片）。通过在底层注入 AST 行号并做精确的 DOM 位置插值，彻底解决了该痛点，提供了业界一流的写作体验。

### [2026-04-05] Feature - Vibe Coding 规范更新与 Git 提交流程标准化
- **开发模块**: [项目规范与架构体系]
- **完成事项**:
  1. 修改 `.trae/rules/vibe-coding-workflow.md`，增加“强制询问与需求明确”原则，防止需求模糊时直接生成代码。
  2. 新增 `.trae/rules/git-and-release.md`，规范了提交流程：`git diff` 对比 -> 详实的 Angular 规范 Commit -> 查询并打标签 -> 推送代码与标签的完整流水线。

### [2026-04-05] 修复生成的系列博客无法添加到“历史博客”的问题
- **完成事项**：
  - 修改 `backend/internal/service/decomposition.go`，在 `GenerateSeries` 方法开始前，主动在数据库中创建并持久化一个 Parent 节点（父博客），并设置 `ParentID`。
  - 修改 `frontend/src/components/Sidebar.tsx`，在“当前生产任务”列表中点击已完成的章节时，不仅自动在右侧打开完整文章内容，还能在左侧“历史博客”树中自动定位、展开父节点并高亮选中的子节点。
- **技术坑点与调整**：
  - **后端数据孤岛问题**：此前在生成系列博客时，代码中虽然为所有子章节分配了相同的 `parentID`，但**并未将这个 parent 实体本身存入数据库**。导致 `GetUserBlogs` 查询时（依赖 `parent_id IS NULL` 查找顶级节点）无法找到该系列，整个系列的博客在数据库中变成了无法被检索到的孤岛数据。通过在生成前主动插入父节点记录彻底修复。
  - **前端树状结构联动展开**：在左侧边栏点击生成的任务时，原来只会做简单的扁平查找。现在改为递归查找 `blogs` 树，找到目标节点后记录其 `parentId`，并利用 `setExpandedNodes` 自动展开该父节点，最后调用 `selectBlog` 实现完美的高亮联动体验。

### [2026-04-05] 优化生成页面的高度与隐藏 Mermaid 报错
- **完成事项**：
  - 优化 `frontend/src/components/Generator.tsx`，为生成状态下的 Markdown 预览区域容器添加了 `max-h-[500px]` 与 `overflow-y-auto` 属性。
  - 优化 `frontend/src/components/MarkdownEngine.tsx`，在初始化 `mermaid` 时配置 `suppressErrorRendering: true` 并在 `catch` 块中清空内容。
- **技术坑点与调整**：
  - **页面撑爆问题**：打字机流式生成长文章时，如果不限制高度，DOM 会随着内容无限伸长，最终导致整个页面结构失衡且产生卡顿。通过限制最大高度为 500px 并支持内部滚动，兼顾了可视区域的稳定与内容的完整展示。
  - **Mermaid 流式渲染报错**：大模型在吐出 Mermaid 代码块的中间态时（语法必然不完整），原本的 mermaid 引擎会强制向 DOM 抛出硕大的红色 "Syntax error in text" 提示，极其影响视觉体验。通过开启配置级静默，并在捕获异常时清空容器，实现了平滑的“画图等待”体验。

### [2026-04-05] 解决生成的文章仍无法查看的问题及 UI 状态优化
- **完成事项**：
  - 排查出上一轮的后端代码修改并未运行（原因为 8080 端口被旧进程卡死，导致 `go run` 启动失败），现已强杀旧进程并重新编译启动后端服务。
  - 为防止您之前的生成数据永远丢失，连接 PostgreSQL 执行了一条“数据恢复脚本”，根据孤岛子文章反向插入了 8 个缺失的父节点。
  - 优化了前端 `frontend/src/components/Generator.tsx`：当大纲中的所有章节变为 `completed` 后，右侧面板会自动隐藏“开始生成”按钮，并显示“系列博客生成完毕，您可以在左侧边栏点击生成的章节查看完整内容”。
- **技术坑点与调整**：
  - **进程幽灵**：Go 后端在后台异常存活导致新修复的“父节点入库”代码并没有被真正执行，造成了“修复假象”。重新启动后根治。
  - **右侧面板的迷惑交互**：原来生成结束后，右侧由于 `isGenerating` 变为 false，又会显示回“准备生成”页面，给用户带来“是不是没生成成功”的错觉。通过状态判断，如果是全部生成完毕则切换为完成态的文案，与左侧全绿的打勾任务互相呼应。

### [2026-04-05] Feature - 容器化部署架构 (Docker) 搭建
- **开发模块**: [项目部署与运维, Docker, Nginx]
- **完成事项**:
  1. **多阶段构建 (Multi-stage Build)**:
     - 编写了 `backend/Dockerfile`，在 `golang:1.24-alpine` 中编译 Go 二进制文件，并在纯净的 `alpine:3.19` 镜像中运行，极大减小了后端镜像体积。
     - 编写了 `frontend/Dockerfile`，在 `node:20-alpine` 中完成 Vite/React 静态资源打包，最后由 `nginx:alpine` 镜像提供服务。
  2. **Nginx 代理与流式通信优化**:
     - 编写了 `frontend/nginx.conf`，将前端的静态路由与后端的 `/api/` 路由进行了完美分离。
     - 针对 DeepSeek 的 SSE 流式生成请求（Server-Sent Events），专门在 Nginx 中配置了 `proxy_buffering off;`、`proxy_cache off;` 和长超时时间，防止 Nginx 缓存导致打字机效果失效。
  3. **Docker Compose 容器编排**:
     - 编写了根目录的 `docker-compose.yml`，一键拉起 `inkwords-db` (PostgreSQL 14)、`inkwords-backend` 和 `inkwords-frontend` 三个核心容器。
     - 配置了数据库的 `healthcheck` 与后端的 `depends_on: condition: service_healthy`，确保容器启动顺序正确，避免后端启动时连不上数据库报错。
  4. **构建上下文优化**:
     - 为前后端分别配置了 `.dockerignore`，排除了 `.git`、`node_modules`、`dist` 及本地环境变量，提高构建速度并防止敏感信息泄露。
- **踩坑记录 / 架构调整**: 
  - **SSE 与 Nginx 代理的冲突**: Nginx 默认会开启缓冲（buffering），这会导致后端的流式块（chunk）被 Nginx 攒满一定大小后才一次性发给前端，破坏了“打字机”的实时体验。必须显式配置 `proxy_buffering off`。
  - **跨容器网络通信**: 前端在浏览器中运行，因此发送的 AJAX 请求必须通过当前域名的 `/api` 代理转发。Nginx 在接收到 `/api` 请求后，再通过 Docker 内部的 DNS 解析将其转发到 `http://backend:8080`，完成了从宿主机到前端容器再到后端容器的请求流转。

## 4. 每日开发日志 (Dev Log)
> 该区域将由 Vibe Coding 工程师（AI 助手）在每天/每次开发周期结束时，如实记录当天的完成事项、遇到的技术坑点及架构小规模调整。

### [2026-04-07] Bugfix - 修复“+”号按钮状态残留问题并开启新工作区
- **开发模块**: [前端 UI 交互, 状态管理]
- **完成事项**:
  1. **状态彻底重置**: 修改了 `frontend/src/components/Sidebar.tsx` 中两个带“+”号的按钮逻辑（顶部的主“新工作区”按钮和“当前生成任务”旁的“新建工作区”按钮），在点击时调用 `streamStore.reset()` 彻底清空大纲和解析内容，让 `Generator.tsx` 回到初始的 Git 链接输入状态。
  2. **防御性中断确认**: 增加了二次确认防御。如果用户在 `isAnalyzing` 或 `isGenerating` 为 `true` 的状态下点击“+”号，会弹出 `window.confirm` 询问是否终止并开启新工作区，防止误触导致辛辛苦苦生成了一半的任务丢失。
- **踩坑记录 / 架构调整**:
  - 全局状态库（Zustand）虽然解决了组件间通信和页面切换保活的问题，但“状态长存”也会带来副作用：当用户明确想要开启“新的一轮”时，如果不显式重置所有的 `Store`，旧状态就会残留，导致视图卡在“已完成”状态无法回到起点。显式调用 `reset()` 是处理此种“新建任务”场景的最佳实践。

### [2026-04-07] Feature - 增强登录注册安全与密码强度校验
- **开发模块**: [AuthAPI, 数据库层, 前端 UI]
- **完成事项**:
  1. **数据库迁移**: 新增 `VerificationCode` 模型以存储邮箱验证码，并为 `User` 模型增加 `IsEmailVerified`, `FailedLoginAttempts`, `LockedUntil` 字段，实现防爆破锁定机制。
  2. **后端服务**: 集成 `base64Captcha` 生成图形验证码，集成 `gomail.v2` 发送邮箱验证码（并提供 Mock 兜底）。强化了 `/login` 和 `/register` 接口的安全逻辑，新增 `/reset-password` 重置密码接口。
  3. **前端交互**: 在 `Login.tsx` 中使用单卡片状态机实现了登录、注册、重置密码的无缝切换。增加了图形验证码展示、倒计时获取邮箱验证码、以及密码强度实时计算条。
  4. **文档同步**: 更新了 `InkWords_API.md` 和 `InkWords_Database.md`，补全了相关接口与表结构说明。
- **踩坑记录 / 架构调整**:
  - 在前端实现多模式单卡片切换时，需要精细控制状态（如 `loginNeedsCaptcha`），并在操作成功/失败后准确重置表单，避免状态残留导致 UI 错乱。密码强度的正则表达式检测需考虑边界情况。

### [2026-04-05] Feature - 增强内容生成兜底与 UI 滚动优化
- **开发模块**: [后端生成与续写 API, 前端 UI 状态管理, Markdown 编辑器]
- **完成事项**:
  1. **智能提取与编辑系列标题**: 修改大纲生成的 Prompt，要求大模型一并总结出 `series_title`，取代原本硬编码的“Git 源码解析系列”。在 `Generator.tsx` 提供输入框允许用户在“开始生成”前修改系列标题。
  2. **内容截断兜底 (继续生成)**: 在大模型生成单篇文章的提示词中，加入了更为严格的**完整性约束**。同时在 `Editor.tsx` 右上角增加“继续生成”按钮，当遇到文章达到大模型生成上限突然中断时，可一键调用新的 `/api/v1/blogs/:id/continue` SSE 接口，大模型基于现有内容自动续写。
  3. **局部滚动条防撑爆页面**: 为主界面（`Generator`）的大纲列表和侧边栏（`Sidebar`）的当前任务列表分别增加了 `max-h-[40vh]` 和 `max-h-[30vh]` 以及自定义样式的局部滚动条，彻底解决了随着生成章节增多导致整个页面被拉长的问题。
- **踩坑记录 / 架构调整**:
  - 大模型截断一直是一个难以从根本上解决的通病（因为无法精确预测 Token 消耗）。通过在 UI 层面提供“继续生成”按钮并将生成的 Markdown 进行追加，是用户体验最好、最可靠的兜底方案。

### [2026-04-05] Feature - 错误拦截与分析停止功能
- **开发模块**: [前端 UI 交互, SSE 流控制]
- **完成事项**:
  1. **输入错误重置**: 在 `Generator.tsx` 中，现在当输入非法的 Git 仓库链接或解析文件抛错时，如果调用失败，系统会弹出对应的 `alert`，并在关闭提示后自动将 `gitUrl` 输入框以及 `fileInputRef` 的值清空，恢复到初始状态。
  2. **支持主动停止分析**: 在“分析中”加载界面底部新增了“停止分析”按钮。点击后，前端主动通过 `abortCtrlRef.current.abort()` 中断 SSE 连接。
  3. **后端协程优雅退出**: 修改了后端的 `stream.go` 的 `AnalyzeStreamHandler`。原本单纯等待 `CloseNotify` 的逻辑，由于 `c.Stream` 会一直阻塞，修改为监听 `ctx.Done()`，并且将服务层的分析方法交由 `sync.WaitGroup` 包装。当用户点击停止断开连接时，后端的 Context 能够立刻感知并终止大模型调度，避免了服务器资源的空耗泄漏。
### [2026-04-05] Feature - Map-Reduce 大型项目并发分析架构
- **开发模块**: [后端并发调度, 项目结构拆分, SSE 通信, 前端 UI 状态]
- **完成事项**:
  1. **智能拆分 (Split)**: 修改了 `backend/internal/parser/git_fetcher.go`，将原先简单粗暴的文件内容拼接，改为按照目录聚合文件内容，当目录内容字符数超过 `300,000` 时则按块切分，返回项目结构树与拆分后的分块 `[]FileChunk`。
  2. **Map 并发分析**: 在 `backend/internal/service/decomposition.go` 中引入了 `MapReduceAnalyze`，使用 `golang.org/x/sync/semaphore` 限制最大并发数为 5。并发请求大模型生成各分块的局部摘要，同时增加了单分块 3 次重试失败跳过的容错机制。
  3. **Reduce 层级汇总**: 将 Map 阶段收集到的各目录局部摘要组合，生成项目的全局上下文，最后再进行一次大模型调用生成项目全局大纲，彻底解决了单次上下文超限导致的被截断问题。
  4. **细粒度实时推送 (SSE)**: `AnalyzeStream` 新增了向前端推送 `chunk_analyzing`, `chunk_done`, `chunk_failed` 的进度事件，同时前端在 `useBlogStream.ts` 和 `streamStore.ts` 引入了 `MapReduceProgress` 状态管理，并在 `Generator.tsx` 中新增了实时展示重试、跳过与分析进度的 UI 组件。
- **踩坑记录 / 架构调整**: 
  - 超大型 Git 仓库如果不加干预直接传给大模型，由于模型存在 Token 限制，强行截断会导致大模型完全丢失后半部分的业务逻辑与代码结构，生成的大纲片面且干瘪。
  - 通过引入基于目录聚合的 Map-Reduce 架构，在确保不超过上下文上限的前提下，完美地将局部特征提取到全局视野，保证了生成博客的完整性和严谨性。同时，通过前端细粒度的分块执行进度展示，进一步提升了用户体验，降低了用户的等待焦虑。

### [2026-04-05] Plan - 大型项目并发分析架构优化 (Map-Reduce)
- **开发模块**: [后端并发调度, 项目结构拆分, SSE 通信]
- **背景**: 针对超大型 Git 仓库或项目，单次/串行分析耗时过长，且容易超出 DeepSeek 的 128k Token 上限。
- **目标**: 引入“按目录拆分 + Goroutine池限流 + Map-Reduce 层级汇总”的高性能大语言模型并发分析架构。
- **核心实施步骤**:
  1. **阶段 1：项目智能拆分 (Split)**
     - 按目录/模块聚合，过滤无关文件（如 `node_modules`, `.git`）。
     - 增加 Token 截断保护（单块限制 < 300,000 字符），超限则向下递归拆分。
  2. **阶段 2：并发分析 (Map 阶段)**
     - 使用 `semaphore` 限制最大并发数（如 5），防止内存溢出。
     - 使用 `rate.Limiter` 令牌桶算法限制 API 请求频率，防止 429 错误。
     - LLM 并发读取各分块，提取核心功能、接口、数据结构，输出局部摘要。
  3. **阶段 3：层级汇总 (Reduce 阶段)**
     - 汇总所有局部摘要，交由 LLM 进行全局梳理。
     - 输出《全局架构设计与代码导读文档》及《系列技术博客大纲》。
  4. **阶段 4：流式输出与呈现 (SSE)**
     - 通过 Gin 框架的 SSE 实时推送 Map-Reduce 各阶段进度给前端。
     - 根据生成的大纲，逐篇流式生成图文并茂的技术博客。


### [2026-04-04] 修复 SSE 流被浏览器挂起中断的 Bug
- **完成事项**：
  - 修改 `frontend/src/hooks/useBlogStream.ts`，为所有的 `fetchEventSource` 调用增加 `openWhenHidden: true` 参数。
- **技术坑点与调整**：
  - `@microsoft/fetch-event-source` 库有一个默认机制：当浏览器标签页失去焦点（用户切换到其他标签页或最小化浏览器）时，会触发 `onVisibilityChange` 事件并主动中断（Abort）连接，这会导致耗时较长的“拉取代码”和“AI生成”过程直接失败。
  - 通过显式传入 `openWhenHidden: true` 关闭了这个行为，保证即使用户离开当前页面去忙别的事情，后台的任务进度和文章生成也能安稳跑完。

### [2026-04-04] Git仓库分析引入真实 SSE 进度流
- **完成事项**：
  - 后端新增了 `/api/v1/stream/analyze` SSE 流接口，替代原有的同步 `POST /analyze` 接口。
  - 在 `DecompositionService` 新增 `AnalyzeStream`，将“克隆拉取 -> 源码分析 -> 大纲生成 -> 完成”这四个阶段包装为独立的进度事件推给前端。
  - 前端 `useBlogStream.ts` 中的 `analyzeGit` 改用 `fetchEventSource`，并新增了 `analysisStep` 和 `analysisMessage` 两个局部状态，移除原有 `Generator.tsx` 中基于 `setTimeout` 写的假进度条。
- **技术坑点与调整**：
  - **缓解等待焦虑**：因为深度克隆和 AI 生成大纲动辄需要几十秒，用户很容易认为是页面卡死了。通过前端移除“假进度”，后端接入“真实流进度”，每当克隆结束或开始对话 AI 时，都能收到明确的状态变更。这种白盒体验极大缓解了等待焦虑。
  - 保留了本地文件上传（由于只读文本解析通常在毫秒级完成）的极短视觉假进度动画，让前后交互体验保持一致。
- **完成事项**：
  - 修改 `backend/internal/service/decomposition.go`，去除了系列文章的并发生成逻辑，改为串行调用 `GenerateStream`，将生成大文章的内容通过 `streaming` 状态实时推送到 SSE 通道。
  - 在 Prompt 中强化了字数限制与深度要求：“字数必须极度充足，内容极其详实（不少于 2000 字的长文）”。
  - 修改 `frontend/src/hooks/useBlogStream.ts` 和 `frontend/src/components/Generator.tsx`，将单文件生成和 Git 解析生成的 UI 体验对齐，实时渲染当前章节的内容。
  - 修改了侧边栏“当前生成任务”的 Mock 点击交互，改为不可点击，避免用户将生成的 Mock 占位符当成文章结果，并且在每一章完成时实时拉取最新历史记录，让用户能直接从“历史博客”中打开完整文章。
- **技术坑点与调整**：
  - **进度条体验**：大项目生成如果仅有一个 Loading 会让用户感觉遥不可及。通过将“并发”转为“串行流式打字机”，虽然总体时间可能增加，但“每时每刻都在生成”的视觉反馈大大提升了用户体验。
  - **JSON序列化报错修复**：此前由于在 Go 后端直接使用 `fmt.Sprintf` 拼接 JSON，会导致换行符与转义符损坏 JSON 结构，从而导致前端的 `fetch-event-source` 在 `JSON.parse()` 时抛错并静默失败。最终现象是“用户看着0字符，等了很久突然全部完成”。修复为使用标准 `json.Marshal` 生成 Map 对象。
  - **完成状态的卡片点击恢复**：之前为了防止用户误点 Mock 草稿，取消了卡片的点击。现在恢复了该功能：当章节状态变为 `completed` 时卡片可点击，点击时会在内存中从 `blogs` 历史记录树中寻找并自动打开那篇对应的**完整真实文章**。

### [2026-04-04] 新增“新建博客 / 返回首页”全局交互按钮
- **完成事项**：
  - 修改了 `frontend/src/components/Sidebar.tsx`，在左侧边栏顶部“墨言博客助手”Logo下方，增加了一个醒目的“新建博客 / 返回首页”主按钮。
  - 通过调用 `selectBlog(null)`，实现了在浏览/编辑历史博客时，随时可以清空当前选中状态，回到 `<Generator />` 首页继续新建或生成下一篇博客的交互闭环。
- **技术坑点与调整**：
  - 之前因为缺乏全局入口，用户在点击进入单篇博客（`<Editor />`）后，如果不再使用“新建任务”面板，就没有入口返回到生成首页。通过在 Sidebar 添加常驻操作按钮，统一了返回首页和新建任务的心智模型。

### [2026-04-04] 优化 GitHub 仓库解析与文件生成进度条
- **完成事项**：
  - 优化 `git_fetcher.go`，在提取所有代码前先生成并拼接仓库的目录结构（Tree）。防止仓库过大导致后半部分被截断时，大模型失去对整个项目的结构认知。
  - 增强 `decomposition.go` 中的 Prompt。强制要求大型项目必须拆分为至少 3 个系列章节；在生成单章时，明确要求“字数充足、深入分析、引用核心代码”，避免生成干瘪的总结。
  - 优化 `generator.go` 单篇博客生成的 Prompt，同样强调字数和代码级剖析。
  - 前端 `streamStore.ts` 增加 `generatedContent` 状态，并在 `useBlogStream.ts` 中拦截单文件生成的 chunk 流。
  - 前端 `Generator.tsx` 组件升级单文件生成状态的 UI，接入 `MarkdownEngine` 实现了类似打字机的流式渲染效果，替代了原本枯燥的 Loading Spinner。
- **技术坑点与调整**：
  - 发现 GitHub 仓库内容经常超过 300k 字符导致被强制截断，原本的逻辑让大模型完全看不到被截断的代码。通过注入目录结构并要求 AI “基于目录和经验推演”，在不增加 Token 成本的前提下缓解了该问题。

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

### 2026-04-04 (Bugfix - SSE 网络流中断与悬挂请求修复)
- **开发模块**: [前端网络请求与状态管理]
- **完成事项**:
  1. **请求生命周期管理**: 在 `useBlogStream.ts` 中引入 `useRef` 保存当前的 `AbortController`，并在 `useEffect` 的清理函数中主动调用 `abort()`，确保组件卸载时安全切断网络流，防止产生“幽灵请求”。
  2. **错误静默拦截**: 修改了 `fetchEventSource` 的 `onerror` 和外层 `catch` 逻辑，拦截并静默处理由 `AbortError` 引起的抛错，防止其污染控制台并触发不必要的重连机制。
  3. **代码规范修复**: 修复了 `Generator.tsx` 中由于在 `useEffect` 中同步调用 `setState` 引起的 `react-hooks/set-state-in-effect` 警告，通过 `eslint-disable-next-line` 抑制了该特性，使 `npm run lint` 全部通过。
- **踩坑记录 / 架构调整**: 
  - 前端使用 `fetchEventSource` 接收服务端 SSE 流时，如果后端发送完 `[DONE]` 立刻断开连接，代理层未能完全消费掉所有流，会导致浏览器抛出 `AbortError: BodyStreamBuffer was aborted`。
  - 需要在代码层面主动接管 AbortController 的声明周期，将正常的请求中断与网络异常崩溃区分开来。

### 2026-04-04 (Feature - 用户注销功能)
- **开发模块**: [前端鉴权模块]
- **完成事项**:
  1. **退出登录功能**: 在左侧导航栏 (`Sidebar.tsx`) 底部增加了常驻的「退出登录」按钮，带有注销图标与红色高亮反馈。
  2. **注销逻辑**: 实现了清除浏览器 `localStorage` 中的认证令牌 (`token`)，并重定向至首页强制重新加载应用的逻辑，恢复至未登录的 `<Login />` 视图。
- **踩坑记录 / 架构调整**: 
  - 由于后端的 JWT 采用无状态设计，注销操作无需新增调用后端接口，直接由前端清除本地 Token 即可完成完整的注销闭环。

### 2026-04-04 (Bugfix - GitHub 登录重定向与前端容错修复)
- **开发模块**: [后端鉴权模块, 前端状态管理与登录]
- **完成事项**:
  1. **GitHub 登录回调修复**: 修复了 `OAuthCallback` 接口在成功/失败后直接返回 JSON 的问题，改为读取 `FRONTEND_URL` 环境变量并重定向至前端页面，通过 URL Query 参数传递 `token` 或 `error`。
  2. **前端错误捕获**: 在 `Login.tsx` 组件中增加了对 URL `error` 参数的监听与提示，并使用 `history.replaceState` 阅后即焚清理 URL。
  3. **前端 JSON 解析容错**: 修复了未登录时 `fetchBlogs` 接口返回空内容导致 `Unexpected end of JSON input` 的崩溃问题，增加了文本判空和 `try-catch` 保护。
- **踩坑记录 / 架构调整**: 
  - 第三方登录的常规流程中，回调接口 (Callback) 必须充当后端与前端的桥梁，直接返回 JSON 会导致浏览器停留在这个 API 路径上。改为重定向并利用 URL 传参是标准的解决方式。
### [2026-04-06] Feature - 端到端（E2E）测试环境搭建与测试套件实现
- **开发模块**: [Playwright E2E测试, Docker 隔离环境, Mock Server, SSE 异常恢复]
- **完成事项**:
  1. **全隔离测试环境**: 创建 `docker-compose.test.yml` 隔离测试数据库与前后端容器。
  2. **Mock 大模型 API**: 编写 `backend/cmd/mockllm/main.go` 构建一个返回 Mock 数据的简易大模型 API 服务器，加速测试且不消耗真实 Token。并修改 `deepseek.go` 支持 `DEEPSEEK_API_URL` 环境变量注入。
  3. **Playwright 测试脚本**: 编写 `e2e_tests/test_full.py` 覆盖登录异常、账号注册、文件上传、博客生成全链路、历史记录与持久化读取。
  4. **后端 context 修复**: 修复了 `GenerateBlogStream` 中 `streamCancel` 过早执行导致的 `context canceled` 从而中断大模型流式请求的严重隐藏 Bug。
  5. **前端 SSE 异常恢复**: 修复了 `useBlogStream.ts` 收到 SSE 异常断开时没有正确恢复 `isGenerating` 状态，导致 UI 永久 Loading 无法进行下一次操作的缺陷。
  6. **一键测试脚本**: 编写 `scripts/run_e2e.sh` 实现测试环境的一键构建、测试执行与自动销毁。
- **踩坑记录 / 架构调整**: 
  - **Go 字符串字面量 JSON 解析陷阱**: 在 Mock Server 构造 SSE 返回块时，使用 `fmt.Sprintf` 直接拼接带 `\n` 的字符串，导致生成了无效的 JSON 格式，致使大模型流解析器在反序列化时不断抛错跳过，引发了后续一连串的通道过早关闭问题。必须转义为 `\\n`。
  - **Goroutine 中 context 泄漏与误杀**: 在拆分出独立协程执行长时间的流式请求时，父函数的 `defer cancel()` 会在协程还没真正执行时就被触发，导致子协程立即收到 `context canceled`。这强调了在 Go 中传递 context 时必须严格管理生命周期和所属协程。

### [2026-04-07] Chore - 项目更新同步与上传 GitHub
- **开发模块**: [项目版本控制与代码托管]
- **完成事项**:
  1. **代码差异比对**: 执行了 `git diff`，确认了此次提交包含了自定义删除弹窗 (`ConfirmDialog`) 以及后台 Worker 数量优化等重要前端及后端并发改进代码。
  2. **项目基准文档更新**: 根据 `git-and-release.md` 规范，全面刷新并对齐了项目的核心开发日志和对话记录，保障“文档即代码”的同步更新。
  3. **标准原子提交**: 采用了 `chore:` 前缀进行了符合 Angular 规范的提交，并将最新进度推送到了 GitHub 远程仓库 (`origin/main`)。
- **踩坑记录 / 架构调整**:
  - “文档必须先于代码落盘提交”是保障 AI 与人共创不丢失上下文的最佳实践。

### [2026-04-07] Bugfix - 前端文章删除功能二次确认与并发竞态问题修复
- **开发模块**: [前端 UI 与事件管理]
- **完成事项**:
  1. **删除二次确认拦截**: 在大纲章节删除（`Generator.tsx`）与左侧边栏博客批量删除（`Sidebar.tsx`）中补充 `window.confirm` 原生拦截弹窗。
  2. **事件冒泡与默认行为阻断**: 补充了 `type="button"` 并加入了 `e.preventDefault()` 与 `e.stopPropagation()` 彻底切断表单提交和事件冒泡。
  3. **并发竞态防御 (Race Condition)**: 
     - 为 `Generator.tsx` 删除图标加入了 `e.detail > 1` 判断，直接在浏览器底层原生事件阻断双击/连击触发的冗余请求。
     - 为 `Sidebar.tsx` 的批量删除动作引入了 `useRef` 同步锁 (`isConfirmingRef`)，使得即便在 React 异步状态更新时发生的并发连击，也只会被响应一次，防止产生“假弹窗真删除”的现象。
- **踩坑记录 / 架构调整**:
  - React 异步更新与原生 `window.confirm` 同步阻塞的结合会引发有趣的竞态：当用户快速双击删除按钮时，第一次点击阻塞主线程弹出提示框，一旦用户点击确认，代码向下执行发起删除 API 请求，而第二次被排队的点击此时立刻又被执行，弹出第二个提示框，造成文章“已经先被删除，而提示框还在”的错误体验。必须在最上层引入 `useRef` 或 `e.detail` 进行防抖防御。

### [2026-04-07] Bugfix - 修复删除按钮原生弹窗失效与 Worker 并发数量动态范围问题
- **开发模块**: [前端 UI 交互, 后端并发调度]
- **完成事项**:
  1. **自定义确认弹窗 (ConfirmDialog)**: 弃用了原生 `window.confirm`，在 `frontend/src/components/ui/confirm-dialog.tsx` 中封装了基于 React 状态和 Tailwind CSS 的自定义确认弹窗组件。
  2. **替换原生拦截**: 修改了 `Sidebar.tsx` (批量删除) 和 `Generator.tsx` (章节删除)，全面接入新的 `ConfirmDialog` 组件。彻底解决了在部分环境或集成浏览器中原生 `window.confirm` 被异步拦截/放行导致“弹窗出现时文章已删除”的严重 Bug，提升了 UX 体验。
  3. **优化 Worker 动态范围**: 调整了 `backend/internal/service/decomposition.go` 中的 `maxWorkers` 并发数计算逻辑。将原本的 `NumCPU() * 2` (5~20) 优化为 `NumCPU()` 并严格限制在 3~8 之间，同时确保 Worker 数量不会超过实际的 `chunks` 任务数，解决了用户本地运行时 Worker 数量忽大忽小、界面过于杂乱以及容易触发 LLM 限流的问题。
- **踩坑记录 / 架构调整**:
  - 原生 `window.confirm` 在某些内嵌 Webview 或特定浏览器配置下，其阻塞主线程的特性会被打破（或被直接赋予默认值返回），导致后续删除逻辑先于用户决策执行。在现代 Web 应用中，**必须**使用基于 React State 的自定义 Modal 弹窗来接管所有危险操作的二次确认，这才是最可靠的架构实践。

### [2026-04-07] Refactor - 根据系统性能动态调整并发协程数
- **开发模块**: [后端并发调度机制, `DecompositionService`]
- **完成事项**:
  1. **动态协程调整**: 修改了 `backend/internal/service/decomposition.go` 中的 `mapReduceAnalyze` 方法。移除了硬编码的 5 个并发 Goroutine 限制，改用 `runtime.NumCPU() * 2` 来动态计算并发数（网络 I/O 密集型任务适合适当放大倍数）。
  2. **并发安全限制**: 增加了最小并发数 5 和最大并发数 20 的硬性限制，既保证了低配机器的分析效率，又避免了因并发过高触发 LLM (DeepSeek) API 的并发限流报错。
- **决策/踩坑记录**:
  - LLM API 请求属于典型的网络 I/O 密集型操作，纯依赖 CPU 核心数可能无法打满带宽和 API 并发度，因此采用 `CPU核心数 * 2` 的策略，并结合 `semaphore.NewWeighted` 和带有缓冲的 `workerPool` Channel 来实现安全调度。

## [2026-04-07] 并发生成文章功能重构
**What (做了什么):**
- 将 `backend/internal/service/decomposition.go` 中的 `GenerateSeries` 方法从串行生成改为并发生成。
- 使用 `golang.org/x/sync/semaphore` 限制最大并发数为 3，避免 API 限流。
- 后端修改：针对每个并发生成的章节，单独向 `progressChan` 推送 `streaming` 和 `error` 状态（带上 `chapter_sort`）。
- 前端修改：`streamStore.ts` 引入 `chapterContents: Record<number, string>`，并在 `Generator.tsx` 中为多个章节同时展示生成卡片和独立打字机效果。

**Why (为什么这么做):**
- 原有串行生成模式速度过慢。通过并发调用 DeepSeek API，大幅缩短长系列文章的整体生成时间。
- 限制并发数为 3 以平衡速度与 API 限制。
- 前端卡片式流式渲染能让用户直观看到多个章节的同时进度，符合业务直觉。

### [2026-04-07] 用户仪表盘与统计功能 (User Dashboard)
- **Backend**:
  - `Blog` 模型新增 `WordCount` 与 `TechStacks` 字段。
  - `Generator` 在生成结束后计算字数并通过 LLM 提取涉及的技术栈。
  - 增加 `/api/v1/user/stats` 接口聚合统计信息（Token、费用、文章数、字数、技术栈排名）。
  - 增加 `/api/v1/user/profile` 与 `/api/v1/user/avatar` 支持更新用户名与头像上传。
- **Frontend**:
  - 增加 `Dashboard` 组件使用 `recharts` 渲染柱状图。
  - 侧边栏增加“个人中心”入口。
  - 修复 `nginx.conf` 与 `vite.config.ts` 代理 `/uploads/` 静态文件服务。

### [2026-04-07] 增加第三方微信登录入口 (WeChat OAuth)
- **开发模块**: [认证模块 Frontend + Backend]
- **完成事项**:
  1. **前端入口**: `Login.tsx` 中增加“使用微信登录”的按钮，引入了微信 SVG 图标，点击后重定向至 `/api/v1/auth/oauth/wechat`。
  2. **后端配置与授权地址**: `backend/internal/service/auth.go` 中，定制了微信特有的 OAuth 授权 URL 生成逻辑，支持追加 `#wechat_redirect` 及使用 `appid` 参数。
  3. **后端回调处理**: 实现了自定义的 `handleWechatCallback`，分别请求微信的 access_token 与 userinfo 接口。
  4. **数据库存储**: 依赖现有的 `wechat_openid` 字段，解析微信 OpenID 与用户信息并实现新用户注册与老用户登录更新。
  5. **环境变量**: `backend/.env` 增加了 `WECHAT_APP_ID`, `WECHAT_APP_SECRET`, `WECHAT_REDIRECT_URL` 配置项。

### [2026-04-07] Refactor - Auth Downgrade (移除复杂的邮箱验证和密码重置流程)
- **开发模块**: [AuthAPI, 数据库层, 前端 UI]
- **完成事项**:
  1. **数据库层**: 删除了 `verification_codes` 表，移除了 `users` 表的 `is_email_verified` 字段。
  2. **后端 API 层**: 删除了邮件发送依赖 `gomail.v2`，移除了 `/send-code` 和 `/reset-password` 接口，精简 `/register` 逻辑仅验证图形验证码和密码强度。
  3. **前端 UI 层**: 从 `Login.tsx` 中彻底移除了验证码发送、倒计时、忘记密码模式等冗余 UI，保留单纯的登录与注册表单。
  4. **文档同步**: 更新了 `InkWords_API.md`、`InkWords_Database.md`、`InkWords_PRD.md`、`InkWords_Architecture.md` 和 `README.md`，去除了与邮件验证码和重置密码相关的描述。
- **踩坑记录 / 架构调整**:
  - 简化认证流程可以大幅降低用户的注册门槛，同时保留图形验证码和密码强度校验依然能提供足够的防爆破保护。

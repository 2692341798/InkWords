# 墨言博客助手 (InkWords) 项目架构文档

## 1. 系统整体架构设计

### 1.1 系统拓扑结构
墨言博客助手采用 **前后端分离的 Monorepo** 管理模式，整体架构分为三层：
- **展现层 (Frontend)**：基于 React 18 (Vite)，负责 OAuth 登录重定向、页面渲染、文件预处理、Markdown 实时预览、用量大屏展示。
- **业务层 (Backend)**：基于 Go (Gin)，负责 GitHub/微信 OAuth 回调处理、路由分发、文档/源码解析、Prompt 策略调度、DeepSeek LLM 接口交互及 Token 计量。
- **数据与存储层 (Storage)**：使用 PostgreSQL 14+ 存储博客快照、用户历史记录、第三方平台 OAuth2 授权 Token。采用“阅后即焚”策略，不在服务器持久化存储任何用户源文件（PDF/Word/Git）。

### 1.2 核心通信协议
- **常规请求**：基于 HTTP/RESTful API 交互（JSON 格式）。
- **流式响应与保活 (SSE & Heartbeat)**：在向前端推送 AI 生成文本时，使用 Server-Sent Events (SSE) 技术，并通过定时 Goroutine 发送 `event: ping` 心跳包，防止代理层（如 Nginx）因长时间无数据流而切断长连接。
- **大模型无缝续写 (Auto Continuation)**：底层 `deepseek.go` 客户端封装了 `FinishReason` 校验。如果发现返回截断（如 `length` 超过 token 限制），`generator.go` 会在服务端无缝触发新的一轮请求，并将后续输出直接拼接推送给前端，实现用户侧大文本生成的“无感”接续体验。
- **鉴权协议**：系统内部采用 **JWT (JSON Web Token)** 进行无状态用户鉴权；对外接入第三方平台登录或发文时，采用标准 **OAuth2.0** 协议。其中，第三方登录回调阶段通过 HTTP 重定向（携带 Token/Error）将控制权交还前端处理路由和鉴权状态。

## 2. 前端架构设计 (React 18)

### 2.1 目录结构与技术栈
- `src/components/`: 可复用的 UI 组件（基于 Shadcn UI + Tailwind CSS + `react-markdown` + 原生 Mermaid 渲染），保持极简浅色阅读风。
- `src/features/`: 核心业务模块（工作区 Uploader、历史侧边栏 Sidebar、渲染器 Renderer）。（当前实现在 App.tsx 中搭建了基础双栏布局）
- `src/store/`: 基于 **Zustand** 的全局状态管理，负责处理极简状态下的 Token、当前激活的博客 ID、以及 SSE 流式追加的 Markdown 内容（`streamStore.ts` 中管理了大纲 `outline` 与章节生成状态）。
- `src/services/`: 封装所有 HTTP/SSE 请求逻辑，统一管理接口异常。
- `src/hooks/`: 自定义 Hooks（如 `useBlogStream.ts` 封装了 `analyzeGit` 与 `generateSeries`，并利用 `@microsoft/fetch-event-source` 维持 POST 流）。
- **第三方工具库**：使用 `jszip` 处理前端轻量级的多文件/系列打包导出功能，减轻后端文件 I/O 压力。

### 2.2 核心渲染器架构 (Renderer)
- **MarkdownEngine**：基于 `react-markdown` 配合 GitHub 样式（去除边框），负责实时渲染流式抵达的文本。为了防止生成极长文章时撑爆页面导致卡顿，该容器设置了最大高度并支持内部滚动。引入了自定义的 Rehype 插件 `rehypeSourceLine`，通过拦截 AST 为所有 HTML 元素注入源文件的 `data-source-line` 属性，为双向精准滚动提供底层锚点支持。
- **Editor 双向滚动同步机制**：编辑区 (`Editor.tsx`) 实现了高度定制的防抖双向滚动逻辑。不采用简单的百分比同步，而是通过读取视口内的 `data-source-line` 属性，进行上、下文元素的行号与真实 `offsetTop` 偏移量的精准插值计算。这确保了即便是遇到巨大的图片或 Mermaid 流程图，左右两侧的文本依然能够像素级完美对齐。
- **MermaidViewer**：基于原生 Mermaid API 进行渲染（替代 `rehype-mermaid` 解决异步渲染冲突）。通过开启 `suppressErrorRendering: true` 并在捕获异常时清空容器，彻底静默了 LLM 流式输出中间态时的语法报错（"Syntax error in text"），保障极简平滑的视觉体验。同时自定义拦截逻辑，**强制注入默认主题配置**，移除所有由于 LLM 幻觉可能携带的 `style` 或 `classDef` 样式属性。

## 3. 后端架构设计 (Go 1.21+)

### 3.1 核心分层与依赖注入
Go 后端采用经典的“三层架构”，并通过依赖注入（Dependency Injection）实现各层解耦，方便单元测试。
- **Controller/API 层 (`internal/api`)**：基于 Gin 框架，处理 HTTP/SSE 请求的路由分发、参数校验（绑定 Request DTO）、JWT 鉴权。
- **Service 层 (`internal/service`)**：核心业务逻辑承载。包含 `DocService` (处理文档流转)、`GeneratorService` (组装 Prompt 与模型调度生成，原 LLMService)、`BlogService` (数据库事务控制与历史快照)、`PublishService` (管理平台发布)。
- **Repository 层 (`internal/repo`)**：封装底层存储逻辑（PostgreSQL），提供 `UserRepo`, `BlogRepo`, `TokenRepo` 等接口。
- **LLM 层 (`internal/llm`)**：负责封装底层针对 DeepSeek API 等外部大模型的直接请求，提供流式 (`stream=true`) 与非流式客户端接口。

### 3.2 解析器模块 (Parser)
负责将外部数据源转化为统一格式的纯文本。

- **GitFetcher**: 
  - 通过 `os/exec` 原生调用 `git clone --depth 1` 命令浅克隆目标仓库到临时目录。
  - **抗截断优化 (目录树生成)**：在拼接所有源码内容前，系统会先遍历并生成一份完整的**项目目录结构树 (Tree)**。这是为解决超大型项目超过 Token 上限导致代码被截断时，大模型失去对全局结构的认知而引入的关键架构优化。
  - 自动过滤 `.git`, `node_modules`, 二进制文件及常见的依赖锁定文件。
  - 解析完成后触发“阅后即焚”机制，立即删除临时目录。
- **DocParser**:
  - 统一使用 `os.CreateTemp` 获取临时文件（解决部分解析库需要文件句柄或真实文件路径的问题），并通过 `defer os.Remove` 确保物理销毁，严格遵守“阅后即焚”策略。
  - 支持 `.md`, `.txt` 等纯文本直接读取。
  - 支持 `.pdf` 格式（基于 `github.com/ledongthuc/pdf`）。
  - 支持 `.docx` (Word) 格式（基于 `github.com/nguyenthenguyen/docx`），并包含定制的 `stripXMLTags` XML 标签清洗策略。

### 3.3 Integration 模块 (第三方 OpenAPI 交互)
- 负责管理与掘金、CSDN、知乎等第三方平台的交互。
- 架构包含 **OAuth2 Handler**（处理重定向与 Callback 换取 Token）和 **Publisher Client**（封装各平台不同的发文 OpenAPI 接口，统一屏蔽差异并处理 Rate Limit 限流重试）。

## 4. 核心业务流转设计

### 4.1 单篇博客生成流转 (SSE Pipeline)
1. 前端上传文件/URL -> API 层接收并校验 Token，通过解析器提取纯文本内容。
2. 前端携带纯文本内容（`source_content`），向后端 POST `/api/v1/stream/generate` 发起 SSE 流式请求。
3. Service 层组装系统级 Prompt（强制“小白友好”、“加代码解释”、“无样式 Mermaid”）。
4. 通过 `DeepSeekClient` 调用 DeepSeek 接口，开启 `stream=true`。
5. 后端 Gin 通过 `c.Stream` 将模型返回的 Token 逐片 (Chunk) 转换为 SSE 事件（`event: chunk` 和 `event: done`）推送到前端。
6. 模型输出完毕，Service 层的 Goroutine 拦截到结束标志后，将完整的 Markdown 落库到 PostgreSQL，流程结束。
7. **异常截断兜底 (Continue Generation)**：若生成的内容因模型长度上限被截断，用户可通过前端触发 `/api/v1/blogs/:id/continue` 接口，后端读取当前文章内容并请求大模型“继续完成上文未写完的内容”，通过 SSE 实时追加至现有文章中。

### 4.2 大项目流式分析与串行生成机制
当 Parser (`GitFetcher` 或 `DocParser`) 提取的文本超长（系统判定为大项目或长篇教程）时，引入 **Map-Reduce 并发分析架构** 与 **按需精准读取机制 (Precise On-Demand Code Feeding)**：
1. **智能拆分 (Split)**：`GitFetcher` 按目录层级聚合文件内容，当单块超过 300,000 字符时自动拆分为多个 `FileChunk`，并附带项目全局树状结构。
2. **并发分析 (Map 阶段)**：`DecompositionService` 利用 Goroutine 池与 `semaphore` 信号量控制最大并发数（默认为 5 个并发 Worker）。针对每个 `FileChunk`，系统向大模型发起局部摘要请求（支持 3 次重试失败跳过）。期间，后端会主动向下推送 `{"step": 2, "message": "...", "data": {"status": "chunk_analyzing", "index": 1}}` 等细粒度事件，前端在大项目分析期间（Map-Reduce 阶段）展示多个 Worker 的并发执行状态（如分析中、重试、成功、失败），缓解等待焦虑。
3. **层级汇总 (Reduce 阶段)**：所有局部摘要拼接后，与全局目录结构合并，发起一次全局大纲规划请求。**关键优化**：此时大模型不仅会生成系列标题、章节的标题和摘要，还必须严格根据目录树输出该章节强相关的**文件路径列表 (`files` 数组)**。大纲生成后，允许用户对章节进行标题和摘要的修改，支持新增、删除、上移、下移，定制化程度极高。
4. **内容生成 (精准按需读取 + 串行打字机渲染)**：
   - 大纲确认后，前端携带 `outline` 数组及 Git 仓库地址请求 `POST /api/v1/stream/generate`。系列博客不会一拥而上导致并发熔断，而是通过大模型队列**串行**逐篇生成。
   - **极致上下文缩减**：后端在生成前，会临时将 Git 仓库克隆到本地。在串行生成每一篇文章时，**不再**将整个项目的摘要塞给大模型，而是严格按照该章节大纲中指明的 `files` 数组，精准读取并仅喂入这几个特定文件的真实源码。
   - 这确保了单次 API 调用的上下文被极大地缩减（通常控制在 10万 字符以内），大模型能够“看清”底层源码细节，彻底消除了长文截断和内容空洞的问题。
5. **断点保存与联动状态流**：每生成完其中一篇文章，立即落库并带有相同的 `ParentID` 与各自的 `ChapterSort`。前端右侧面板实时呈现当前正在生成的章节及总体进度，并更新该章节的 UI 状态（`generating` -> `completed`）。如果在生成过程中用户切换页面，前端的 `StreamStore` 会在后台**静默保持 SSE 连接**，文章依然会在后台安稳生成并落库，切回页面时进度无缝衔接。

## 5. 安全与部署架构

### 5.1 数据加密与脱敏
- **敏感数据存储 (OAuth2 Tokens)**：对掘金、CSDN等第三方平台的 Access Token/Refresh Token 必须在入库前采用 AES 等对称加密算法进行加密存储，出库时解密。禁止明文存储任何敏感凭证。
- **配置与密钥管理**：诸如 DeepSeek API Key、数据库密码、JWT Secret、加密盐值等敏感信息，必须通过环境变量 (`.env`) 或专用的配置中心注入到 Go 程序中，严禁硬编码至源码中或提交至版本控制系统。

### 5.2 防刷与 API 安全 (Rate Limit)
- **频率限制与防滥用**：在 Gin 的全局 Middleware 层，引入基于 Redis/内存缓存的 IP 粒度与 User 粒度限流器（Rate Limiter）。例如，限制单一用户每小时最多生成 5 篇文章，防止恶意爬虫消耗高昂的 LLM Token。
- **文件体积限制防护**：Nginx/Gin 必须在入口层直接拦截大于 50MB 的 Payload，防止恶意上传导致的 OOM（内存溢出）攻击。

### 5.3 网关与传输安全 (SSL & CORS)
- **传输加密**：生产环境必须强制启用 HTTPS，所有前后端交互（尤其是包含 Token 和 Cookie 的请求）必须通过加密信道传输。
- **跨域资源共享 (CORS)**：配置严格的 CORS 策略，只允许白名单中的前端域名发起跨域请求（含 SSE 请求）。
- **反向代理 (Nginx/Caddy)**：前端通过 Nginx 部署静态资源并处理 SSL 证书，同时将 `/api` 和 `/stream` 等后端请求反向代理到 Go Gin 服务，隐藏后端真实端口，并对前端路由开启 HTML5 History Mode 支持。

## 6. 容器化部署架构 (Docker)

本项目采用标准的 **Docker Compose** 编排方案，包含三个核心容器：
1. **Frontend (前端容器)**：基于 `nginx:alpine`，通过多阶段构建（Multi-stage Build）先在 Node.js 环境中打包 React 静态产物，再由 Nginx 提供静态文件服务，并将 `/api/` 路径的请求反向代理至后端容器（针对 SSE 请求做了特殊优化，关闭了 proxy_buffering）。
2. **Backend (后端容器)**：基于 `alpine`，通过多阶段构建在 Go 环境中编译出二进制执行文件，极大减小了生产环境的镜像体积（注：运行镜像中已显式安装 `git` 依赖以支持代码仓库克隆）。
3. **DB (数据库容器)**：基于 `postgres:14-alpine`，持久化挂载数据卷（Volume），并配置了 healthcheck 以确保后端服务在数据库就绪后再启动。
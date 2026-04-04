# 墨言博客助手 (InkWords) 项目架构文档

## 1. 系统整体架构设计

### 1.1 系统拓扑结构
墨言博客助手采用 **前后端分离的 Monorepo** 管理模式，整体架构分为三层：
- **展现层 (Frontend)**：基于 React 18 (Vite)，负责 OAuth 登录重定向、页面渲染、文件预处理、Markdown 实时预览、用量大屏展示。
- **业务层 (Backend)**：基于 Go (Gin)，负责 GitHub/微信 OAuth 回调处理、路由分发、文档/源码解析、Prompt 策略调度、DeepSeek LLM 接口交互及 Token 计量。
- **数据与存储层 (Storage)**：使用 PostgreSQL 14+ 存储博客快照、用户历史记录、第三方平台 OAuth2 授权 Token。采用“阅后即焚”策略，不在服务器持久化存储任何用户源文件（PDF/Word/Git）。

### 1.2 核心通信协议
- **常规请求**：基于 HTTP/RESTful API 交互（JSON 格式）。
- **流式响应**：采用 **SSE (Server-Sent Events)** 协议，前端单向接收由后端转发的 DeepSeek 模型的流式（Stream）Markdown 输出，提供打字机效果。
- **鉴权协议**：系统内部采用 **JWT (JSON Web Token)** 进行无状态用户鉴权；对外接入第三方平台发文时，采用标准 **OAuth2.0** 协议管理 Access Token 与 Refresh Token。

## 2. 前端架构设计 (React 18)

### 2.1 目录结构与技术栈
- `src/components/`: 可复用的 UI 组件（基于 Shadcn UI + Tailwind CSS + `react-markdown` + `rehype-mermaid`），保持极简浅色阅读风。
- `src/features/`: 核心业务模块（工作区 Uploader、历史侧边栏 Sidebar、渲染器 Renderer）。
- `src/store/`: 基于 **Zustand** 的全局状态管理，负责处理极简状态下的 Token、当前激活的博客 ID、以及 SSE 流式追加的 Markdown 内容（`streamStore.ts`）。
- `src/services/`: 封装所有 HTTP/SSE 请求逻辑，统一管理接口异常。
- `src/hooks/`: 自定义 Hooks（如 `useBlogStream` 用于封装 `@microsoft/fetch-event-source` 维持 POST 流，`useAuth`）。

### 2.2 核心渲染器架构 (Renderer)
- **MarkdownEngine**：基于 `react-markdown` 配合 GitHub 样式（去除边框），负责实时渲染流式抵达的文本。
- **MermaidViewer**：基于 `rehype-mermaid`，并自定义拦截逻辑，**强制注入默认主题配置**，移除所有由于 LLM 幻觉可能携带的 `style` 或 `classDef` 样式属性，保障图表的极简统一。

## 3. 后端架构设计 (Go 1.21+)

### 3.1 核心分层与依赖注入
Go 后端采用经典的“三层架构”，并通过依赖注入（Dependency Injection）实现各层解耦，方便单元测试。
- **Controller/API 层 (`internal/api`)**：基于 Gin 框架，处理 HTTP/SSE 请求的路由分发、参数校验（绑定 Request DTO）、JWT 鉴权。
- **Service 层 (`internal/service`)**：核心业务逻辑承载。包含 `DocService` (处理文档流转)、`GeneratorService` (组装 Prompt 与模型调度生成，原 LLMService)、`BlogService` (数据库事务控制与历史快照)、`PublishService` (管理平台发布)。
- **Repository 层 (`internal/repo`)**：封装底层存储逻辑（PostgreSQL），提供 `UserRepo`, `BlogRepo`, `TokenRepo` 等接口。
- **LLM 层 (`internal/llm`)**：负责封装底层针对 DeepSeek API 等外部大模型的直接请求，提供流式 (`stream=true`) 与非流式客户端接口。

### 3.2 Parser 模块 (阅后即焚解析器)
- **文档解析器 (`DocParser`)**：上传的 PDF/Word 被直接加载到内存中的 `io.Reader`，或使用极短生命周期的临时文件。提取纯文本并清洗乱码后，触发 `defer os.Remove` 或让 GC 回收，彻底防止文件滞留。
- **Git 仓库分析器 (`GitFetcher`)**：通过 `go-git` 库进行 Shallow Clone（浅克隆），内置一份黑名单规则（自动过滤 `.git`, `node_modules`, `.png`, `.exe` 等非文本文件）。提取所有有效源码后拼接成单一长文本，克隆目录随即删除。

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

### 4.2 大项目并发/串行调度机制
当 Parser 提取的文本超长（系统判定为大项目或长篇教程）时：
1. **大纲规划 (串行)**：Service 层先向 DeepSeek 发送一次请求，要求其根据项目全局代码生成一份“系列博客大纲”（例如分为 3 篇文章）。
2. **内容生成 (并发 + 队列)**：大纲确认后，系统通过 Goroutine 池（Worker Pool，限制并发数防止 DeepSeek 接口 Rate Limit）**并发或异步排队**向模型发送请求。
3. **断点保存**：每生成完其中一篇文章，立即落库并更新其状态为“已完成”，支持用户在网络中断后点击“继续生成”未完成的篇章。

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
# 墨言博客助手 (InkWords) - 架构设计与工程规范

## 1. 整体架构 (Monorepo)
项目采用前后端分离的 Monorepo 结构，根目录隔离：
- **`frontend/`**: 包含所有前端界面、状态管理和客户端逻辑。
- **`backend/`**: 包含所有的 RESTful API 服务、数据库交互、第三方登录与大模型通信。
- **`docker-compose.yml`**: 项目唯一的容器化编排入口。

## 2. 核心技术栈
### 2.1 前端 (Frontend)
- **核心框架**: React 18 + Vite
- **UI 库**: Tailwind CSS + Shadcn UI + Recharts
- **状态管理**: Zustand (含多 store：`blogStore`, `streamStore`, `authStore`)
- **流式通信**: `@microsoft/fetch-event-source` 维持 SSE 连接
- **Markdown 渲染**: `react-markdown` 配合 `rehype-highlight`、`remark-gfm` 和 `mermaid`。

### 2.2 后端 (Backend)
- **核心语言**: Go 1.21+
- **Web 框架**: Gin (`github.com/gin-gonic/gin`)
- **依赖注入**: 后端通过明确的构造函数（如 `NewAuthAPI(authService)`）进行依赖注入，降低 `api` 层和 `service` 层、全局变量之间的耦合，便于单元测试。
- **数据库 ORM**: GORM (`gorm.io/gorm` + `gorm.io/driver/postgres`)
- **认证与安全**: 
  - JWT Token (长短效签发) + GitHub OAuth (`golang.org/x/oauth2`)
  - 图形验证码防刷 (`github.com/mojocn/base64Captcha`)
  - 密码强度与连续登录失败防爆破锁定 (`LockedUntil`)
- **并发架构**: 引入了 Go 原生的 Goroutine 池与 `x/sync/semaphore` 信号量控制（动态范围 3~8），保障并发生成稳定且不超限。
- **特大型项目保护 (Map-Reduce)**:
  - **Map 阶段**: 按目录分块(针对 Git 仓库)或按字数智能段落分块(针对大于 1,000,000 字符的长文本文件)并发提炼局部摘要，当遇到 LLM 限流时启用带随机抖动的**指数退避 (Exponential Backoff)**。
  - **Reduce 阶段**: 当局部摘要过多（>20个）时，自动触发 **Tree Reduce** 多级树状汇总，将局部摘要分组提炼成中间层摘要后，再进行全局大纲合并，最高支持 15,000,000 字符上限。
- **深度剖析与博客再生 (Deep Generation & Regeneration)**:
  - **思考模式与 JSON 模式**：通过调用 DeepSeek 的 `Thinking` 模式加强逻辑推理，并在大纲生成、技术栈提取等场景强制启用 `json_object` 模式配合极低的 `Temperature` (如 0.1) 确保严格的结构化稳定输出。
  - **上下文注入**：在 `regenerate` 更新重写阶段，系统会从数据库提取旧版博客（截断至 50万 字符）并注入 Prompt，指导大模型基于最新源码进行“松散参考重写”，有效避免优秀沉淀丢失。
- **数据推送**: 基于标准 HTTP `text/event-stream` 实现 SSE 推送机制。

### 2.3 基础设施 (Infrastructure)
- **数据库**: PostgreSQL 14 (Docker volume 挂载持久化)
- **本地知识库导出**: 挂载宿主机目录到 `/app/obsidian` 用于生成包含 YAML Frontmatter 的 Markdown 笔记，直接接入用户的 Obsidian Vault
- **代理与网关**: Nginx (构建前端静态页面并反向代理后端 `/api/` 路径，配置 `client_max_body_size 100M` 以支持大文件解析)
- **大语言模型**: DeepSeek-V4-Flash API (支持 128k 输出及 1M Token 上下文)

## 3. 并发生成架构
在处理项目到系列博客的生成时，后端采取如下架构：
1. **模块扫描与卡片选择**：针对 Git 源码，使用无盘 `ls-tree` 与 GitHub API 极速提取一级核心代码目录；并行调用 LLM 结合根目录 `README` 内容智能推断每个目录的中文简介，生成模块卡片供用户多选，实现按需定向解析。
2. **Map-Reduce 分析**：针对选中的模块和超大文件（>5万字）进行预切片处理，并行调用 LLM 抽取各个分块摘要，最后合并生成结构化 JSON 大纲（包含子章节 `outline`）。
3. **多协程并发生成**：
   - 接收到前端下发的大纲后，启动多个 `goroutine` 为每个章节并行生成内容。
   - 使用 `semaphore.NewWeighted(3)` 将全局并发数严格限制为 3。
   - 每个 `goroutine` 均拥有独立的错误隔离环境，通过同一个 `progressChan` 向前端推送包含自身 `chapter_sort` ID 的 Chunk（数据切片）。
4. **系列导读生成**：
   - 所有单篇博客生成完毕后（`wg.Wait()` 返回），主流程自动触发一次 AI 调用，生成“系列导读”文章，将其作为整个系列的父节点，将各个单篇博客串联成专栏。
5. **前端批量更新防卡顿**：
   - 前端接收到密集交织的 SSE Chunk 时，使用 `pendingUpdates` 缓冲队列进行暂存。
   - 通过 `setTimeout(200ms)` 的节流（Throttle）机制，将缓冲区内的文本批量合并，定期只触发一次 Zustand 状态更新和 React DOM 重绘。
   - 极大缓解了多章节 Markdown 同时渲染导致的主线程卡死。

## 4. 部署架构 (Docker-First)
- **前端镜像**: 采用多阶段构建（Node.js 安装依赖并构建，Nginx 轻量级运行并作为反向代理网关）。映射宿主机 `80` 和 `5173` 端口（以解决 GitHub OAuth 回调端口兼容）。
- **后端镜像**: 采用多阶段构建（Go 官方镜像编译，Alpine 运行）。使用 `FRONTEND_URL` 和 `DATABASE_URL` 环境变量控制运行逻辑。
- **数据库**: PostgreSQL，初始化 `inkwords_db`。
- **容器互联**: 全部服务处于 `inkwords_default` 内部网络，后端连接数据库通过服务名 `db:5432` 互通。

## 5. 全局缓存机制 (Prompt Caching)
- **目标**：降低 DeepSeek Token 消耗，提高首字响应速度 (TTFT)。
- **原生支持**：全面拥抱 DeepSeek V4 API 级别的原生前缀缓存 (Prompt Caching)。
- **Prompt 结构重构**：将数百万字的巨量源码 `sourceContent` 提取至 `system` 消息并置于请求最前，将易变的“指令”置于 `user` 消息并置于请求尾部，以最大化原生缓存的命中率，将长文本输入成本降低 80% 以上。

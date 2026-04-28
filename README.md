# 墨言博客助手 (InkWords)

## 1. 项目简介 (About)
**墨言博客助手 (InkWords)** 是一款基于 **DeepSeek API** 的智能化技术博客写作辅助平台。
该平台致力于解决技术人员“写文档难、拆解长篇教程繁琐”的痛点。通过强大的源码/文档解析与 LLM 逻辑重构能力，它能将复杂的本地文档或 Git 仓库（如官方教程、开源项目）快速转化为结构清晰、内容详实的系列技术博客。

## 2. 核心特性 (Features)
- 🚀 **大型项目 Map-Reduce 并发拆解**：针对超长代码库或超大文本文件（>5万字），独创“智能分块 -> Goroutine并发 Map 局部摘要 -> Reduce 全局汇总”的并发架构，彻底解决大模型 Token 上限与上下文遗忘问题。
- 📚 **模块选择与系列生成**：自动极速扫描 Git 仓库提取核心模块，通过分析 README 智能生成每个目录模块的简短描述，用户自由勾选后，系统不仅并发生成单篇博客，最后还会自动生成一篇“系列导读”文章串联全集。
- ⚡ **全链路流式体验 (SSE)**：从分析 Git 仓库目录结构、拉取源码到底层大模型并行生成文章，全链路采用 Server-Sent Events (SSE) 技术实时推送进度与数据流。让耗时较长的大型项目解析过程彻底“白盒化”，拒绝黑盒等待焦虑。
- 🎯 **精准按需阅读与后台静默生成**：大纲级别绑定源码文件，生成时仅动态提取强相关代码，拒绝“假大空”。流式生成任务全局接管，支持后台静默生成，切换页面或断网重连无缝衔接。
- 🔄 **博客松散参考再生 (Regeneration)**：在仓库更新后的重写阶段，系统会将旧版博客作为上下文注入大模型，基于最新源码进行“松散参考重写”，既保留了历史优秀的业务解释，又确保代码逻辑的绝对实时。单篇博客严格要求“单点聚焦与深度剖析”，深度挖掘核心技术点。
- 📑 **超大文件解析支持**：支持本地上传高达 100MB 的 PDF/Word/MD 技术文档，前后端与网关链路已做深度适配。支持根据大文件内容智能评估并拆分系列大纲，像处理 Git 仓库一样并发生成系列博客。
- ✍️ **沉浸式极简创作体验**：内置类似 Notion 的双栏 Markdown 二次编辑器。首创基于底层 AST 行号注入的像素级双向滚动同步算法；支持对生成的图表进行纯净无样式（无 `style` 污染）的原生 Mermaid 渲染，并具备强大的正则表达式错误兼容与自动修复机制。
- 📦 **全能导出与本地知识库直通**：支持系列导出 Markdown ZIP、系列合并导出 PDF（封面 + 目录 + 正文），并支持前端勾选历史博客由浏览器离线构建 `.zip` 压缩包批量导出。支持通过 Obsidian Local REST API 将博客一键导入宿主机的本地 Obsidian Vault，并按 Karpathy LLM Wiki Pattern 自动生成 `sources/`、`concepts/`、`entities/` 卡片与双向链接，实现知识复利沉淀。
- 🛡️ **极致安全与双态鉴权**：源文件解析采用“阅后即焚”策略，不在服务器进行任何物理滞留；内置常规账号密码与 GitHub OAuth2.0 一键授权双体系；集成图形验证码与防爆破锁定安全机制。
- 🐳 **容器化开箱即用**：全面支持 Docker 化部署，提供前后端与数据库的一键编排，内置 Nginx 反向代理与流式通信优化。
- 🚀 **支持随时中止与断点续传**：前端提供“停止生成”按钮，利用 AbortController 中断请求；后端透传 Context 以立刻释放大模型调用资源。用户随时可以点击“继续生成”，系统自动跳过已完成章节，无缝接续同一个系列生成，拒绝数据孤岛。
- 📱 **响应式阅读体验**：在开始生成系列博客时，大纲自动呈现手风琴式折叠，为正文腾出充足的阅读空间；卡片排版兼容响应式，拒绝溢出。
- 📊 **用户仪表盘**：直观展示 Token 消耗、生成字数与最高频技术栈图表。

## 3. 技术栈与架构 (Tech Stack)
本项目采用前后端完全分离的 **Monorepo** 结构。

- **前端 (`frontend/`)**：
  - React 18 + Vite + Tailwind CSS v4 + Shadcn UI
  - Zustand (全局状态管理，接管 SSE 长连接保活)
  - 自定义 Rehype/Remark 插件 (AST 行号注入与图表拦截)
- **后端 (`backend/`)**：
  - Go (1.24+) + Gin 框架 (依赖注入分层架构)
  - Map-Reduce 并发调度引擎 (`x/sync/semaphore`)
  - SSE (Server-Sent Events) 打字机流式推送与空闲超时打断机制
  - GORM + PostgreSQL 14+
  - DeepSeek V4 API 原生前缀缓存 (Prompt Caching)
  - JWT & OAuth2.0 无状态鉴权
- **运维与测试**：
  - Docker & Docker Compose (多阶段构建)
  - Nginx (代理路由与流式防缓冲优化)
  - Playwright (全链路端到端 E2E 测试)

## 4. 快速开始 (Quick Start)

### 4.1 推荐：Docker 一键部署
项目已提供完整的容器化支持，只需一行命令即可拉起前后端与数据库：
```bash
# 使用 Docker Compose 一键启动
docker compose down && docker compose up -d --build
```
启动前请先在 `backend/.env` 中配置必要环境变量（例如 `DEEPSEEK_API_KEY`、`DATABASE_URL`、`JWT_SECRET`、`OBSIDIAN_REST_API_KEY`）。如需启用证书校验，另外在宿主机设置 `OBSIDIAN_REST_API_CERT_PATH` 指向插件证书；本机开发可选择设置 `OBSIDIAN_REST_API_INSECURE_SKIP_VERIFY=true` 跳过 TLS 校验。
由于后端仅提供 API 接口，前端服务由独立的 Nginx 容器代理。项目启动后：
1. **必须通过前端入口**访问：`http://localhost` (映射于宿主机 80 端口)。

### 4.2 本地开发环境运行
如果您需要进行二次开发，请确保本地已安装 Node.js 18+、Go 1.24+ 和 PostgreSQL 14+。

**启动后端：**
```bash
cd backend
cp .env  # 并配置数据库与 API 密钥
go mod tidy
go run ./cmd/server/main.go
```
*后端服务默认运行在 `http://localhost:8080`*

**启动前端：**
```bash
cd frontend
npm install
npm run dev
```
*前端页面默认运行在 `http://localhost:5173`*

## 5. 文档索引与 AI 协作指南 (AI Collaboration)
本项目深度拥抱 **Vibe Coding**（AI 辅助编程）理念。在项目根目录下，我们维护了 `.trae/rules` 作为全栈开发的核心护栏。

⚠️ **【重要提示给 AI 助手与开发者】**
在着手开发任何新功能、修改现有逻辑之前，**必须**先阅读 `.trae/documents/` 目录下的核心基准文档。严禁脱离文档上下文“闭门造车”。项目基准文档索引如下：

- 📖 [1. 产品需求文档 (PRD)](.trae/documents/InkWords_PRD.md)
- 🏗️ [2. 项目架构文档 (Architecture)](.trae/documents/InkWords_Architecture.md)
- 💾 [3. 数据库设计文档 (Database)](.trae/documents/InkWords_Database.md)
- 🔌 [4. API 接口设计文档 (API)](.trae/documents/InkWords_API.md)
- 📅 [5. 开发计划与日志 (Plan & Log)](.trae/documents/InkWords_Development_Plan_and_Log.md)
- 💬 [6. AI 对话与决策摘要 (Conversation Log)](.trae/documents/InkWords_Conversation_Log.md)

**Documentation as Code (代码与文档强同步)**：
当业务逻辑、表结构或接口路由发生变更时，AI 助手**必须在修改代码的同一个执行上下文中**同步更新上述基准文档，并在日志中记录变动。

# 墨言知识训练平台 (InkWords Trainer)

把资料变成知识，把知识变成能力。

## 1. 项目简介 (About)
**墨言知识训练平台 (InkWords Trainer)** 是一款面向个人知识沉淀与复习的本地知识工作台：把你的源码仓库/技术文档/课件包快速整理成可持续增长的 Obsidian 知识库（LLM Wiki Pattern），并提供“知识漫游复习”工作台帮助你把知识真正记住、用起来。

它依然支持将资料进一步生成小白友好、可复现的系列技术博客，但 README 的主叙事将更聚焦在“知识沉淀 → 复习闭环 → 可选的对外输出（博客/导出）”。

## 2. 核心特性 (Features)
- 🚀 **大型项目 Map-Reduce 并发拆解**：针对超长代码库或超大文本文件（>5万字），独创“智能分块 -> Goroutine并发 Map 局部摘要 -> Reduce 全局汇总”的并发架构，彻底解决大模型 Token 上限与上下文遗忘问题。
- 📚 **模块选择与系列生成**：自动极速扫描 Git 仓库提取核心模块，通过分析 README 智能生成每个目录模块的简短描述，用户自由勾选后，系统不仅并发生成单篇博客，最后还会自动生成一篇“系列导读”文章串联全集。
- 🧩 **系列历史结构保底**：系列章节在正式流式生成前会先创建子博客草稿；即使某个章节生成失败，历史博客仍会保留父子结构，便于查看、重试和排障，不再退化为只剩导读。
- ⚡ **全链路流式体验 (SSE)**：从分析 Git 仓库目录结构、拉取源码到底层大模型并行生成文章，全链路采用 Server-Sent Events (SSE) 技术实时推送进度与数据流。让耗时较长的大型项目解析过程彻底“白盒化”，拒绝黑盒等待焦虑。
- 🎯 **精准按需阅读与后台静默生成**：大纲级别绑定源码文件，生成时仅动态提取强相关代码，拒绝“假大空”。流式生成任务全局接管，支持后台静默生成，切换页面或断网重连无缝衔接。
- 🔄 **博客松散参考再生 (Regeneration)**：在仓库更新后的重写阶段，系统会将旧版博客作为上下文注入大模型，基于最新源码进行“松散参考重写”，既保留了历史优秀的业务解释，又确保代码逻辑的绝对实时。单篇博客严格要求“单点聚焦与深度剖析”，深度挖掘核心技术点。
- 📑 **超大文件与 ZIP 课件解析支持**：支持本地上传高达 888MB 的 PDF/DOCX/Markdown/TXT 技术文档与 ZIP 课件包；ZIP 会自动完成白名单筛选、去重聚合并返回解析摘要，再进入既有系列分析链路。
- 🎛️ **创作场景切换 (`scenario_mode`)**：生成器新增“电子书解读 / 开卷复习 / 小白教程”三种中文场景卡片；`scenario_mode` 负责定义任务目标，`article_style` 继续控制写法风格，并在分析、大纲、单篇生成、系列章节和系列导读链路中统一生效。
- 🔒 **场景锁定与只读回显**：用户可以在生成大纲前自由切换创作场景；大纲一旦生成，场景选择区会自动隐藏，并在大纲头部显示“当前创作场景”只读标签，避免大纲与正文使用不同场景语义。
- 🩹 **文件来源判定兼容修复**：文件上传分析链路会显式发送 `source_type=file`，后端也会在仅携带 `source_content` 且未传 `git_url` 时自动兜底识别为文件来源，避免旧静态资源或缓存请求误触发 `git_url is required for git source type`。
- 🧼 **正文纯净输出保护**：系统会在流式正文输出与润色结果应用前剥离 `<think>` 思考标签、推理过程以及“好的，收到你的需求”“作为高级全栈架构师……”这类对话式前言，避免 AI 思考内容污染最终正文。
- ✍️ **沉浸式极简创作体验**：内置类似 Notion 的双栏 Markdown 二次编辑器。首创基于底层 AST 行号注入的像素级双向滚动同步算法；支持对生成的图表进行纯净无样式（无 `style` 污染）的原生 Mermaid 渲染，并具备强大的正则表达式错误兼容与自动修复机制。
- **手写博客入口**：侧边栏新增“写博客”，一键创建空白草稿并直接进入编辑器进行手写写作。
- **语音输入**：写博客编辑器支持浏览器语音转写输入，转写内容实时写入正文并插入到光标处。
- **全文润色**：支持对当前草稿进行流式润色，前端提供“润色预览”与一键应用到正文，提升二次编辑效率。
- **文章类型/风格与模板管理**：生成器支持选择文章类型（通用/小白手把手/备考复习）；编辑器提供“模板管理”入口，可查看默认写作要求并按用户保存自定义覆盖（空字符串恢复默认）。
- 📦 **全能导出与本地知识库直通**：支持前端勾选历史博客，由浏览器离线构建 `.zip` 压缩包批量导出，或调用后端流式接口将完整项目系列一键打包下载。独创 Docker Volume 挂载技术，支持将博客一键导入宿主机的本地 Obsidian Vault，并按 Karpathy LLM Wiki Pattern 自动生成 `sources/`、`concepts/`、`entities/` 卡片与双向链接，实现知识复利沉淀。
- 🧠 **知识漫游复习工作台**：登录后可从侧边栏进入“知识漫游复习”，支持随机抽题、手动选文两种入口，以及 `light_recall / detailed_qa` 两种训练模式；所有提示、追问和总结反馈均以中文返回，并保留最近复习记录。
- 🛡️ **极致安全与双态鉴权**：源文件解析采用“阅后即焚”策略，不在服务器进行任何物理滞留；内置常规账号密码与 GitHub OAuth2.0 一键授权双体系；集成图形验证码与防爆破锁定安全机制。
- 🐳 **容器化开箱即用**：全面支持 Docker 化部署，提供前后端与数据库的一键编排，内置 Nginx 反向代理与流式通信优化。
- 🚀 **支持随时中止与断点续传**：前端提供“停止生成”按钮，利用 AbortController 中断请求；后端透传 Context 以立刻释放大模型调用资源。用户随时可以点击“继续生成”，系统自动跳过已完成章节，无缝接续同一个系列生成，拒绝数据孤岛。
- 📱 **响应式阅读体验**：在开始生成系列博客时，大纲自动呈现手风琴式折叠，为正文腾出充足的阅读空间；卡片排版兼容响应式，拒绝溢出。
- 📊 **用户仪表盘**：直观展示 Token 消耗、生成字数与最高频技术栈图表。
- 🧩 **代码规模治理**：持续将超大文件拆分为可复用模块（Hooks/子组件/同包多文件），保持高内聚低耦合，降低维护成本。

## 3. 技术栈与架构 (Tech Stack)
本项目采用前后端完全分离的 **Monorepo** 结构。

- **前端 (`frontend/`)**：
  - React + Vite + Tailwind CSS v4 + Shadcn UI
  - Zustand (全局状态管理，接管 SSE 长连接保活)
  - 目录边界：`src/pages`（页面级）/ `src/components`（组件）/ `src/hooks`（编排）/ `src/services`（请求层）/ `src/store`（全局状态）
  - 自定义 Rehype/Remark 插件 (AST 行号注入与图表拦截)
- **后端 (`backend/`)**：
  - Go + Gin 框架 (依赖注入分层架构)
  - 目录边界：`internal/domain/*`（领域切片）/ `internal/transport/http`（HTTP 适配）/ `internal/infra/*`（基础设施能力）
  - Map-Reduce 并发调度引擎 (`x/sync/semaphore`)
  - SSE (Server-Sent Events) 打字机流式推送与空闲超时打断机制
  - GORM + PostgreSQL 14+
  - DeepSeek V4 API 原生前缀缓存 (Prompt Caching)
  - JWT & OAuth2.0 无状态鉴权
- **运维与测试**：
  - Docker & Docker Compose (多阶段构建)
  - Nginx (代理路由与流式防缓冲优化)
  - Playwright (全链路端到端 E2E 测试)

## 5. 快速开始 (Quick Start)

### 5.1 推荐：Docker 一键部署
项目已提供完整的容器化支持。Docker Compose 运行时约定统一从 `backend/.env` 读取环境变量，再拉起前后端与数据库：
```bash
# 标准启动命令
docker compose --env-file backend/.env up -d --build
```
如需应用新代码或完整重启，请使用：
```bash
# 标准重启命令
docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build
```

启动前请先在 `backend/.env` 中配置必要环境变量：
- **必须配置**：`DEEPSEEK_API_KEY`、`JWT_SECRET`、`OBSIDIAN_REST_API_KEY`、`OBSIDIAN_VAULT_PATH`
- **Docker 运行时建议显式维护**：`POSTGRES_USER`、`POSTGRES_PASSWORD`、`POSTGRES_DB`
- **按需覆盖**：`FRONTEND_URL`、`REDIS_URL`、`OBSIDIAN_REST_API_BASE_URL`、`OBSIDIAN_WIKI_DIR`

当前 Docker 默认仅暴露前端入口 `http://localhost`；`backend`、`db`、`redis` 仅在 Docker 内部网络 `inkwords-network` 中互通，不再默认暴露宿主机端口。当前 Docker 本地开发仍默认通过 `backend/.env` 中的 `OBSIDIAN_REST_API_INSECURE_SKIP_VERIFY=true` 访问 Obsidian Local REST API，避免把宿主机错误文件挂载成证书；如需启用严格证书校验，请显式配置 `OBSIDIAN_REST_API_CERT_PATH` 指向真实插件证书，并将 `OBSIDIAN_REST_API_INSECURE_SKIP_VERIFY=false`。

如果你需要在 Docker 模式下使用 GitHub OAuth，本地回调地址应配置为 `http://localhost/api/v1/auth/callback/github`；如果你运行的是 Vite 本地开发服务器，再改回 `http://localhost:5173/api/v1/auth/callback/github`。

`OBSIDIAN_VAULT_PATH` 不再提供任何机器私有的默认绝对路径。请显式把它设置为你的 Obsidian `wiki/` 根目录，否则容器启动前的 Compose 渲染会直接报错，避免静默挂载到错误位置。
如遇导出到 Obsidian 提示“无法解析 Obsidian 目录列表响应”，请确认后端版本已兼容 Obsidian Local REST API 目录列表 `{ "files": [...] }` 返回格式。
如遇上传 PDF/DOCX/Markdown/TXT/ZIP 后仍提示 `git_url is required for git source type`，请先执行标准重启命令并强制刷新浏览器，以确保前端静态资源与后端兼容逻辑同步生效。

### 5.1.1 ZIP 课件包上传说明
- 支持上传 `.zip` 课件包，后端会自动扫描其中的受支持文档与代码文本文件。
- 当前支持的文档类包括：`.pdf`、`.docx`、`.md`、`.markdown`、`.txt`。
- 当前支持的代码/文本类包括：`.go`、`.js`、`.ts`、`.tsx`、`.jsx`、`.py`、`.java`、`.cpp`、`.c`、`.h`、`.hpp`、`.rs`、`.sql`、`.sh`、`.json`、`.yaml`、`.yml`。
- 当前不支持 `.doc`、`rar/7z/tar.gz`、图片 OCR 与音视频转写。
- 上传成功后，前端会显示 ZIP 解析摘要，例如保留、去重、忽略与失败数量。

### 5.1.2 创作场景说明
- **电子书解读**：适合经典著作、长文档、概念材料，输出更偏向篇章结构、观点提炼和白话解读。
- **开卷复习**：适合课件、实验指导、考试范围资料，输出更偏向考点清单、步骤模板、易错点和速查结构。
- **小白教程**：适合 Git 仓库、项目教程、官方文档，输出更偏向环境准备、目录结构、主链路和排错说明。
- **默认推荐**：
  - Git 仓库默认推荐 `beginner_walkthrough`
  - 文件上传默认推荐 `ebook_interpretation`
- **交互约束**：
  - 只能在生成大纲前切换创作场景。
  - 大纲生成后会隐藏场景选择区，并在大纲头部显示当前场景的只读标签。
- **兼容策略**：旧前端即便未显式发送 `scenario_mode`，后端也会按 `source_type` 自动兜底，避免现有链路回归。

### 5.1.3 知识漫游复习说明
- 侧边栏新增“知识漫游复习”入口，登录后可直接进入独立工作台。
- 当前支持两种进入方式：
  - 随机抽一篇
  - 选择文章复习
- 当前支持两种训练模式：
  - `light_recall`：轻提示复述
  - `detailed_qa`：细致问答
- 运行依赖：
  - 后端需要能访问 Obsidian `wiki/` 目录（由 `OBSIDIAN_WIKI_DIR` 指定，默认 `wiki`）。
  - PostgreSQL 会持久化 `review_sessions` 与 `review_turns`，用于保存会话状态与轮次记录。
- 若进入页面后提示无法获取复习题卡，请优先检查：
  - 本地 Obsidian 知识库是否已存在可复习的 `wiki` 页面
  - `backend/.env` 中 Obsidian 相关变量是否已配置
  - 是否已执行 `docker compose down && docker compose up -d --build` 让前后端与迁移保持一致

### 5.1.4 流程型入口说明
- 当用户未打开具体博客编辑器时，应用默认先进入 `HomeEntry`，而不是直接落到生成器表单。
- 首页会提供两条主路径：
  - `生成博客`
  - `知识复习`
- 首页底部保留继续上次任务与最近记录，方便从流程入口回到真实工作内容。
- 进入 `Generator` 或 `KnowledgeReview` 后，主区一次只展示当前步骤；其余步骤通过共享 `StepStrip` 以预览或进度态呈现。

由于后端仅提供 API 接口，前端服务由独立的 Nginx 容器代理。项目启动后：
1. **必须通过前端入口**访问：`http://localhost` (映射于宿主机 80 端口)。

### 5.2 本地开发环境运行
如果您需要进行二次开发，请确保本地已安装 Node.js 18+、Go 1.25+ 和 PostgreSQL 14+。

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

### 5.3 仓库文件整理与大文件策略
- 本仓库不追踪构建产物/大文件：`backend/server`、`backend/inkwords-server`、`backend/bin/*`、`pdf/*.pdf`、`dogfood-output/` 等均应保持为本地产物或通过脚本/CI 生成。
- `dogfood-output/` 可能包含本地调试截图与浏览器存储（含 token），只允许本地存在，禁止提交进 Git。

## 6. 文档索引与 AI 协作指南 (AI Collaboration)
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

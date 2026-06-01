# 墨言知识训练平台 (InkWords Trainer)

把资料变成知识，把知识变成能力。

## 1. 项目简介 (About)
**墨言知识训练平台 (InkWords Trainer)** 是一款面向个人知识沉淀与复习的本地知识工作台：把你的源码仓库/技术文档/课件包快速整理成可持续增长的 Obsidian 知识库（LLM Wiki Pattern），并提供“知识漫游复习”工作台帮助你把知识真正记住、用起来。

它依然支持将资料进一步生成小白友好、可复现的系列技术博客，但 README 的主叙事将更聚焦在“知识沉淀 → 复习闭环 → 可选的对外输出（博客/导出）”。

## 2. 核心特性 (Features)
- **一键知识摄入（Ingest）**：支持解析 Git 仓库、本地文档（PDF/DOCX/Markdown/TXT）与 ZIP 课件包，自动生成符合 Karpathy LLM Wiki Pattern 的 `sources/`、`concepts/`、`entities/` 卡片与双向链接。
- **动态提示词类型锁定**：文件 Analyze 阶段会先自动识别内容类型，并锁定最匹配的提示词 profile；例如心理学/沟通类书籍会显示“心理学经典解读”，技术手册会显示“技术资料讲解”，后续正文生成会沿用同一个 profile，避免大纲和正文语义漂移。
- **系列章节质量门禁基础**：后端已为系列章节四段式流水线补齐 `理解 -> 草稿 -> 审稿 -> 终稿` 的结构化类型基础，并对“机制解释、案例、复现、修订动作”增加硬门禁校验，防止不完整章节结果直接流入后续补强阶段。
- **系列级稳定前缀与章节质量主链路**：后端已抽出 `buildSeriesSharedPromptPrefix()` 固定系列标题、读者画像、总大纲和统一质量门禁，并把系列章节主链路切换为 `理解 -> 草稿 -> 审稿 -> 定向补强` 四段式；只有终稿补强阶段才会向前端持续流式输出正文，避免中间态草稿污染用户阅读体验。
- **DeepSeek usage / 缓存命中可视化**：系列章节终稿补强完成后，前端进度卡会直接显示当前章节的 “缓存命中 / 未命中” 摘要，底层数据来自后端追加的 `usage` 事件（`prompt_tokens / completion_tokens / prompt_cache_hit_tokens / prompt_cache_miss_tokens`）。
- **中文 PDF 兼容回退**：PDF 默认先走 Go 原生提取；若检测到严重乱码，会自动回退到 `pdftotext` 恢复文本，减少中文嵌入字体 PDF 直接解析失败的概率。
- **知识库“热缓存 + 索引 + 日志”自动维护**：自动更新 `wiki/index.md`、`wiki/hot.md` 与 `wiki/log.md`，避免知识孤岛，让知识库可持续演进。
- **知识漫游复习工作台（核心主链路）**：支持“随机抽一篇 / 手动选文”两种入口；推荐卡支持 `用这篇开始 / 提问开始 / 再抽一篇`，并提供 `light_recall / detailed_qa` 两种训练模式。会话会基于文章内容进行结构化追问，回答后明确给出“本轮目标 / 答到的点 / 漏掉的点 / 下一步建议”，帮助判断是否真正讲对重点。
- **流程型入口与单步聚焦体验**：默认入口 `HomeEntry` 帮你在“知识复习 / 内容生成”之间选择路径，页面同一时间只聚焦一个主步骤，减少平铺与注意力分散。
- **生成器三步流与步内进度**：博客生成工作台固定为“选择来源 → 配置解析 → 确认大纲”三步；解析/分析进度展示在“配置解析”内部，正文生成进度展示在“确认大纲”内部，不再跳入独立进度页。
- **全链路流式体验（SSE）**：从解析、分析到生成与润色，全链路使用 SSE 推送进度与内容，让过程可见、可中断、可续写。
- **系列章节质量阶段可视化**：系列章节生成进度卡会直接显示 `理解章节 / 生成草稿 / 技术审稿 / 定向补强 / 输出终稿` 等中文阶段标签，帮助用户理解当前流水线停留在哪一步。
- **阅后即焚的安全策略**：解析源文件只在内存中处理，任务完成后立即清理临时产物；对用户可见正文做净化，剥离 `<think>` 与对话式前言。
- **可选的对外输出：博客生成与导出**：在知识沉淀基础上，支持生成可复现的系列技术博客、导读串联，以及批量导出为 Markdown ZIP 或直通 Obsidian Vault。
- **超大规模资料处理**：对超长文本/大型仓库支持 Map-Reduce 分块并发分析，降低上下文遗忘并提升稳定性。

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

如果你需要在 Docker 模式下使用 GitHub OAuth，Compose 现在会默认把回调地址固定为 `http://localhost/api/v1/auth/callback/github`，避免错误跳回 Vite 本地开发端口 `5173`。如果你需要在 Docker 下覆盖这个地址（例如远程域名调试），请显式设置 `DOCKER_GITHUB_REDIRECT_URL`。如果你运行的是 Vite 本地开发服务器，再继续使用 `backend/.env` 中的 `GITHUB_REDIRECT_URL=http://localhost:5173/api/v1/auth/callback/github`。

`OBSIDIAN_VAULT_PATH` 不再提供任何机器私有的默认绝对路径。请显式把它设置为你的 Obsidian `wiki/` 根目录，否则容器启动前的 Compose 渲染会直接报错，避免静默挂载到错误位置。
如遇导出到 Obsidian 提示“无法解析 Obsidian 目录列表响应”，请确认后端版本已兼容 Obsidian Local REST API 目录列表 `{ "files": [...] }` 返回格式。
如遇上传 PDF/DOCX/Markdown/TXT/ZIP 后仍提示 `git_url is required for git source type`，请先执行标准重启命令并强制刷新浏览器，以确保前端静态资源与后端兼容逻辑同步生效。

### 5.1.1 ZIP 课件包上传说明
- 支持上传 `.zip` 课件包，后端会自动扫描其中的受支持文档与代码文本文件。
- 当前支持的文档类包括：`.pdf`、`.docx`、`.md`、`.markdown`、`.txt`。
- `.pdf` 当前采用“主提取 + 回退恢复”双通道策略：若主提取结果出现严重乱码，后端会自动尝试 `pdftotext`；若仍无法可靠恢复，则返回稳定中文提示，建议改用可复制文本的 PDF、DOCX 或 Markdown。
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
- **动态提示词类型**：
  - 当来源是文件上传时，Analyze 会自动识别当前资料的内容类型，并返回一个已锁定的提示词 profile。
  - 大纲生成后，页面除了显示“当前创作场景”，还会显示“当前提示词类型”只读标签，例如“心理学经典解读”“开卷复习材料”“技术资料讲解”。
  - 后续单篇生成、系列章节生成与系列导读都会沿用这个已锁定的 profile，不需要用户再次选择。
- **兼容策略**：旧前端即便未显式发送 `scenario_mode`，后端也会按 `source_type` 自动兜底，避免现有链路回归。
- **分类失败兜底**：如果内容类型识别失败或返回非法值，系统会按当前创作场景自动回退到默认提示词，并把回退原因展示在 Analyze 结果里。

### 5.1.3 知识漫游复习说明
- 侧边栏新增“知识漫游复习”入口，登录后可直接进入独立工作台。
- 当前支持两种进入方式：
  - 随机抽一篇
  - 选择文章复习
- 推荐题卡默认暴露三种动作：
  - `用这篇开始`
  - `提问开始`
  - `再抽一篇`
- 当前支持两种训练模式：
  - `light_recall`：轻提示复述
  - `detailed_qa`：细致问答
- 若当前题卡 `available_modes` 不包含 `detailed_qa`，则 `提问开始` 会保持禁用态，避免前端发起后端不支持的模式。
- 首页“继续上次任务”只会恢复未完成的复习会话；已完成会话会回退到最近博客任务或默认入口，不再显示为“会话仍可继续”。
- 运行依赖：
  - 后端需要能访问 Obsidian `wiki/` 目录（由 `OBSIDIAN_WIKI_DIR` 指定，默认 `wiki`）。
  - PostgreSQL 会持久化 `review_sessions` 与 `review_turns`，用于保存会话状态与轮次记录。
- 若进入页面后提示无法获取复习题卡，请优先检查：
  - 本地 Obsidian 知识库是否已存在可复习的 `wiki` 页面
  - `backend/.env` 中 Obsidian 相关变量是否已配置
  - 是否已执行 `docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build` 让前后端与迁移保持一致

### 5.1.4 流程型入口说明
- 当用户未打开具体博客编辑器时，应用默认先进入 `HomeEntry`，而不是直接落到生成器表单。
- 首页会提供两条主路径：
  - `生成博客`
  - `知识复习`
- 首页底部保留继续上次任务与最近记录，方便从流程入口回到真实工作内容。
- 进入 `Generator` 或 `KnowledgeReview` 后，主区一次只展示当前步骤；其余步骤通过共享 `StepStrip` 以预览或进度态呈现。

### 5.1.5 系列章节质量流水线验证
- 章节生成链路当前已升级为 `understanding -> drafting -> reviewing -> revising -> streaming -> usage -> completed/error`。
- 前端进度卡会把这些状态展示为中文“质量阶段”，并在收到 `usage` 后展示“缓存命中 / 未命中”摘要。
- 推荐验证命令：

```bash
cd backend
go test ./...
```

```bash
cd frontend
npm run test -- --run
```

```bash
docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build
docker compose --env-file backend/.env ps
curl -I http://localhost
```

- 推荐始终使用 `docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build`；这样 Compose 会显式加载 `backend/.env`，避免因 `OBSIDIAN_VAULT_PATH` 未注入而在解析 bind mount 时直接报错。

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

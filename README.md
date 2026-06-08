# 墨言知识训练平台 (InkWords Trainer)

把资料变成知识，把知识变成能力。

## 最近更新
- `2026-06-08`：完成一轮全仓安全清理，删除 `llm-stream` 下未接入的 generation 占位/空桥接代码，收窄过渡层 `StreamAPI` 依赖，并移除前端 `GeneratorProgressStage` 兼容壳与两处 store 中未消费状态；不改变对外功能、API 和默认访问入口。

## 项目定位
InkWords Trainer 是一个面向个人知识沉淀、知识复习与可选内容输出的本地知识工作台。

它的主链路不再只是“把资料生成博客”，而是：

`资料摄入 -> 知识整理 -> Obsidian 沉淀 -> 知识漫游复习 -> 可选输出为博客 / PDF / Obsidian`

当前项目的主叙事是“知识训练平台”：
- 你可以导入 Git 仓库、技术文档、PDF、ZIP 课件包。
- 系统会把资料整理成适合长期维护的知识卡片与结构化内容。
- 你可以在“知识漫游复习”工作台里反复训练这些内容。
- 如果需要，再把整理后的内容生成系列博客、导出 PDF，或直通 Obsidian Vault。

## 核心能力
- **知识摄入**：支持 Git 仓库、本地文档（PDF / DOCX / Markdown / TXT）和 ZIP 课件包解析，并生成适合知识沉淀的结构化内容。
- **知识库沉淀**：围绕 Obsidian LLM Wiki Pattern 组织 `sources/`、`concepts/`、`entities/`、索引页与热缓存页。
- **知识漫游复习**：提供独立的复习工作台，支持随机抽题、手动选文、轻提示复述与细致问答。
- **动态提示词锁定**：文件 Analyze 会自动识别内容类型并锁定提示词 profile，避免大纲和正文语义漂移。
- **任务式生成链路**：生成主链路已经切到“创建任务 -> 队列消费 -> SSE 回放”的后台任务模式。
- **任务式解析与导出**：大文件 / ZIP 解析、系列 PDF 导出已开始接入统一任务中心。
- **系列生成质量流水线**：章节生成过程支持 `理解 -> 草稿 -> 审稿 -> 定向补强 -> 输出终稿` 的分阶段可视化。
- **可选内容输出**：支持生成系列博客、续写、润色、导出 Markdown / PDF、导出到 Obsidian。

## 当前阶段
项目最近一轮变化的重点，不是新增一个零散功能，而是整体运行形态和主链路的升级：

- **产品定位升级**：从“博客助手”升级为“知识训练平台”。
- **生产形态升级**：从单体后端演进为 Docker Compose 多服务架构。
- **任务中心升级**：生成、解析、导出逐步统一到 `job_tasks + RabbitMQ + SSE` 模型。
- **复习链路升级**：知识漫游复习从固定模板问答升级为文章驱动的结构化追问。
- **代码边界升级**：`parser-service`、`review-service`、`export-service` 已完成服务目录归属；`core-api` 与 `llm-stream` 正在做深拆分。

## 系统架构
项目采用前后端分离的 Monorepo 结构：

- `frontend/`：React 18 + Vite + Tailwind CSS + shadcn/ui + Zustand
- `backend/`：Go + Gin + PostgreSQL + RabbitMQ + Redis
- `docker-compose.yml`：当前唯一的容器化编排入口

### 生产形态
当前标准运行形态是“前端单入口 + 后端多服务”：

- `frontend`：Nginx 静态站点与统一网关
- `core-api`：核心业务 API、任务创建 / 查询 / SSE 回放、用户 / 博客主事实写入
- `llm-stream`：流式生成执行与 generation worker
- `parser-service`：文件 / ZIP 解析与 parse worker
- `export-service`：PDF 导出与 export worker
- `review-service`：知识漫游复习服务
- `db`：PostgreSQL
- `redis`：缓存与状态辅助
- `rabbitmq`：任务队列
- `obsidian-bridge`：容器访问宿主机 Obsidian Local REST API 的桥接服务

### 对外入口
- 默认公开入口始终是 `http://localhost`
- 所有页面访问都应走前端网关，而不是直接访问后端端口
- 在宿主机 `:80` 冲突时，可临时用 `FRONTEND_PORT=8088` 覆盖为 `http://localhost:8088`

### 网关分流
前端 Nginx 会按路径把请求分发到不同服务：

- `/api/v1/stream/*` -> `llm-stream`
- `/api/v1/project/parse` -> `parser-service`
- `/api/v1/review/*` -> `review-service`
- `/api/v1/blogs/:id/export*` -> `export-service`
- 其余 `/api/*` -> `core-api`

## 关键工作流
### 1. 资料摄入
- Git 仓库：扫描目录结构、识别核心模块、按需做大纲分析
- 文件上传：支持 PDF / DOCX / Markdown / TXT
- ZIP 课件包：支持安全解压、白名单筛选、文本聚合与解析摘要
- 大文件保护：超长文本走 Map-Reduce 分块分析

### 2. 提示词与场景控制
- 创作场景支持：
  - `电子书解读`
  - `开卷复习`
  - `小白教程`
- 文件 Analyze 会额外锁定 `prompt_profile`
- 大纲生成后，场景与提示词类型都会以只读标签形式展示

### 3. 任务中心
- **生成任务**：前端先创建 generation task，再订阅 `/api/v1/tasks/:id/stream`
- **解析任务**：ZIP 与 `50MB` 以上普通单文件默认走 parse task
- **导出任务**：系列 PDF 默认走 export task，完成后通过受控下载接口取回文件

### 4. 知识漫游复习
- 提供独立入口，不再和生成器混成一个工作流
- 支持“随机抽一篇 / 手动选文”
- 支持 `light_recall` 和 `detailed_qa`
- 回答后会返回“本轮目标 / 答到的点 / 漏掉的点 / 下一步建议”

### 5. 内容输出
- 系列博客生成
- 单篇续写与润色
- Markdown / ZIP 导出
- PDF 异步导出
- 导出到 Obsidian Vault

## 快速开始
### 1. 准备环境变量
先复制环境文件：

```bash
cp backend/.env.example backend/.env
```

至少需要确认以下变量：
- `DEEPSEEK_API_KEY`
- `JWT_SECRET`
- `OBSIDIAN_REST_API_KEY`
- `OBSIDIAN_VAULT_PATH`

常用默认值已在 `backend/.env.example` 中提供，例如：
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DB`
- `RABBITMQ_URL`
- `RABBITMQ_EXCHANGE`
- `RABBITMQ_GENERATION_QUEUE`
- `RABBITMQ_PARSE_QUEUE`
- `RABBITMQ_EXPORT_QUEUE`

### 2. 启动整套服务

```bash
docker compose --env-file backend/.env up -d --build
```

如需完整重启：

```bash
docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build
```

如需只扩容流式生成服务：

```bash
docker compose --env-file backend/.env up -d --build --scale llm-stream=3
```

### 3. 验证运行状态

```bash
docker compose --env-file backend/.env ps
curl -I http://localhost
curl --fail http://localhost/api/v1/ping
```

预期结果：
- `frontend / core-api / llm-stream / parser-service / export-service / review-service` 为 `Up (healthy)` 或等价健康状态
- `http://localhost` 可访问
- `/api/v1/ping` 可返回成功响应

### 4. 端口冲突处理
当宿主机 `:80` 被占用时，可临时覆盖前端端口：

```bash
FRONTEND_PORT=8088 docker compose --env-file backend/.env up -d --build frontend
```

然后通过 `http://localhost:8088` 访问。

## 本地开发
### 后端本地开发
本地开发 / 集成调试仍保留聚合入口：

```bash
cd backend
cp .env.example .env
go mod tidy
go run ./cmd/server/main.go
```

说明：
- 本地聚合入口默认运行在 `http://localhost:8080`
- `cmd/server` 主要用于本地开发和集成调试
- Docker 生产形态默认不再使用这个聚合入口

### 前端本地开发

```bash
cd frontend
npm install
npm run dev
```

前端开发服务器默认运行在 `http://localhost:5173`。

## 目录结构
```text
InkWords/
├── frontend/                    # React 前端
├── backend/                     # Go 后端
│   ├── cmd/server/              # 本地聚合入口
│   ├── internal/                # 共享领域 / transport / infra
│   ├── services/
│   │   ├── core-api/            # 核心 API 服务
│   │   ├── llm-stream/          # 流式生成服务
│   │   ├── parser-service/      # 解析服务
│   │   ├── export-service/      # 导出服务
│   │   └── review-service/      # 复习服务
│   └── shared/                  # 服务间共享基础层
├── docs/runbooks/               # 运行与排障手册
├── .trae/documents/             # 项目基准文档
└── docker-compose.yml           # 多服务编排入口
```

## 技术栈
### 前端
- React 18
- Vite
- Tailwind CSS
- shadcn/ui
- Zustand
- `@microsoft/fetch-event-source`

### 后端
- Go 1.25+
- Gin
- GORM
- PostgreSQL 14+
- RabbitMQ
- Redis
- DeepSeek API

### 基础设施
- Docker
- Docker Compose
- Nginx
- Obsidian Local REST API

## 当前微服务化进度
当前可以把项目理解为“多服务已落地，核心双服务仍在深拆”：

- 已完成：
  - `parser-service`、`review-service`、`export-service` 的服务自有入口与装配收口
  - Docker Compose 多服务生产形态
  - Nginx 单入口网关分流
  - 生成 / 解析 / 导出任务中心基础链路
- 进行中：
  - `core-api / llm-stream` 深拆分第一轮
  - 生成结果由 `core-api` 逐步回收最终业务落库
  - legacy 共享 transport 和共享写入边界的进一步收口

## 运行与验证建议
当你修改了以下内容之一时，建议优先跑微服务冒烟检查：
- `docker-compose.yml`
- `frontend/nginx.conf`
- `backend/services/*/cmd/main.go`
- 任务中心、健康检查、服务入口、网关路由相关逻辑

推荐参考：
- [微服务冒烟检查 Runbook](docs/runbooks/microservices-smoke-check.md)
- [core-blog-task-boundary.md](docs/runbooks/core-blog-task-boundary.md)
- [review-db-migration.md](docs/runbooks/review-db-migration.md)

## 文档索引
项目采用 Docs-as-Code，以下文档与代码需保持同步：

- [产品需求文档 PRD](.trae/documents/InkWords_PRD.md)
- [项目架构文档 Architecture](.trae/documents/InkWords_Architecture.md)
- [数据库设计文档 Database](.trae/documents/InkWords_Database.md)
- [API 接口文档 API](.trae/documents/InkWords_API.md)
- [开发计划与日志 Plan & Log](.trae/documents/InkWords_Development_Plan_and_Log.md)
- [对话与决策摘要 Conversation Log](.trae/documents/InkWords_Conversation_Log.md)

## 开发约束
- 修改业务逻辑、接口或表结构时，需要同步更新上述文档
- 默认通过 Docker Compose 验证完整运行形态
- 前端界面文本以中文为准
- 生成 / 解析源文件遵循“阅后即焚”，不持久化原始文件实体

## 说明
如果你希望把它当作“博客生成平台”来用，也完全可以；只是当前项目的长期定位已经从“生成一篇文章”升级为“围绕资料沉淀、知识复习与可选输出的知识训练闭环”。

# 墨言知识训练平台 (InkWords Trainer)

把资料变成知识，把知识变成能力。

## 最近更新
- `2026-06-29`：完成代码治理清理收尾：消除 54% 的重复 Go 文件、消除全部 service→monolith 依赖、前端 lint 清零、CI 新增 golangci-lint 与 deadcode 阻塞门禁、前端打包体积缩减 43%（gzip）。质量基线文档已同步更新。
- `2026-06-08`：完成一轮全仓安全清理，删除 `llm-stream` 下未接入的 generation 占位/空桥接代码，收窄过渡层 `StreamAPI` 依赖，并移除前端 `GeneratorProgressStage` 兼容壳与两处 store 中未消费状态；不改变对外功能、API 和默认访问入口。

[English README](./README_EN.md)

## 项目简介

InkWords Trainer 是一个面向个人知识沉淀、知识复习与内容输出的本地知识工作台。它不再只服务于“生成一篇博客”，而是围绕完整知识闭环来设计：

`资料摄入 -> 知识整理 -> Obsidian 沉淀 -> 知识漫游复习 -> 可选输出为博客 / PDF / Obsidian`

你可以把它理解为一个以中文交互为主的知识训练平台：

- 支持导入 Git 仓库、技术文档、PDF、DOCX、Markdown、TXT 和 ZIP 课件包
- 支持把内容整理为适合长期维护的知识卡片与结构化材料
- 支持在“知识漫游复习”工作台里进行持续训练
- 支持把整理后的内容继续输出为系列博客、Markdown、PDF 或 Obsidian 知识库内容

## 核心能力

- **资料摄入**：支持 Git 仓库扫描、本地文件解析、ZIP 课件包安全解压与聚合分析
- **知识沉淀**：围绕 Obsidian LLM Wiki Pattern 组织 `sources/`、`concepts/`、`entities/`、索引页与热缓存页
- **知识复习**：提供独立复习工作台，支持随机抽题、手动选文、轻提示复述与结构化追问
- **内容生成**：支持单篇生成、系列生成、续写、润色和系列导读自动串联
- **任务中心**：生成、解析、导出逐步统一到 `job_tasks + RabbitMQ + SSE` 的后台任务模型
- **异步导出**：支持 Markdown / ZIP 导出、系列 PDF 导出、导出到 Obsidian Vault
- **动态提示词锁定**：文件 Analyze 阶段自动识别内容类型并锁定 `prompt_profile`
- **质量流水线**：系列章节生成支持 `理解 -> 草稿 -> 审稿 -> 定向补强 -> 输出终稿` 的阶段化可视化

## 当前架构

项目采用前后端分离的 Monorepo 结构：

- `frontend/`：React 19 + Vite + Tailwind CSS + shadcn/ui + Zustand
- `backend/`：Go + Gin + GORM + PostgreSQL + RabbitMQ + Redis
- `docker-compose.yml`：项目唯一的容器化编排入口

### 生产形态

当前标准运行形态是“前端单入口 + 后端多服务”：

- `frontend`：Nginx 静态站点与统一网关
- `core-api`：核心业务 API、任务创建/查询、SSE 回放、用户与博客主事实写入
- `llm-stream`：流式生成执行与 generation worker
- `parser-service`：文件 / ZIP 解析与 parse worker
- `export-service`：PDF 导出与 export worker
- `review-service`：知识漫游复习服务
- `db`：PostgreSQL
- `redis`：缓存与状态辅助
- `rabbitmq`：任务队列
- `obsidian-bridge`：容器访问宿主机 Obsidian Local REST API 的桥接服务

### 对外访问入口

- 默认公开入口始终是 `http://localhost`
- 页面访问必须走前端网关，不直接面向后端端口
- 当宿主机 `:80` 被占用时，可通过 `FRONTEND_PORT=8088` 临时切换到 `http://localhost:8088`

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
- 长文本保护：超长内容走 Map-Reduce 分块分析

### 2. 场景与提示词控制

- 创作场景支持：`电子书解读`、`开卷复习`、`小白教程`
- 文件 Analyze 会额外锁定 `prompt_profile`
- 大纲生成后，场景与提示词类型会以只读标签展示给用户

### 3. 任务中心

- 生成任务：前端先创建 generation task，再订阅 `/api/v1/tasks/:id/stream`
- 解析任务：ZIP 与 `50MB` 以上普通单文件默认走 parse task
- 导出任务：系列 PDF 默认走 export task，完成后通过受控下载接口取回文件

### 4. 知识漫游复习

- 提供独立入口，不再与生成器混在同一条主链路中
- 支持“随机抽一篇 / 手动选文”
- 支持 `light_recall` 和 `detailed_qa`
- 选中文章后先展示原文预览，再进入复述输入
- 回答后返回“本轮目标 / 答到的点 / 漏掉的点 / 下一步建议”
- 当用户表达“忘了 / 记不清”时，系统会先给简短提示，再按需提供原文摘录

### 5. 内容输出

- 系列博客生成
- 单篇续写与润色
- Markdown / ZIP 导出
- PDF 异步导出
- 导出到 Obsidian Vault

## 快速开始

### 1. 准备环境

建议先准备：

- Docker
- Docker Compose
- DeepSeek API Key
- 可选的 Obsidian Local REST API 环境

复制环境变量模板：

```bash
cp backend/.env.example backend/.env
```

至少需要检查以下变量：

- `DEEPSEEK_API_KEY`
- `JWT_SECRET`
- `OBSIDIAN_REST_API_KEY`
- `OBSIDIAN_VAULT_PATH`

`backend/.env.example` 中已经提供了常用默认项，例如：

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

- `frontend`、`core-api`、`llm-stream`、`parser-service`、`export-service`、`review-service` 进入健康状态
- `http://localhost` 可访问
- `/api/v1/ping` 返回成功响应

### 4. 处理端口冲突

当宿主机 `:80` 被占用时，可临时覆盖前端端口：

```bash
FRONTEND_PORT=8088 docker compose --env-file backend/.env up -d --build frontend
```

然后通过 `http://localhost:8088` 访问。

## 本地开发

### 后端开发

本地开发和集成调试仍保留聚合入口：

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

### 前端开发

```bash
cd frontend
npm install
npm run dev
```

前端开发服务器默认运行在 `http://localhost:5173`。

### 常用前端命令

```bash
cd frontend
npm run dev
npm run build
npm run lint
npm run test
```

## 目录结构

```text
InkWords/
├── frontend/                    # React 前端与 Nginx 静态站点构建源
├── backend/                     # Go 后端与多服务实现
│   ├── cmd/server/              # 本地聚合调试入口
│   ├── internal/                # 共享领域、基础设施与过渡层
│   ├── services/
│   │   ├── core-api/            # 核心 API 服务
│   │   ├── llm-stream/          # 流式生成服务
│   │   ├── parser-service/      # 解析服务
│   │   ├── export-service/      # 导出服务
│   │   └── review-service/      # 复习服务
│   ├── db/                      # 数据库初始化脚本
│   └── scripts/                 # 辅助脚本
├── docs/runbooks/               # 运行与排障手册
├── .trae/documents/             # PRD / 架构 / API / 数据库等项目基准文档
├── docker-compose.yml           # 多服务编排入口
└── README_EN.md                 # 英文说明文档
```

## 技术栈

### 前端
- React 19
- Vite 8
- Tailwind CSS 4
- shadcn/ui
- Zustand
- `@microsoft/fetch-event-source`

### 后端

- Go 1.25
- Gin
- GORM
- PostgreSQL 14
- RabbitMQ
- Redis
- DeepSeek API

### 基础设施

- Docker
- Docker Compose
- Nginx
- Obsidian Local REST API

## 当前微服务化进度
当前可以把项目理解为"多服务已落地，旧业务层已退役"：

- 已完成：
  - 五个后端服务的自有入口、领域与装配收口
  - Docker Compose 多服务生产形态
  - Nginx 单入口网关分流
  - 生成 / 解析 / 导出任务中心基础链路
  - 删除旧 `cmd/*` 包装与不可达的 domain/service/transport/prompt/cache/mq 代码
- 进行中：
  - 生成结果由 `core-api` 逐步回收最终业务落库
  - 将 `internal/infra/db` 与 `internal/model` 持久化桥接迁移到 shared/service-owned 边界

## 运行与排障建议

当你修改以下内容之一时，建议优先执行微服务冒烟检查：

- `docker-compose.yml`
- `frontend/nginx.conf`
- `backend/services/*/cmd/main.go`
- 任务中心、健康检查、服务入口、网关路由相关逻辑

推荐参考：

- [微服务冒烟检查 Runbook](./docs/runbooks/microservices-smoke-check.md)
- [核心任务边界说明](./docs/runbooks/core-blog-task-boundary.md)
- [Review 数据库迁移说明](./docs/runbooks/review-db-migration.md)
- [服务镜像边界说明](./docs/runbooks/service-image-boundaries.md)

## 文档索引

项目采用 Docs-as-Code，以下文档应与代码保持同步：

- [产品需求文档 PRD](./.trae/documents/InkWords_PRD.md)
- [项目架构文档 Architecture](./.trae/documents/InkWords_Architecture.md)
- [数据库设计文档 Database](./.trae/documents/InkWords_Database.md)
- [API 接口文档 API](./.trae/documents/InkWords_API.md)
- [开发计划与日志 Plan & Log](./.trae/documents/InkWords_Development_Plan_and_Log.md)
- [对话与决策摘要 Conversation Log](./.trae/documents/InkWords_Conversation_Log.md)

## 代码治理与质量门禁

项目采用 CI 强制质量门禁，每次 PR 和 push main 均执行以下阻塞检查：

| 层级 | 检查 | 阻塞 |
| --- | --- | --- |
| Backend | `go vet` | ✅ |
| Backend | `golangci-lint` | ✅ |
| Backend | `go test ./...` | ✅ |
| Frontend | `npm run lint` (`--max-warnings=0`) | ✅ |
| Frontend | `npm run deadcode` (knip) | ✅ |
| Frontend | `npm test` | ✅ |
| Frontend | `npm run test:coverage` | ✅ |
| Frontend | `npm run build` | ✅ |
| Config | `docker compose config` | ✅ |
| Smoke | Docker 多服务启动 + 健康检查 + 网关冒烟 | ✅ |

### 当前质量基线

- **重复 Go 文件**：12 组 24 个文件（均为 domain 镜像，已完成 de-duplication，从 22 组 52 文件缩减）
- **service → monolith 依赖**：0（已全部消除，仅剩测试引用）
- **后端覆盖**：35.4%
- **前端覆盖**：38.83%
- **前端 lint**：0 发现
- **前端死码**：2 未使用文件、5 未使用依赖、1 未使用 devDep（存量已知）
- **主 bundle gzip**：335 kB

以上基线与详细验证命令参见 [docs/qa/code-cleanup-baseline.md](docs/qa/code-cleanup-baseline.md)。

## 开发约束

- 修改业务逻辑、接口或表结构时，需要同步更新上述项目文档
- 默认通过 Docker Compose 验证完整运行形态
- 默认公开入口是 `http://localhost`
- 前端界面文本以中文为准
- 生成与解析源文件遵循“阅后即焚”，不持久化原始文件实体

## 说明

如果你只想把它当作“博客生成平台”来使用，也完全没问题；只是当前项目的长期定位已经升级为“围绕资料沉淀、知识复习与可选输出的知识训练闭环”。

# InkWords Microservices Long-Term Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在保持 `http://localhost` 单入口、现有前端交互和核心 API 语义稳定的前提下，把 InkWords 从“多服务过渡态”推进为“可观测、可独立扩容、具备明确边界和异步任务能力”的长期可演进微服务架构。

**Architecture:** 继续保留 `frontend(Nginx) -> core-api / llm-stream / parser-service / export-service / review-service` 的现有生产形态，把后续演进重点放在四条主线：一是补齐 RabbitMQ 生成任务闭环；二是补齐服务级可观测性、健康检查和运行契约；三是通过“写入归属 + 任务模型 + 渐进拆库”硬化服务边界；四是把 parse/export 等重资源链路逐步异步化，最终形成统一任务中心和独立发布边界。

**Tech Stack:** Go 1.25 + Gin + GORM + PostgreSQL 14 + RabbitMQ + Redis + Docker Compose + Nginx + React + Zustand + `@microsoft/fetch-event-source`

---

## 0. 背景与约束

- 当前生产形态已经是 Docker Compose 多服务：`core-api / llm-stream / parser-service / export-service / review-service`，并由前端 Nginx 聚合为单一入口。
- 当前主要短板不是“有没有拆服务”，而是“服务自治是否成立”：任务消息发布闭环未完全硬化、共享数据库写入边界仍然模糊、服务健康检查和链路观测不足。
- 本计划按“先稳态、再自治、最后平台化”推进，避免把拆库、异步化、观测治理、CI/CD 一次性混在同一个改动里。
- 本计划默认仍以 Docker Compose 为统一验证入口；如果未来引入 Kubernetes，应另开专题计划，不在本计划内强推。

## 1. 目标态

### 1.1 阶段目标

- **Phase A：稳态收口**
  - 补齐生成任务从 `core-api` 到 `llm-stream` 的 RabbitMQ 发布和消费闭环。
  - 为所有服务补齐基础健康检查、request id、结构化日志和最小化运行指标。
- **Phase B：服务边界硬化**
  - 明确每个服务的“读写归属”和“跨服务调用规则”。
  - 收敛 `cmd/server` 的生产心智，避免开发态聚合入口继续反向塑造生产边界。
- **Phase C：重资源任务异步化**
  - 将 `parser-service` 和 `export-service` 逐步升级为任务消费型服务。
  - 对外继续保持同步入口或任务入口兼容，不强行一次切断旧链路。
- **Phase D：数据自治与独立发布**
  - 形成服务级数据归属矩阵。
  - 视收益决定是否推进“同实例不同库”或“独立实例”拆分。

### 1.2 完成定义

- `core-api` 不再直接承担长耗时生成执行，只承担接单、鉴权、任务查询和事件回放。
- `llm-stream` 可独立扩容且不会要求前端直连服务实例。
- `parser-service`、`export-service` 至少具备升级为任务消费型服务的统一任务模型。
- 所有对外服务均具备 `/health` 或 `/ready`、request id、结构化日志、Compose 健康检查。
- 文档、Runbook、CI 校验与代码保持同步。

## 2. 现状证据

- 生产形态以多服务和单网关入口为准，而不是 `cmd/server` 聚合入口。
- 当前 RabbitMQ consumer 已在 `llm-stream` 侧存在，但 `core-api` 侧消息发布器尚未正式注入。
- 当前 `review-service` 已具备独立 `REVIEW_DATABASE_URL`，说明“渐进式拆库”路径在项目内已经被验证可行。
- 当前多数服务仍共享 `backend/internal/service`、`backend/internal/infra/db` 和同一个核心数据库连接心智，需要逐步收敛。

## 3. 文件地图

**核心任务与消息链路**
- Modify: `backend/cmd/core-api/main.go`
- Modify: `backend/cmd/llm-stream/main.go`
- Modify: `backend/internal/domain/task/service.go`
- Modify: `backend/internal/infra/mq/rabbitmq.go`
- Modify: `backend/internal/transport/http/v1/routes.go`
- Modify: `backend/internal/transport/http/v1/api/task.go`
- Test: `backend/internal/domain/task/service_test.go`
- Test: `backend/internal/domain/stream/task_consumer_test.go`

**可观测性与服务契约**
- Create: `backend/internal/transport/http/middleware/request_id.go`
- Create: `backend/internal/transport/http/middleware/request_logger.go`
- Create: `backend/internal/transport/http/api/health.go`
- Modify: `backend/cmd/core-api/main.go`
- Modify: `backend/cmd/llm-stream/main.go`
- Modify: `backend/cmd/parser-service/main.go`
- Modify: `backend/cmd/export-service/main.go`
- Modify: `backend/cmd/review-service/main.go`
- Modify: `docker-compose.yml`

**边界治理**
- Modify: `backend/internal/infra/db/db.go`
- Modify: `backend/internal/service/*`
- Modify: `backend/internal/domain/*`
- Modify: `backend/cmd/server/main.go`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Database.md`

**解析与导出异步化**
- Modify: `backend/cmd/parser-service/main.go`
- Modify: `backend/cmd/export-service/main.go`
- Create: `backend/internal/domain/task/parse_*.go`
- Create: `backend/internal/domain/task/export_*.go`
- Modify: `frontend/src/services/*`
- Modify: `frontend/src/hooks/*`

**交付与验证**
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Create: `docs/runbooks/microservices-smoke-check.md`

---

### Task 1: 补齐生成链路 RabbitMQ 发布闭环

**目标：** 让 `core-api` 创建任务后真正发布 `generation.requested`，由 `llm-stream` 统一消费，形成当前最关键的服务间异步链路。

**Files:**
- Modify: `backend/cmd/core-api/main.go`
- Modify: `backend/internal/infra/mq/rabbitmq.go`
- Modify: `backend/internal/domain/task/service.go`
- Test: `backend/internal/domain/task/service_test.go`

- [ ] **Step 1: 收敛发布器注入点**
  - 在 `core-api` 启动链路中显式初始化 RabbitMQ channel 和 publisher，禁止继续使用 `taskdomain.NewService(taskRepo, nil)` 的占位方式。
  - 约束：若 RabbitMQ 未配置，可在启动时报出明确错误，或通过开关进入兼容只读模式，但不得静默退化为“任务创建成功但不投递”。

- [ ] **Step 2: 锁定发布成功/失败语义**
  - 更新 `backend/internal/domain/task/service_test.go`，新增两个测试：
    - 创建任务时发布成功，任务状态保持 `queued`
    - 发布失败时接口返回错误，且不出现“前端看到成功、后台无 worker 消费”的假成功

- [ ] **Step 3: 接入启动参数与连接回收**
  - 在 `backend/cmd/core-api/main.go` 中复用现有 `RABBITMQ_URL / RABBITMQ_EXCHANGE`。
  - 和 `llm-stream` 一样接入 `signalContext` 生命周期，确保优雅停机时关闭 channel 和 connection。

- [ ] **Step 4: 验证**
  - Run: `cd backend && go test ./internal/domain/task ./internal/infra/mq -count=1`
  - Run: `docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build`
  - Expected:
    - 创建任务后，RabbitMQ 队列有消息进入或被消费者及时消费
    - `llm-stream` 日志中可见消费记录
    - `/api/v1/tasks/:id/stream` 能收到状态推进

### Task 2: 收口生成入口，统一任务式 SSE 为默认主链路

**目标：** 把当前“旧 `/stream/*` 直连 SSE”和“任务式 SSE”之间的关系收敛为“任务式为默认、旧链路为兼容回滚”。

**Files:**
- Modify: `backend/internal/transport/http/v1/api/task.go`
- Modify: `backend/internal/transport/http/v1/routes.go`
- Modify: `frontend/src/services/generationTasks.ts`
- Modify: `frontend/src/hooks/generator/useSeriesGenerator.ts`
- Modify: `frontend/src/store/streamStore.ts`

- [ ] **Step 1: 明确对外契约**
  - 在前端生成入口默认先调 `/api/v1/tasks/generation`，再订阅 `/api/v1/tasks/:id/stream`。
  - 保留旧 `/api/v1/stream/*` 路由，但只作为紧急回滚路径，不再作为默认实现。

- [ ] **Step 2: 统一任务载荷模型**
  - 为 `generate_single`、`generate_series`、`continue`、`polish` 约定统一 `kind + payload` 模型。
  - 前端不再散落拼装多套请求结构，统一放到 `frontend/src/services/generationTasks.ts`。

- [ ] **Step 3: 统一事件映射**
  - 让 `streamStore` 只消费一种标准 SSE 事件模型，不再理解“两套不同源头的 chunk 协议”。
  - Why：只有统一事件模型，后续 parser/export 异步化时前端才能复用任务框架。

- [ ] **Step 4: 验证**
  - Run: `cd frontend && npm test -- --run`
  - Run: `cd frontend && npm run build`
  - 手工验证：
    - 单篇生成
    - 系列生成
    - 继续生成
    - 润色
    - 取消任务

### Task 3: 为所有服务补齐健康检查、request id 与结构化日志

**目标：** 解决“服务拆了，但排障和监控没有跟上”的问题，先建立微服务最基础的运行契约。

**Files:**
- Create: `backend/internal/transport/http/middleware/request_id.go`
- Create: `backend/internal/transport/http/middleware/request_logger.go`
- Create: `backend/internal/transport/http/api/health.go`
- Modify: `backend/cmd/core-api/main.go`
- Modify: `backend/cmd/llm-stream/main.go`
- Modify: `backend/cmd/parser-service/main.go`
- Modify: `backend/cmd/export-service/main.go`
- Modify: `backend/cmd/review-service/main.go`
- Modify: `docker-compose.yml`

- [ ] **Step 1: 增加统一健康检查路由**
  - 每个服务暴露：
    - `GET /api/v1/ping` 保持兼容
    - `GET /health` 返回进程存活
    - `GET /ready` 返回依赖就绪状态
  - `ready` 至少检查数据库连接；`llm-stream` 额外检查 RabbitMQ 连接配置可用性。

- [ ] **Step 2: 增加 request id 中间件**
  - 为所有 HTTP 请求注入 `X-Request-ID`。
  - 若上游已带 request id，优先透传；否则后端生成。

- [ ] **Step 3: 结构化日志落地**
  - 日志最少包含：`service`、`request_id`、`path`、`method`、`status`、`latency_ms`。
  - Why：当前多服务形态如果没有统一日志字段，定位跨服务问题成本会快速飙升。

- [ ] **Step 4: Compose 健康检查**
  - 在 `docker-compose.yml` 为 `core-api / llm-stream / parser-service / export-service / review-service / frontend` 增加 `healthcheck`。
  - 调整 `depends_on` 以依赖健康而不是仅依赖容器启动。

- [ ] **Step 5: 验证**
  - Run: `docker compose --env-file backend/.env up -d --build`
  - Run: `docker compose --env-file backend/.env ps`
  - Expected: 所有服务为 `healthy` 或 `Up (healthy)`。

### Task 4: 制定并落地“服务写入归属矩阵”

**目标：** 从“共享数据库随意写”推进到“共享数据库但写入有归属”，这是后续拆库前必须先完成的边界治理。

**Files:**
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `README.md`
- Modify: `backend/internal/domain/*`
- Modify: `backend/internal/service/*`

- [ ] **Step 1: 输出表级归属矩阵**
  - 建议初版：
    - `users`、`oauth_tokens`、`user_prompt_settings` -> `core-api`
    - `blogs`、`job_tasks`、`job_task_events` -> `core-api` 为事实归属，`llm-stream` 仅通过任务模型写事件或通过明确仓储边界写结果
    - `review_sessions`、`review_turns` -> `review-service`

- [ ] **Step 2: 清点跨服务直接写库点**
  - 重点扫描 `backend/internal/service/` 中直接使用 `db.DB` 的生成、续写、分析和导出逻辑。
  - 标记哪些必须短期保留，哪些可以通过 repo/interface 转义。

- [ ] **Step 3: 把“允许直接写”收敛为显式接口**
  - 对必须跨边界写入的路径建立明确仓储接口，避免继续从 service 层裸写全局 `db.DB`。
  - Why：如果现在不先把写点收口，后续拆库只会演变成全局搜索替换灾难。

- [ ] **Step 4: 验证**
  - 产出一份文档化的“表归属 + 服务职责 + 临时例外”清单，并在架构文档中固化。

### Task 5: 收敛 `cmd/server` 到开发/集成调试用途

**目标：** 防止开发态聚合入口继续模糊生产服务边界，让团队心智和部署心智保持一致。

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `docs/runbooks/microservices-smoke-check.md`

- [ ] **Step 1: 明确标签**
  - 在 README、架构文档、代码注释中明确写清：
    - `cmd/server` 仅用于本地集成调试
    - Docker 生产形态不使用该入口

- [ ] **Step 2: 避免新功能默认只接在聚合入口**
  - 任何新增服务能力优先接入对应 `cmd/*` 服务，不得只接在 `cmd/server` 后“回头再拆”。

- [ ] **Step 3: 准备退场标准**
  - 约定当以下条件满足后，可以考虑冻结甚至移除 `cmd/server`：
    - 所有核心链路已具备多服务本地验证脚本
    - 开发态已经能用 Compose 满足多数集成调试需求

### Task 6: 推进 `parser-service` 异步化

**目标：** 把大文件解析和 ZIP 课件解析从同步 HTTP 请求中拆出来，降低解析高峰对网关和 API 线程的挤压。

**Files:**
- Modify: `backend/cmd/parser-service/main.go`
- Create: `backend/internal/domain/task/parse_task.go`
- Create: `backend/internal/domain/task/parse_consumer.go`
- Modify: `frontend/src/services/*`
- Modify: `frontend/src/hooks/*`

- [ ] **Step 1: 定义 parse 任务 subtype**
  - 如：
    - `parse_file`
    - `parse_archive`

- [ ] **Step 2: 保持外部兼容**
  - 初期可以采用“双入口”：
    - 同步 `/api/v1/project/parse` 保留
    - 新增任务式入口 `/api/v1/tasks/parse`
  - 等任务式验证稳定后，再决定是否收口同步入口。

- [ ] **Step 3: 标准化解析结果**
  - `result_json` 至少包含：
    - `source_content`
    - `archive_summary`
    - `warnings`

- [ ] **Step 4: 验证**
  - 普通文件
  - ZIP 课件包
  - 大文件超时与错误提示

### Task 7: 推进 `export-service` 异步化

**目标：** 把 PDF 导出和 Obsidian 导出收口为后台任务，隔离 Chromium 和批量写 Vault 的资源风险。

**Files:**
- Modify: `backend/cmd/export-service/main.go`
- Create: `backend/internal/domain/task/export_task.go`
- Create: `backend/internal/domain/task/export_consumer.go`
- Modify: `frontend/src/services/*`
- Modify: `frontend/src/components/*`

- [ ] **Step 1: 定义 export 任务 subtype**
  - `export_markdown_zip`
  - `export_pdf`
  - `export_obsidian_single`
  - `export_obsidian_series`

- [ ] **Step 2: 先异步化最重路径**
  - 优先顺序：
    - `export_pdf`
    - `export_obsidian_series`
    - 其余再跟进

- [ ] **Step 3: 明确幂等与重试语义**
  - PDF 允许重试覆盖同一输出。
  - Obsidian 批量导出必须增加幂等保护，避免重复写入同一批卡片。

- [ ] **Step 4: 验证**
  - 单篇导出
  - 系列 PDF 导出
  - 单篇/系列 Obsidian 导出
  - 失败重试和取消

### Task 8: 规划数据自治路线，而不是立刻全面拆库

**目标：** 先用“同实例不同库”和“归属矩阵”验证边界，再决定是否进入真正的独立实例拆分。

**Files:**
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `docs/runbooks/review-db-migration.md`
- Create: `docs/runbooks/core-blog-task-boundary.md`

- [ ] **Step 1: 先保持 review 路线**
  - 把 `review-service` 独立库的经验沉淀成可复用模板。

- [ ] **Step 2: 评估下一个最适合拆的数据边界**
  - 建议优先评估：
    - `job_tasks / job_task_events`
    - 或 `blogs` 生成结果相关写入

- [ ] **Step 3: 给出拆库门槛**
  - 只有在以下条件满足时才推进独立实例：
    - 写入归属清晰
    - 跨服务读写已接口化
    - 回滚方案和迁移 Runbook 准备完成

### Task 9: 把微服务验证前置到 CI 与 Runbook

**目标：** 让“多服务能否正常工作”从人工经验变成可重复验证的工程流程。

**Files:**
- Modify: `.github/workflows/ci.yml`
- Create: `docs/runbooks/microservices-smoke-check.md`
- Modify: `README.md`

- [ ] **Step 1: CI 补充 Compose 冒烟**
  - 在现有 `docker compose config` 基础上，增加最小启动和健康检查脚本。

- [ ] **Step 2: Runbook 固化**
  - 新建 `docs/runbooks/microservices-smoke-check.md`，明确以下检查：
    - 服务启动
    - 网关访问
    - 任务创建
    - 任务 SSE
    - review 可用
    - parser/export 基础通路

- [ ] **Step 3: 提交前核对文档同步**
  - 每次涉及 API、数据库、Compose、路由边界调整，必须同步更新：
    - `README.md`
    - `.trae/documents/InkWords_API.md`
    - `.trae/documents/InkWords_Architecture.md`
    - `.trae/documents/InkWords_Database.md`
    - `.trae/documents/InkWords_Development_Plan_and_Log.md`

## 4. 里程碑建议

- **Milestone 1：两周内**
  - 完成 Task 1-3
  - 目标：生成链路任务闭环稳定、服务健康可见、日志可追踪

- **Milestone 2：一个月内**
  - 完成 Task 4-5
  - 目标：服务写入边界形成文档化契约，生产与开发心智一致

- **Milestone 3：一到两个月**
  - 完成 Task 6-7
  - 目标：parse/export 重资源链路具备任务化能力

- **Milestone 4：长期演进**
  - 完成 Task 8-9
  - 目标：形成稳定的数据自治路线和可审计的微服务交付流程

## 5. 风险与回滚

- **风险 1：任务链路切换导致前端回归**
  - 回滚：保留旧 `/api/v1/stream/*` 直连链路，前端可通过 feature flag 或配置切回。

- **风险 2：消息发布已接入，但状态回写不完整**
  - 回滚：先保持任务 API 灰度，仅对系列生成启用。

- **风险 3：过早拆库导致跨服务读写爆炸**
  - 回滚：继续停留在共享实例 + 写入归属矩阵，不推进物理拆分。

- **风险 4：异步化后用户感知“更慢”**
  - 回滚：保留同步路径用于小请求；前端补齐排队中、运行中、重试中、可取消的状态提示。

## 6. 验证基线

- `cd backend && go test ./...`
- `cd frontend && npm test -- --run`
- `cd frontend && npm run build`
- `docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build`
- `docker compose --env-file backend/.env ps`
- `curl -I http://localhost`

## 7. 参考

- [InkWords_Microservices_RFC.md](file:///Users/huangqijun/Documents/墨言博客助手/InkWords/.trae/documents/InkWords_Microservices_RFC.md)
- [InkWords_Architecture.md](file:///Users/huangqijun/Documents/墨言博客助手/InkWords/.trae/documents/InkWords_Architecture.md)
- [InkWords_API.md](file:///Users/huangqijun/Documents/墨言博客助手/InkWords/.trae/documents/InkWords_API.md)
- [docker-compose.yml](file:///Users/huangqijun/Documents/墨言博客助手/InkWords/docker-compose.yml)
- [nginx.conf](file:///Users/huangqijun/Documents/墨言博客助手/InkWords/frontend/nginx.conf)


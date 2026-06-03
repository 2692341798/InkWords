# InkWords RabbitMQ 事件驱动微服务设计

## 1. 目标

- 在保留现有 `http://localhost` 单入口与 `/api/*` 对外路径不变的前提下，把 InkWords 从“路径分流的多服务过渡态”升级为“可独立演进的事件驱动微服务架构”。
- 保留现有 `core-api / llm-stream / parser-service / export-service / review-service` 的服务拆分成果，不回退到单体。
- 为 `LLM 生成`、`文档解析`、`导出/知识摄入` 三类长耗时任务引入 RabbitMQ，降低长请求占用 Web 进程、连接和内存的风险。
- 保持前端中文界面与现有核心交互体验不回归，尤其是生成链路的 SSE 体验。
- 采用渐进式迁移，每一阶段都允许通过 Docker Compose 一键重启验证，并具备清晰回滚路径。

## 2. 背景与现状

### 2.1 当前真实形态

InkWords 当前已经不是“单体待拆分”，而是“多服务已落地，但内部治理尚未完全收口”的状态：

- 前端由独立 Nginx 容器提供静态资源与反向代理。
- 后端已拆为 `core-api`、`llm-stream`、`parser-service`、`export-service`、`review-service` 五个服务。
- PostgreSQL 负责业务数据持久化，`review-service` 已支持独立数据库。
- Redis 作为缓存与可选辅助能力。
- `obsidian-bridge` 作为容器到宿主机 Obsidian REST API 的桥接。

这套形态已经满足“多服务部署”，但还存在几个结构性问题：

- 长任务仍大量绑定在 HTTP 请求生命周期里，尤其是 `stream/generate`、`project/parse`、`export/pdf`、`export/obsidian*`。
- `cmd/server` 聚合入口仍保留，说明代码心智仍兼容单体，服务边界没有完全硬化。
- 部分领域边界重复，例如 `project` 与 `fileparse` 在 parse 能力上存在重叠。
- 服务间尚未建立事件总线，导致长任务调度、失败补偿、状态观测、重试语义都偏弱。

### 2.2 当前已落地的边界

- `core-api`
  - 负责认证、用户、博客基础 CRUD、部分项目元数据接口。
- `llm-stream`
  - 负责 `/api/v1/stream/*` 与 `/api/v1/blogs/:id/(continue|polish)`。
- `parser-service`
  - 负责 `/api/v1/project/parse`。
- `export-service`
  - 负责 `/api/v1/blogs/:id/export*`。
- `review-service`
  - 负责 `/api/v1/review/*`。
- `frontend/nginx.conf`
  - 负责按路径把外部请求分流到对应服务。

因此，本次设计的重点不是“重新发明微服务边界”，而是为现有服务增加一层统一的异步任务基础设施。

## 3. 设计原则

- 最小改动优先：优先复用现有服务入口、路由和领域服务，不做无关重构。
- 单入口不变：外部用户始终通过 `http://localhost` 与 `/api/*` 访问。
- API 兼容优先：能保持前端请求路径不变，就不引入额外 base URL、额外端口或前端环境分叉。
- 长任务异步化：超过典型 Web 请求预算的任务，优先改为“创建任务 -> 投递事件 -> 后台执行 -> 状态回写”。
- Why-first 注释：跨服务边界、消息投递、幂等控制、回滚逻辑必须说明为什么这样做。
- Docker-First：本地、测试、生产继续以 Docker Compose 作为一致入口。

## 4. 目标架构

### 4.1 总体结构

保留当前单入口网关与已有服务划分，在内部新增 RabbitMQ 事件总线：

- `frontend`
  - 浏览器端 React 应用，继续只访问 `/api/*`。
- `gateway`
  - 前端 Nginx 单入口，继续负责路径级反向代理。
- `core-api`
  - 负责同步短事务 API、任务创建、任务状态查询。
- `llm-stream`
  - 负责生成类任务消费、流式输出协调、生成结果落库。
- `parser-service`
  - 负责解析类任务消费与结果产出。
- `export-service`
  - 负责 PDF 导出、Obsidian 导出、知识摄入任务消费。
- `review-service`
  - 保持同步服务，不优先异步化。
- `rabbitmq`
  - 负责长任务事件总线、重试和死信隔离。
- `postgres`
  - 继续作为共享实例；短期保持共享库/共享 schema 思路，`review-service` 可维持独立 DB。

### 4.2 为什么选 RabbitMQ 而不是 Kafka / NATS

- 你已明确选择 RabbitMQ。
- 当前任务规模更像“业务任务队列”，不是“高吞吐日志流”。
- 需要明确的队列语义、ACK、重试、死信和延迟补偿，RabbitMQ 更贴合当前阶段。
- 对 Compose 本地开发更友好，运维门槛低于 Kafka。

## 5. 服务职责与边界

### 5.1 `core-api`

职责：

- 用户认证、用户资料、博客列表与博客详情等同步短事务。
- 任务接单入口。
- 任务状态查询接口。
- 作为“写请求入口”的统一协调者。

不负责：

- 长时间 LLM 推理执行。
- 大文件解析执行。
- Chromium PDF 渲染执行。

建议新增能力：

- `POST /api/v1/tasks/generation`
- `POST /api/v1/tasks/parse`
- `POST /api/v1/tasks/export`
- `GET /api/v1/tasks/:id`

### 5.2 `llm-stream`

职责：

- 消费生成类任务。
- 执行 `analyze / scan / generate / continue / polish`。
- 将正文 chunk、阶段状态、usage 和最终结果写回任务状态表。
- 对接前端 SSE 订阅能力。

不负责：

- 新任务创建与授权判定。
- 非生成类同步查询。

### 5.3 `parser-service`

职责：

- 消费解析类任务。
- 执行普通文档解析、ZIP 课件解析、Git 获取预处理。
- 输出 `source_content`、`archive_summary` 等结构化结果。

### 5.4 `export-service`

职责：

- 消费导出类任务。
- 执行 PDF 渲染。
- 执行 Obsidian 导出与知识摄入。

说明：

`export-service` 当前同时包含“导出”和“知识摄入”两类能力，短期先保留一个服务，待观察瓶颈后再考虑拆为 `pdf-renderer` 与 `obsidian-sync`。

### 5.5 `review-service`

职责：

- 保持知识复习、session、history、hint 的同步请求处理。
- 继续读取 Obsidian wiki 作为复习源。

说明：

`review-service` 当前不属于高耗时任务中心，不纳入第一阶段事件驱动改造。

## 6. 任务模型与事件模型

### 6.1 统一任务模型

新增统一任务表，例如 `job_tasks`，用于承载所有异步任务：

- `id`
- `task_type`
  - `generation`
  - `parse`
  - `export`
- `task_subtype`
  - 例如 `generate_series`、`parse_archive`、`export_pdf`
- `status`
  - `pending`
  - `queued`
  - `running`
  - `streaming`
  - `succeeded`
  - `failed`
  - `cancelled`
- `requested_by`
- `payload_json`
- `result_json`
- `error_message`
- `idempotency_key`
- `retry_count`
- `started_at`
- `finished_at`
- `created_at`
- `updated_at`

### 6.2 统一事件主题

建议采用以下 routing key：

- `generation.requested`
- `generation.progress`
- `generation.completed`
- `generation.failed`
- `parse.requested`
- `parse.completed`
- `parse.failed`
- `export.requested`
- `export.completed`
- `export.failed`

### 6.3 事件流转

以“系列生成”为例：

1. 前端调用 `core-api` 创建生成任务。
2. `core-api` 校验用户权限、请求参数与限额。
3. `core-api` 插入 `job_tasks(status=pending)`。
4. `core-api` 将任务消息投递到 RabbitMQ，并把任务状态更新为 `queued`。
5. `llm-stream` consumer 拉取消息并 ACK 前先把任务状态更新为 `running`。
6. `llm-stream` 执行生成过程，持续把阶段状态和 chunk 写回数据库。
7. 任务成功后写 `result_json` 并标记 `succeeded`。
8. 任务失败后写 `error_message` 并标记 `failed`。

## 7. SSE 与前端交互策略

### 7.1 生成链路

生成类任务继续保留 SSE 体验，但 SSE 不再直接绑定一次长 HTTP 调用，而是改为：

- 前端先创建任务，拿到 `task_id`。
- 前端再通过 SSE 订阅 `/api/v1/tasks/:id/stream` 一类接口。
- `core-api` 或 `llm-stream` 根据任务状态表中的增量内容输出 SSE 事件。

这样做的原因：

- 把任务执行与用户连接解耦，浏览器断连不等于任务立刻丢失。
- 后端可以更清晰地区分“任务取消”和“前端订阅断开”。
- 后续更容易支持重连续播。

### 7.2 解析与导出链路

第一阶段不强制引入 SSE，采用轮询更稳妥：

- 前端创建任务后轮询 `GET /api/v1/tasks/:id`。
- 当状态为 `succeeded` 时读取结果。

后续如有必要，再给解析或导出补 SSE。

## 8. 数据一致性与幂等策略

### 8.1 数据库策略

- 短期维持共享 Postgres 实例。
- `core-api`、`llm-stream`、`parser-service`、`export-service` 共用主业务库。
- `review-service` 继续维持独立 DB 或独立 URL。
- 本阶段不做“每服务独立数据库”的强隔离迁移。

### 8.2 幂等策略

所有异步任务入口必须支持 `idempotency_key`：

- 前端重复点击不应生成多个完全相同的任务。
- `core-api` 在创建任务前先按 `requested_by + task_type + idempotency_key` 查重。
- consumer 在重复收到相同任务时，先检查任务状态再决定是否跳过。

### 8.3 失败补偿

- RabbitMQ 使用死信队列隔离多次失败的任务。
- 任务失败不自动删除，保留错误信息供前端展示与人工重试。
- 导出类与解析类优先采用“用户显式重试”，而不是无限自动重试。

## 9. Docker 与部署设计

### 9.1 新增服务

在 `docker-compose.yml` 中新增：

- `rabbitmq`
  - 默认镜像可用 `rabbitmq:3-management-alpine`
  - 对内服务名使用 `rabbitmq`

### 9.2 现有服务调整

- `core-api` 增加 RabbitMQ 连接配置，用于投递任务。
- `llm-stream` 增加 RabbitMQ 连接配置，用于消费生成任务。
- `parser-service` 增加 RabbitMQ 连接配置，用于消费解析任务。
- `export-service` 增加 RabbitMQ 连接配置，用于消费导出任务。
- `frontend` 与 `review-service` 在第一阶段不强制依赖 RabbitMQ。

### 9.3 环境变量建议

- `RABBITMQ_URL`
- `RABBITMQ_EXCHANGE`
- `RABBITMQ_GENERATION_QUEUE`
- `RABBITMQ_PARSE_QUEUE`
- `RABBITMQ_EXPORT_QUEUE`
- `RABBITMQ_DEAD_LETTER_EXCHANGE`

## 10. 分阶段迁移计划

### 10.1 Phase A：架构收口

目标：

- 明确当前五服务是正式运行形态。
- 降低 `cmd/server` 的生产地位，仅保留本地集成调试用途。
- 清理重复的 parse 边界与过重的 bootstrap。

产出：

- 服务边界文档同步。
- 各服务启动入口和路由注册方式统一。

### 10.2 Phase B：生成链路事件化

目标：

- 先把 `llm-stream` 改成真正的后台任务消费者。

产出：

- `job_tasks` 表。
- 生成任务创建接口。
- RabbitMQ publisher / consumer。
- SSE 订阅模型。

原因：

这是全项目收益最大、风险最高的长任务链路，优先改造最有价值。

### 10.3 Phase C：解析链路事件化

目标：

- 将 `project/parse` 从同步上传解析改为“上传后入队处理”。

注意：

- 文件上传本身仍可能需要 HTTP 接收，但解析执行必须异步化。

### 10.4 Phase D：导出链路事件化

目标：

- 把 PDF 导出与 Obsidian 导出改为后台任务。

注意：

- PDF 渲染与知识摄入都属于资源重任务，应避免长期占用 Web 请求线程。

## 11. 回滚策略

### 11.1 路由级回滚

- 如果 RabbitMQ 任务模型出现问题，可让 Nginx 和 `core-api` 暂时回退到现有同步接口。
- 保留现有路径不变，回滚成本只在服务内部。

### 11.2 服务级回滚

- 每个服务仍保留 HTTP 入口，不在第一阶段删除现有同步处理逻辑，直到异步链路稳定。
- `cmd/server` 在过渡期间保留，作为本地对照入口。

### 11.3 数据级回滚

- 新增的任务表不会影响既有博客、用户、复习主数据。
- 回滚时只需停用任务入口并关闭消费者，不需要迁移主业务表。

## 12. 风险与已知局限

- 共享数据库仍意味着严格意义上的数据自治尚未完成。
- 生成类任务若同时写数据库和推 SSE，必须明确事务与推送顺序，否则容易出现状态漂移。
- 大文件上传若完全走同步上传，再异步解析，仍要小心容器磁盘与临时文件清理。
- `export-service` 内部目前职责偏多，后续仍可能需要继续拆分。
- 如果前端需要“刷新页面后继续看到旧进度”，则必须设计任务结果缓存与 SSE 重连协议。

## 13. 验收标准

- 继续通过 `http://localhost` 访问应用。
- 前端仍只请求 `/api/*`，不新增面向用户的后端端口。
- `llm-stream`、`parser-service`、`export-service` 的长任务可脱离单个 HTTP 请求独立执行。
- 任一任务都具备明确的 `task_id`、状态流转、错误信息与重试入口。
- Docker Compose 仍支持：
  - `docker compose up -d --build`
  - `docker compose down && docker compose up -d --build`
- 关键链路可验证：
  - 登录成功
  - 系列生成成功
  - 文档解析成功
  - PDF/Obsidian 导出成功
  - 复习链路不回归

## 14. 非目标

- 本次不引入服务发现、服务网格、Kubernetes。
- 本次不做每服务独立数据库的彻底拆库。
- 本次不改动前端整体产品交互，只在必要处补任务编排。
- 本次不改造 `review-service` 为异步任务服务。

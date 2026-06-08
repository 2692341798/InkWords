# InkWords Core-API 与 LLM-Stream 深层拆分设计

**类型**：Technical Design / Backend Architecture  
**范围**：`core-api` 与 `llm-stream` 的深层服务自治拆分  
**日期**：2026-06-04  
**状态**：待评审

## 1. 背景

Phase 1 已完成：

- `review-service`
- `parser-service`
- `export-service`

这三个服务已经迁入 `backend/services/<service>/` 自有目录，并保持 Docker Compose 多服务 + Nginx 单入口的运行形态不变。

但 `core-api` 与 `llm-stream` 仍保留明显的“共享业务核心 + 独立入口”过渡态特征：

- 两个服务继续共享 `internal/service/generator*.go`
- 两个服务继续共享 `internal/service/decomposition_*.go`
- `llm-stream` 仍通过共享业务逻辑直接写入 `blogs / users`
- 两个服务的路由与 HTTP 适配仍高度依赖共享 `internal/transport/http/v1`

因此，当前真正尚未拆开的核心耦合只剩 `core-api` 与 `llm-stream`。

## 2. 目标与非目标

### 2.1 目标

- 让 `core-api` 与 `llm-stream` 具备接近完全自治的服务目录结构
- 让 `llm-stream` 只承担生成/分析/流式事件生产，不再直接写业务事实表
- 让 `core-api` 成为 `blogs / users / task` 等核心事实表的唯一业务写入方
- 保持前端单入口、现有 `/api/*` 路径和 Docker Compose 运行方式不变
- 为下一阶段进一步拆库或独立扩容打下明确边界

### 2.2 非目标

- 不改前端页面或前端路由结构
- 不改数据库实例拆分策略
- 不引入服务网格、Kafka 或新的复杂基础设施
- 不改对外 API URL 与基础请求语义

## 3. 现状问题

### 3.1 `core-api` 当前问题

- 启动入口仍直接装配共享 `BlogService`、`DecompositionService`、`TaskService`
- `project analyze` 与任务创建逻辑仍依赖共享 use case
- `transportv1.RegisterCore` 继续作为共享路由总入口

### 3.2 `llm-stream` 当前问题

- 启动入口仍直接装配共享 `GeneratorService`、`DecompositionService`
- `transportv1.RegisterStream` 仍是共享 transport 入口
- 生成、续写、润色相关逻辑仍通过共享 use case 和共享数据写入路径运作

### 3.3 最关键的技术债

- `GeneratorService / DecompositionService` 通过全局 `db.DB` 直接写 `blogs / users`
- `StreamAPI` 只是共享 use case 和共享 transport 之间的中间适配层
- `internal/transport/http/v1/routes.go` 继续同时控制多服务主路由

## 4. 目标架构

### 4.1 `core-api`

目录目标：

```text
backend/services/core-api/
  cmd/
  app/
    bootstrap/
    config/
  domain/
    auth/
    user/
    blog/
    project/
    task/
  infra/
    db/
    mq/
    cache/
  transport/
    http/
      middleware/
      v1/
```

职责目标：

- 提供用户态 API
- 创建/查询/取消任务
- 负责博客事实持久化
- 负责 token 记账
- 负责项目扫描与非流式协调逻辑
- 负责把任务最终结果写回 `blogs / users`

### 4.2 `llm-stream`

目录目标：

```text
backend/services/llm-stream/
  cmd/
  app/
    bootstrap/
    config/
  domain/
    stream/
    generation/
  infra/
    db/
    mq/
    llm/
  transport/
    http/
      middleware/
      v1/
```

职责目标：

- 承担分析、生成、续写、润色等 LLM 用例
- 承担 RabbitMQ generation worker
- 负责生成流式事件与任务结果
- 不再直接写 `blogs / users`

### 4.3 `shared`

`shared` 只允许保留：

- JWT 基础件
- 通用 response
- request id / tracing / logging 基础件
- PostgreSQL / RabbitMQ 连接工厂
- 稳定错误模型
- 任务消息 schema

`shared` 不再允许保留：

- `GeneratorService`
- `DecompositionService`
- `StreamAPI`
- 带 blog/project/stream/task 业务语义的 use case、repository、handler

## 5. 写入边界重定义

### 5.1 `llm-stream` 允许写入

- `job_tasks.status`
- `job_tasks.result_json`
- `job_task_events`

### 5.2 `llm-stream` 禁止写入

- `blogs`
- `users.tokens_used`
- 任何其它业务事实表

### 5.3 `core-api` 负责的最终持久化

`core-api` 新增任务结果持久化用例，统一负责：

- 单篇生成落库
- 系列父子博客结构落库
- 续写内容回写
- 润色内容回写
- token 计账

## 6. 新的数据流

1. 前端继续调用 `core-api` 创建 generation/analyze/continue/polish 任务
2. `core-api` 写入 `job_tasks` 并投递 MQ
3. `llm-stream` 消费任务，执行分析/生成/续写/润色
4. `llm-stream` 持续写入 `job_task_events`
5. `llm-stream` 将结构化完成结果写入 `job_tasks.result_json`
6. `core-api` 在任务完成或回放阶段执行结果持久化，将结果落回 `blogs / users`
7. 前端仍通过 `core-api` 订阅任务流，外部体验不变

## 7. 目录与代码迁移目标

### 7.1 需要迁出的共享业务核心

从共享目录迁出的重点对象：

- `internal/service/generator*.go`
- `internal/service/decomposition_analyze*.go`
- `internal/service/decomposition_generate*.go`
- `internal/service/decomposition_scan*.go`
- `internal/transport/http/v1/api/stream_api.go`
- `internal/transport/http/v1/routes.go` 中 `core-api / llm-stream` 专属部分

### 7.2 归位方向

- 生成、续写、润色、分析主链路 -> `services/llm-stream/domain/generation`
- 流式 handler、SSE 和 worker 编排 -> `services/llm-stream/domain/stream`
- 博客结果落库、token 记账、任务结果持久化 -> `services/core-api/domain/blog`、`services/core-api/domain/task`
- `core-api` 项目编排与请求校验 -> `services/core-api/domain/project`
- `core-api` 和 `llm-stream` 各自拥有私有 transport 路由

## 8. 迁移分期

### Phase 2A：transport 与 bootstrap 私有化

- 新建 `services/core-api/app/bootstrap`
- 新建 `services/core-api/transport/http/v1`
- 新建 `services/llm-stream/app/bootstrap`
- 新建 `services/llm-stream/transport/http/v1`
- 停止让共享 `routes.go` 和 `stream_api.go` 继续控制这两个服务主入口

### Phase 2B：generation use case 迁入 `llm-stream`

- 把分析、生成、续写、润色的核心 use case 迁入 `services/llm-stream/domain/generation`
- 让 `llm-stream` 拥有自己的 LLM infra 与 worker 协调逻辑

### Phase 2C：结果持久化迁入 `core-api`

- `core-api` 新增任务结果持久化用例
- 接管 `blogs` 与 `users.tokens_used` 的最终写入
- `llm-stream` 停止直接操作业务表

### Phase 2D：删除旧共享核心

- 删除旧共享 `generator / decomposition / stream api` 对应部分
- 清空共享 `transport` 中这两个服务的主入口职责
- 只保留极薄兼容层或基础件

## 9. 测试策略

### 9.1 单元测试

- `core-api` 私有 transport 路由测试
- `llm-stream` 私有 transport 路由测试
- 任务结果持久化器测试：
  - 单篇生成落库
  - 系列结构落库
  - 续写落库
  - 润色落库
  - token 计账

### 9.2 回归测试

- 保持 `/api/v1/tasks/*` 行为验证
- 保持 `/api/v1/stream/*` 回滚路径验证
- 保持 `/api/v1/blogs/:id/(continue|polish)` 兼容验证

### 9.3 集成验证

- `cd backend && go test ./...`
- `docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build`
- `docker compose --env-file backend/.env ps`
- `curl -I http://localhost`
- `curl http://localhost/api/v1/ping`
- 至少一条 “create task -> stream -> persist” 冒烟链路

## 10. 回滚策略

- 每个 phase 单独提交，不把 2A / 2B / 2C / 2D 混在一个提交里
- 保持旧任务协议不变
- 保持前端单入口与路径不变
- 如果 2B 失败，可先回退到“transport 私有化完成，但 generation 仍指向旧实现”
- 如果 2C 失败，可先回退到“结果持久化仍走旧路径”，不回退目录边界

## 11. 风险

- `generator / decomposition` 代码归属复杂，真实拆分量可能大于 Phase 1
- `llm-stream` 停止直写业务表后，任务结果持久化协议必须一次定义清楚
- 如果阶段切得不稳，容易在 `core-api` 和 `llm-stream` 之间形成新的中间过渡层

## 12. 设计结论

第二阶段应直接推进 `core-api` 与 `llm-stream` 的深层拆分，目标不是“再做一层入口目录”，而是：

- 让两者拥有各自的 `app / domain / transport / infra`
- 让共享层退化为最薄基础层
- 让 `llm-stream` 只负责生成与任务事件
- 让 `core-api` 成为业务事实表的唯一写入方

这条路线最符合“几乎完全自治”的目标，也为后续真正的独立扩容、拆库或服务进一步分治提供了稳定边界。

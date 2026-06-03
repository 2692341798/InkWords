# InkWords Docker 微服务化设计（Phase 1）

## 1. 目标

- 把后端拆为两个可独立扩容的服务：`core-api` 与 `llm-stream`
- 保持对外 API URL 不变，前端无需修改 base URL
- 通过 Nginx 按路径分流请求，实现仅 `llm-stream` 独立扩容
- 保持 SSE 稳定性（禁缓冲、长超时、断连取消语义不回归）
- 本阶段不引入 MQ/服务网格/拆库，优先最小改动

## 2. 现状与边界

现有服务入口：
- `frontend`：Nginx 静态站点 + `/api/*` 反代
- `backend`：Go 单体（`backend/cmd/server`），注册所有 `/api/v1/*`

现有路由边界：
- `/api/v1/stream/*`（SSE：scan/analyze/generate）
- `/api/v1/blogs/:id/continue`、`/api/v1/blogs/:id/polish`（SSE）
- 其它：auth/user/blog/project/review/export 等

重计算链路（优先拆分对象）：
- LLM 流式分析/生成（SSE + 高并发受限 + CPU/内存压力）

## 3. 目标架构（Phase 1）

### 3.1 服务拆分

1) `core-api`
- 承载：除流式生成外的所有 API
- 路由包含：`/api/v1/auth`、`/api/v1/user`、`/api/v1/blogs`（不含 continue/polish）、`/api/v1/project`、`/api/v1/review`

2) `llm-stream`
- 承载：仅流式链路
- 路由包含：
  - `/api/v1/stream/*`
  - `/api/v1/blogs/:id/continue`
  - `/api/v1/blogs/:id/polish`

### 3.2 网关分流（frontend/nginx.conf）

分流策略：
- `/api/v1/stream/` 前缀 → `llm-stream`
- `/api/v1/blogs/*/continue`、`/api/v1/blogs/*/polish` → `llm-stream`
- 其余 `/api/` → `core-api`

SSE 网关要求：
- 关闭缓冲
- 关闭缓存
- 读取/写入超时放宽（小时级）

### 3.3 数据与一致性（Phase 1）

- `core-api` 与 `llm-stream` 共享 PostgreSQL 与 schema
- `llm-stream` 仍可直接写入 `blogs` 与累加 `users.tokens_used`，保持现有事务边界与错误处理策略
- 后续如需进一步收口一致性，可再引入“内部写入 API”或 outbox，但不在本阶段范围

## 4. 代码组织方案（Monorepo 内）

### 4.1 新增两个可执行入口

- `backend/cmd/core-api/main.go`
- `backend/cmd/llm-stream/main.go`

两者共享：
- 配置加载方式（环境变量/`.env`）
- DB/Redis 初始化
- Gin 基础中间件（含鉴权中间件）

差异：
- 路由注册不同（core-api 不注册 stream，llm-stream 只注册 stream 与 continue/polish）

### 4.2 路由注册拆分

新增两个注册函数（放在 `backend/internal/transport/http/v1`）：
- `RegisterCore(...)`
- `RegisterStream(...)`

并保证：
- 现有 `routes.go` 的 handler 校验逻辑复用
- 单体入口 `cmd/server` 可保留（作为回滚/对照），或在 Phase 1 完成后再删除

## 5. Docker Compose 方案

### 5.1 服务定义

新增两个服务：
- `core-api`：build backend，运行 core-api 二进制
- `llm-stream`：build backend，运行 llm-stream 二进制

保持现有：
- `frontend`（Nginx）
- `db`（PostgreSQL）
- `redis`（可选）
- `obsidian-bridge`（保持）

### 5.2 扩容方式

- 仅扩容生成服务：`docker compose up -d --build --scale llm-stream=3`

## 6. 验证计划

### 6.1 本地 Compose 冒烟

- 启动：`docker compose --env-file backend/.env up -d --build`
- 验证：
  - 登录：`POST /api/v1/auth/login`
  - SSE 生成：`POST /api/v1/stream/generate`（观察流式刷新）
  - 继续生成：`POST /api/v1/blogs/:id/continue`
  - 润色：`POST /api/v1/blogs/:id/polish`
  - 非流式 API：`GET /api/v1/blogs`、`GET /api/v1/user/profile`

### 6.2 回滚策略

- 网关分流回滚：把 `frontend/nginx.conf` 的 upstream 回切到单个 `backend`
- 服务回滚：继续使用原 `backend` 服务（单体入口）作为稳定对照

## 7. 风险与已知局限

- Phase 1 共享数据库，暂不具备“服务级数据隔离”
- Nginx 分流需谨慎处理 `location` 匹配优先级，避免 `continue/polish` 误落到 core-api
- `llm-stream` 水平扩容后，仍需依赖现有并发限制策略防止 LLM 限流

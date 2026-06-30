# 微服务冒烟检查 Runbook

本文档用于把 InkWords 的“多服务是否还能正常协同工作”从口头经验收敛成可重复执行的检查流程。  
目标覆盖长期微服务计划 `Task 9` 要求的六类检查：服务启动、网关访问、任务创建、任务 SSE、review 可用、parser/export 基础通路。

## 1. 适用场景

- 合并涉及 Docker Compose、服务入口、健康检查、任务中心、Nginx 代理的改动前后
- 本地执行 `docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build` 后
- CI 中做最小微服务冒烟验证时

## 2. 前置条件

- 已配置 `backend/.env`
- 至少具备以下变量：
  - `DEEPSEEK_API_KEY`
  - `JWT_SECRET`
  - `OBSIDIAN_REST_API_KEY`
  - `OBSIDIAN_VAULT_PATH`
- 本地 Docker / Docker Compose 可用
- 若要验证生成、解析、导出与 review 的完整业务链路，需准备一个可登录账号和一份可用的 Obsidian `wiki/`
- Phase 1 当前基线：`parser-service`、`review-service`、`export-service` 的服务私有入口与装配应分别收口在 `backend/services/parser-service/`、`backend/services/review-service/`、`backend/services/export-service/`；如果你刚修改过这些目录，本 Runbook 应视为必跑项

## 2.5 P0 最小回归集

每次改动以下任一文件后，至少执行本节 4 步，再决定是否进入更完整的冒烟流程：

- `docker-compose.yml`
- `frontend/nginx.conf`
- `backend/services/*/cmd/main.go`
- `.github/workflows/ci.yml`

Why:
- 这 4 步是现有 Runbook 中最容易回归、但也最能快速暴露编排与网关退化的最小集合。
- 先把它们收敛成固定门槛，可以避免“容器能启动，但 parser worker 已被静默禁用”这类问题再次漏过。

### P0-1 渲染 Compose

```bash
docker compose --env-file backend/.env config
```

预期结果：
- Compose 渲染成功
- `core-api / llm-stream / parser-service / export-service / review-service / frontend` 都在输出中出现

### P0-2 检查健康状态

```bash
docker compose --env-file backend/.env ps
```

预期结果：
- `core-api / llm-stream / parser-service / export-service / review-service / frontend` 均处于 `Up` 或 `healthy`

### P0-3 检查网关

```bash
curl -I http://localhost
curl --fail http://localhost/api/v1/ping
```

预期结果：
- `http://localhost` 返回 `200 OK`
- `/api/v1/ping` 返回成功响应，证明 Nginx 到 `core-api` 的代理仍然可用

### P0-4 检查 parser 任务链路

创建一条 `POST /api/v1/tasks/parse` 任务，并确认：

- `parser-service` 日志中不再出现 `parse consumer disabled`
- 任务最终能看到结果事件，或拿到成功快照
- 若本轮只做 CI 最小门禁，至少检查 `parser-service` 日志里没有 worker 被静默禁用的标记



### 3.1 渲染 Compose

```bash
docker compose --env-file backend/.env config
```

预期结果：
- Compose 渲染成功
- `core-api / llm-stream / parser-service / export-service / review-service / frontend` 都在输出中出现

### 3.2 启动多服务

```bash
docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build
```

### 3.3 检查容器健康状态

```bash
docker compose --env-file backend/.env ps
```

预期结果：
- `core-api`
- `llm-stream`
- `parser-service`
- `export-service`
- `review-service`
- `frontend`

以上服务均显示 `Up (healthy)` 或等价健康状态。

### 3.4 检查网关入口

```bash
curl -I http://localhost
curl http://localhost/api/v1/ping
```

预期结果：
- `http://localhost` 返回 `200 OK`
- `/api/v1/ping` 返回成功响应，证明 Nginx 到 `core-api` 的代理仍可用

### 3.5 检查网关路由归属

浏览器与 Vite 始终只访问统一网关。当前路由契约如下：

| 外部路径 | 上游服务 | 关键约束 |
| --- | --- | --- |
| `/api/v1/tasks/:id/stream` | `core-api` | SSE 禁用缓冲与缓存 |
| `/api/v1/stream/*` | `llm-stream` | 兼容旧流式入口 |
| `/api/v1/blogs/:id/continue` 或 `/polish` | `llm-stream` | 兼容旧编辑器流式入口 |
| `/api/v1/project/parse` | `parser-service` | 支持大文件上传与长超时 |
| `/api/v1/review/*` | `review-service` | review 独立领域入口 |
| `/api/v1/blogs/:id/export*` | `export-service` | 兼容直接导出入口 |
| 其余 `/api/*` | `core-api` | 鉴权、用户、博客、任务中心 |
| `/uploads/*` | `core-api` | 上传资源 |

路由专项测试：

```bash
cd frontend
npm test -- --run src/gatewayRouting.test.ts
```

### 3.6 Vite 微服务开发拓扑

需要前端热更新时，通过 Compose 暴露同一个 Nginx 网关，不发布五个后端服务端口：

```bash
FRONTEND_PORT=8081 \
FRONTEND_URL=http://localhost:5173 \
DOCKER_GITHUB_REDIRECT_URL=http://localhost:5173/api/v1/auth/callback/github \
docker compose --env-file backend/.env up -d --build

cd frontend
INKWORDS_GATEWAY_ORIGIN=http://localhost:8081 npm run dev
```

访问 `http://localhost:5173`。预期链路为：`Vite -> Nginx :8081 -> owning service`。

## 4. 任务中心检查

### 4.1 生成任务创建

建议通过前端 UI 登录后触发“生成博客”，或用已登录态的 Bearer Token 调用：

```bash
curl -X POST http://localhost/api/v1/tasks/generation \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "kind": "generate_single",
    "payload": {
      "title": "CI smoke title",
      "source_type": "topic",
      "topic": "Go context 入门"
    },
    "idempotency_key": "smoke-generate-single"
  }'
```

预期结果：
- 返回 `202 Accepted`
- 响应体包含 `task_id`
- 响应体包含 `stream_url`

### 4.2 任务 SSE

使用上一步返回的 `stream_url`：

```bash
curl -N \
  -H "Authorization: Bearer <your-token>" \
  http://localhost<stream_url>
```

预期结果：
- 能收到标准 SSE 事件
- 至少看到 `status` 推进或 `chunk` 事件
- 失败时能收到稳定错误信息，而不是连接被静默断开

## 5. review-service 可用性

### 5.1 推荐验证方式

- 登录后进入“知识漫游复习”工作台
- 检查是否能成功拿到推荐题卡
- 检查是否能发起一次复习会话并落下一条 turn

### 5.2 最小接口验证

如果已有 Bearer Token，可验证 review 入口仍可达：

```bash
curl -H "Authorization: Bearer <your-token>" \
  http://localhost/api/v1/review/today
```

预期结果：
- 不是网关 404
- 不是容器连接失败
- 若返回业务错误，也应是可读 JSON，而不是代理层异常

## 6. parser-service 基础通路

### 6.1 同步兼容路径

- 上传一个 `50MB` 以下的普通 Markdown / TXT 文件
- 预期仍可走同步 `/api/v1/project/parse`

### 6.2 任务式路径

- 上传一个 `.zip` 课件包，或一个 `50MB` 以上的普通文件
- 预期前端先创建 `/api/v1/tasks/parse`
- 随后订阅 `/api/v1/tasks/:id/stream`
- 最终在任务结果中拿到：
  - `source_content`
  - `archive_summary`
  - `warnings`

## 7. export-service 基础通路

### 7.1 PDF 导出任务化

- 在侧边栏选择一个系列并导出 PDF
- 预期前端先调用 `POST /api/v1/tasks/export`
- 随后等待统一任务流完成
- 最终通过 `GET /api/v1/tasks/:id/download` 下载 PDF

### 7.2 关键观察点

- `export-service` 日志中能看到消费 `export.requested`
- 导出完成后下载成功
- 下载成功后共享目录中的 PDF 被删除，不出现长期堆积

## 8. 失败排查顺序

### 8.1 服务启动失败
- 先看：

```bash
docker compose --env-file backend/.env ps
docker compose --env-file backend/.env logs --no-color
```

### 8.2 网关不可用
- 检查 `frontend` 是否 healthy
- 检查 `core-api` 是否 healthy
- 检查 `frontend/nginx.conf` 是否仍代理 `/api/` 到 `core-api`
- `404`：先核对专用 location 是否位于通用 `/api/` fallback 之前，以及路径是否仍属于对应服务
- `502/503/504`：检查 owning service 的健康状态、容器名和 `inkwords-network`；前端会统一显示“服务暂时不可用，请稍后重试”

### 8.3 Task SSE 有状态但前端迟迟不更新

- 确认 `/api/v1/tasks/:id/stream` 命中 `core-api` 专用 SSE location，而不是通用 `/api/`
- 确认该 location 包含 `proxy_buffering off`、`proxy_cache off`、`X-Accel-Buffering: no` 和长读超时
- 使用 `curl -N` 观察事件是否逐条到达；若容器日志已有事件但客户端批量收到，优先排查代理缓冲

### 8.4 任务创建成功但无推进
- 检查 `rabbitmq` 是否存活
- 检查 `core-api` 是否真的发布了任务消息
- 检查对应 worker（`llm-stream / parser-service / export-service`）日志是否有消费记录

### 8.5 review 不可用
- 检查 `review-service` 健康状态
- 检查 `REVIEW_DATABASE_URL`
- 检查 `OBSIDIAN_WIKI_DIR` 与 `OBSIDIAN_VAULT_PATH`

### 8.6 Vite 或 OAuth 回调异常

- Vite 请求 `502`：确认 `INKWORDS_GATEWAY_ORIGIN` 与 Compose 的 `FRONTEND_PORT` 指向同一端口
- OAuth 回到静态网关而不是 Vite：确认 `FRONTEND_URL=http://localhost:5173`
- GitHub callback 直接 `404`：确认 `DOCKER_GITHUB_REDIRECT_URL=http://localhost:5173/api/v1/auth/callback/github`，并确保 Vite 正在运行、`/api` 正常代理到网关
- 不要把 `core-api:8080` 或其他容器地址配置进浏览器环境变量；这些名称只在 Compose 网络内可解析

## 9. 提交前最小核对清单

- `README.md` 已同步当前启动与验证方式
- `.github/workflows/ci.yml` 已覆盖最小微服务冒烟
- `docker compose --env-file backend/.env config` 通过
- `docker compose --env-file backend/.env ps` 显示服务健康
- `curl -I http://localhost` 成功

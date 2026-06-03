# InkWords 微服务化（面向独立扩容）RFC

状态：草案（可共创迭代）

## 1. 背景与现状

InkWords 当前形态为：
- `frontend/`：React 静态站点，由 Nginx 提供静态资源并反向代理 `/api/*`
- `backend/`：Go + Gin 单体 API，内部已按领域切片组织（auth/user/blog/project/stream/review）
- `PostgreSQL`：持久化 blogs/users/review 等业务数据
- `Redis`：缓存与可选能力
- `Obsidian Local REST API + Vault`：导出与复习数据源（通过 `obsidian-bridge` 侧车转发）

现有路由边界（以 Gin 分组为准）：
- `/api/v1/auth` → `internal/domain/auth/*`
- `/api/v1/user` → `internal/domain/user/*`
- `/api/v1/blogs` → `internal/domain/blog/*`（导出包含 Markdown Zip / PDF / Obsidian）
- `/api/v1/project` → `internal/domain/project/*`（scan/analyze/parse）
- `/api/v1/stream` → `internal/domain/stream/*`（analyze/generate/scan，SSE）
- `/api/v1/review` → `internal/domain/review/*`

其中可独立扩容的“重计算/重资源”路径：
- LLM 流式分析/生成：`/api/v1/stream/*` 与 `/api/v1/blogs/:id/continue|polish`（SSE + 高并发受限）
- 文件解析：`/api/v1/project/parse`（大文件上传 + PDF 解析与回退工具链）
- PDF 导出：`/api/v1/blogs/:id/export/pdf`（Chromium headless）
- Obsidian 导出：`/api/v1/blogs/:id/export/obsidian*`（并发抽取 entities/concepts + 写入 Vault）

参考：
- `.trae/documents/InkWords_Architecture.md`
- `.trae/documents/InkWords_API.md`
- `.trae/documents/InkWords_Database.md`
- `backend/internal/transport/http/v1/routes.go`
- `backend/cmd/server/main.go`

## 2. 目标与非目标

### 2.1 目标

- 支持对“重计算能力”独立扩容与独立发布，避免后端整体被 LLM/PDF/解析拖慢
- 保持前端使用单一入口域名（`/api/*`）不变，尽可能降低客户端改动
- 保持 SSE 体验与稳定性不下降
- 基于 Docker Compose 落地“微服务形态”，并支持按服务独立扩容
- 渐进式迁移：每一步都可回滚，且单体形态仍可在本地保留（Docker Compose）

### 2.2 非目标（Phase 1 不做）

- 不追求一次性拆成“严格数据库隔离”的终极微服务
- 不强制引入消息队列/事件总线/服务网格（除非后续证明必要）
- 不改变现有对外 API 的语义与 URL（仅在网关层做路由分发）

## 3. 关键约束

- 前端流式请求约束：必须使用 `@microsoft/fetch-event-source` 发起 POST（已有约束）
- 网关必须对 SSE 禁用缓冲、拉长超时、支持长连接（现有 Nginx 已做）
- 后端 LLM 调用并发必须受信号量控制，避免限流/内存溢出（现有约束）
- Vault 挂载：Docker 环境必须显式绑定 Vault 目录，并通过环境变量配置 wiki 根目录

## 4. 目标架构（渐进式）

### 4.1 Phase 0（现状）

- `frontend` + `gateway(Nginx)` + `backend-monolith` + `postgres` + `redis` + `obsidian-bridge`

### 4.2 Phase 1（最小微服务化：先解决独立扩容）

拆出 1 个强独立扩容的服务：

1) `inkwords-core-api`
- 负责：auth/user/blog/review + 业务写入与查询
- 仍保留：export(Zip/PDF/Obsidian) 的 HTTP 入口（Phase 1 可不拆）

2) `inkwords-llm-stream`
- 负责：`/api/v1/stream/*` + `/api/v1/blogs/:id/continue|polish`（保持原有 URL，对外由网关分流）
- 负责：LLM 调用、Map-Reduce 分析、SSE 写出
- 访问：PostgreSQL（写 blogs/status/tokens_used 等）、Redis（如需）

网关（Nginx）：
- `/api/v1/stream/*`、`/api/v1/blogs/*/continue`、`/api/v1/blogs/*/polish` → `inkwords-llm-stream`
- 其余 `/api/*` → `inkwords-core-api`

### 4.3 Phase 2（进一步拆重资源：按需）

按实际瓶颈决定是否继续拆：

3) `inkwords-ingest`
- 负责：`/api/v1/project/parse`（大文件上传 + 文档/ZIP 解析）
- 可选：`/api/v1/stream/scan` 的 Git 快扫（如果实际 CPU/IO 压力显著）

4) `inkwords-pdf-renderer`
- 负责：`/api/v1/blogs/:id/export/pdf`（Chromium headless）
- 建议：改为异步任务 + 对象存储（后续再做）

5) `inkwords-obsidian-sync`
- 负责：`/api/v1/blogs/:id/export/obsidian*`
- 配合：Vault bind mount + 对外限流/重试策略

## 5. 服务边界与代码拆分策略（Monorepo 内）

仓库仍保持 Monorepo：
- `frontend/` 不变
- `backend/` 内拆成多个可独立编译/镜像的 Go 程序（多个 `cmd/*` 入口）

建议的后端目录演进：
- `backend/cmd/core-api/`：core-api main
- `backend/cmd/llm-stream/`：llm-stream main
- `backend/internal/domain/*`：尽量复用现有领域切片
- `backend/internal/transport/http/v1/`：按服务拆分路由注册（避免 core 引入 stream 路由）

Phase 1 的最小改动原则：
- 优先通过“复制/裁剪路由注册 + 复用 domain handler/service”实现拆进程
- 暂不做大规模包迁移；仅在编译边界上把 `core` 与 `stream` 依赖分开

## 6. 身份鉴权与内部信任模型

对外：
- 继续使用 JWT Bearer Token
- 网关只做转发，不解析业务

服务内：
- Phase 1：`core-api` 与 `llm-stream` 共享 JWT 校验配置（同一个 `JWT_SECRET` 或同一套公私钥）
- 后续：如需要更强隔离，可将 JWT 切为非对称签名（RS256），各服务仅持有公钥

## 7. 数据与一致性策略

Phase 1（共享数据库，渐进式）：
- `core-api` 与 `llm-stream` 共享 PostgreSQL 与相同 schema
- 数据写入边界保持“现状优先”：生成服务仍可直接写 `blogs` 与 `users.tokens_used`（减少改动）

Phase 2+（演进方向，非强制）：
- 通过明确“表/写路径归属”逐步消除跨服务写入
- 若要拆库：以 `blogs` 为核心边界，优先把生成结果写入改为 `llm-stream -> core-api(内部写入 API)` 或事件化（outbox）

## 8. SSE 与网关/Ingress 要求（Kubernetes）

SSE 关键点：
- 禁用 proxy buffering
- 超时放宽（至少 1 小时级别）
- 保持 HTTP/1.1 与连接复用策略可控

## 9. Vault 挂载与 Obsidian 集成（Docker）

约定：
- Vault 目录以 bind mount 挂载进需要访问 Vault 的服务（通常为 `core-api` 与 `obsidian-sync`，以及 `review` 读取侧）
- Vault 的 `wiki/` 根目录通过环境变量显式配置（例如 `OBSIDIAN_WIKI_DIR`），不允许隐式默认到机器私有路径

风险：
- Vault 是有状态共享目录，需明确并发写入策略与写入幂等规则（尤其是批量 Ingest）

## 10. Docker Compose 交付物（Phase 1 最小集）

- `docker-compose.yml`：新增 `core-api` 与 `llm-stream` 两个服务，并允许按服务独立扩容
- `frontend/nginx.conf`：按路径把流式路由分流到 `llm-stream`，其余路由分流到 `core-api`
- `backend`：新增两个 `cmd` 入口，分别编译为不同二进制

## 11. 迁移步骤（可回滚）

### Step A：准备多入口编译
- 新增 `cmd/core-api` 与 `cmd/llm-stream`
- 拆分路由注册：core-api 仅注册非 stream 路由，llm-stream 仅注册 stream/continue/polish 路由
- 两个服务都加载同一套配置与依赖（DB/Redis/JWT）

### Step B：本地 Compose 验证（保持一键启动）
- Compose 中新增两个后端服务，网关按路径分流
- 验证：登录、列表、生成（SSE）、继续生成、润色

## 12. 验收标准（Phase 1）

- 外部 URL 不变，前端无需改 base URL
- `/api/v1/stream/*` SSE 可稳定工作，且断连/取消语义仍可向下传播
- `llm-stream` 可独立扩容，且扩容不影响 core-api 的 p95 延迟
- 日志可区分服务维度；至少具备基础的 request_id 贯通

## 13. 开放问题（下一轮共创）

- 是否把 `/api/v1/project/parse` 也纳入 Phase 1 拆分（依据：上传/解析对 core-api 的资源挤压程度）
- PDF 导出是否需要尽早拆出（Chromium 的资源隔离策略）
- Obsidian 批量导出在 Vault PVC 场景下的并发写入幂等策略（避免重复写/冲突）

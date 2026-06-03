# Task 7：`export_pdf` 异步化设计

## 1. 背景与目标

InkWords 当前已经完成：

- `core-api / llm-stream / parser-service / export-service / review-service` 的多服务拆分
- 生成链路任务化：`generation` 已接入 `job_tasks + RabbitMQ + SSE`
- 解析链路任务化起步：`parse` 已支持 `.zip` 与大文件异步解析

但 `export-service` 仍保留同步导出模型：

- `GET /api/v1/blogs/:id/export/pdf` 会在请求生命周期内完成 Chromium 渲染并直接返回 PDF
- `POST /api/v1/blogs/:id/export/obsidian` 与 `POST /api/v1/blogs/:id/export/obsidian/series` 仍是同步写 Vault

这会带来两个问题：

- PDF 渲染会长时间占用 HTTP 请求与 Chromium 资源，峰值时容易拖慢 `export-service`
- 导出链路还没有复用统一任务中心，状态查询、取消、重试与失败观测都弱于生成/解析链路

本设计只聚焦 `Task 7` 的第一步：先把最重的 `export_pdf` 异步化。

### 目标（DoD）

- 新增任务式导出入口，支持创建 `export_pdf` 任务
- `export-service` 通过 RabbitMQ 消费 `export_pdf` 任务并生成 PDF
- 前端导出 PDF 时改为“创建任务 -> 订阅任务 SSE -> 下载结果文件”
- 对外仍保持 `http://localhost` 与 `/api/*` 单入口不变
- 同步 PDF 导出入口继续保留，作为灰度与回滚路径

### 非目标（Not in scope）

- 本轮不实现 `export_obsidian_single`
- 本轮不实现 `export_obsidian_series`
- 本轮不实现 `export_markdown_zip` 任务化
- 本轮不同时改造所有导出按钮与批量导出模式
- 本轮不引入对象存储（S3 / MinIO）作为文件结果存储

## 2. 方案选择

### 方案 A：轻量双入口方案（推荐）

- 保留现有同步 PDF 导出接口
- 新增任务式导出入口 `POST /api/v1/tasks/export`
- 仅支持 `kind=export_pdf`
- 任务成功后不把 PDF 二进制写入数据库，而是返回下载元数据，再通过单独下载接口取文件

优点：

- 最小改动，可灰度、可回滚
- 完全复用现有任务中心与 SSE 模型
- 不改变现有同步导出调用方语义

缺点：

- 同步/异步两套 PDF 导出入口会并存一段时间

### 方案 B：透明替换同步 PDF 导出入口

- 保持当前 PDF 导出接口路径不变
- 但接口不再直接返回 PDF，而是返回任务句柄

优点：

- 表面上入口更少

缺点：

- 会直接破坏既有前端和历史调用方的响应语义
- 回归风险高，不符合当前阶段“最小改动 + 可回滚”的原则

### 方案 C：导出服务内自建异步模型

- 不走统一任务中心
- 在 `export-service` 内部单独维护导出队列或 goroutine 模型

优点：

- 可以只改一个服务

缺点：

- 会重复造轮子
- 无法和现有 `job_tasks + SSE` 统一
- 长期维护成本更高

### 选择结论

采用 **方案 A：轻量双入口方案**。

Why：

- 这与 `generation`、`parse` 的任务化路径完全一致，最容易复用已有代码和心智
- `export_pdf` 的主要难点是“后台生成后如何下载文件”，而不是任务创建本身；双入口可以把这个问题收敛为一个清晰的下载语义设计
- 保留同步入口后，即使任务式导出出现回归，也可以快速回切前端

## 3. 范围与边界

### 3.1 本轮纳入

- `export_pdf` 任务 subtype
- 导出任务创建接口
- `export-service` PDF worker consumer
- 任务结果下载接口
- 前端单个 PDF 导出按钮的任务化接入

### 3.2 本轮不纳入

- Obsidian 导出异步化
- ZIP Markdown 导出异步化
- 批量 PDF 导出任务化
- 文件长期持久化存储
- 跨服务下载 CDN / 对象存储代理

## 4. API 与任务模型设计

## 4.1 任务 subtype

新增导出任务 subtype：

- `export_pdf`

`task_type` 统一使用既有模型中的：

- `export`

## 4.2 创建任务接口

新增接口：

`POST /api/v1/tasks/export`

请求头：

- `Authorization: Bearer <token>`

请求体最小结构：

```json
{
  "kind": "export_pdf",
  "payload": {
    "blog_id": "123e4567-e89b-12d3-a456-426614174000"
  },
  "idempotency_key": "export-pdf:123e4567-e89b-12d3-a456-426614174000"
}
```

成功响应：

```json
{
  "task_id": "223e4567-e89b-12d3-a456-426614174000",
  "status": "queued",
  "stream_url": "/api/v1/tasks/223e4567-e89b-12d3-a456-426614174000/stream"
}
```

约束：

- `kind` 当前仅允许 `export_pdf`
- `payload.blog_id` 必填，表示系列父节点 blog ID
- `idempotency_key` 建议由前端按 `export-pdf:<blog_id>` 生成，避免用户连续点击产生重复任务

## 4.3 任务结果模型

`result_json` 不保存 PDF 文件内容，只保存下载元数据：

```json
{
  "file_token": "exp_pdf_xxx",
  "filename": "系列标题.pdf",
  "content_type": "application/pdf",
  "expires_at": "2026-06-03T23:59:59Z"
}
```

设计原则：

- 文件结果与任务状态解耦
- 数据库只保存任务结果的索引信息，不保存大二进制
- 任务成功后，前端通过下载接口再获取 PDF 文件

## 4.4 下载接口

新增接口：

`GET /api/v1/tasks/:id/download`

语义：

- 只允许下载当前用户自己创建且已成功完成的导出任务结果
- 当前仅支持 `export_pdf`

成功响应：

- `200`
- `Content-Type: application/pdf`
- `Content-Disposition: attachment; filename="<filename>"`

失败响应：

- `404`：任务不存在、无权限、结果文件已过期或不存在
- `409`：任务尚未完成
- `400`：任务类型不支持下载

Why：

- 让下载授权与任务归属统一收口到 `core-api`
- 避免前端直接访问 `export-service` 内部文件路径或宿主机路径

## 5. 后端架构设计

## 5.1 `core-api`

职责：

- 校验导出任务请求
- 创建 `task_type=export`、`task_subtype=export_pdf` 的任务
- 发布 `export.requested` RabbitMQ 消息
- 查询任务状态与 SSE
- 提供下载接口并做用户归属校验

建议改动：

- 任务领域新增 `CreateExportTaskInput`
- 任务 handler 新增 `CreateExportTask`
- 路由新增 `POST /api/v1/tasks/export`
- 路由新增 `GET /api/v1/tasks/:id/download`

## 5.2 `export-service`

职责：

- 消费 `export.requested`
- 对 `export_pdf` 执行 PDF 导出
- 生成临时文件
- 回写 `result_json`

建议改动：

- `backend/cmd/export-service/main.go`
  - 新增 RabbitMQ consumer 启动逻辑
- `backend/internal/domain/task/export_consumer.go`
  - 新增导出任务 consumer
- `backend/internal/domain/task/export_task.go`
  - 新增导出任务 DTO / 发布契约

## 5.3 复用现有 PDF 生成逻辑

本轮不重写 PDF 核心实现，应直接复用现有同步导出服务中的：

- 系列博客查询
- HTML 渲染
- Chromium 打印
- 文件名生成

Why：

- 当前目标是异步化，不是重新设计 PDF 渲染引擎
- 只有这样才能把改动限定在任务入口、worker 与下载语义层

## 6. 文件生命周期与下载语义

## 6.1 临时文件存储

worker 生成的 PDF 文件存放在受控临时目录，例如：

- `os.CreateTemp("", "inkwords-export-*.pdf")`

任务成功后：

- 记录临时文件路径到进程内下载仓储或受控文件映射
- `result_json` 只记录 `file_token` 与元数据

说明：

- `result_json` 不直接暴露服务器真实文件路径
- 下载时由 `core-api` 通过 `file_token` 解析到真实文件

## 6.2 下载后的清理策略

本轮采用简单可控策略：

- 下载成功后立即删除 PDF 临时文件
- 同时删除 `file_token -> file_path` 的映射
- 若用户未下载，则按 TTL 过期

建议 TTL：

- `15 分钟`

Why：

- 足够覆盖“任务完成后立即下载”的主流程
- 不需要额外引入清理守护进程的复杂度

## 6.3 过期与重复下载

- 首次下载成功后，文件立即清理，后续再次下载返回 `404`
- 若任务已成功但结果已过期，也返回 `404`

这是刻意选择的最小方案。

非目标：

- 本轮不支持“多次重复下载同一个 PDF 结果”
- 如后续需要多次下载，再升级为对象存储或持久化下载缓存

## 7. 幂等、取消与重试

## 7.1 幂等

`export_pdf` 的幂等语义采用：

- 同一用户 + 同一 `blog_id` + 同一 `idempotency_key`
- 若存在未完成任务，则复用已有任务

不做内容级哈希幂等。

Why：

- PDF 导出允许重复生成覆盖同一输出
- 当前最主要目的是避免“用户连续点击按钮造成重复任务”

## 7.2 取消

- 任务创建后可沿用既有 `POST /api/v1/tasks/:id/cancel`
- worker 在关键阶段检查任务取消状态

取消语义：

- 若还未开始 Chromium 渲染，直接停止
- 若 PDF 已经生成但还未写回成功状态，优先清理临时文件并标记 `cancelled`
- 若已成功完成，则视为不可取消

## 7.3 重试

失败后允许用户重新发起任务。

本轮不做：

- 自动重试队列
- 死信队列回放界面

## 8. 前端交互设计

## 8.1 导出 PDF 按钮

本轮只改“导出 PDF”主入口，不改其它导出按钮。

前端流程：

1. 用户点击“导出 PDF”
2. 前端调用 `POST /api/v1/tasks/export`
3. 前端订阅 `stream_url`
4. 任务完成后，前端调用 `GET /api/v1/tasks/:id/download`
5. 浏览器触发 PDF 下载

## 8.2 中文交互文案

建议文案：

- 创建任务中：`正在创建 PDF 导出任务...`
- 执行中：`正在生成 PDF，请稍候...`
- 成功：`PDF 已生成，开始下载`
- 失败：`PDF 导出失败，请稍后重试`

## 8.3 回滚策略

前端保留同步 PDF 导出调用能力。

若任务式导出出现问题：

- 可通过前端开关或代码回切恢复同步导出
- 不需要回滚后端全部导出逻辑

## 9. 错误处理设计

后端：

- 无效 `blog_id`：返回 `400`
- 系列不存在或无权限：返回 `404`
- Chromium 渲染失败：任务标记 `failed`，写稳定错误信息
- 下载时任务未成功：返回 `409`
- 下载时结果文件丢失或过期：返回 `404`

前端：

- 创建任务失败：toast 提示中文错误
- SSE 中收到 `error`：显示中文错误
- 下载失败：提示“PDF 已生成但下载失败，请重试”

## 10. 测试与验证计划

## 10.1 后端测试

按 TDD 顺序至少覆盖：

- `CreateExportTask` 成功发布 `export.requested`
- `CreateExportTask` 发布失败时返回错误
- `export_consumer` 成功后写入 `result_json`
- `export_consumer` 失败时写入 `failed`
- `GET /api/v1/tasks/:id/download` 的成功、未完成、无权限、过期场景

## 10.2 前端测试

- 导出 PDF 时先创建任务，再订阅 SSE
- 任务成功后调用下载接口
- 任务失败时展示中文错误

## 10.3 Docker 验证

必须执行：

```bash
docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build
docker compose --env-file backend/.env ps
curl -I http://localhost
```

## 10.4 人工冒烟

至少验证：

- 单个系列导出 PDF 成功
- 下载得到 `.pdf` 文件
- 任务失败时前端能感知
- 用户取消任务时不会遗留临时文件

## 11. 风险与后续

### 当前风险

- `core-api` 下载接口需要持有结果文件映射，若进程重启，尚未下载的结果文件会失效
- 单机临时文件方案不适合未来多副本部署

### 当前接受的原因

- 本轮目标是用最小成本验证 `export_pdf` 任务化
- 当前 Docker Compose 本地与单机部署形态可以接受这一限制

### 后续扩展方向

- 将结果文件迁移到对象存储
- 扩展 `export_obsidian_series`，并引入更严格的幂等语义
- 让批量 PDF 导出也接入统一任务中心

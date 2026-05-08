# 后端 DDD 垂直切片渐进迁移（Stream + Project Domain）设计文档

**类型**：Technical Design / Refactor Spec  
**范围**：后端目录升级（不改变对外 API 行为，不改 DB schema）  
**迁移领域**：Stream（SSE） + Project（scan/analyze/parse）  
**迁移深度**：Phase 1（domain 落地边界，内部复用现有 service/parser 作为过渡依赖）  

## 1. 背景与动机

目前后端的流式能力与项目解析能力主要集中在：
- `internal/api/stream_*.go`（SSE handler + quota 校验 + request binding）
- `internal/api/project.go`（非流式 scan/analyze/parse）
- `internal/service/generator.go`、`internal/service/decomposition_*.go`（大模型生成与 Map-Reduce 分析）
- `internal/parser/*`（Git 拉取与文件解析）

在完成 `internal/domain/{blog,user,auth}` 后，需要继续将 Stream 与 Project 的“对外接口层”迁移到对应领域切片，以统一后端目录结构与依赖组装方式，并为后续的深拆与测试打基础。

## 2. 目标与非目标

### 2.1 目标
- 新增 `internal/domain/stream` 与 `internal/domain/project`
- 将 `internal/api/stream_*.go` 与 `internal/api/project.go` 变为薄适配层：保留原路由绑定/方法名，内部转发到 domain handler
- `cmd/server/main.go` 统一完成依赖组装（repo/service/handler -> api）
- 对外行为不变：
  - SSE 事件类型（chunk/progress/result/done/ping/error）与输出格式不变
  - quota 校验、未授权行为与状态码保持不变
  - `project/analyze` 的 `source_content` 拼接逻辑保持不变

### 2.2 非目标（Phase 1 不做）
- 不将 `GeneratorService` / `DecompositionService` 深度拆解搬迁到 domain（仅作为依赖注入到 domain service）
- 不改变现有并发、超时、背景上下文 `context.WithoutCancel` 等行为
- 不统一 response 结构（现存 stream API 对错误返回 `{error: ...}` 与其他模块不一致，Phase 1 保持不动）

## 3. 总体方案（Phase 1）

### 3.1 Stream Domain

- `internal/domain/stream` 提供：
  - `Handler`：实现原 `StreamAPI` 的 5 个 endpoint handler（generate/continue/polish/analyze/scan）
  - `Service`：仅作为“编排层”，封装对 `GeneratorService`/`DecompositionService` 的调用（不改其内部实现）
  - `DTO`：迁移 `GenerateRequest` / `PolishRequest`（保持字段一致）

### 3.2 Project Domain

- `internal/domain/project` 提供：
  - `Handler`：实现 `ScanGithubRepo` / `Analyze` / `Parse`（与原行为一致）
  - `Service`：编排 `GitFetcher` / `DocParser` / `DecompositionService`（继续复用现有实现）
  - `DTO`：`ScanRequest` / `AnalyzeRequest`

### 3.3 依赖组装与薄适配

- `internal/api/stream_api.go` 与 `internal/api/project.go`：
  - 增加 `*domain.Handler` 字段
  - 方法体变为单行转发（如 `a.streamDomainHandler.GenerateBlogStreamHandler(c)`）
- `cmd/server/main.go`：
  - 统一 new 并注入 `GeneratorService`、`DecompositionService`、`GitFetcher`、`DocParser`
  - `NewStreamAPIWithDeps(...)` / `NewProjectAPIWithDeps(...)` 注入 domain handler

## 4. 验证与验收

- `cd backend && go test ./...` 通过
- 与原接口行为一致（重点检查）：
  - `/api/v1/stream/generate`、`/api/v1/stream/analyze`、`/api/v1/stream/scan` 的 SSE event 类型与 payload 格式不变
  - `/api/v1/project/scan` 的 `code` 返回仍为 `0`
  - `/api/v1/project/analyze` 的 `source_content` 拼接逻辑不变


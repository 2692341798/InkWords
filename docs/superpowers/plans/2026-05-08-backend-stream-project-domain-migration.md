# 后端 Stream + Project Domain 垂直切片迁移 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Stream（SSE）与 Project（scan/analyze/parse）相关接口迁移到 `internal/domain/stream` 与 `internal/domain/project`，并将 `internal/api` 对应文件薄化为转发层；保持对外行为不变。

**Architecture:** Phase 1 只迁移 handler 编排边界；内部继续复用 `service.GeneratorService`、`service.DecompositionService` 与 `parser` 组件作为依赖。依赖组装收口到 `cmd/server/main.go`。

**Tech Stack:** Go + Gin + SSE + GORM + LLM + MapReduce + Git parser

---

## Part A：Stream Domain

### Task A1: 新增 domain/stream（dto/service/handler）

**Files:**
- Create: `backend/internal/domain/stream/dto.go`
- Create: `backend/internal/domain/stream/service.go`
- Create: `backend/internal/domain/stream/handler.go`

- [ ] **Step A1.1: dto 迁移**
- [ ] **Step A1.2: service 编排层（包装 GeneratorService/DecompositionService）**
- [ ] **Step A1.3: handler 迁移 stream_generate/continue/polish/analyze/scan 的逻辑**
- [ ] **Step A1.4: 回归**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

### Task A2: 薄化 internal/api/stream_*.go

**Files:**
- Modify: `backend/internal/api/stream_api.go`
- Modify: `backend/internal/api/stream_generate.go`
- Modify: `backend/internal/api/stream_continue.go`
- Modify: `backend/internal/api/stream_polish.go`
- Modify: `backend/internal/api/stream_analyze.go`
- Modify: `backend/internal/api/stream_scan.go`
- Modify: `backend/internal/api/stream_types.go`（如需迁移/删减）

- [ ] **Step A2.1: StreamAPI 增加 streamDomainHandler 字段**
- [ ] **Step A2.2: NewStreamAPIWithDeps 注入 domain handler；保留旧 NewStreamAPI 兼容或改为调用 WithDeps**
- [ ] **Step A2.3: 每个 handler 文件改为单行转发**
- [ ] **Step A2.4: 回归**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

---

## Part B：Project Domain

### Task B1: 新增 domain/project（dto/service/handler）

**Files:**
- Create: `backend/internal/domain/project/dto.go`
- Create: `backend/internal/domain/project/service.go`
- Create: `backend/internal/domain/project/handler.go`

- [ ] **Step B1.1: dto：ScanRequest / AnalyzeRequest**
- [ ] **Step B1.2: service：编排 GitFetcher/DocParser/DecompositionService**
- [ ] **Step B1.3: handler：迁移 internal/api/project.go 的 3 个 endpoint**
- [ ] **Step B1.4: 回归**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

### Task B2: 薄化 internal/api/project.go

**Files:**
- Modify: `backend/internal/api/project.go`

- [ ] **Step B2.1: ProjectAPI 增加 projectDomainHandler 字段**
- [ ] **Step B2.2: NewProjectAPIWithDeps 注入 domain handler**
- [ ] **Step B2.3: ScanGithubRepo/Analyze/Parse 改为单行转发**
- [ ] **Step B2.4: 回归**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

---

## Part C：DI 收口与文档同步

### Task C1: cmd/server/main.go 统一组装 stream/project 依赖

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step C1.1: 统一创建 GeneratorService/DecompositionService/GitFetcher/DocParser**
- [ ] **Step C1.2: 注入 StreamAPIWithDeps / ProjectAPIWithDeps**
- [ ] **Step C1.3: 回归**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

### Task C2: 文档同步

**Files:**
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_PRD.md`

- [ ] **Step C2.1: 追加 Stream/Project Domain 迁移记录（声明无 API/DB 行为变更）**

---

## Part D：最终验证与提交推送

- [ ] **Step D1: go test**

Run:
```bash
cd backend && go test ./...
```

- [ ] **Step D2: commit & push**


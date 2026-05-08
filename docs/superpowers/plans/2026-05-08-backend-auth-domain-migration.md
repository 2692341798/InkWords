# 后端 Auth Domain 垂直切片迁移 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Auth 相关逻辑迁移到 `backend/internal/domain/auth`（repo/service/handler/dto），并将 `internal/api/auth.go` 薄化为转发层；保持路由与对外行为不变。

**Architecture:** 先盘点路由与现有实现，再落地 domain/auth；迁移完成后在 `cmd/server/main.go` 统一组装依赖；每个里程碑执行 `go test ./...` 回归。

**Tech Stack:** Go + Gin + GORM + OAuth2 + JWT + base64Captcha

---

### Task 1: 现状盘点

- [ ] **Step 1.1: 定位 auth 路由注册与 handler**

Run:
```bash
cd backend
rg -n \"authGroup\\.(GET|POST)\\(\" cmd/server/main.go
rg -n \"type AuthAPI|func \\(a \\*AuthAPI\\)\" internal/api/auth.go
```

- [ ] **Step 1.2: 定位 auth service 入口与错误类型**

Run:
```bash
cd backend
rg -n \"type AuthService|ErrEmailExistsBindRequired|GetAuthURL\\(|HandleCallback\\(|Register\\(|Login\\(|BindGithub\\(|GenerateCaptcha\\(\" internal/service/auth.go
```

---

### Task 2: 新增 domain/auth（repo/service/handler/dto）

**Files:**
- Create: `backend/internal/domain/auth/dto.go`
- Create: `backend/internal/domain/auth/repository.go`
- Create: `backend/internal/domain/auth/service.go`
- Create: `backend/internal/domain/auth/handler.go`

- [ ] **Step 2.1: dto**

包含：
- `RegisterRequest`、`LoginRequest`、`BindGithubRequest`

- [ ] **Step 2.2: repository 接口 + GORM 实现**

包含：
- `GetUserByEmail` / `GetUserByGithubIDOrEmail` / `CreateUser` / `SaveUser`

- [ ] **Step 2.3: service 迁移**

将 `internal/service/auth.go` 的核心逻辑迁移到 `domain/auth/service.go`（保持行为与错误消息不变）：
- OAuth config + `GetAuthURL` / `HandleCallback`
- Captcha generate/verify
- Register/Login/BindGithub

- [ ] **Step 2.4: handler 迁移**

将 `internal/api/auth.go` 的 HTTP 适配迁移到 `domain/auth/handler.go`：
- `OAuthRedirect` / `OAuthCallback`
- `Register` / `Login` / `BindGithub` / `GetCaptcha`

- [ ] **Step 2.5: 编译回归**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

---

### Task 3: API 薄化（auth.go 转发到 domain handler）

**Files:**
- Modify: `backend/internal/api/auth.go`

- [ ] **Step 3.1: AuthAPI 增加 authDomainHandler 字段**
- [ ] **Step 3.2: 5 个路由方法全部改为转发**
- [ ] **Step 3.3: 删除无用 imports 与对 internal/service 错误类型的依赖**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

---

### Task 4: DI 收口（cmd/server/main.go 组装 auth domain 并注入）

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/api/auth.go`（如需 NewAuthAPIWithDeps）

- [ ] **Step 4.1: main.go 组装 authRepo/authService/authHandler**
- [ ] **Step 4.2: 通过 NewAuthAPIWithDeps 注入 AuthAPI**
- [ ] **Step 4.3: 回归**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

---

### Task 5: 移除旧实现与文档同步

**Files:**
- Delete: `backend/internal/service/auth.go`（确认无引用后）
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_PRD.md`

- [ ] **Step 5.1: 删除 internal/service/auth.go（如 grep 确认仅剩 domain 使用）**
- [ ] **Step 5.2: 文档追加 Auth Domain 迁移记录（无 API/DB 行为变更）**

---

### Task 6: 最终验证

- [ ] **Step 6.1: go test**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS


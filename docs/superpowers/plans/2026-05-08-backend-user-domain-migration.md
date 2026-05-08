# 后端 User Domain 垂直切片迁移 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 User 相关逻辑迁移到 `backend/internal/domain/user`（repo/service/handler/dto），并将 `internal/api/user.go` 薄化为转发层；保持路由与对外 JSON 结构不变。

**Architecture:** 采用渐进式迁移；每个 endpoint 迁移后跑 `go test ./...`；依赖由 `cmd/server/main.go` 统一组装注入。

**Tech Stack:** Go + Gin + GORM + PostgreSQL

---

### Task 1: 现状盘点

- [ ] **Step 1.1: 定位 user 路由注册与处理器**

Run:
```bash
cd backend
rg -n \"userGroup\\.(GET|PUT|POST)\\(\" cmd/server/main.go
rg -n \"type UserAPI|func \\(a \\*UserAPI\\)\" internal/api/user.go
```

- [ ] **Step 1.2: 定位 user service 的 DB 访问点**

Run:
```bash
cd backend
rg -n \"type UserService|GetUserByID|UpdateUsername|UpdateAvatarURL|GetUserStats\" internal/service/user.go
```

---

### Task 2: 新增 domain/user 骨架与最小单测

**Files:**
- Create: `backend/internal/domain/user/dto.go`
- Create: `backend/internal/domain/user/repository.go`
- Create: `backend/internal/domain/user/service.go`
- Create: `backend/internal/domain/user/handler.go`
- Test: `backend/internal/domain/user/handler_test.go`

- [ ] **Step 2.1: dto 定义（保持对外 JSON 一致）**

- [ ] **Step 2.2: repository 接口 + GORM 实现（基于 gorm.DB）**

- [ ] **Step 2.3: service 封装业务逻辑（默认 token_limit、平台连接判断、统计聚合）**

- [ ] **Step 2.4: handler 实现 GetProfile / UpdateProfile / UploadAvatar / GetUserStats（保持 code/message/data 结构）**

- [ ] **Step 2.5: handler 单测（未授权 + profile 正常路径）**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

---

### Task 3: API 薄化（user.go 转发到 domain handler）

**Files:**
- Modify: `backend/internal/api/user.go`

- [ ] **Step 3.1: 将 4 个方法替换为 domain handler 调用**
- [ ] **Step 3.2: 删除 user.go 中不再需要的 imports / types**
- [ ] **Step 3.3: 回归**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

---

### Task 4: 依赖组装收口到 cmd/server/main.go

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/api/user.go`（如需要增加 WithDeps 构造器）

- [ ] **Step 4.1: main.go 组装 user repo/service/handler 并注入 UserAPI**
- [ ] **Step 4.2: 回归**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

---

### Task 5: 文档同步（最小增量）

**Files:**
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`

- [ ] **Step 5.1: 追加 User Domain 迁移记录**

---

### Task 6: 最终验证

- [ ] **Step 6.1: go test**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS


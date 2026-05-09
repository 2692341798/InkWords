# 后端 Blog Domain 垂直切片迁移 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不改变对外 API 行为的前提下，将 Blog 相关逻辑从水平分层逐步迁移到 `backend/internal/domain/blog` 垂直切片结构，并补齐可测试的 DI 边界（handler/service/repository）。

**Architecture:** 新增 `domain/blog`（repo -> service -> handler），`internal/api` 暂保留为薄适配层，仅负责路由注册与转发；逐路由迁移并回归测试，确保可回滚。

**Tech Stack:** Go + Gin + GORM + PostgreSQL

---

## 0. 现状盘点（先读再改）

**目标：** 明确当前 Blog 路由分布、BlogAPI 调用链、DB 访问位置，避免迁移中遗漏功能（导出 ZIP/PDF、Obsidian 同步等）。

- [ ] **Step 0.1: 定位 Blog 路由注册位置**

Run:
```bash
cd backend
rg -n \"NewBlogAPI\\(|ExportSeries|ExportToObsidian|/api/v1/blog\" -S .
```
Expected: 找到 router 注册文件与 BlogAPI 的使用点。

- [ ] **Step 0.2: 列出 blog 相关 api 文件**

Run:
```bash
cd backend
ls internal/api | rg '^blog_.*\\.go$' -n
```

---

## 1. 新增 domain/blog 骨架（不接线）

**Files:**
- Create: `backend/internal/domain/blog/repository.go`
- Create: `backend/internal/domain/blog/service.go`
- Create: `backend/internal/domain/blog/handler.go`
- Create: `backend/internal/domain/blog/dto.go`

- [ ] **Step 1.1: 创建 repository 接口与 GORM 实现骨架**

Create `backend/internal/domain/blog/repository.go`:
```go
package blog

import (
	"context"

	"github.com/google/uuid"

	"inkwords-backend/internal/model"
)

type Repository interface {
	GetByID(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) (*model.Blog, error)
	ListByUser(ctx context.Context, userID uuid.UUID, page int, size int) ([]model.Blog, error)
	CreateDraft(ctx context.Context, userID uuid.UUID) (*model.Blog, error)
	Update(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, updates map[string]any) error
	BatchDelete(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error
}
```

Then add GORM impl stub in same file:
```go
type GormRepository struct{}

func NewGormRepository() *GormRepository { return &GormRepository{} }
```

- [ ] **Step 1.2: 创建 service 骨架**

Create `backend/internal/domain/blog/service.go`:
```go
package blog

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}
```

- [ ] **Step 1.3: 创建 handler 骨架（Gin 适配）**

Create `backend/internal/domain/blog/handler.go`:
```go
package blog

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}
```

- [ ] **Step 1.4: 创建 dto（blog 域自用请求体）**

Create `backend/internal/domain/blog/dto.go`:
```go
package blog

import "github.com/google/uuid"

type BatchDeleteRequest struct {
	BlogIDs []uuid.UUID `json:"blog_ids" binding:"required"`
}
```

- [ ] **Step 1.5: 编译验证**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS（此时仅新增骨架，不影响行为）

---

## 2. 迁移路由 1：GET /api/v1/blogs（列表）

**Files:**
- Modify: `backend/internal/api/blog_list.go`
- Modify: `backend/internal/domain/blog/repository.go`
- Modify: `backend/internal/domain/blog/service.go`
- Modify: `backend/internal/domain/blog/handler.go`
- Test: `backend/internal/domain/blog/handler_test.go`

- [ ] **Step 2.1: 在 repo 实现 ListByUser（复用现有 GORM 查询逻辑）**

实现策略：
- 使用现有 `internal/db` 的 DB 句柄（Phase 1 可直接引用 `db.DB`，后续再进一步 DI）
- 必须通过 GORM 参数化查询，避免字符串拼接

- [ ] **Step 2.2: service 实现 ListByUser 并保持与原返回结构一致**

- [ ] **Step 2.3: handler 实现 GetUserBlogs（读取 user_id、page、size，并返回原有 JSON 结构）**

注意：必须保持原 `code/message/data` 的结构与状态码不变。

- [ ] **Step 2.4: api 层改为薄适配，调用 domain handler**

`internal/api/blog_list.go` 中的 `(*BlogAPI).GetUserBlogs`：
- 保留函数签名与路由绑定不变
- 内部改为调用 `blogHandler.GetUserBlogs(c)`

- [ ] **Step 2.5: 写 handler 单测（至少覆盖鉴权缺失 + 正常路径）**

Create `backend/internal/domain/blog/handler_test.go`（示例骨架）：
```go
package blog_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"inkwords-backend/internal/domain/blog"
)

type fakeRepo struct{}

func (f *fakeRepo) GetByID(ctx any, userID uuid.UUID, blogID uuid.UUID) (*any, error) { return nil, nil }

func TestGetUserBlogs_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	svc := blog.NewService(nil)
	h := blog.NewHandler(svc)

	r.GET("/blogs", func(c *gin.Context) { h.GetUserBlogs(c) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/blogs?page=1&size=20", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}
```
（后续 Step 会把 fakeRepo 补齐为符合接口的实现，并补正常路径测试。）

- [ ] **Step 2.6: 回归**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

---

## 3. 迁移路由 2：POST /api/v1/blogs/draft（创建草稿）

**Files:**
- Modify: `backend/internal/api/blog_draft.go`
- Modify: `backend/internal/domain/blog/repository.go`
- Modify: `backend/internal/domain/blog/service.go`
- Modify: `backend/internal/domain/blog/handler.go`
- Test: `backend/internal/domain/blog/handler_test.go`

- [ ] **Step 3.1: repo 实现 CreateDraft（与现有 BlogService 行为一致）**
- [ ] **Step 3.2: handler 实现 CreateDraftBlog（保持返回 BlogNode 的结构一致）**
- [ ] **Step 3.3: api 层薄适配**
- [ ] **Step 3.4: 单测覆盖（鉴权缺失 + 正常路径）**
- [ ] **Step 3.5: 回归测试**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

---

## 4. 迁移路由 3：PUT /api/v1/blogs/:id（更新）

**Files:**
- Modify: `backend/internal/api/blog_update.go`
- Modify: `backend/internal/domain/blog/*`
- Test: `backend/internal/domain/blog/handler_test.go`

- [ ] **Step 4.1: repo 实现 Update（只允许更新原先允许字段）**
- [ ] **Step 4.2: handler 实现 UpdateBlog（保持状态码/响应结构一致）**
- [ ] **Step 4.3: api 薄适配**
- [ ] **Step 4.4: 回归**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

---

## 5. 迁移路由 4：导出（ZIP/PDF/Obsidian）

**Files:**
- Modify: `backend/internal/api/blog_export.go`
- Modify: `backend/internal/domain/blog/*`

- [ ] **Step 5.1: 将导出逻辑保留在旧 BlogService（Phase 1），domain service 先做转发**

策略：导出逻辑依赖较多（zip/pdf/os 临时文件），Phase 1 先不拆深，只把入口迁到 domain handler，内部继续调用现有 `internal/service.BlogService`，减少风险。

- [ ] **Step 5.2: api 薄适配**
- [ ] **Step 5.3: 回归**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

---

## 6. 收口：依赖组装与文档同步

**Files:**
- Modify: `backend/cmd/server/main.go`（或实际的 router/di 初始化文件，以 Step 0.1 结果为准）
- Modify: `.trae/documents/InkWords_Architecture.md`

- [ ] **Step 6.1: 在启动时组装 blog domain 依赖（repo -> service -> handler）并注入到 api 薄层**
- [ ] **Step 6.2: 文档补充 domain/blog 的目录规范与迁移原则**
- [ ] **Step 6.3: 最终回归**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS


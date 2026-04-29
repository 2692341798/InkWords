# 手写博客入口（草稿创建）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在左侧侧边栏新增“写博客”入口，点击后立即创建一篇空白草稿并进入现有双面板编辑器进行手写写作。

**Architecture:** 前端点击入口调用后端新增 `POST /api/v1/blogs/draft` 创建顶级草稿记录；创建成功后将草稿插入本地博客树并选中，从而复用现有 `Editor.tsx` 的编辑/预览/自动保存能力。

**Tech Stack:** Go + Gin + GORM；React 18 + Zustand + Tailwind + shadcn/ui；@microsoft/fetch-event-source（既有）。

---

## 文件结构（本次会改动/新增）

**后端**
- Modify: [main.go](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/backend/cmd/server/main.go)（路由注册）
- Modify: [blog.go](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/backend/internal/api/blog.go)（新增 handler）
- Modify: [blog.go](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/backend/internal/service/blog.go)（新增 service 方法 + 可注入构造）
- Add: `backend/internal/service/blog_draft_test.go`（TDD：sqlite 内存库验证落库行为）

**前端**
- Modify: [Sidebar.tsx](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/frontend/src/components/Sidebar.tsx)（新增“写博客”按钮）
- Modify: [blogStore.ts](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/frontend/src/store/blogStore.ts)（新增 createDraftBlog action）

**文档**
- Modify: [InkWords_API.md](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/.trae/documents/InkWords_API.md)（记录新接口）

---

### Task 1: 后端草稿创建（Service，TDD）

**Files:**
- Add: `backend/internal/service/blog_draft_test.go`
- Modify: `backend/internal/service/blog.go`
- Modify: `backend/go.mod`（加入 sqlite driver，仅用于测试）

- [ ] **Step 1: 写 failing test（应创建一条顶级草稿记录）**

```go
package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

func TestBlogService_CreateDraftBlog_createsTopLevelDraft(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&model.Blog{}))

	s := NewBlogServiceWithDB(db)
	userID := uuid.New()

	node, err := s.CreateDraftBlog(context.Background(), userID)
	require.NoError(t, err)
	require.NotEmpty(t, node.ID)
	require.Equal(t, userID, node.UserID)
	require.Nil(t, node.ParentID)
	require.Equal(t, 0, node.ChapterSort)
	require.Equal(t, "未命名博客", node.Title)
	require.Equal(t, "", node.Content)
	require.Equal(t, "manual", node.SourceType)
}
```

- [ ] **Step 2: 运行测试并确认失败**

Run:
```bash
cd backend && go test ./... -run TestBlogService_CreateDraftBlog_createsTopLevelDraft -v
```

Expected: FAIL（`NewBlogServiceWithDB` / `CreateDraftBlog` 未定义）

- [ ] **Step 3: 写最小实现让测试变绿**

实现点：
- `NewBlogServiceWithDB(db *gorm.DB) *BlogService`
- `CreateDraftBlog(ctx, userID) (*model.Blog, error)` 或返回 `BlogNode`
- 插入 `model.Blog{UserID:userID, ParentID:nil, ChapterSort:0, Title:"未命名博客", Content:"", SourceType:"manual", IsSeries:false, Status:0, TechStacks:[]}`

- [ ] **Step 4: 运行测试确认通过**

Run:
```bash
cd backend && go test ./... -run TestBlogService_CreateDraftBlog_createsTopLevelDraft -v
```

Expected: PASS

---

### Task 2: 后端 API（POST /api/v1/blogs/draft）

**Files:**
- Modify: `backend/internal/api/blog.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: 写 failing test（API 返回 200 且 data 为草稿）**

（若时间受限，本次可用 service 层测试兜底 + 手动验证 API；但优先补 API 层最小测试。）

- [ ] **Step 2: 实现 handler**

行为：
- 从 `c.Get("user_id")` 取登录用户
- 调用 `blogService.CreateDraftBlog(...)`
- 返回 `{code:200,message:"success",data:<BlogNode>}`

---

### Task 3: 前端入口（Sidebar + store）

**Files:**
- Modify: `frontend/src/components/Sidebar.tsx`
- Modify: `frontend/src/store/blogStore.ts`

- [ ] **Step 1: blogStore 增加 createDraftBlog**

行为：
- `POST /api/v1/blogs/draft`（带 Authorization）
- 成功后把返回的草稿节点插入 `blogs` 顶部（与后端 created_at DESC 一致）
- `selectBlog(draft)`，从而进入 Editor

- [ ] **Step 2: Sidebar 顶部新增“写博客”按钮**

交互：
- 放在“新工作区”按钮下方
- 点击后调用 `createDraftBlog`
- 失败 toast 提示

---

### Task 4: 验证与文档

- [ ] **Step 1: 后端全量测试**

Run:
```bash
cd backend && go test ./... -v
```

- [ ] **Step 2: Docker Compose 验证**

Run:
```bash
docker compose down && docker compose up -d --build
```

手动验证：
- 访问 `http://localhost`
- 登录后点击“写博客”
- 自动进入编辑器；标题/内容修改后 2 秒内自动保存
- 刷新页面后草稿仍在“历史博客”列表最上方

- [ ] **Step 3: 更新 API 文档**

在 `.trae/documents/InkWords_API.md` 增加：
- `POST /api/v1/blogs/draft`：创建空白草稿（顶级博客，后续可扩展到 series）


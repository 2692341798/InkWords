# Project Directory Structure Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 InkWords 目录结构按目标态落地（后端 `domain/transport/infra`，前端 `pages/services`），在不改变对外 API 行为与核心业务逻辑的前提下完成可回滚的渐进迁移。

**Architecture:** 后端先抽离路由注册到 `internal/transport/http/v1`，再搬迁基础设施目录（db/cache/llm/parser），最后搬迁 HTTP middleware 与 legacy api 包；前端先把页面级组件迁入 `pages/`，再引入 `services/` 并收敛 SSE 请求入口。

**Tech Stack:** Go + Gin + GORM + PostgreSQL；React + Vite + Tailwind + shadcn/ui + Zustand；SSE 使用 `@microsoft/fetch-event-source`。

---

## Scope & Constraints

- 不改 DB schema
- 不改变对外 API 路由、响应结构与鉴权策略
- 每个阶段可独立回滚（一次只搬迁一个目录或一个薄层）

---

## File Map (Target vs Current)

### Backend

- Current
  - `backend/internal/db` → Target: `backend/internal/infra/db`
  - `backend/internal/cache` → Target: `backend/internal/infra/cache`
  - `backend/internal/llm` → Target: `backend/internal/infra/llm`
  - `backend/internal/parser` → Target: `backend/internal/infra/parser`
  - `backend/internal/middleware` → Target: `backend/internal/transport/http/middleware`
  - `backend/internal/api` → Target: `backend/internal/transport/http/v1/api`（或保持 api 仅为薄适配，后续再移）
  - `backend/cmd/server/main.go`：路由注册内联 → Target: 调用 `internal/transport/http/v1` 的注册函数

### Frontend

- Current: `frontend/src/components/{Dashboard,Editor,Generator,Login}.tsx`
- Target: `frontend/src/pages/{Dashboard,Editor,Generator,Login}.tsx`
- Add: `frontend/src/services/`（先落地 SSE wrapper，再逐步迁移请求逻辑）

---

## Task 0: Create an isolated worktree (recommended)

**Files:** none

- [ ] **Step 1: Create a worktree branch**

```bash
cd "/Users/huangqijun/Documents/墨言博客助手/InkWords"
git worktree add ../InkWords-dir-migration -b agent/dir-migration
```

- [ ] **Step 2: Install deps / ensure baseline tests pass**

```bash
cd ../InkWords-dir-migration/backend && go test ./...
cd ../InkWords-dir-migration/frontend && npm ci && npm test
```

Expected: both pass.

---

## Task 1: Extract v1 route registration into transport layer (backend)

**Files:**
- Create: `backend/internal/transport/http/v1/routes.go`
- Test: `backend/internal/transport/http/v1/routes_test.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add `backend/internal/transport/http/v1/routes.go`**

```go
package v1

import "github.com/gin-gonic/gin"

type AuthHandlers struct {
	Register      gin.HandlerFunc
	Login         gin.HandlerFunc
	BindGithub    gin.HandlerFunc
	GetCaptcha    gin.HandlerFunc
	OAuthRedirect gin.HandlerFunc
	OAuthCallback gin.HandlerFunc
}

type UserHandlers struct {
	GetProfile    gin.HandlerFunc
	UpdateProfile gin.HandlerFunc
	UploadAvatar  gin.HandlerFunc
	GetUserStats  gin.HandlerFunc
}

type BlogHandlers struct {
	GetUserBlogs           gin.HandlerFunc
	CreateDraftBlog        gin.HandlerFunc
	BatchDeleteBlogs       gin.HandlerFunc
	UpdateBlog             gin.HandlerFunc
	ExportSeries           gin.HandlerFunc
	ExportSeriesPDF        gin.HandlerFunc
	ExportToObsidian       gin.HandlerFunc
	ExportSeriesToObsidian gin.HandlerFunc
	ContinueBlog           gin.HandlerFunc
	PolishBlog             gin.HandlerFunc
}

type ProjectHandlers struct {
	ScanGithubRepo gin.HandlerFunc
	Analyze        gin.HandlerFunc
	Parse          gin.HandlerFunc
}

type StreamHandlers struct {
	ScanStreamHandler     gin.HandlerFunc
	AnalyzeStreamHandler  gin.HandlerFunc
	GenerateStreamHandler gin.HandlerFunc
}

type Handlers struct {
	Auth    AuthHandlers
	User    UserHandlers
	Blog    BlogHandlers
	Project ProjectHandlers
	Stream  StreamHandlers
}

func Register(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers Handlers) {
	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", handlers.Auth.Register)
			authGroup.POST("/login", handlers.Auth.Login)
			authGroup.POST("/bind-github", handlers.Auth.BindGithub)
			authGroup.GET("/captcha", handlers.Auth.GetCaptcha)
			authGroup.GET("/oauth/:provider", handlers.Auth.OAuthRedirect)
			authGroup.GET("/callback/:provider", handlers.Auth.OAuthCallback)
		}

		userGroup := v1.Group("/user")
		userGroup.Use(authMiddleware)
		{
			userGroup.GET("/profile", handlers.User.GetProfile)
			userGroup.PUT("/profile", handlers.User.UpdateProfile)
			userGroup.POST("/avatar", handlers.User.UploadAvatar)
			userGroup.GET("/stats", handlers.User.GetUserStats)
		}

		blogGroup := v1.Group("/blogs")
		blogGroup.Use(authMiddleware)
		{
			blogGroup.GET("", handlers.Blog.GetUserBlogs)
			blogGroup.POST("/draft", handlers.Blog.CreateDraftBlog)
			blogGroup.DELETE("", handlers.Blog.BatchDeleteBlogs)
			blogGroup.PUT("/:id", handlers.Blog.UpdateBlog)
			blogGroup.GET("/:id/export", handlers.Blog.ExportSeries)
			blogGroup.GET("/:id/export/pdf", handlers.Blog.ExportSeriesPDF)
			blogGroup.POST("/:id/export/obsidian", handlers.Blog.ExportToObsidian)
			blogGroup.POST("/:id/export/obsidian/series", handlers.Blog.ExportSeriesToObsidian)
			blogGroup.POST("/:id/continue", handlers.Blog.ContinueBlog)
			blogGroup.POST("/:id/polish", handlers.Blog.PolishBlog)
		}

		projectGroup := v1.Group("/project")
		projectGroup.Use(authMiddleware)
		{
			projectGroup.POST("/scan", handlers.Project.ScanGithubRepo)
			projectGroup.POST("/analyze", handlers.Project.Analyze)
			projectGroup.POST("/parse", handlers.Project.Parse)
		}

		streamGroup := v1.Group("/stream")
		streamGroup.Use(authMiddleware)
		{
			streamGroup.POST("/scan", handlers.Stream.ScanStreamHandler)
			streamGroup.POST("/analyze", handlers.Stream.AnalyzeStreamHandler)
			streamGroup.POST("/generate", handlers.Stream.GenerateStreamHandler)
		}
	}
}
```

- [ ] **Step 2: Add router smoke tests**

```go
package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegister_RoutesAreReachable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()

	authMiddleware := func(c *gin.Context) { c.Next() }

	ok := func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) }

	Register(r, authMiddleware, Handlers{
		Auth: AuthHandlers{
			Register:      ok,
			Login:         ok,
			BindGithub:    ok,
			GetCaptcha:    ok,
			OAuthRedirect: ok,
			OAuthCallback: ok,
		},
		User: UserHandlers{
			GetProfile:    ok,
			UpdateProfile: ok,
			UploadAvatar:  ok,
			GetUserStats:  ok,
		},
		Blog: BlogHandlers{
			GetUserBlogs:           ok,
			CreateDraftBlog:        ok,
			BatchDeleteBlogs:       ok,
			UpdateBlog:             ok,
			ExportSeries:           ok,
			ExportSeriesPDF:        ok,
			ExportToObsidian:       ok,
			ExportSeriesToObsidian: ok,
			ContinueBlog:           ok,
			PolishBlog:             ok,
		},
		Project: ProjectHandlers{
			ScanGithubRepo: ok,
			Analyze:        ok,
			Parse:          ok,
		},
		Stream: StreamHandlers{
			ScanStreamHandler:     ok,
			AnalyzeStreamHandler:  ok,
			GenerateStreamHandler: ok,
		},
	})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodGet, "/api/v1/user/profile"},
		{http.MethodGet, "/api/v1/blogs"},
		{http.MethodPost, "/api/v1/project/scan"},
		{http.MethodPost, "/api/v1/stream/generate"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s %s expected 200, got %d", tc.method, tc.path, w.Code)
		}
	}
}
```

- [ ] **Step 3: Wire `cmd/server/main.go` to use transport router**

Modify: `backend/cmd/server/main.go`

1) Add imports:

```go
transportv1 "inkwords-backend/internal/transport/http/v1"
```

2) Replace the current inline `/api/v1` route registration block with:

```go
transportv1.Register(r, middleware.AuthMiddleware(), transportv1.Handlers{
	Auth: transportv1.AuthHandlers{
		Register:      authAPI.Register,
		Login:         authAPI.Login,
		BindGithub:    authAPI.BindGithub,
		GetCaptcha:    authAPI.GetCaptcha,
		OAuthRedirect: authAPI.OAuthRedirect,
		OAuthCallback: authAPI.OAuthCallback,
	},
	User: transportv1.UserHandlers{
		GetProfile:    userAPI.GetProfile,
		UpdateProfile: userAPI.UpdateProfile,
		UploadAvatar:  userAPI.UploadAvatar,
		GetUserStats:  userAPI.GetUserStats,
	},
	Blog: transportv1.BlogHandlers{
		GetUserBlogs:           blogAPI.GetUserBlogs,
		CreateDraftBlog:        blogAPI.CreateDraftBlog,
		BatchDeleteBlogs:       blogAPI.BatchDeleteBlogs,
		UpdateBlog:             blogAPI.UpdateBlog,
		ExportSeries:           blogAPI.ExportSeries,
		ExportSeriesPDF:        blogAPI.ExportSeriesPDF,
		ExportToObsidian:       blogAPI.ExportToObsidian,
		ExportSeriesToObsidian: blogAPI.ExportSeriesToObsidian,
		ContinueBlog:           streamAPI.ContinueBlogStreamHandler,
		PolishBlog:             streamAPI.PolishBlogStreamHandler,
	},
	Project: transportv1.ProjectHandlers{
		ScanGithubRepo: projectAPI.ScanGithubRepo,
		Analyze:        projectAPI.Analyze,
		Parse:          projectAPI.Parse,
	},
	Stream: transportv1.StreamHandlers{
		ScanStreamHandler:     streamAPI.ScanStreamHandler,
		AnalyzeStreamHandler:  streamAPI.AnalyzeStreamHandler,
		GenerateStreamHandler: streamAPI.GenerateBlogStreamHandler,
	},
})
```

- [ ] **Step 4: Run backend tests**

Run:

```bash
cd backend && go test ./...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/cmd/server/main.go backend/internal/transport/http/v1/routes.go backend/internal/transport/http/v1/routes_test.go
git commit -m "refactor(backend): register v1 routes via transport layer

中文：将 /api/v1 路由注册从 main.go 抽离到 internal/transport/http/v1，保持对外接口不变，便于后续目录搬迁与渐进重构。"
```

---

## Task 2: Move infrastructure packages into `internal/infra` (backend)

**Files:**
- Move: `backend/internal/db` → `backend/internal/infra/db`
- Move: `backend/internal/cache` → `backend/internal/infra/cache`
- Move: `backend/internal/llm` → `backend/internal/infra/llm`
- Move: `backend/internal/parser` → `backend/internal/infra/parser`
- Modify: all Go files importing old paths (expected: multiple files)

- [ ] **Step 1: Move directories**

```bash
git mv backend/internal/db backend/internal/infra/db
git mv backend/internal/cache backend/internal/infra/cache
git mv backend/internal/llm backend/internal/infra/llm
git mv backend/internal/parser backend/internal/infra/parser
```

- [ ] **Step 2: Update import paths**

Replace imports (examples):
- `inkwords-backend/internal/db` → `inkwords-backend/internal/infra/db`
- `inkwords-backend/internal/cache` → `inkwords-backend/internal/infra/cache`
- `inkwords-backend/internal/llm` → `inkwords-backend/internal/infra/llm`
- `inkwords-backend/internal/parser` → `inkwords-backend/internal/infra/parser`

- [ ] **Step 3: Run backend tests**

```bash
cd backend && go test ./...
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend
git commit -m "refactor(backend): move infra packages under internal/infra

中文：将 db/cache/llm/parser 等基础设施目录迁移至 internal/infra，以对齐 domain/transport/infra 三分法。"
```

---

## Task 3: Move HTTP middleware under `internal/transport/http/middleware` (backend)

**Files:**
- Move: `backend/internal/middleware/auth.go` → `backend/internal/transport/http/middleware/auth.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Move middleware directory**

```bash
git mv backend/internal/middleware backend/internal/transport/http/middleware
```

- [ ] **Step 2: Update imports in `main.go`**

Replace:
- `inkwords-backend/internal/middleware` → `inkwords-backend/internal/transport/http/middleware`

- [ ] **Step 3: Run backend tests**

```bash
cd backend && go test ./...
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend
git commit -m "refactor(backend): move auth middleware under transport/http

中文：将鉴权中间件迁移到 internal/transport/http/middleware，明确其属于 HTTP 传输适配层。"
```

---

## Task 4: Relocate legacy `internal/api` package (backend, optional but aligns target)

**Files:**
- Move: `backend/internal/api` → `backend/internal/transport/http/v1/api`
- Modify: `backend/cmd/server/main.go`
- Modify: moved package imports inside `api/*.go` (if any)

- [ ] **Step 1: Move package directory**

```bash
git mv backend/internal/api backend/internal/transport/http/v1/api
```

- [ ] **Step 2: Update package import path in `main.go`**

Replace:
- `inkwords-backend/internal/api` → `inkwords-backend/internal/transport/http/v1/api`

- [ ] **Step 3: Ensure internal imports are consistent**

If any `api/*.go` referenced old infra paths, ensure they match Task 2 after infra migration.

- [ ] **Step 4: Run backend tests**

```bash
cd backend && go test ./...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend
git commit -m "refactor(backend): relocate legacy api package into transport/http/v1

中文：将 internal/api 迁移到 internal/transport/http/v1/api，进一步收敛 HTTP 相关代码位置；对外 API 行为不变。"
```

---

## Task 5: Move page-level components into `src/pages` (frontend)

**Files:**
- Move: `frontend/src/components/Dashboard.tsx` → `frontend/src/pages/Dashboard.tsx`
- Move: `frontend/src/components/Editor.tsx` → `frontend/src/pages/Editor.tsx`
- Move: `frontend/src/components/Generator.tsx` → `frontend/src/pages/Generator.tsx`
- Move: `frontend/src/components/Login.tsx` → `frontend/src/pages/Login.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: any imports referencing these components

- [ ] **Step 1: Create pages dir and move files**

```bash
git mv frontend/src/components/Dashboard.tsx frontend/src/pages/Dashboard.tsx
git mv frontend/src/components/Editor.tsx frontend/src/pages/Editor.tsx
git mv frontend/src/components/Generator.tsx frontend/src/pages/Generator.tsx
git mv frontend/src/components/Login.tsx frontend/src/pages/Login.tsx
```

- [ ] **Step 2: Update `App.tsx` imports**

Modify: `frontend/src/App.tsx`

```ts
import { Generator } from '@/pages/Generator'
import { Editor } from '@/pages/Editor'
import { Login } from '@/pages/Login'
import { Dashboard } from '@/pages/Dashboard'
```

- [ ] **Step 3: Fix relative imports inside moved pages (if any)**

Example (if present after move):
- `./editor/EditorHeader` should become `../components/editor/EditorHeader` or use alias `@/components/editor/EditorHeader`

- [ ] **Step 4: Run frontend checks**

```bash
cd frontend && npm test
cd frontend && npm run build
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/App.tsx frontend/src/pages
git commit -m "refactor(frontend): move page components into src/pages

中文：将 Dashboard/Editor/Generator/Login 归类为页面级组件迁移到 src/pages，保持现有渲染逻辑不变。"
```

---

## Task 6: Add `src/services` and centralize SSE auth handling (frontend)

**Files:**
- Create: `frontend/src/services/sse.ts`
- Test: `frontend/src/services/sse.test.ts`
- Modify: SSE call sites
  - `frontend/src/hooks/generator/useProjectScanner.ts`
  - `frontend/src/hooks/generator/useProjectAnalyzer.ts`
  - `frontend/src/hooks/generator/useFileParser.ts`
  - `frontend/src/hooks/generator/useSeriesGenerator.ts`
  - `frontend/src/hooks/usePolishStream.ts`
  - `frontend/src/pages/Editor.tsx` (after Task 5)

- [ ] **Step 1: Add `services/sse.ts`**

```ts
import { fetchEventSource } from '@microsoft/fetch-event-source'

export type SSEOptions = Omit<Parameters<typeof fetchEventSource>[1], 'headers'> & {
  headers?: Record<string, string>
  requireAuth?: boolean
}

export const buildAuthHeader = (token: string | null) => {
  if (!token) return {}
  return { Authorization: `Bearer ${token}` }
}

export const fetchEventSourceWithAuth = (url: string, options: SSEOptions) => {
  const token = localStorage.getItem('token')
  const headers: Record<string, string> = {
    ...(options.headers ?? {}),
    ...(options.requireAuth === false ? {} : buildAuthHeader(token)),
  }

  return fetchEventSource(url, {
    ...options,
    headers,
    async onopen(response) {
      if (response.status === 401) {
        localStorage.removeItem('token')
        window.location.reload()
        throw new Error('登录已过期，请重新登录')
      }
      await options.onopen?.(response)
    },
  })
}
```

- [ ] **Step 2: Add unit tests**

```ts
import { describe, expect, it } from 'vitest'
import { buildAuthHeader } from './sse'

describe('buildAuthHeader', () => {
  it('returns empty object when token missing', () => {
    expect(buildAuthHeader(null)).toEqual({})
  })

  it('returns Bearer token header when token present', () => {
    expect(buildAuthHeader('t')).toEqual({ Authorization: 'Bearer t' })
  })
})
```

- [ ] **Step 3: Update SSE call sites to use `fetchEventSourceWithAuth`**

Example change pattern:

Before:

```ts
await fetchEventSource('/api/v1/stream/scan', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': token ? `Bearer ${token}` : ''
  },
  body: JSON.stringify({ git_url: gitUrl }),
  ...
})
```

After:

```ts
import { fetchEventSourceWithAuth } from '@/services/sse'

await fetchEventSourceWithAuth('/api/v1/stream/scan', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({ git_url: gitUrl }),
  ...
})
```

- [ ] **Step 4: Run frontend checks**

```bash
cd frontend && npm test
cd frontend && npm run build
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/services/sse.ts frontend/src/services/sse.test.ts frontend/src/hooks frontend/src/pages/Editor.tsx
git commit -m "refactor(frontend): centralize SSE auth handling in services

中文：新增 src/services/sse.ts 统一 SSE 请求的鉴权 header 与 401 处理，减少 hooks/pages 中的重复逻辑。"
```

---

## Task 7: End-to-end verification via Docker Compose (system smoke)

**Files:** none

- [ ] **Step 1: Rebuild and start**

```bash
docker compose down && docker compose up -d --build
```

- [ ] **Step 2: Smoke check**

- Open: `http://localhost`
- Verify:
  - 登录页可打开（或已有 token 自动进入）
  - `/api/v1/ping` 返回 `pong`
  - 生成/扫描任意 Git 仓库流程可正常建立 SSE 连接

---

## Docs & Governance

- 若 Task 2~6 进入真实落地并产生 import/目录重大变更，建议同步更新：
  - `.trae/documents/InkWords_Architecture.md`
  - `.trae/documents/InkWords_Development_Plan_and_Log.md`（记录迁移批次与回滚点）

---

## Plan Self-Review Checklist

- [ ] 每个 Task 都能单独落地并回滚
- [ ] backend `go test ./...` 与 frontend `npm test && npm run build` 在每个批次都可通过
- [ ] `internal/domain/*` 的现有迁移成果不被打断（仅移动路由与基础设施目录）


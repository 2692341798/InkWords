# InkWords Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement architecture refactoring (DI, comments), OAuth security enhancements (Bind Account), UI performance fixes, and a hard token limit.

**Architecture:** 
1. Decouple global `db.DB` from services/APIs by injecting `*gorm.DB`.
2. Add a `BindGithub` flow for OAuth users whose email already exists locally.
3. Add a token quota interceptor in stream/generation handlers.
4. Flatten the blog tree in `Sidebar.tsx` using `useMemo` to fix React rendering performance.
5. Translate all backend JSON error messages to Chinese.

**Tech Stack:** Go, Gin, GORM, React, Zustand, Tailwind CSS.

---

### Task 1: Architecture - Dependency Injection & Comments

**Files:**
- Modify: `backend/internal/service/auth.go`
- Modify: `backend/internal/service/user.go` (Create this file to move logic from `api/user.go`)
- Modify: `backend/internal/api/auth.go`
- Modify: `backend/internal/api/user.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Create UserService and inject DB**
Move business logic from `UserAPI` to `UserService` and inject `*gorm.DB` into both `AuthService` and `UserService`.

- [ ] **Step 2: Update APIs to use injected services**
Refactor `NewAuthAPI` and `NewUserAPI` to accept their respective services.

- [ ] **Step 3: Update `main.go` initialization**
Pass `db.DB` to the services, and pass the services to the APIs.

- [ ] **Step 4: Add Godoc to all public methods in these files**

- [ ] **Step 5: Verify the backend compiles and runs**
Run: `cd backend && go build -o server ./cmd/server`
Expected: PASS

---

### Task 2: Business Logic - Token Quota Interceptor

**Files:**
- Modify: `backend/internal/api/stream.go`
- Modify: `backend/internal/api/project.go`

- [ ] **Step 1: Add quota check helper**
Create a helper function to check if `TokensUsed >= TokenLimit` (default 100,000 if 0).

- [ ] **Step 2: Apply interceptor to generation endpoints**
Inject the check into `AnalyzeStreamHandler`, `GenerateBlogStreamHandler`, and `ContinueBlogStreamHandler`.
Return Chinese error: `"您的 Token 额度已耗尽，请升级订阅或联系管理员"`

- [ ] **Step 3: Apply interceptor to project endpoints**
Inject the check into `Analyze` and `Parse`.

- [ ] **Step 4: Verify the backend compiles**
Run: `cd backend && go build -o server ./cmd/server`
Expected: PASS

---

### Task 3: OAuth Security - Bind Account Flow

**Files:**
- Modify: `backend/internal/service/auth.go`
- Modify: `backend/internal/api/auth.go`
- Modify: `frontend/src/components/Login.tsx`

- [ ] **Step 1: Update OAuth Callback Logic**
If email exists but `GithubID` is null, redirect to frontend with `?bind_required=true&email=xxx&github_id=yyy&avatar_url=zzz&username=uuu`.

- [ ] **Step 2: Add `BindGithub` endpoint**
Create `POST /api/v1/auth/bind-github` to accept `email`, `password`, and the GitHub info. Verify password, update user with GitHub info, and return JWT.

- [ ] **Step 3: Update Frontend Login Component**
Add state for `bindRequired` and a form to enter the password for the existing email. Call the new bind endpoint.

---

### Task 4: Frontend UX & Error Translation

**Files:**
- Modify: `frontend/src/components/Sidebar.tsx`
- Modify: `backend/internal/api/user.go`
- Modify: `backend/internal/api/auth.go`
- Modify: `backend/internal/api/blog.go`

- [ ] **Step 1: Fix React Performance in Sidebar**
Use `useMemo` to create a flat map of `id -> BlogNode` and `parentId -> children[]` to avoid deep recursion during click events.

- [ ] **Step 2: Add Username Validation**
In `UpdateProfile`, validate `len(req.Username) >= 2 && len(req.Username) <= 20`.

- [ ] **Step 3: Translate Backend Errors**
Replace all English error messages in `c.JSON(..., gin.H{"message": "..."})` with Chinese equivalents (e.g., "参数格式错误", "用户不存在", "文件过大，最大限制 2MB").

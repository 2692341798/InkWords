# Framework Compliance Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the current InkWords project closer to the declared software engineering standards for monorepo structure, backend architecture, frontend architecture, and Docker-first deployment.

**Architecture:** The repo direction is already correct, so this plan keeps changes incremental instead of rewriting the project. The main strategy is to tighten dependency boundaries, centralize request logic, standardize UI/documentation rules, and harden Docker configuration without breaking current user-facing flows.

**Tech Stack:** Go 1.21+, Gin, GORM, PostgreSQL, Redis, React 18, Vite, Tailwind CSS, shadcn-style UI primitives, Zustand, Docker Compose, Nginx

---

## Scope

**In scope**
- Backend DI cleanup and constructor cleanup
- Backend external error contract cleanup
- Frontend service extraction from stores/pages/components
- Frontend Chinese UI text normalization
- Frontend JSDoc backfill for exported hooks and complex components
- Docker Compose portability and security hardening

**Out of scope for this plan**
- Full domain migration of every legacy backend service under `backend/internal/service`
- Large-scale redesign of page layouts or state model
- Splitting every 500+ line file in one pass

## File Map

**Backend composition and transport**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/transport/http/v1/api/stream_api.go`
- Modify: `backend/internal/transport/http/v1/api/project.go`
- Modify: `backend/internal/transport/http/v1/api/user.go`
- Modify: `backend/internal/transport/http/v1/api/blog_api.go`
- Modify: `backend/internal/transport/http/v1/routes.go`

**Backend error/output cleanup**
- Modify: `backend/internal/domain/blog/handler.go`
- Modify: `backend/internal/domain/stream/handler.go`
- Modify: `backend/internal/service/generator.go`
- Reuse pattern from: `backend/internal/domain/review/handler.go`

**Frontend request-layer cleanup**
- Create: `frontend/src/services/blog.ts`
- Create: `frontend/src/services/auth.ts`
- Modify: `frontend/src/store/blogStore.ts`
- Modify: `frontend/src/components/Sidebar.tsx`
- Modify: `frontend/src/pages/Login.tsx`
- Modify: `frontend/src/pages/Dashboard.tsx`
- Modify: `frontend/src/pages/Editor.tsx`
- Modify: `frontend/src/hooks/generator/useFileParser.ts`

**Frontend standards polish**
- Modify: `frontend/src/pages/Login.tsx`
- Modify: `frontend/src/pages/Dashboard.tsx`
- Modify: `frontend/src/components/Sidebar.tsx`
- Modify: `frontend/src/hooks/useKnowledgeReview.ts`
- Modify: `frontend/src/hooks/useBlogStream.ts`
- Modify: `frontend/src/pages/Generator.tsx`
- Review/remove: `frontend/src/store/index.ts`

**Docker and docs**
- Modify: `docker-compose.yml`
- Modify: `README.md`
- Optional create: `.env.example` entries or `backend/.env.example` updates if missing

### Task 1: Backend DI Boundary Cleanup

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/transport/http/v1/api/stream_api.go`
- Modify: `backend/internal/transport/http/v1/api/project.go`
- Modify: `backend/internal/transport/http/v1/api/user.go`
- Modify: `backend/internal/transport/http/v1/api/blog_api.go`
- Test: `backend/internal/transport/http/v1/routes_test.go`

- [ ] **Step 1: Snapshot current backend behavior**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/transport/http/v1/... ./internal/domain/... ./cmd/server/...
```

Expected:
- Existing transport and domain tests pass before refactor.

- [ ] **Step 2: Remove transport-layer self-construction**

Implementation target:
- Keep `New*APIWithDeps(...)` constructors as the only constructors used by production wiring.
- Stop transport constructors from creating services via `db.DB` or `service.New...`.
- Treat `backend/cmd/server/main.go` as the composition root for concrete dependency creation.

Required edits:
- Delete or stop using `NewStreamAPI(...)` in `stream_api.go` if it constructs services internally.
- Apply the same rule to `project.go`, `user.go`, and `blog_api.go`.
- Ensure `main.go` passes ready-made dependencies into transport APIs.

Verification:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
grep -R "service.NewPromptRequirementsService(db.DB)" backend/internal/transport/http/v1/api || true
grep -R "db.DB" backend/internal/transport/http/v1/api || true
```

Expected:
- No transport constructor builds services from `db.DB`.

- [ ] **Step 3: Keep route registration fail-safe without request-path surprises**

Implementation target:
- Replace panic-heavy validation in `routes.go` with startup-time explicit error reporting where practical.
- If full signature change is too wide for one pass, keep `validateHandlers` only as startup validation and add clear comments explaining it is composition-root-only behavior.

Verification:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/transport/http/v1/...
```

Expected:
- Route registration tests still pass.

- [ ] **Step 4: Run focused regression tests**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/transport/http/v1/... ./internal/domain/stream/... ./internal/domain/project/... ./internal/domain/user/... ./internal/domain/blog/...
```

Expected:
- No transport/domain regression.

- [ ] **Step 5: Commit**

```bash
git add backend/cmd/server/main.go \
  backend/internal/transport/http/v1/api/stream_api.go \
  backend/internal/transport/http/v1/api/project.go \
  backend/internal/transport/http/v1/api/user.go \
  backend/internal/transport/http/v1/api/blog_api.go \
  backend/internal/transport/http/v1/routes.go \
  backend/internal/transport/http/v1/routes_test.go
git commit -m "refactor(backend): tighten transport dependency injection boundaries"
```

### Task 2: Backend Error Contract and Public API Docs

**Files:**
- Modify: `backend/internal/domain/blog/handler.go`
- Modify: `backend/internal/domain/stream/handler.go`
- Modify: `backend/internal/service/generator.go`
- Modify: `backend/internal/domain/stream/service.go`
- Modify: `backend/internal/domain/user/service.go`
- Modify: `backend/internal/domain/user/handler.go`

- [ ] **Step 1: Record current leak points**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
grep -R 'err.Error()' backend/internal/domain/blog backend/internal/domain/stream
```

Expected:
- Raw error exposure locations are visible before cleanup.

- [ ] **Step 2: Standardize external error messages**

Implementation target:
- Replace direct `err.Error()` JSON responses in blog and stream handlers with stable user-facing messages.
- Preserve root cause internally with wrapped errors or logs.
- Reuse the pattern already demonstrated in `backend/internal/domain/review/handler.go`.

Required edits:
- `blog/handler.go`: map internal failures to stable Chinese messages such as `获取博客列表失败`, `创建草稿失败`, `更新博客失败`, `批量删除失败`.
- `stream/handler.go`: avoid emitting raw backend error strings over SSE `error` events; emit stable event text and log the original error server-side.
- `generator.go`: stop printing DB failures to stdout only; return or log structured errors.

- [ ] **Step 3: Add missing Godoc on exported types and methods touched by this work**

Implementation target:
- Add standard Godoc comments to exported structs/functions in:
  - `backend/internal/domain/stream/service.go`
  - `backend/internal/domain/user/service.go`
  - `backend/internal/domain/user/handler.go`

Verification:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/blog/... ./internal/domain/stream/... ./internal/domain/user/... ./internal/service/...
```

Expected:
- Domain/service tests still pass after message/doc cleanup.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/domain/blog/handler.go \
  backend/internal/domain/stream/handler.go \
  backend/internal/service/generator.go \
  backend/internal/domain/stream/service.go \
  backend/internal/domain/user/service.go \
  backend/internal/domain/user/handler.go
git commit -m "fix(backend): standardize external errors and add api docs"
```

### Task 3: Frontend Request Layer Extraction

**Files:**
- Create: `frontend/src/services/blog.ts`
- Create: `frontend/src/services/auth.ts`
- Modify: `frontend/src/store/blogStore.ts`
- Modify: `frontend/src/components/Sidebar.tsx`
- Modify: `frontend/src/pages/Login.tsx`
- Modify: `frontend/src/pages/Dashboard.tsx`
- Modify: `frontend/src/pages/Editor.tsx`
- Modify: `frontend/src/hooks/generator/useFileParser.ts`

- [ ] **Step 1: Snapshot frontend baseline**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm test -- --runInBand
npm run build
```

Expected:
- Current tests and build pass before extraction.

- [ ] **Step 2: Create service-layer wrappers**

Implementation target:
- `src/services/blog.ts` owns:
  - fetch blog tree
  - create draft
  - update blog
  - batch delete
  - export series to Obsidian
  - export series PDF
- `src/services/auth.ts` owns:
  - login
  - register
  - captcha fetch
  - auth token header helper

Design rule:
- Service functions return parsed business payloads or throw normalized `Error`.
- Do not redirect with `window.location.href` inside services.

- [ ] **Step 3: Slim down store and UI callers**

Implementation target:
- `blogStore.ts` keeps state transitions and optimistic updates, but delegates network calls to `services/blog.ts`.
- `Sidebar.tsx`, `Login.tsx`, `Dashboard.tsx`, `Editor.tsx`, and `useFileParser.ts` stop using raw `fetch` for business APIs where a service wrapper exists.

Verification:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
grep -R "fetch(" frontend/src/pages frontend/src/components frontend/src/store frontend/src/hooks | grep -v "frontend/src/services" || true
```

Expected:
- Remaining raw `fetch` usage is either zero or intentionally limited and documented.

- [ ] **Step 4: Run focused frontend regression checks**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm test -- --runInBand
npm run build
```

Expected:
- No TypeScript or build regressions after service extraction.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/services/blog.ts \
  frontend/src/services/auth.ts \
  frontend/src/store/blogStore.ts \
  frontend/src/components/Sidebar.tsx \
  frontend/src/pages/Login.tsx \
  frontend/src/pages/Dashboard.tsx \
  frontend/src/pages/Editor.tsx \
  frontend/src/hooks/generator/useFileParser.ts
git commit -m "refactor(frontend): centralize request logic into services"
```

### Task 4: Frontend Standards Polish

**Files:**
- Modify: `frontend/src/pages/Login.tsx`
- Modify: `frontend/src/pages/Dashboard.tsx`
- Modify: `frontend/src/components/Sidebar.tsx`
- Modify: `frontend/src/hooks/useKnowledgeReview.ts`
- Modify: `frontend/src/hooks/useBlogStream.ts`
- Modify: `frontend/src/pages/Generator.tsx`
- Review/remove: `frontend/src/store/index.ts`

- [ ] **Step 1: Normalize user-facing text to Chinese**

Implementation target:
- Replace mixed-language strings:
  - `name@example.com` -> Chinese placeholder text
  - `alt="captcha"` -> Chinese alt text
  - `alt="Avatar"` -> Chinese alt text
- Check button labels, placeholders, empty states, and loading states in touched files.

- [ ] **Step 2: Add JSDoc to exported hooks and complex components**

Required targets:
- `useKnowledgeReview.ts`
- `useBlogStream.ts`
- `Generator.tsx`
- `Sidebar.tsx`

JSDoc rule:
- Explain purpose, key inputs/outputs, and any important side-effect boundary.

- [ ] **Step 3: Remove dead or demo-only store surface**

Implementation target:
- Confirm whether `frontend/src/store/index.ts` is unused.
- If unused, delete it.
- If kept, document why it exists and prevent confusion with a comment or rename.

- [ ] **Step 4: Address file-size warning pragmatically**

Implementation target:
- Split `Sidebar.tsx` if the service extraction still leaves it over the 500-line warning.
- Preferred extraction:
  - batch action toolbar
  - tree node renderer
  - export action helpers

Verification:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm test -- --runInBand
npm run build
```

Expected:
- Frontend still builds, and touched tests stay green.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/pages/Login.tsx \
  frontend/src/pages/Dashboard.tsx \
  frontend/src/components/Sidebar.tsx \
  frontend/src/hooks/useKnowledgeReview.ts \
  frontend/src/hooks/useBlogStream.ts \
  frontend/src/pages/Generator.tsx \
  frontend/src/store/index.ts
git commit -m "docs(frontend): align ui text and jsdoc with project standards"
```

### Task 5: Docker Compose Hardening and Entry-Point Cleanup

**Files:**
- Modify: `docker-compose.yml`
- Modify: `README.md`
- Optional modify: `backend/.env.example`

- [ ] **Step 1: Snapshot deployment baseline**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose config
```

Expected:
- Current Compose file renders successfully before edits.

- [ ] **Step 2: Harden Compose networking and secrets**

Implementation target:
- Define an explicit internal network such as `inkwords-network`.
- Attach `db`, `redis`, `backend`, `frontend`, and `obsidian-bridge` to that network.
- Move hardcoded database credentials out of versioned literal values and into `.env` or Compose variable substitution.

- [ ] **Step 3: Tighten host exposure defaults**

Implementation target:
- Keep frontend on `http://localhost`.
- Remove default host exposure for backend, Redis, and PostgreSQL unless a documented debug profile requires them.
- Replace machine-specific default bind mount path for `OBSIDIAN_VAULT_PATH` with an env-only variable and README instructions.

- [ ] **Step 4: Update operational docs**

Implementation target:
- `README.md` must clearly state:
  - standard startup command
  - standard restart command
  - frontend is the primary entrypoint
  - required environment variables
  - optional debug-only port exposure if retained

Verification:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose config
docker compose down && docker compose up -d --build
```

Expected:
- Compose validates successfully.
- App remains reachable at `http://localhost`.

- [ ] **Step 5: Commit**

```bash
git add docker-compose.yml README.md backend/.env.example
git commit -m "chore(deploy): harden compose networking and runtime config"
```

## Follow-up After Core Compliance

- Split `backend/internal/domain/review/service.go` if it grows further beyond the warning threshold.
- Evaluate whether `backend/internal/service/decomposition_generate.go` should be migrated into a domain slice or split by scenario/output stage.
- Consider adding ESLint `max-lines` and a Go lint rule for function/file complexity to enforce the stated standards automatically.

## Validation Matrix

- Backend compile/test:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./...
```

- Frontend build/test:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm test -- --runInBand
npm run build
```

- Docker validation:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose config
docker compose down && docker compose up -d --build
```

- Manual verification:
- Open `http://localhost`
- Log in and fetch captcha
- Open dashboard and verify profile/avatar UI text is Chinese
- Fetch blog tree from sidebar
- Create/update/delete a draft blog
- Run stream scan/analyze/generate once

## Self-Review

- Spec coverage: the plan addresses every major gap found in the review: backend DI, backend error contract, frontend request layering, Chinese UI text, JSDoc/Godoc, Docker hardening, and file-size risk management.
- Placeholder scan: no `TODO`, `TBD`, or unnamed tasks are left in the implementation steps.
- Type consistency: the plan uses existing repo paths and current module names consistently.

Plan complete and saved to `docs/superpowers/plans/2026-05-28-framework-compliance-remediation.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using `executing-plans`, batch execution with checkpoints

**Which approach?**

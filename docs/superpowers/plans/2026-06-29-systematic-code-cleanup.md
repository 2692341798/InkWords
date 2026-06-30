# InkWords Systematic Code Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不改变现有 API、任务协议、数据库 schema 和默认访问入口的前提下，建立可持续质量门禁，消除关键错误吞噬与前端安全风险，并分批收口 `internal/*`、`shared/*`、`services/*` 之间的重复和双轨实现。

**Architecture:** 清理按“门禁 -> 行为刻画 -> 风险修复 -> 单一所有者迁移 -> 删除 -> 冒烟验收”推进。每个工作流独立分支、独立 PR；生产服务目录是最终所有者，`shared` 只保留真正跨服务的平台能力，legacy `internal` 代码在所有入口完成切换后删除。

**Tech Stack:** Go 1.25.4, Gin, GORM, PostgreSQL, RabbitMQ, Redis, React 19.2, TypeScript 5.9, Vite 8, Vitest 3.2, ESLint 9, Zustand, Docker Compose, GitHub Actions

---

## Completion Ledger (2026-06-29)

The original Tasks 0–13 are complete on `codex/governance-baseline` and map to these commits:

| Task | Implementing commit(s) |
| --- | --- |
| 0 | `c73a39f` |
| 1 | `e8155f0` |
| 2 | `6154870` |
| 3 | `6ee7e65` |
| 4 | `a00fc7f` |
| 5 | `c937afb` |
| 6 | `9158fa5` |
| 7 | `2578f27`, `fc572ff` |
| 8 | `de65cb0` |
| 9 | `ed2102c` |
| 10 | `caeafbb` |
| 11 | `8058d75`, `389e93f`, `006f5ea` |
| 12 | `5c549bf` |
| 13 | `eeaf8bc`, `1b3adf3`, `d492a26` |

Closeout verification is recorded in `2026-06-29-systematic-code-cleanup-closeout.md`. Local code, coverage, bundle, Compose config, clean Docker build, health, and gateway smoke gates pass. Publication and remote CI remain intentionally pending explicit user authorization.

## 1. Current Baseline

- Branch: `main` tracking `origin/main`; worktree clean at plan creation.
- Backend: `go test ./...` passes; `go vet ./...` passes; statement coverage is `37.8%`.
- Frontend: 41 test files / 138 tests pass; production build passes.
- Frontend lint: fails with 2 errors and 3 warnings.
- Duplicate Go code: 22 exact-content groups covering 52 non-test files.
- Production services still import these legacy packages through `internal/service`: `internal/domain/blog`, `internal/infra/db`, `internal/infra/llm`, `internal/model`, `internal/prompt`, `internal/service`.
- Main bundle: about 1.81 MB / 586 KB gzip; Vite reports chunks larger than 500 KB.

## 2. Non-Goals

- Do not change external HTTP paths or response envelopes.
- Do not change RabbitMQ routing keys or task payload schema.
- Do not change database tables or run destructive migrations.
- Do not redesign the UI while fixing lint, safety, or bundle boundaries.
- Do not delete `cmd/server` until it has either been rewired to service-owned packages or explicitly retired in a separate decision.
- Do not combine dependency upgrades with legacy deletion.

## 3. Pull Request Map and Dependency Order

| PR | Branch | Scope | Depends on |
|---|---|---|---|
| 1 | `codex/governance-baseline` | Restore lint; add deterministic quality commands | none |
| 2 | `codex/governance-coverage` | Add coverage and dead-code reports; no deletions | PR 1 |
| 3 | `codex/auth-error-handling` | Fix auth persistence errors with tests | PR 1 |
| 4 | `codex/markdown-safety` | Mermaid safety and lazy loading | PR 1 |
| 5 | `codex/parser-platform-dedup` | Select parser owner and remove exact duplicates | PR 2 |
| 6 | `codex/llm-platform-dedup` | Select DeepSeek owner and remove duplicate infra | PR 2 |
| 7 | `codex/worker-delivery-errors` | MQ Ack/Nack and export I/O error handling | PR 2 |
| 8 | `codex/service-characterization` | Move behavior coverage to production service packages | PRs 3, 5, 6, 7 |
| 9 | `codex/core-api-legacy-exit` | Remove core-api dependency on `internal/service` | PR 8 |
| 10 | `codex/llm-stream-legacy-exit` | Move generation application logic to llm-stream ownership | PR 8 |
| 11 | `codex/legacy-entrypoint-exit` | Rewire local aggregate entrypoint and delete unreachable legacy code | PRs 9, 10 |
| 12 | `codex/governance-finalize` | Enforce architecture, coverage, bundle and docs gates | PRs 1-11 |

PRs 3 and 4 may run in parallel. PRs 5, 6, and 7 may run in parallel after PR 2. PRs 9-11 must remain sequential.

## Task 0: Create the Cleanup Branch and Capture the Baseline

**Files:**
- Create: `docs/qa/code-cleanup-baseline.md`
- Do not modify production code.

- [ ] **Step 1: Verify the repository state**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git status --short --branch
git rev-parse --show-toplevel
```

Expected:

```text
## main...origin/main
/Users/huangqijun/Documents/墨言博客助手/InkWords
```

- [ ] **Step 2: Create the branch**

```bash
git switch -c codex/governance-baseline
```

Expected: current branch is `codex/governance-baseline`.

- [ ] **Step 3: Capture backend baseline**

```bash
cd backend
go test ./... -coverprofile=/tmp/inkwords-before.cover
go tool cover -func=/tmp/inkwords-before.cover | tail -n 1
go vet ./...
```

Expected: tests and vet pass; total statement coverage is approximately `37.8%`.

- [ ] **Step 4: Capture frontend baseline**

```bash
cd ../frontend
npm test
npm run build
npm run lint
```

Expected: tests/build pass; lint fails with the known 2 errors and 3 warnings. Any additional failure must be investigated before cleanup starts.

- [ ] **Step 5: Record the baseline**

Create `docs/qa/code-cleanup-baseline.md` with these sections and actual command outputs:

```markdown
# Code Cleanup Baseline

## Commit
- Base commit: `ff4c51696bbc032d0255af98ad545e8af7a3dce8`

## Backend
- Full tests: pass
- Vet: pass
- Statement coverage: 37.8%

## Frontend
- Tests: 41 files / 138 tests pass
- Build: pass
- Lint: 2 errors / 3 warnings
- Main bundle: 1.81 MB / 586 KB gzip

## Known Structural Debt
- 22 exact duplicate groups / 52 files
- production services still depend on internal/service
```

- [ ] **Step 6: Commit the baseline document**

```bash
git add docs/qa/code-cleanup-baseline.md
git commit -m "docs(qa): capture code cleanup baseline"
```

## Task 1: Restore Frontend Lint and Make It a Required Gate

**Files:**
- Modify: `frontend/src/hooks/usePolishStream.ts`
- Modify: `frontend/src/hooks/useKnowledgeReview.test.tsx`
- Modify: `frontend/src/pages/Editor.tsx`
- Modify: `frontend/package.json`
- Modify: `.github/workflows/ci.yml`
- Test: existing frontend tests plus lint/build

- [ ] **Step 1: Verify the existing buffer lifecycle contract**

`frontend/src/lib/streamFlushBuffer.test.ts` already contains `drops buffered text when cancelled before flushing`. Keep that test unchanged as the lifecycle contract for the hook refactor:

```ts
it('drops buffered text when cancelled before flushing', () => {
  vi.useFakeTimers()
  const received: string[] = []
  const buffer = createTextChunkBuffer((chunk) => received.push(chunk))

  buffer.push('draft')
  buffer.cancel()
  vi.runAllTimers()

  expect(received).toEqual([])
})
```

- [ ] **Step 2: Run the focused test before modifying the hook**

```bash
cd frontend
npx vitest run src/lib/streamFlushBuffer.test.ts --configLoader runner
```

Expected: PASS. If the assertion fails, fix the buffer contract before refactoring the hook.

- [ ] **Step 3: Replace render-time ref mutation with stable lazy state**

In `frontend/src/hooks/usePolishStream.ts`, replace `draftBufferRef` initialization with:

```ts
const [draftBuffer] = useState(() =>
  createTextChunkBuffer((chunk) => {
    setDraft((previous) => previous + chunk)
  }),
)

useEffect(() => () => draftBuffer.cancel(), [draftBuffer])
```

Replace every `draftBufferRef.current?.push/flush/cancel` call with `draftBuffer.push/flush/cancel`, and add `draftBuffer` to callbacks' dependency arrays.

- [ ] **Step 4: Mark the test capture container as a ref-shaped test value**

The React lint rule explicitly recognizes ref-shaped containers by a `Ref` suffix. Rename the hoisted `capturedHook` object and all references to `capturedHookRef`; keep the synchronous server-render test behavior unchanged:

```tsx
const { capturedHookRef } = vi.hoisted(() => ({
  capturedHookRef: {
    current: null as null | ReturnType<typeof useKnowledgeReview>,
  },
}))

function HookHarness() {
  capturedHookRef.current = useKnowledgeReview()
  return null
}
```

Run lint immediately. If the rule still rejects the test-only ref container, replace this harness with a DOM-capable `renderHook` test in a separate dependency-approved change; do not suppress `react-hooks/immutability`.

- [ ] **Step 5: Correct Editor callback dependencies**

For callbacks reading `editorRef`, include `editorRef` in the dependency array. Do not disable `react-hooks/exhaustive-deps`.

Example:

```ts
const moveCaretToVoiceEnd = useCallback(() => {
  // existing body
}, [editorRef])
```

- [ ] **Step 6: Require zero warnings**

Change the package script to:

```json
"lint": "eslint . --max-warnings=0"
```

- [ ] **Step 7: Run the frontend quality set**

```bash
npm run lint
npm test
npm run build
```

Expected: all three commands exit 0; 138 or more tests pass.

- [ ] **Step 8: Add lint to CI**

In `.github/workflows/ci.yml`, add after `npm ci` and before tests:

```yaml
      - name: npm run lint
        working-directory: frontend
        run: npm run lint
```

- [ ] **Step 9: Commit**

```bash
git add frontend/src/hooks/usePolishStream.ts \
  frontend/src/hooks/useKnowledgeReview.test.tsx \
  frontend/src/pages/Editor.tsx \
  frontend/package.json \
  .github/workflows/ci.yml
git commit -m "fix(frontend): restore lint quality gate"
```

## Task 2: Add Go Lint, Coverage, and Dead-Code Reports Without Deleting Code

**Files:**
- Create: `backend/.golangci.yml`
- Modify: `backend/go.mod`
- Modify: `backend/go.sum`
- Create: `frontend/knip.json`
- Modify: `frontend/package.json`
- Modify: `frontend/package-lock.json`
- Modify: `frontend/vite.config.ts`
- Modify: `.github/workflows/ci.yml`
- Modify: `docs/qa/code-cleanup-baseline.md`

- [ ] **Step 1: Add the Go lint configuration**

Create `backend/.golangci.yml`:

```yaml
version: "2"

run:
  timeout: 5m

linters:
  default: none
  enable:
    - bodyclose
    - dupl
    - errcheck
    - errorlint
    - gocritic
    - gocyclo
    - gosec
    - govet
    - ineffassign
    - nilerr
    - noctx
    - staticcheck
    - unparam
    - unused
  settings:
    gocyclo:
      min-complexity: 20
```

- [ ] **Step 2: Install pinned Go analysis tools and generate the first reports**

Install golangci-lint `v2.12.2` into the developer tool path, and record `deadcode` as a Go 1.25 tool dependency so its resolved version is pinned in `go.mod`/`go.sum`:

```bash
cd backend
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
go get -tool golang.org/x/tools/cmd/deadcode@latest
golangci-lint run ./... 2>&1 | tee /tmp/inkwords-golangci-baseline.txt
go tool deadcode -test ./... 2>&1 | tee /tmp/inkwords-deadcode-baseline.txt
```

Expected: the first runs may report findings. The `go get -tool` command resolves and pins an exact x/tools version; subsequent runs use `go tool deadcode`. Classify findings into `must-fix`, `legacy`, and `false-positive`; do not add broad path exclusions.

- [ ] **Step 3: Add Knip and Vitest coverage packages**

```bash
cd ../frontend
npm install --save-dev knip@^6 @vitest/coverage-v8@3.2.4
```

Expected: `package.json` and `package-lock.json` change; Vitest and coverage provider remain on the same 3.2.4 version.

- [ ] **Step 4: Add Knip configuration**

Create `frontend/knip.json`:

```json
{
  "$schema": "https://unpkg.com/knip@6/schema.json",
  "entry": ["src/main.tsx", "vite.config.ts"],
  "project": ["src/**/*.{ts,tsx}", "vite.config.ts"]
}
```

Add scripts:

```json
"deadcode": "knip",
"test:coverage": "vitest run --coverage --configLoader runner"
```

- [ ] **Step 5: Configure coverage collection without an arbitrary threshold**

Add to `frontend/vite.config.ts`:

```ts
test: {
  coverage: {
    provider: 'v8',
    reporter: ['text-summary', 'json-summary', 'lcov'],
    include: ['src/**/*.{ts,tsx}'],
    exclude: ['src/**/*.test.{ts,tsx}', 'src/components/ui/**'],
  },
},
```

If TypeScript rejects the `test` field, import `defineConfig` from `vitest/config` rather than duplicating the Vite config.

- [ ] **Step 6: Capture reports**

```bash
npm run deadcode 2>&1 | tee /tmp/inkwords-knip-baseline.txt
npm run test:coverage
```

Expected: coverage generates `coverage/coverage-summary.json`. Knip may report candidates; no candidate is deleted in this task.

- [ ] **Step 7: Add non-destructive CI checks**

Required gates immediately:

```yaml
      - name: go vet
        working-directory: backend
        run: go vet ./...

      - name: backend coverage
        working-directory: backend
        run: go test ./... -coverprofile=coverage.out

      - name: frontend coverage
        working-directory: frontend
        run: npm run test:coverage
```

Keep golangci-lint and Knip in report-only mode until their reviewed baseline is committed. Do not hide exit codes with `|| true` once they become required.

- [ ] **Step 8: Commit**

```bash
git add backend/.golangci.yml backend/go.mod backend/go.sum \
  frontend/knip.json frontend/package.json \
  frontend/package-lock.json frontend/vite.config.ts .github/workflows/ci.yml \
  docs/qa/code-cleanup-baseline.md
git commit -m "chore(quality): add static analysis and coverage baselines"
```

## Task 3: Make Auth Login-State Persistence Fail Explicitly

**Files:**
- Create: `backend/services/core-api/domain/auth/service_test.go`
- Modify: `backend/services/core-api/domain/auth/service.go`

- [ ] **Step 1: Create a repository fake with controllable Save failure**

```go
type fakeRepository struct {
    user      *User
    saveErr   error
    saveCalls int
}

func (f *fakeRepository) GetByEmail(context.Context, string) (*User, error) {
    clone := *f.user
    return &clone, nil
}

func (f *fakeRepository) Save(context.Context, *User) error {
    f.saveCalls++
    return f.saveErr
}
```

Implement the remaining `Repository` methods to fail the test if unexpectedly invoked.

- [ ] **Step 2: Write a failing test for invalid-password state persistence**

```go
func TestLogin_ReturnsInternalErrorWhenFailedAttemptCannotBePersisted(t *testing.T) {
    hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
    require.NoError(t, err)
    repo := &fakeRepository{
        user: &User{ID: uuid.New(), Email: "user@example.com", PasswordHash: string(hash)},
        saveErr: errors.New("database unavailable"),
    }

    _, _, err = NewService(repo).Login(context.Background(), "user@example.com", "wrong-password", "", "")

    require.ErrorContains(t, err, "persist failed login state")
    require.Equal(t, 1, repo.saveCalls)
}
```

- [ ] **Step 3: Run the test and verify it fails**

```bash
cd backend
go test ./services/core-api/domain/auth -run TestLogin_ReturnsInternalErrorWhenFailedAttemptCannotBePersisted -count=1
```

Expected: FAIL because the current code returns only `邮箱或密码错误`.

- [ ] **Step 4: Propagate persistence errors**

Replace both ignored `Save` calls:

```go
if err := s.repo.Save(ctx, user); err != nil {
    return "", nil, fmt.Errorf("persist failed login state: %w", err)
}
```

and:

```go
if err := s.repo.Save(ctx, user); err != nil {
    return "", nil, fmt.Errorf("persist successful login state: %w", err)
}
```

- [ ] **Step 5: Add success-path and reset-failure tests**

Assert that a valid password clears `FailedLoginAttempts` and `LockedUntil`, and that a reset `Save` failure does not issue a JWT.

- [ ] **Step 6: Verify the auth package and core-api**

```bash
go test ./services/core-api/domain/auth ./services/core-api/...
go vet ./services/core-api/...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add backend/services/core-api/domain/auth/service.go \
  backend/services/core-api/domain/auth/service_test.go
git commit -m "fix(auth): propagate login state persistence failures"
```

## Task 4: Harden and Lazy-Load Markdown Rendering

**Files:**
- Create: `frontend/src/components/markdown/MermaidBlock.tsx`
- Create: `frontend/src/components/markdown/MermaidBlock.test.tsx`
- Create: `frontend/src/components/markdown/CodeBlock.tsx`
- Modify: `frontend/src/components/MarkdownEngine.tsx`
- Modify: `frontend/src/components/editor/EditorBody.tsx`
- Modify: `frontend/src/components/generator/GeneratorStatus.tsx`

- [ ] **Step 1: Write a malicious Mermaid regression test**

The test must render a Mermaid diagram containing a link or HTML payload and assert that the resulting container contains no `script`, inline event handler, `javascript:` URL, or `foreignObject`.

```tsx
expect(container.querySelector('script')).toBeNull()
expect(container.innerHTML).not.toMatch(/onerror\s*=|onclick\s*=|javascript:|foreignObject/i)
```

- [ ] **Step 2: Run the test against current behavior**

```bash
cd frontend
npx vitest run src/components/markdown/MermaidBlock.test.tsx --configLoader runner
```

Expected: FAIL or prove that the test environment cannot execute Mermaid. If Mermaid needs a browser DOM, move this one test to the project's browser-capable test lane; do not weaken the assertion.

- [ ] **Step 3: Extract MermaidBlock and enable strict mode**

Move Mermaid initialization/rendering out of `MarkdownEngine.tsx` and configure:

```ts
mermaid.initialize({
  startOnLoad: false,
  securityLevel: 'strict',
  suppressErrorRendering: true,
})
```

Keep the existing style-line removal only if tests prove it is still required.

- [ ] **Step 4: Remove fallback HTML interpolation**

Replace fallback `innerHTML` with React-rendered markup:

```tsx
return renderError
  ? <pre><code>{chart}</code></pre>
  : <div ref={containerRef} />
```

Only Mermaid-produced SVG may enter the container, under strict mode and the malicious-input regression test.

- [ ] **Step 5: Lazy-load heavy renderers**

```tsx
const MermaidBlock = lazy(() => import('./markdown/MermaidBlock'))
const CodeBlock = lazy(() => import('./markdown/CodeBlock'))
```

Wrap each with a small `Suspense` fallback. `MarkdownEngine.tsx` must no longer statically import `mermaid` or `react-syntax-highlighter`.

- [ ] **Step 6: Verify tests, lint, and bundle output**

```bash
npm run lint
npm test
npm run build 2>&1 | tee /tmp/inkwords-build-after-markdown.txt
```

Expected: all commands pass; Mermaid and syntax highlighter appear in lazy chunks; the main entry gzip size is lower than the 586 KB baseline.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/components/MarkdownEngine.tsx \
  frontend/src/components/markdown \
  frontend/src/components/editor/EditorBody.tsx \
  frontend/src/components/generator/GeneratorStatus.tsx
git commit -m "fix(markdown): isolate secure lazy renderers"
```

## Task 5: Consolidate Parser Infrastructure to One Owner

**Decision:** `shared/platform/parser` remains the single platform implementation because both `core-api` project analysis and `parser-service` consume it. Delete service-owned and legacy exact copies only after imports/tests move.

**Files:**
- Modify: `backend/services/parser-service/app/bootstrap/bootstrap.go`
- Modify: `backend/services/parser-service/domain/parse/service.go`
- Modify: parser-service parse tests importing service-owned parser infra
- Delete: `backend/services/parser-service/infra/parser/*.go`
- Delete later: `backend/internal/infra/parser/*.go`
- Modify: `backend/services/architecture_test.go`

- [ ] **Step 1: Prove shared and service-owned parser behavior is identical**

```bash
cd backend
go test ./shared/platform/parser ./services/parser-service/infra/parser -count=1
```

Expected: both packages pass the same parser/fetcher tests.

- [ ] **Step 2: Add an architecture test forbidding parser-service's duplicate infra package**

```go
func TestParserServiceUsesSharedParserPlatform(t *testing.T) {
    assertTreeDoesNotContainImport(t, "parser-service", "inkwords-backend/services/parser-service/infra/parser")
}
```

Use or introduce one shared file-walk helper rather than copying another `WalkDir` loop.

- [ ] **Step 3: Run the new test and verify it fails**

```bash
go test ./services -run TestParserServiceUsesSharedParserPlatform -count=1
```

Expected: FAIL on current imports.

- [ ] **Step 4: Switch parser-service imports**

Replace:

```go
parserinfra "inkwords-backend/services/parser-service/infra/parser"
```

with:

```go
parserinfra "inkwords-backend/shared/platform/parser"
```

Update `domain/parse/service.go`, bootstrap, and tests. No parser behavior changes are allowed in this PR.

- [ ] **Step 5: Verify service behavior before deletion**

```bash
go test ./shared/platform/parser ./services/parser-service/... ./services -count=1
go test -race ./services/parser-service/domain/parse -count=1
```

Expected: PASS.

- [ ] **Step 6: Delete the service-owned duplicate package**

Delete `backend/services/parser-service/infra/parser/` only after `rg` returns no import:

```bash
rg -n "services/parser-service/infra/parser" backend || true
```

- [ ] **Step 7: Determine whether legacy internal parser remains reachable**

```bash
go list -deps ./cmd/... | rg '^inkwords-backend/internal/infra/parser$' || true
go tool deadcode -test ./... | rg 'internal/infra/parser' || true
```

If `cmd/server` still reaches it, rewire `cmd/server` to `shared/platform/parser` first. Then delete `backend/internal/infra/parser/` and its duplicate tests.

- [ ] **Step 8: Verify and commit**

```bash
go test ./...
go vet ./...
git diff --check
git add backend/services/parser-service backend/shared/platform/parser \
  backend/internal/infra/parser backend/services/architecture_test.go
git commit -m "refactor(parser): consolidate platform implementation"
```

## Task 6: Consolidate DeepSeek Infrastructure to shared/platform/llm

**Decision:** `shared/platform/llm` is the single provider implementation. Service/application packages may depend on a narrow interface, but no second DeepSeek client implementation may remain.

**Files:**
- Modify: `backend/internal/service/*.go` imports as an intermediate migration
- Modify: related internal service tests
- Delete: `backend/internal/infra/llm/*.go`
- Modify: `backend/services/architecture_test.go`

- [ ] **Step 1: Prove the current implementations and tests are exact duplicates**

```bash
cd backend
cmp internal/infra/llm/deepseek.go shared/platform/llm/deepseek.go
cmp internal/infra/llm/output_sanitize.go shared/platform/llm/output_sanitize.go
go test ./internal/infra/llm ./shared/platform/llm -count=1
```

Expected: `cmp` exits 0 and both test packages pass.

- [ ] **Step 2: Add the architecture rule first**

Extend `TestServicesUseSharedHTTPRuntimeContract` or add a focused test that rejects `inkwords-backend/internal/infra/llm` imports outside a temporary allowlist. The allowlist may contain only `internal/service` during this PR and must be removed in Task 10.

- [ ] **Step 3: Replace all internal service imports**

Replace:

```go
"inkwords-backend/internal/infra/llm"
```

with:

```go
llm "inkwords-backend/shared/platform/llm"
```

in generator, decomposition, prompt profile, quality pipeline, and Obsidian export files and their tests. Keep public type names unchanged through the import alias.

- [ ] **Step 4: Run the affected high-value tests**

```bash
go test ./internal/service/... -run 'Generator|Decomposition|PromptProfile|SeriesQuality|Obsidian' -count=1
go test ./shared/platform/llm -count=1
```

Expected: PASS with no request-shape, usage, cache-hit, reasoning, or sanitization regression.

- [ ] **Step 5: Delete duplicate LLM infra and tests**

```bash
rg -n "internal/infra/llm" backend || true
```

Expected before deletion: no production/test imports except architecture-test strings. Delete `backend/internal/infra/llm/`.

- [ ] **Step 6: Verify and commit**

```bash
go test ./...
go vet ./...
git diff --check
git add backend/internal/service backend/internal/infra/llm \
  backend/shared/platform/llm backend/services/architecture_test.go
git commit -m "refactor(llm): use shared DeepSeek platform client"
```

## Task 7: Handle Worker Delivery and Export I/O Errors

**Files:**
- Modify: `backend/services/parser-service/domain/parse/task_consumer.go`
- Modify: `backend/services/parser-service/domain/parse/task_consumer_test.go`
- Modify: `backend/services/export-service/domain/export/consumer.go`
- Modify: `backend/services/export-service/domain/export/worker_test.go`
- Modify: `backend/services/llm-stream/cmd/main.go`
- Add focused llm-stream consumer tests in a service-owned package
- Modify: `backend/services/export-service/domain/export/handler.go`
- Create: `backend/services/export-service/domain/export/handler_test.go`

- [ ] **Step 1: Introduce a delivery acknowledgment interface in each consumer package**

```go
type deliveryAcknowledger interface {
    Ack(multiple bool) error
    Nack(multiple bool, requeue bool) error
}
```

Keep RabbitMQ's delivery behind this interface so failures can be tested without a broker.

- [ ] **Step 2: Write tests for Ack/Nack failures**

Cover these exact cases:

- successful work + Ack failure returns/logs a consumer error and does not pretend success;
- failed work + Nack failure records both the work error and the Nack error;
- malformed non-retryable payload is Acked once;
- transient work failure is Nacked once with `requeue=true`.

- [ ] **Step 3: Run tests and verify failure**

```bash
cd backend
go test ./services/parser-service/domain/parse ./services/export-service/domain/export -run 'Ack|Nack|Delivery' -count=1
```

Expected: new tests fail while errors are ignored.

- [ ] **Step 4: Return or log acknowledgment errors with task/request metadata**

Use wrapped errors:

```go
if err := delivery.Ack(false); err != nil {
    return fmt.Errorf("ack task %s: %w", taskID, err)
}
```

Do not log raw task payloads or generated content.

- [ ] **Step 5: Make ZIP writes fail the response**

Replace `continue` and ignored `Write` errors with immediate failure before the archive is finalized:

```go
file, err := zipWriter.Create(filename)
if err != nil {
    return fmt.Errorf("create zip entry %q: %w", filename, err)
}
if _, err := io.WriteString(file, body); err != nil {
    return fmt.Errorf("write zip entry %q: %w", filename, err)
}
```

Move archive construction into a testable function that writes to an `io.Writer`; the HTTP handler maps internal errors to a stable generic message.

- [ ] **Step 6: Check io.Copy and cleanup errors separately**

An `io.Copy` failure is a request failure and must be recorded. Failure to remove a temporary file after a successful response is cleanup-only and should be logged, not sent after headers are committed.

- [ ] **Step 7: Verify race, package, and full tests**

```bash
go test -race ./services/parser-service/domain/parse ./services/export-service/domain/export ./services/llm-stream/... -count=1
go test ./...
go vet ./...
```

- [ ] **Step 8: Commit worker and export changes separately**

```bash
git add backend/services/parser-service backend/services/export-service/domain/export/consumer.go
git commit -m "fix(workers): surface delivery acknowledgement failures"

git add backend/services/export-service/domain/export/handler.go \
  backend/services/export-service/domain/export/handler_test.go
git commit -m "fix(export): propagate archive and response write failures"
```

## Task 8: Move Characterization Coverage to Production Service Packages

**Files:**
- Create tests under:
  - `backend/services/core-api/domain/auth/`
  - `backend/services/core-api/domain/blog/`
  - `backend/services/core-api/domain/project/`
  - `backend/services/core-api/domain/user/`
  - `backend/services/llm-stream/domain/stream/`
- Modify: `docs/qa/code-cleanup-baseline.md`

- [ ] **Step 1: Generate package-level coverage**

```bash
cd backend
go test ./services/core-api/... ./services/llm-stream/... -coverprofile=/tmp/services-before.cover
go tool cover -func=/tmp/services-before.cover
```

- [ ] **Step 2: Port tests by behavior, not by copying package names**

For each legacy test, recreate it against the production package's public constructor and interface. Required behavior matrix:

| Package | Required tests before deletion |
|---|---|
| core-api/auth | register conflict, invalid password, lockout, persistence failure, OAuth provider failure |
| core-api/blog | ownership filtering, draft creation, batch delete, update not-found |
| core-api/project | quota failure, Git fetch failure, outline failure, ZIP parse, ordinary file parse |
| core-api/user | profile read/update, prompt setting merge failure, avatar validation |
| llm-stream/stream | single generation, series generation, continue, polish, analyze, scan, cancellation, task result building |

- [ ] **Step 3: Keep tests deterministic**

Use fake repositories, `httptest.Server`, fake LLM/provider interfaces, temporary directories, and in-memory SQLite only where repository behavior is the subject. No test may call GitHub, DeepSeek, Obsidian, RabbitMQ, or PostgreSQL over the network.

- [ ] **Step 4: Enforce package floors only after the tests exist**

Initial targets:

- each changed production domain package: at least 70% statement coverage;
- auth state machine and task result builders: all success and error branches explicitly asserted;
- global backend coverage may not fall below 37.8% before legacy deletion changes the denominator.

- [ ] **Step 5: Run the production package suite uncached**

```bash
go test ./services/core-api/... ./services/llm-stream/... -count=1 -coverprofile=/tmp/services-after.cover
go tool cover -func=/tmp/services-after.cover
```

- [ ] **Step 6: Commit one domain at a time**

```bash
git commit -m "test(auth): characterize production auth domain"
git commit -m "test(blog): characterize production blog domain"
git commit -m "test(project): characterize production project domain"
git commit -m "test(user): characterize production user domain"
git commit -m "test(stream): characterize production generation domain"
```

## Task 9: Remove core-api's Dependency on internal/service

**Files:**
- Modify: `backend/services/core-api/app/bootstrap/bootstrap.go`
- Modify: `backend/services/core-api/domain/user/service.go`
- Create: `backend/services/core-api/app/projectanalysis/service.go`
- Move/adapt analysis-only helpers from `backend/internal/service/decomposition_analyze*.go` and `decomposition_scan.go`
- Create tests under `backend/services/core-api/app/projectanalysis/`
- Modify: `backend/services/architecture_test.go`

- [ ] **Step 1: Add CheckQuota to the service-owned user domain**

Implement quota checking through `services/core-api/domain/user.Repository`; preserve the current token-limit behavior with a fake-repository test.

- [ ] **Step 2: Introduce a project-analysis application service**

Define only the port required by `domain/project`:

```go
type Service struct {
    llm      LLMClient
    prompts PromptResolver
}

func (s *Service) ScanProjectModules(ctx context.Context, gitURL string) ([]project.ModuleCard, error)
func (s *Service) GenerateOutline(ctx context.Context, source string, mode prompt.ScenarioMode) (project.OutlineResult, error)
```

Use `shared/platform/llm` behind a narrow interface. Move only scan/outline logic; do not move series generation into core-api.

- [ ] **Step 3: Write adapter tests before switching bootstrap**

Assert module mapping, outline mapping, invalid provider response, and context cancellation.

- [ ] **Step 4: Switch bootstrap wiring**

Remove:

```go
"inkwords-backend/internal/service"
```

Wire `projectanalysis.Service` and `userdomain.Service` directly into `projectdomain.NewService`.

- [ ] **Step 5: Tighten architecture tests**

Add:

```go
func TestCoreAPIDoesNotImportLegacyInternalPackages(t *testing.T) {
    assertTreeDoesNotContainImport(t, "core-api", "inkwords-backend/internal/")
}
```

- [ ] **Step 6: Verify**

```bash
go test ./services/core-api/... ./services -count=1
go list -deps ./services/core-api/... | rg '^inkwords-backend/internal' || true
```

Expected: tests pass and the dependency query prints nothing.

- [ ] **Step 7: Commit**

```bash
git add backend/services/core-api backend/services/architecture_test.go
git commit -m "refactor(core-api): remove legacy service dependency"
```

## Task 10: Move Generation Application Logic Under llm-stream Ownership

**Files:**
- Create: `backend/services/llm-stream/app/generation/`
- Modify: `backend/services/llm-stream/app/bootstrap/bootstrap.go`
- Modify: `backend/services/llm-stream/domain/stream/service.go`
- Move generation/decomposition tests from legacy package to service-owned package
- Modify: `backend/services/architecture_test.go`

- [ ] **Step 1: Replace concrete legacy types with stream-domain ports**

Define interfaces matching only the calls in `domain/stream/service.go`:

```go
type Generator interface {
    GenerateBlogStreamWithProfile(context.Context, uuid.UUID, string, string, prompt.ScenarioMode, string, prompt.PromptProfile, chan<- string, chan<- error)
    GeneratePolishDraftStream(context.Context, string, string, chan<- string, chan<- error)
    BuildGenerateSingleTaskResult(context.Context, string, string) (GenerateSingleResult, error)
}

type QuotaChecker interface {
    CheckQuota(uuid.UUID) error
}
```

Define a separate `Decomposition` port for series, continue, analyze, scan, and task-result methods. Update `Service` to depend on interfaces, not `*internal/service.*`.

- [ ] **Step 2: Run stream tests**

```bash
go test ./services/llm-stream/domain/stream -count=1
```

Expected: PASS with fakes; domain package no longer imports `internal/service`.

- [ ] **Step 3: Move single-generation and polish application code first**

Move `generator.go`, message builders, prompt resolution, usage aggregation, and their tests into `services/llm-stream/app/generation`. Replace `internal/domain/blog/contracts` with service-owned persistence interfaces already implemented by `streamdomain.NewGeneratedBlogPersistence`.

- [ ] **Step 4: Move analysis and series code in bounded commits**

Use this order, running focused tests after each move:

1. prompt requirements and profile resolution;
2. scan and analyze helpers;
3. single generation and polish;
4. continue generation;
5. series outline and series quality pipeline;
6. task-result usage aggregation.

Each move must preserve the existing DeepSeek request shape, `thinking`, token caps, cache usage fields, and `task_only` persistence semantics.

- [ ] **Step 5: Switch llm-stream bootstrap**

Construct the new generation application services with `shared/platform/llm`, service-owned persistence implementations, and service-owned quota checking. Remove the `internal/service` import.

- [ ] **Step 6: Tighten the architecture gate**

```go
func TestLLMStreamDoesNotImportLegacyInternalPackages(t *testing.T) {
    assertTreeDoesNotContainImport(t, "llm-stream", "inkwords-backend/internal/")
}
```

- [ ] **Step 7: Verify targeted, race, and full suites**

```bash
go test ./services/llm-stream/... ./shared/platform/llm -count=1
go test -race ./services/llm-stream/... -count=1
go test ./...
go vet ./...
```

- [ ] **Step 8: Commit each migration slice separately**

Use commit subjects:

```text
refactor(stream): introduce generation application ports
refactor(stream): move prompt and analysis ownership
refactor(stream): move single generation and polish ownership
refactor(stream): move continue and series generation ownership
refactor(stream): remove legacy service dependency
```

## Task 11: Rewire the Aggregate Development Entrypoint and Delete Legacy Code

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/transport/http/v1/routes.go` or delete after route migration
- Delete only after reachability proof: obsolete packages under `backend/internal/`
- Modify: `README.md`
- Modify: `docs/runbooks/core-blog-task-boundary.md`

- [ ] **Step 1: Decide and encode the aggregate entrypoint behavior**

Keep `cmd/server` as a local-development compatibility entrypoint, but wire it from service-owned routers/handlers. It must not own a second copy of business logic.

- [ ] **Step 2: Add an entrypoint architecture test**

Reject imports from legacy business packages in `cmd/server`; allow service-owned and shared runtime packages only.

- [ ] **Step 3: Rewire one route group at a time**

Order: auth/user/blog -> project -> task -> stream -> parse/export/review. After each group:

```bash
go test ./cmd/server ./services/... -count=1
```

- [ ] **Step 4: Produce reachability and dependency reports**

```bash
go list -deps ./cmd/... ./services/... | rg '^inkwords-backend/internal'
go tool deadcode -test ./... > /tmp/inkwords-deadcode-final.txt
```

Every candidate package must satisfy all conditions:

- no import in `go list -deps` output;
- reported unreachable or absent from all entrypoint dependency graphs;
- no build-tag, generator, script, migration, reflection, or documentation contract depends on it;
- production service characterization tests cover the replacement.

- [ ] **Step 5: Delete one package group per commit**

Recommended order:

1. duplicate `internal/infra/*` packages already consolidated to `shared/platform`;
2. legacy domain copies whose service-owned equivalents are tested;
3. legacy transport routes;
4. legacy `internal/service` only after core-api, llm-stream, and `cmd/server` no longer import it.

- [ ] **Step 6: Verify after every deletion commit**

```bash
go test ./...
go vet ./...
golangci-lint run ./...
git diff --check
```

Do not batch multiple package groups if one verification run fails.

## Task 12: Split Large Files Only Along Proven Responsibility Boundaries

**Files:**
- Refactor candidates:
  - `backend/services/llm-stream/domain/stream/handler.go`
  - `frontend/src/components/Sidebar.tsx`
  - `frontend/src/pages/Editor.tsx`
  - `frontend/src/store/streamStore.ts`
- Preserve public APIs and existing tests.

- [ ] **Step 1: Measure functions/components, not only file lines**

Use golangci `gocyclo`/`gocognit` and ESLint complexity reports to identify functions causing risk. A file over 400 lines is a review trigger, not automatic proof of bad design.

- [ ] **Step 2: Split the stream handler by operation**

Target structure:

```text
domain/stream/handler.go             shared Handler and helpers
domain/stream/handler_generate.go    single/series generation
domain/stream/handler_continue.go    continue
domain/stream/handler_polish.go      polish
domain/stream/handler_analyze.go     analyze/scan
domain/stream/handler_events.go      SSE event mapping/flush
```

Move code without behavior changes, run handler tests, then commit. Do not combine the split with new error semantics.

- [ ] **Step 3: Split Editor by behavior hooks**

Extract continuation streaming and save lifecycle into focused hooks while keeping `Editor` as composition:

```text
hooks/useContinueStream.ts
hooks/useEditorAutosave.ts
pages/Editor.tsx
```

Add hook tests before moving code.

- [ ] **Step 4: Split Sidebar by visible sections**

Keep selection state in the existing owner; extract presentational sections only. Avoid introducing a new global store.

- [ ] **Step 5: Verify after each mechanical split**

```bash
cd backend && go test ./services/llm-stream/domain/stream -count=1
cd ../frontend && npm run lint && npm test && npm run build
```

## Task 13: Finalize Quality Gates, Documentation, and Smoke Validation

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`
- Modify: `docs/qa/code-cleanup-baseline.md`
- Modify: relevant runbooks

- [ ] **Step 1: Make reviewed static checks blocking**

CI must block on:

```text
go test ./...
go vet ./...
golangci-lint run ./...
npm run lint
npm run deadcode
npm run test:coverage
npm run build
docker compose config
microservices smoke
```

`govulncheck ./...` may run nightly and before releases because it needs current vulnerability data; any reachable high-impact finding blocks release.

- [ ] **Step 2: Add coverage ratchets**

- Store backend and frontend coverage summaries as CI artifacts.
- Reject new code that reduces a touched production package below its recorded post-cleanup baseline.
- Require at least 70% statement coverage for the migrated service-owned domain packages.
- Do not enforce an artificial 80% global threshold while untested bootstrap/infrastructure code remains.

- [ ] **Step 3: Add a bundle budget**

Record the post-lazy-load main entry gzip size and fail CI if it grows by more than 5% without an explicit reviewed update to the budget file.

- [ ] **Step 4: Update documentation from actual manifests**

Correct React 18 references to React 19 and document the final ownership model:

```text
services/*     business/application ownership
shared/kernel stable cross-service value types and contracts
shared/platform cross-service infrastructure adapters
internal/*     no production business logic after cleanup
```

- [ ] **Step 5: Run final local verification**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./... -count=1 -coverprofile=/tmp/inkwords-final.cover
go tool cover -func=/tmp/inkwords-final.cover | tail -n 1
go vet ./...
golangci-lint run ./...
go tool deadcode -test ./...

cd ../frontend
npm run lint
npm run deadcode
npm run test:coverage
npm run build

cd ..
OBSIDIAN_VAULT_PATH=/tmp/obsidian-vault docker compose --env-file backend/.env.example config
git diff --check
git status --short --branch
```

- [ ] **Step 6: Run Docker smoke validation when Docker credentials/environment are available**

```bash
mkdir -p /tmp/obsidian-vault/wiki
OBSIDIAN_VAULT_PATH=/tmp/obsidian-vault \
DEEPSEEK_API_KEY=smoke-placeholder \
JWT_SECRET=smoke-placeholder-secret \
OBSIDIAN_REST_API_KEY=smoke-placeholder \
docker compose --env-file backend/.env.example up -d --build

curl --fail http://localhost/api/v1/ping
docker compose --env-file backend/.env.example ps
docker compose --env-file backend/.env.example down -v
```

Expected: all application containers become healthy and the gateway ping succeeds. Use non-production disposable data only because `down -v` removes the smoke volumes.

- [ ] **Step 7: Update the baseline document with before/after evidence**

Include:

- duplicate group/file counts;
- remaining `services -> internal` dependencies;
- backend and frontend coverage;
- lint/dead-code finding counts;
- main bundle gzip size;
- exact verification commands and dates.

- [ ] **Step 8: Final documentation commit**

```bash
git add README.md docs/qa/code-cleanup-baseline.md docs/runbooks .github/workflows/ci.yml
git commit -m "docs(governance): record cleanup ownership and quality gates"
```

## 4. Stop Conditions and Rollback Rules

Stop the current PR and investigate if any condition occurs:

- an external response body, status code, route, SSE event, or RabbitMQ payload changes unexpectedly;
- a migrated production package has lower behavior coverage than the legacy package it replaces;
- `go list -deps` still shows a supposedly deleted owner through another entrypoint;
- parser/LLM output fixtures change without a product requirement;
- main bundle grows after lazy loading;
- a cleanup requires a database migration or public contract change.

Rollback rules:

- configuration, tests, migration, and deletion remain separate commits;
- revert only the smallest failing commit;
- never restore deleted code by copying it into a new location; revert the deletion commit and reassess ownership;
- do not merge a PR with skipped quality gates unless the skip is itself reviewed and time-bounded.

## 5. Definition of Done

- Frontend lint, tests, coverage, dead-code scan, and build pass in CI.
- Backend tests, vet, golangci-lint, coverage, and architecture tests pass in CI.
- Auth persistence and worker delivery failures are covered and no longer silently ignored.
- Mermaid untrusted input has a regression test; heavy renderers are lazy chunks.
- Parser and DeepSeek each have one platform implementation.
- `services/core-api` and `services/llm-stream` no longer import `internal/*`.
- `cmd/server` is either service-owned wiring or explicitly retired.
- Exact duplicate groups are reduced with a recorded before/after count.
- Documentation matches React 19 and the final service/shared ownership model.
- Full Docker smoke validation passes or the exact external environment blocker is recorded.

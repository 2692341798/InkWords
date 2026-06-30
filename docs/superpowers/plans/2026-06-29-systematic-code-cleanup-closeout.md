# InkWords Systematic Cleanup Closeout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将已经完成主体迁移的 `codex/governance-baseline` 分支收敛为一组真实通过的质量门禁，补齐关键错误处理和生产生成链路测试，删除已不可达的 legacy 业务代码，并形成可推送、可审查、可回滚的 Draft PR。

**Architecture:** 本计划只处理当前审计确认的尾项，不重复 parser/LLM 所有权迁移。收尾顺序固定为“工作区卫生 -> Knip 变绿 -> 错误处理 -> 生成链路测试 -> legacy 删除 -> 阈值门禁 -> 发布验收”；`internal/infra/db` 与 `internal/model` 本轮保留，因为 `shared/platform/postgres` 仍依赖它们。

**Tech Stack:** Go 1.25.4, Gin, GORM, RabbitMQ, React 19.2, TypeScript 5.9, Vite 8, Vitest 3.2, Knip 6, golangci-lint 2, Docker Compose, GitHub Actions

---

## Execution Status (2026-06-29)

Local closeout implementation is complete. This block is the authoritative execution ledger for the checklist below.

| Task | Status | Evidence |
| --- | --- | --- |
| 0 | Complete | `d9ec0af` |
| 1 | Complete | `177cb89`; Knip, lint, 144 tests, coverage, build pass |
| 2 | Complete | `3f12888`, `13e6f5d`; focused, race, full tests and vet pass |
| 3 | Complete | `20f464e`; generation coverage 53.4%, race/full tests pass |
| 4 | Complete | `389e93f`, `006f5ea`; only `internal/model` and `internal/infra/db` remain; duplicate scan `0 0` |
| 5 | Complete | `d492a26`; backend 36.8%/53.4%, frontend 38.99/65.65/54.51/38.99, bundle 1,104,462/333,288 bytes |
| 6 local | Complete | Compose config passes; clean Docker build passes after `437b35e`; all services healthy; `/api/v1/ping` returns `pong` |
| 6 publish | Pending authorization | No push, Draft PR, or remote CI monitoring performed |

Notes:

- Local `golangci-lint` could not run because the binary is not installed. CI is pinned to `golangci-lint-action@v8` with v2.12.2.
- Disposable smoke containers and volumes remain running/preserved. `docker compose down -v` was not run because it requires separate destructive approval.
- The first Docker clean build exposed an undeclared `@testing-library/dom` peer dependency; `437b35e` makes it explicit and the rebuilt image passed.

## Current Closeout Baseline

- Current commit: `1b3adf3f5fcb708f8ba398bc21c3050269b6a89b`.
- Branch: `codex/governance-baseline`, 12 commits ahead of `origin/codex/governance-baseline`.
- No pull request exists for the branch.
- Backend: full tests pass outside the socket-restricted sandbox; `go vet ./...` passes; statement coverage is 35.4%.
- Frontend: lint passes; 42 files / 144 tests pass; statement coverage is 38.99%; build passes.
- Blocking defect: `npm run deadcode` fails with 5 CSS-loaded dependencies, 1 unused devDependency, 13 unused exports, 11 unused exported types, and 3 redundant config patterns.
- Remaining ignored errors: llm-stream delivery Ack/Nack and core-api task download `io.Copy`.
- `services/llm-stream/app/generation` has 0% statement coverage.
- Remaining legacy business code: 48 Go files under `backend/internal/service`, legacy domain/transport packages, and four obsolete standalone command wrappers.
- Remaining duplicate production files: 12 groups / 24 files.
- Main frontend bundle: about 1.10 MB / 336 KB gzip; Vite still emits a chunk-size warning.

## Scope Boundary

**Delete in this closeout after verification:**

- `backend/cmd/core-api/`
- `backend/cmd/llm-stream/`
- `backend/cmd/parser-service/`
- `backend/cmd/export-service/`
- `backend/internal/service/`
- `backend/internal/domain/`
- `backend/internal/transport/`
- `backend/internal/prompt/`
- `backend/internal/infra/cache/`
- `backend/internal/infra/mq/`

**Keep in this closeout:**

- `backend/cmd/server/` as the supported aggregate local-development entrypoint.
- `backend/internal/infra/db/` because `shared/platform/postgres` still delegates connection and migration initialization to it.
- `backend/internal/model/` because database migration and the maintenance script still use the shared persistence models.
- `backend/scripts/cleanup.go`.

## Commit Sequence

1. `chore(repo): ignore generated coverage artifacts`
2. `chore(frontend): make knip dead-code gate actionable`
3. `fix(stream): surface generation delivery acknowledgement failures`
4. `fix(task): surface download stream failures`
5. `test(stream): cover service-owned generation application`
6. `refactor(backend): remove obsolete standalone commands`
7. `refactor(backend): remove unreachable legacy business packages`
8. `chore(ci): enforce coverage and bundle budgets`
9. `docs(governance): close systematic cleanup plan`

## Task 0: Normalize the Worktree Without Deleting User Files

**Files:**
- Modify: `.gitignore`
- Add later: `docs/superpowers/plans/2026-06-29-systematic-code-cleanup.md`
- Add later: `docs/superpowers/plans/2026-06-29-systematic-code-cleanup-closeout.md`

- [ ] **Step 1: Confirm the only untracked paths**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git status --short --branch
```

Expected:

```text
## codex/governance-baseline
?? docs/superpowers/plans/2026-06-29-systematic-code-cleanup.md
?? docs/superpowers/plans/2026-06-29-systematic-code-cleanup-closeout.md
?? frontend/coverage/
```

Stop if any other path appears; do not clean or reset unrelated user changes.

- [ ] **Step 2: Ignore generated frontend coverage reports**

Add to the repository `.gitignore`:

```gitignore
# Generated frontend test coverage
frontend/coverage/
```

Do not delete the existing directory. Once ignored, it no longer dirties `git status` and can remain available for local inspection.

- [ ] **Step 3: Verify ignore behavior**

```bash
git check-ignore -v frontend/coverage/coverage-summary.json
git status --short
```

Expected: `git check-ignore` points to the new rule; only the two plan documents and `.gitignore` remain visible.

- [ ] **Step 4: Commit only the ignore rule**

```bash
git add .gitignore
git commit -m "chore(repo): ignore generated coverage artifacts"
```

## Task 1: Make the Knip Gate Green Without Hiding Real Dead Code

**Files:**
- Modify: `frontend/knip.json`
- Modify: `frontend/package.json`
- Modify: `frontend/package-lock.json`
- Modify: `frontend/src/lib/authTokenStore.ts`
- Modify: `frontend/src/pages/homeEntryViewState.ts`
- Modify: `frontend/src/services/generationTasks.ts`
- Modify: `frontend/src/services/project.ts`
- Modify: `frontend/src/services/review.ts`
- Modify: `frontend/src/services/user.ts`
- Modify: `frontend/src/store/streamStore.ts`
- Preserve: `frontend/src/components/ui/**`

- [ ] **Step 1: Capture the current failure**

```bash
cd frontend
npm run deadcode
```

Expected: FAIL with the known 5 unused dependencies, 1 unused devDependency, 13 exports, 11 exported types, and config hints.

- [ ] **Step 2: Configure only evidence-backed Knip exceptions**

Replace `frontend/knip.json` with:

```json
{
  "$schema": "https://unpkg.com/knip@6/schema.json",
  "ignore": ["src/components/ui/**"],
  "ignoreDependencies": [
    "@fontsource-variable/geist",
    "@tailwindcss/typography",
    "shadcn",
    "tailwindcss",
    "tw-animate-css"
  ]
}
```

Why these exceptions are allowed:

- all five dependencies are referenced by `src/index.css` through `@import` or `@plugin`;
- `src/components/ui/**` is generated registry surface, where exporting a wider component family is intentional;
- no application/service/store path is ignored.

- [ ] **Step 3: Remove the truly unused test dependency**

```bash
npm uninstall --save-dev @testing-library/jest-dom
```

Expected: only `package.json` and `package-lock.json` change. Existing tests do not import the package.

- [ ] **Step 4: Make module-private constants private**

In `frontend/src/lib/authTokenStore.ts`, change:

```ts
export const AUTH_TOKEN_STORAGE_KEY = 'token'
export const AUTH_TOKEN_CHANGE_EVENT = 'inkwords:auth-token-changed'
```

to:

```ts
const AUTH_TOKEN_STORAGE_KEY = 'token'
const AUTH_TOKEN_CHANGE_EVENT = 'inkwords:auth-token-changed'
```

Keep the existing exported storage functions unchanged.

- [ ] **Step 5: Remove unnecessary `export` keywords from file-local types**

Make these declarations module-private; their references are currently confined to the same source file:

```text
pages/homeEntryViewState.ts: HomeEntryTargetView
services/generationTasks.ts: SeriesChapter
services/project.ts: ProjectArchiveSummary, ParseProjectResponse, CreateParseTaskResponse, TaskSnapshotResponse
services/review.ts: SessionOutline, ReviewFeedback
services/user.ts: UserTechStackStat
store/streamStore.ts: ChapterPhase, ChapterUsage
```

Example:

```ts
interface SeriesChapter {
  id: string
  title: string
  // keep existing fields unchanged
}
```

Do not change field names or API payload structures.

- [ ] **Step 6: Verify dead-code, type, test, and build gates**

```bash
npm run deadcode
npm run lint
npm test
npm run test:coverage
npm run build
```

Expected: all commands exit 0; at least 144 tests pass; current payload-shape tests remain green.

- [ ] **Step 7: Commit**

```bash
git add frontend/knip.json frontend/package.json frontend/package-lock.json \
  frontend/src/lib/authTokenStore.ts \
  frontend/src/pages/homeEntryViewState.ts \
  frontend/src/services/generationTasks.ts \
  frontend/src/services/project.ts \
  frontend/src/services/review.ts \
  frontend/src/services/user.ts \
  frontend/src/store/streamStore.ts
git commit -m "chore(frontend): make knip dead-code gate actionable"
```

## Task 2: Finish Generation Ack/Nack and Download Stream Error Handling

**Files:**
- Create: `backend/services/llm-stream/cmd/delivery.go`
- Create: `backend/services/llm-stream/cmd/delivery_test.go`
- Modify: `backend/services/llm-stream/cmd/main.go`
- Create: `backend/services/core-api/domain/task/download_writer.go`
- Create: `backend/services/core-api/domain/task/download_writer_test.go`
- Modify: `backend/services/core-api/domain/task/download_handler.go`

- [ ] **Step 1: Write failing delivery acknowledgement tests**

Create `delivery_test.go` with a fake:

```go
type fakeAcknowledger struct {
    ackErr  error
    nackErr error
    acked   int
    nacked  int
}

func (f *fakeAcknowledger) Ack(bool) error {
    f.acked++
    return f.ackErr
}

func (f *fakeAcknowledger) Nack(bool, bool) error {
    f.nacked++
    return f.nackErr
}
```

Required assertions:

```go
func TestAckDelivery_ReturnsWrappedFailure(t *testing.T) {
    ack := &fakeAcknowledger{ackErr: errors.New("channel closed")}
    err := ackDelivery(ack, "malformed generation message")
    require.ErrorContains(t, err, "malformed generation message")
    require.ErrorContains(t, err, "channel closed")
    require.Equal(t, 1, ack.acked)
}

func TestNackDelivery_ReturnsWrappedFailure(t *testing.T) {
    ack := &fakeAcknowledger{nackErr: errors.New("channel closed")}
    err := nackDelivery(ack, uuid.MustParse("11111111-1111-1111-1111-111111111111"))
    require.ErrorContains(t, err, "nack generation task")
    require.Equal(t, 1, ack.nacked)
}
```

- [ ] **Step 2: Run tests and verify they fail to compile**

```bash
cd backend
go test ./services/llm-stream/cmd -run 'AckDelivery|NackDelivery' -count=1
```

Expected: FAIL because the helper functions do not exist.

- [ ] **Step 3: Implement the acknowledgement helpers**

Create `delivery.go`:

```go
package main

import (
    "fmt"

    "github.com/google/uuid"
)

type deliveryAcknowledger interface {
    Ack(multiple bool) error
    Nack(multiple bool, requeue bool) error
}

func ackDelivery(delivery deliveryAcknowledger, reason string) error {
    if err := delivery.Ack(false); err != nil {
        return fmt.Errorf("ack %s: %w", reason, err)
    }
    return nil
}

func nackDelivery(delivery deliveryAcknowledger, taskID uuid.UUID) error {
    if err := delivery.Nack(false, true); err != nil {
        return fmt.Errorf("nack generation task %s: %w", taskID, err)
    }
    return nil
}
```

- [ ] **Step 4: Replace ignored Ack/Nack calls**

In `main.go`, malformed payload handling becomes:

```go
if err := json.Unmarshal(delivery.Body, &message); err != nil {
    log.Printf("invalid generation message payload: %v", err)
    if ackErr := ackDelivery(delivery, "malformed generation message"); ackErr != nil {
        log.Printf("generation delivery acknowledgement failed: %v", ackErr)
    }
    continue
}
```

Business failure becomes:

```go
if err := consumer.HandleGenerationRequested(signalContext, message); err != nil {
    log.Printf("generation task handling failed for %s: %v", message.TaskID, err)
    if nackErr := nackDelivery(delivery, message.TaskID); nackErr != nil {
        log.Printf("generation delivery rejection failed: %v", nackErr)
    }
    continue
}
```

Success becomes:

```go
if ackErr := ackDelivery(delivery, "completed generation task "+message.TaskID.String()); ackErr != nil {
    log.Printf("generation delivery acknowledgement failed: %v", ackErr)
}
```

- [ ] **Step 5: Write the failing download-copy test**

Create `download_writer_test.go`:

```go
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
    return 0, errors.New("client disconnected")
}

func TestCopyDownload_ReturnsWriterFailure(t *testing.T) {
    err := copyDownload(failingWriter{}, strings.NewReader("pdf"))
    require.ErrorContains(t, err, "stream download")
    require.ErrorContains(t, err, "client disconnected")
}
```

- [ ] **Step 6: Implement and use copyDownload**

Create `download_writer.go`:

```go
package task

import (
    "fmt"
    "io"
)

func copyDownload(dst io.Writer, src io.Reader) error {
    if _, err := io.Copy(dst, src); err != nil {
        return fmt.Errorf("stream download: %w", err)
    }
    return nil
}
```

Replace the ignored copy in `DownloadTask`:

```go
if err := copyDownload(c.Writer, file); err != nil {
    _ = c.Error(err)
    c.Abort()
    return
}
```

Do not attempt to write a JSON error after the PDF headers have been committed.

- [ ] **Step 7: Verify focused, race, and full tests**

```bash
go test ./services/llm-stream/cmd ./services/core-api/domain/task -count=1
go test -race ./services/llm-stream/cmd ./services/core-api/domain/task -count=1
go test ./... -count=1
go vet ./...
```

Expected: all commands pass in an environment that permits `httptest` loopback listeners.

- [ ] **Step 8: Commit the two fixes separately**

```bash
git add backend/services/llm-stream/cmd
git commit -m "fix(stream): surface generation delivery acknowledgement failures"

git add backend/services/core-api/domain/task/download_handler.go \
  backend/services/core-api/domain/task/download_writer.go \
  backend/services/core-api/domain/task/download_writer_test.go
git commit -m "fix(task): surface download stream failures"
```

## Task 3: Cover the Service-Owned Generation Application Before Legacy Deletion

**Files:**
- Create: `backend/services/llm-stream/app/generation/prompt_service_test.go`
- Create: `backend/services/llm-stream/app/generation/decomposition_quality_test.go`
- Create: `backend/services/llm-stream/app/generation/decomposition_result_test.go`
- Create: `backend/services/llm-stream/app/generation/generator_service_test.go`
- Create: `backend/services/llm-stream/app/generation/decomposition_continue_test.go`
- Reference: corresponding tests currently under `backend/internal/service/`

- [ ] **Step 1: Capture the zero-coverage baseline**

```bash
cd backend
go test ./services/llm-stream/app/generation -count=1 -coverprofile=/tmp/generation-before.cover
go tool cover -func=/tmp/generation-before.cover | tail -n 1
```

Expected: package compiles with no test files and reports 0.0% coverage.

- [ ] **Step 2: Port prompt-resolution tests**

Move behavior, not imports, from:

```text
internal/service/prompt_requirements_test.go
internal/service/prompt_profile_resolver_test.go
```

Required service-owned test cases:

- invalid scenario falls back to the default scenario;
- user override remains honored;
- profile requirements are prepended;
- classifier failure uses deterministic fallback;
- valid classifier JSON selects the expected profile;
- unknown profile key falls back safely.

Tests must use package `generation` and `shared/kernel/prompt` types. They must not import `internal/service`, `internal/prompt`, or `internal/model`.

- [ ] **Step 3: Port the series-quality state machine tests**

Move these behaviors from `internal/service/series_quality_pipeline_test.go`:

```text
reject missing mechanism/examples
require example and reproduction details
require revision actions
keep shared prompt prefix stable
reject invalid JSON
stream only final stage in order
repair low scorecard drafts
```

Use a fake LLM implementation injected through the existing generation package provider boundary; no test may contact DeepSeek.

- [ ] **Step 4: Port task-result collector tests**

From `internal/service/decomposition_generate_result_test.go`, cover:

```go
func TestSeriesTaskResultCollector_BuildTaskResultIncludesParentAndChapters(t *testing.T)
func TestDecompositionService_TakeGenerateSeriesTaskResultReturnsStoredResultOnce(t *testing.T)
```

Assert prompt, completion, cache-hit, and cache-miss usage fields remain present.

- [ ] **Step 5: Port single-generation and continue tests**

Required behaviors:

- task-only mode does not write blogs directly;
- final structured result contains generated content and real usage;
- persistence error is returned;
- continuation uses injected persistence;
- cancellation stops streaming;
- prompt profile role is included in generated messages.

- [ ] **Step 6: Run focused tests after each file**

```bash
go test ./services/llm-stream/app/generation -count=1
go test -race ./services/llm-stream/app/generation -count=1
```

Expected: PASS without network, RabbitMQ, PostgreSQL, or Obsidian access.

- [ ] **Step 7: Measure coverage and fill only critical gaps**

```bash
go test ./services/llm-stream/app/generation -count=1 -coverprofile=/tmp/generation-after.cover
go tool cover -func=/tmp/generation-after.cover | sort -k3,3n | sed -n '1,80p'
go tool cover -func=/tmp/generation-after.cover | tail -n 1
```

Acceptance:

- package statement coverage is at least 50%;
- every public task result method and quality-pipeline transition has a success and error-path assertion;
- no test relies on the legacy package.

- [ ] **Step 8: Commit**

```bash
git add backend/services/llm-stream/app/generation/*_test.go
git commit -m "test(stream): cover service-owned generation application"
```

## Task 4: Remove Obsolete Standalone Commands and Legacy Business Packages

**Files:**
- Delete: `backend/cmd/core-api/`
- Delete: `backend/cmd/llm-stream/`
- Delete: `backend/cmd/parser-service/`
- Delete: `backend/cmd/export-service/`
- Delete after tests are ported: `backend/internal/service/`
- Delete: `backend/internal/domain/`
- Delete: `backend/internal/transport/`
- Delete: `backend/internal/prompt/`
- Delete: `backend/internal/infra/cache/`
- Delete: `backend/internal/infra/mq/`
- Modify: `backend/services/architecture_test.go`
- Modify: `README.md`
- Preserve: `backend/internal/infra/db/`, `backend/internal/model/`

- [ ] **Step 1: Prove Docker and aggregate entrypoints do not use legacy commands**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
rg -n 'cmd/(core-api|llm-stream|parser-service|export-service)' \
  docker-compose.yml backend/Dockerfile backend/services -g 'Dockerfile' -g '*.go'
```

Expected: no Docker build references to legacy `backend/cmd/*`; all service images build `./services/<service>/cmd`.

- [ ] **Step 2: Add an architecture test requiring the old command directories to be absent**

```go
func TestLegacyStandaloneCommandWrappersAreRemoved(t *testing.T) {
    legacyDirs := []string{"core-api", "llm-stream", "parser-service", "export-service"}
    for _, name := range legacyDirs {
        path := filepath.Join("..", "cmd", name)
        if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
            t.Fatalf("legacy command wrapper %s must be removed", path)
        }
    }
}
```

Add `errors` to the test imports.

- [ ] **Step 3: Delete only the obsolete command wrappers**

Before deletion, run:

```bash
git status --short
```

Delete the four directories listed above. Preserve `backend/cmd/server`.

- [ ] **Step 4: Verify commands and services still compile**

```bash
cd backend
go test ./cmd/server ./services/... -count=1
go test ./services -run TestLegacyStandaloneCommandWrappersAreRemoved -count=1
```

- [ ] **Step 5: Commit command retirement**

```bash
git add backend/cmd backend/services/architecture_test.go
git commit -m "refactor(backend): remove obsolete standalone commands"
```

- [ ] **Step 6: Confirm legacy business packages have no external imports**

Run from `backend/`:

```bash
rg -n '"inkwords-backend/internal/(domain|service|transport|prompt|infra/cache|infra/mq)' \
  cmd services shared pkg scripts --glob '*.go'
```

Expected: no production imports. References inside `services/architecture_test.go` are policy strings and may remain.

- [ ] **Step 7: Capture the pre-delete test baseline**

```bash
go test ./... -count=1
go vet ./...
go tool deadcode -test ./... > /tmp/inkwords-deadcode-before-delete.txt
```

Expected: tests/vet pass; deadcode still lists the legacy packages targeted in this task.

- [ ] **Step 8: Delete the unreachable legacy business packages**

Delete exactly:

```text
backend/internal/service/
backend/internal/domain/
backend/internal/transport/
backend/internal/prompt/
backend/internal/infra/cache/
backend/internal/infra/mq/
```

Do not delete `backend/internal/infra/db/` or `backend/internal/model/`.

- [ ] **Step 9: Verify preserved dependencies and full build**

```bash
go test ./... -count=1
go vet ./...
go list -deps ./cmd/... ./services/... | rg '^inkwords-backend/internal'
go tool deadcode -test ./... > /tmp/inkwords-deadcode-after-delete.txt
```

Expected dependency output contains only the intentionally retained persistence bridge:

```text
inkwords-backend/internal/model
inkwords-backend/internal/infra/db
```

- [ ] **Step 10: Recount exact duplicates**

```bash
find . -type f -name '*.go' -not -name '*_test.go' -exec shasum {} + \
  | awk '{count[$1]++} END {for (h in count) if (count[h]>1) {groups++; files+=count[h]} print groups, files}'
```

Expected: duplicate groups/files are lower than the current 12/24 baseline.

- [ ] **Step 11: Commit legacy deletion**

```bash
git add backend/internal backend/services/architecture_test.go README.md
git commit -m "refactor(backend): remove unreachable legacy business packages"
```

## Task 5: Enforce Coverage and Bundle Budgets in CI

**Files:**
- Create: `backend/scripts/check_coverage.sh`
- Create: `frontend/scripts/check-bundle-budget.mjs`
- Modify: `frontend/vite.config.ts`
- Modify: `frontend/package.json`
- Modify: `.github/workflows/ci.yml`
- Modify: `docs/qa/code-cleanup-baseline.md`

- [ ] **Step 1: Add backend coverage thresholds**

Create executable `backend/scripts/check_coverage.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

check_total() {
  local profile="$1"
  local minimum="$2"
  local label="$3"
  local actual
  actual=$(go tool cover -func="$profile" | awk '/^total:/ {gsub("%", "", $3); print $3}')
  awk -v actual="$actual" -v minimum="$minimum" -v label="$label" 'BEGIN {
    if ((actual + 0) < (minimum + 0)) {
      printf "%s coverage %.2f%% is below %.2f%%\n", label, actual, minimum > "/dev/stderr"
      exit 1
    }
    printf "%s coverage %.2f%% meets %.2f%%\n", label, actual, minimum
  }'
}

go test ./... -coverprofile=/tmp/inkwords-all.cover
check_total /tmp/inkwords-all.cover 35.0 "backend total"

go test ./services/llm-stream/app/generation -coverprofile=/tmp/inkwords-generation.cover
check_total /tmp/inkwords-generation.cover 50.0 "llm-stream generation"
```

- [ ] **Step 2: Verify the script succeeds**

```bash
cd backend
chmod +x scripts/check_coverage.sh
./scripts/check_coverage.sh
```

Expected: both thresholds pass. If generation coverage is below 50%, return to Task 3; do not lower the threshold.

- [ ] **Step 3: Add frontend coverage floors**

In `frontend/vite.config.ts`, extend coverage configuration:

```ts
thresholds: {
  statements: 38,
  branches: 65,
  functions: 54,
  lines: 38,
},
```

These floors are slightly below the verified 38.99/65.65/54.51/38.99 baseline and prevent regression without pretending the current coverage is sufficient.

- [ ] **Step 4: Add a deterministic main-bundle budget script**

Create `frontend/scripts/check-bundle-budget.mjs`:

```js
import { gzipSync } from 'node:zlib'
import { readdirSync, readFileSync } from 'node:fs'
import path from 'node:path'

const assetsDir = path.resolve('dist/assets')
const candidates = readdirSync(assetsDir)
  .filter((name) => /^index-[A-Za-z0-9_-]+\.js$/.test(name))
  .map((name) => ({ name, bytes: readFileSync(path.join(assetsDir, name)) }))

if (candidates.length !== 1) {
  throw new Error(`expected one main index chunk, found ${candidates.length}`)
}

const [{ name, bytes }] = candidates
const raw = bytes.byteLength
const gzip = gzipSync(bytes).byteLength
const limits = { raw: 1_160_000, gzip: 355_000 }

console.log(`${name}: raw=${raw} gzip=${gzip}`)
if (raw > limits.raw || gzip > limits.gzip) {
  throw new Error(`main bundle exceeds budget raw=${limits.raw} gzip=${limits.gzip}`)
}
```

Add:

```json
"check:bundle": "node scripts/check-bundle-budget.mjs"
```

- [ ] **Step 5: Verify frontend thresholds and budget**

```bash
cd frontend
npm run test:coverage
npm run build
npm run check:bundle
```

Expected: coverage and bundle commands exit 0.

- [ ] **Step 6: Pin a compatible golangci-lint action**

Replace the current v6 action with:

```yaml
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v8
        with:
          version: v2.12.2
          working-directory: backend
```

The v8 action supports golangci-lint v2 while remaining compatible with the workflow's current Node runtime assumptions.

- [ ] **Step 7: Wire blocking CI commands**

Backend job:

```yaml
      - name: backend coverage thresholds
        working-directory: backend
        run: ./scripts/check_coverage.sh
```

Frontend job after build:

```yaml
      - name: frontend bundle budget
        working-directory: frontend
        run: npm run check:bundle
```

Keep `npm run deadcode` blocking now that Task 1 makes it green.

- [ ] **Step 8: Record the new verified baselines**

Update `docs/qa/code-cleanup-baseline.md` with:

- backend total and generation package coverage;
- frontend four coverage dimensions;
- raw and gzip main-bundle limits;
- remaining duplicate count;
- remaining intentional `internal/model` and `internal/infra/db` bridge.

- [ ] **Step 9: Commit**

```bash
git add backend/scripts/check_coverage.sh frontend/scripts/check-bundle-budget.mjs \
  frontend/vite.config.ts frontend/package.json .github/workflows/ci.yml \
  docs/qa/code-cleanup-baseline.md
git commit -m "chore(ci): enforce coverage and bundle budgets"
```

## Task 6: Final Verification, Plan Closure, and Draft PR Handoff

**Files:**
- Modify: `docs/superpowers/plans/2026-06-29-systematic-code-cleanup.md`
- Modify: `docs/superpowers/plans/2026-06-29-systematic-code-cleanup-closeout.md`
- Modify: `README.md` only if final ownership text is stale

- [ ] **Step 1: Run the full backend gate**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./... -count=1
go vet ./...
golangci-lint run ./...
go tool deadcode -test ./... > /tmp/inkwords-final-deadcode.txt
./scripts/check_coverage.sh
```

Expected: tests, vet, lint, and coverage gates pass. Review the deadcode report; no deleted legacy path may appear.

- [ ] **Step 2: Run the full frontend gate**

```bash
cd ../frontend
npm run lint
npm run deadcode
npm test
npm run test:coverage
npm run build
npm run check:bundle
```

Expected: all commands pass; at least 144 tests pass.

- [ ] **Step 3: Validate Compose configuration without starting services**

```bash
cd ..
mkdir -p /tmp/obsidian-vault/wiki
OBSIDIAN_VAULT_PATH=/tmp/obsidian-vault \
  docker compose --env-file backend/.env.example config > /tmp/inkwords-compose-final.yml
```

Expected: command exits 0 and all five service-owned Dockerfiles remain referenced.

- [ ] **Step 4: Run the real smoke gate when Docker is available**

Use only disposable CI values:

```bash
OBSIDIAN_VAULT_PATH=/tmp/obsidian-vault \
DEEPSEEK_API_KEY=smoke-placeholder \
JWT_SECRET=smoke-placeholder-secret \
OBSIDIAN_REST_API_KEY=smoke-placeholder \
docker compose --env-file backend/.env.example up -d --build

curl --fail http://localhost/api/v1/ping
docker compose --env-file backend/.env.example ps
```

After capturing results, request explicit approval before running `docker compose down -v`, because it deletes the disposable smoke volumes.

- [ ] **Step 5: Update execution ledgers**

At the top of the original plan, add a completion block mapping Tasks 0–13 to their implementing commits. Mark a task complete only when its final verification command passes. In this closeout plan, check each completed step with `[x]`.

- [ ] **Step 6: Verify the final Git diff**

```bash
git status --short --branch
git diff --check
git log --oneline origin/main..HEAD
git diff --stat origin/main...HEAD
```

Expected:

- no generated `frontend/coverage/` path appears;
- only intended plan/code/config changes are present;
- no unstaged or untracked source files remain.

- [ ] **Step 7: Commit the plan closure**

```bash
git add docs/superpowers/plans/2026-06-29-systematic-code-cleanup.md \
  docs/superpowers/plans/2026-06-29-systematic-code-cleanup-closeout.md \
  README.md
git commit -m "docs(governance): close systematic cleanup plan"
```

- [ ] **Step 8: Publish only after explicit user authorization**

When publication is authorized:

```bash
git push -u origin codex/governance-baseline
gh pr create --draft \
  --base main \
  --head codex/governance-baseline \
  --title "refactor: complete systematic code cleanup" \
  --body-file /tmp/inkwords-cleanup-pr.md
```

The PR body must include:

```markdown
## Summary
- establish blocking lint, dead-code, coverage and bundle gates
- move parser, LLM, core and stream ownership to service/shared packages
- remove obsolete standalone commands and unreachable legacy business code
- preserve the internal database/model bridge for a separate migration

## Verification
- backend tests, vet, golangci-lint and coverage thresholds
- frontend lint, Knip, 144+ tests, coverage thresholds and bundle budget
- Docker Compose config and microservices smoke

## Remaining bounded debt
- shared/platform/postgres still delegates to internal/infra/db and internal/model
```

- [ ] **Step 9: Monitor remote CI**

```bash
gh pr checks --watch
```

If a job fails, inspect the specific run and fix only the failing concern; do not merge while any required check is red.

## Stop Conditions

Stop and split a new plan if any of these occurs:

- deleting legacy packages changes an external route, response body, SSE event, RabbitMQ message, or database table;
- service-owned generation tests require live DeepSeek access;
- `go list -deps` shows an unexpected production dependency on a deleted internal package;
- Knip can only be made green by ignoring application/service/store directories;
- bundle budget requires removing a user-visible feature;
- Docker smoke points to a data migration rather than an entrypoint/configuration issue.

## Definition of Done

- `npm run deadcode` exits 0 without hiding application code.
- llm-stream Ack/Nack and task download copy errors are observable and tested.
- `services/llm-stream/app/generation` has at least 50% statement coverage with deterministic provider fakes.
- obsolete `backend/cmd/{core-api,llm-stream,parser-service,export-service}` wrappers are removed.
- unreachable legacy domain/service/transport/prompt/cache/mq packages are removed.
- only `internal/model` and `internal/infra/db` remain as the explicitly documented persistence bridge.
- backend total coverage is at least 35%; generation package coverage is at least 50%.
- frontend coverage remains at or above 38/65/54/38 and the main bundle stays below 1.16 MB / 355 KB gzip.
- all local gates, Compose config, remote CI, and Docker smoke pass.
- both plan documents are tracked and accurately reflect execution state.

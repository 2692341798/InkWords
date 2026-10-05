# Directory Structure

> Where backend code lives and which service owns what.

## Top-level layout (`backend/`)

| Path | Status | Contents |
|------|--------|----------|
| `services/<svc>/` | **Current** | One deployable per service. Each has `cmd/main.go`, `app/bootstrap/`, `domain/`, optional `app/<usecase>/`, `infra/`, `transport/http/v1/`, and its own `Dockerfile`. |
| `services/architecture_test.go` | Current | Repository-level dependency and packaging rules (see [Architecture Boundaries](./architecture-boundaries.md)). |
| `shared/kernel/` | Current | Stable cross-process value objects and HTTP plumbing: `httpx` (auth middleware, request ID, slog request logger, health, server), `projectcourse` (Blueprint/Checkpoint/Evidence contracts with `Validate()`), `prompt`, `blog`. |
| `shared/platform/` | Current | Infrastructure adapters used by ≥2 services: `cache` (Redis), `llm` (DeepSeek client), `obsidian`, `parser`, `rabbitmq` (dial + message envelopes), `postgres` (thin wrapper — see debt). |
| `internal/` | **Legacy** | `internal/model/*` GORM models and `internal/infra/db/db.go` (global `DB`, AutoMigrate). Still on the production startup path via `shared/platform/postgres`. |
| `cmd/server/main.go` | **Legacy** | Old monolith entrypoint. Not built by any Dockerfile. Do not add wiring here. |
| `pkg/jwt` | Current | JWT helpers used by `shared/kernel/httpx/auth.go` and `core-api/domain/auth`. |
| `pkg/response` | Dead | Imported by nothing. Do not use. |
| `integration/` | Current | `//go:build integration` contract tests (no DB, no secrets). |
| `scripts/` | Current | `//go:build tools` helper mains and `check_coverage.sh`. |
| `db/init/` | Current | Postgres container init only (`00-create-review-db.sql` creates the review DB). Not a migration directory. |

## Per-service layout (as-is)

The nominal layout is `cmd → app/bootstrap → domain/<bounded-context>`, with `transport/http/v1/routes.go` registering routes. In practice the **domain package is the whole vertical slice**:

```text
services/core-api/
├── cmd/main.go                    # godotenv.Load in init(); bootstrap.BuildRouter(); httpx server
├── app/bootstrap/bootstrap.go     # constructs repos → services → handlers, builds gin engine
├── app/projectanalysis/           # the only real application-layer package in core-api
├── domain/<bc>/                   # auth, blog, project, projectcourse, task, user
│   ├── model.go                   # GORM model structs
│   ├── repository.go              # Repository interface + GormRepository
│   ├── service.go                 # business logic, depends on Repository interface
│   ├── handler.go                 # gin handlers, depend on small unexported service interfaces
│   ├── dto.go                     # request/response structs
│   └── *_test.go
├── infra/mq/publisher.go          # RabbitMQ publisher
└── transport/http/v1/routes.go    # route table from a CoreHandlers struct of gin.HandlerFunc
```

Reference wiring: `backend/services/core-api/app/bootstrap/bootstrap.go` (`BuildRouter`) builds `NewGormRepository → NewService → NewHandler` per domain, then calls `corev1.RegisterCoreRoutes`.

Other services follow the same shape with fewer packages: `review-service/domain/review/`, `parser-service/domain/parse/`, `export-service/domain/export/`, `llm-stream/domain/stream/` + `llm-stream/app/{generation,projectcourse}/`, `course-runner/domain/verification/`. Several `infra/{db,mq,...}` directories contain only `.gitkeep`.

**When adding code:** put it in the existing domain package of the owning service, following the `model/repository/service/handler/dto` file split. Put LLM orchestration that spans multiple domain calls in `app/<usecase>/` (pattern: `llm-stream/app/generation`, `core-api/app/projectanalysis`). Register routes in `transport/http/v1/routes.go`, not in bootstrap.

## Service ownership

From `AGENTS.md` §4 — decide the owning service before writing code:

| Service | Owns | Must not own |
|---------|------|--------------|
| `core-api` | Textbook/blog business state, approvals, project courses, job task records, auth/users | LLM execution, parsing execution |
| `parser-service` | Crawling/parsing execution | Textbook persistence |
| `llm-stream` | LLM execution and streaming | Final manuscript authority |
| `course-runner` | Verification runs | Source import, approval |
| `review-service` | Mastery evidence, FSRS scheduling (separate `REVIEW_DATABASE_URL`) | Textbook content |
| `export-service` | Rendering approved revisions | Rewriting revisions |

Do not add a new service unless a documented permission, scaling, or failure-isolation need justifies it.

## Known debt (do not copy)

- `cmd/server/main.go` wires every service into one engine and omits the `ProjectCourse*` handlers that `RegisterCoreRoutes` requires via `must()`; treat it as unmaintained.
- `internal/infra/db/db.go` imports `services/core-api/domain/projectcourse` (legacy → service dependency) and holds a global `var DB *gorm.DB`.
- `shared/platform/postgres/core.go` is a wrapper over legacy `internal/infra/db`, so every service bootstrap still runs legacy AutoMigrate.
- Job-task models are duplicated per service (`core-api/domain/task/model.go`, `internal/model/job_task.go`, and private `jobTask` structs in `llm-stream/domain/stream/task_store.go`, `parser-service/domain/parse/task_store.go`, `export-service/domain/export/task_store.go`). Review models exist in both `internal/model/review.go` and `review-service/domain/review/model.go`.
- `shared/platform/rabbitmq/message.go` holds the cross-process envelope; by the `AGENTS.md` rule it belongs in `shared/kernel`.
- `shared/kernel/auth/` and `shared/kernel/response/` contain only `.gitkeep`.

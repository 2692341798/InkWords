# Architecture Boundaries

> Which dependency rules are enforced by tests, which are only intended, and what new code must respect.

## What `services/architecture_test.go` enforces (as-is)

The test walks `.go` files with `filepath.WalkDir` and checks imports with `strings.Contains` on file contents. It does **not** use `go/parser` or `go list`, so it only sees direct, textual imports.

Enforced rules:

- `backend/cmd/{core-api,llm-stream,parser-service,export-service}` must not exist — service entrypoints live in `services/<svc>/cmd`.
- Each service `Dockerfile` builds `./services/<svc>/cmd`, ends with `CMD ["./<svc>"]`, and is referenced in `docker-compose.yml`.
- The `llm-stream` compose section sets `INKWORDS_TASK_PERSISTENCE_MODE` defaulting to `task_only`.
- Services in `backendServices` must not import a peer service package. **The list currently contains `core-api`, `llm-stream`, `parser-service`, `export-service`, `review-service` — `course-runner` is missing.**
- Per-service/per-directory blacklists of `inkwords-backend/internal/...` imports, plus restrictions for `cmd/server`.
- `parser-service` must not import its own `infra/parser`.

Not enforced by any test:

- Domain packages importing gin, GORM, AMQP, or provider SDKs (19 domain files import gin, 31 import gorm today).
- `transport → application → domain` direction.

## Rules for new code

1. **No peer-service imports.** Cross-service communication uses versioned HTTP or MQ contracts. If two services need the same stable value object or event, put it in `backend/shared/kernel/`.
2. **Adapters:** put an adapter in `backend/shared/platform/` only when at least two services use it; otherwise keep it under `services/<svc>/infra/`.
3. **Extend the architecture test** in the same change whenever you introduce a new dependency rule, add a service, or add a package that must stay isolated. Add new services to `backendServices`.
4. **Do not deepen domain/framework coupling.** Existing domain packages already contain gin handlers and GORM repositories; new code may follow that per-package file split (`handler.go`, `repository.go`), but keep business decisions in `service.go` behind interfaces so the service is testable without gin or a database. Do not import gin, GORM, or AMQP types into `shared/kernel`.
5. **Business code depends on internal ports**, not provider request structs (see [Messaging & External Calls](./messaging-and-external-calls.md#llm-calls)).

## Known debt (do not copy)

- `services/course-runner/app/bootstrap/bootstrap.go` imports `inkwords-backend/services/core-api/domain/task` (`coretask.GormRepository`). This violates the no-peer-import rule and is not caught because `course-runner` is absent from `backendServices`.
- `services/core-api/domain/projectcourse/handler.go` imports `core-api/domain/task` — intra-service, allowed, but it couples two bounded contexts at the handler level.
- The legacy `internal/infra/db` → `services/core-api/domain/projectcourse` import (see [Directory Structure](./directory-structure.md#known-debt-do-not-copy)).

Fixing these is a separate, explicitly scoped change; do not bundle it with feature work.

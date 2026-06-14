# Service Image Boundaries

InkWords runs as a frontend gateway plus five backend services. Each backend service now owns its Docker build recipe instead of sharing one aggregate backend image.

## Service Images

| Service | Dockerfile | Runtime focus |
| --- | --- | --- |
| `core-api` | `backend/services/core-api/Dockerfile` | Auth, users, blogs, task creation/query, upload/static assets |
| `llm-stream` | `backend/services/llm-stream/Dockerfile` | Streaming generation API and generation worker |
| `parser-service` | `backend/services/parser-service/Dockerfile` | File, ZIP, PDF and Git parsing API/worker |
| `export-service` | `backend/services/export-service/Dockerfile` | PDF/Obsidian export API and export worker |
| `review-service` | `backend/services/review-service/Dockerfile` | Knowledge review API and review database access |

`docker-compose.yml` must point each backend service at its own Dockerfile. The root `backend/Dockerfile` is no longer the Compose production path and should not receive new production-only service behavior.

## Boundary Rules

- A service may import its own `backend/services/<service>/...` packages.
- A service must not import packages from another `backend/services/<peer>/...` directory.
- Shared infrastructure can live under `backend/shared/...` when it has no business vocabulary.
- HTTP liveness, readiness, request id, request logging and JWT middleware belong to `backend/shared/kernel/httpx`; service-owned entrypoints should not import `backend/internal/transport/http/middleware`.
- RabbitMQ message envelopes and connection helpers belong to `backend/shared/platform/rabbitmq`; service-owned code should not import `backend/internal/infra/mq`.
- RabbitMQ publishers that depend on one service's business domain belong in that service, such as `backend/services/core-api/infra/mq`.
- LLM provider clients belong to `backend/shared/platform/llm`; service-owned code should not import `backend/internal/infra/llm`.
- Parser/export worker domains should depend on local task callback interfaces instead of importing `backend/internal/domain/task`.
- Legacy `backend/internal/...` imports are temporary migration debt. New business code should move toward the owning service directory instead of expanding shared internal packages.
- Heavy runtime tools belong only in the service that needs them. For example, Chromium lives in `export-service`, while `pdftotext` lives in parser-facing images.

## Verification

Run the boundary tests after changing service Dockerfiles, Compose build rules, or service imports:

```bash
cd backend
go test ./services -count=1
```

Run the Compose render check before rebuilding the stack:

```bash
docker compose --env-file backend/.env config
```

Expected results:

- Each backend service renders with `dockerfile: services/<service>/Dockerfile`.
- `go test ./services -count=1` passes.
- No service imports another service package directly.
- No service imports the legacy internal HTTP middleware package directly.
- No service imports the legacy internal RabbitMQ infra package directly.
- No service imports the legacy internal LLM infra package directly.
- Parser/export domain and infra packages do not import the legacy internal task domain directly.

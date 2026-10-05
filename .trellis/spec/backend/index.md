# Backend Development Guidelines

> How the InkWords Go backend (`backend/`, module `inkwords-backend`, Go 1.25) is actually built, and the rules new code must follow.

Repository-wide product invariants, the beginner-friendly writing contract, and source/LLM safety rules live in the root `AGENTS.md`. These spec files do not repeat them; they translate them into concrete backend coding guidance and record where the current code deviates.

## How to read these files

Every file separates three things:

- **As-is** — what the code does today, with file paths. Match these patterns so new code looks native.
- **Rules for new code** — the `AGENTS.md` target that new or touched code must satisfy, even where old code does not.
- **Known debt (do not copy)** — existing patterns that violate the target. Do not spread them; do not fix them opportunistically inside an unrelated change either (`AGENTS.md` §10: no mixed feature + refactor changes).

## Guidelines Index

| Guide | Description |
|-------|-------------|
| [Directory Structure](./directory-structure.md) | `services/`, `shared/`, legacy `internal/`/`cmd/server`, per-service layout, service ownership |
| [Architecture Boundaries](./architecture-boundaries.md) | What `services/architecture_test.go` enforces, layering, cross-service rules |
| [Database Guidelines](./database-guidelines.md) | GORM repositories, AutoMigrate reality vs goose target, CAS, transactions, idempotency, JSONB |
| [Error Handling](./error-handling.md) | Sentinel errors, wrapping, handler error mapping, HTTP/SSE response shapes |
| [Logging Guidelines](./logging-guidelines.md) | stdlib `log` vs `slog`, request logging, redaction requirements |
| [Messaging & External Calls](./messaging-and-external-calls.md) | RabbitMQ envelopes/consumers, LLM client, timeouts, retries, model selection |
| [Quality Guidelines](./quality-guidelines.md) | Tests, fakes, sqlite, coverage, commands, file size, Godoc, config |

## Pre-development checklist

1. Identify the owning service (see [Directory Structure](./directory-structure.md#service-ownership)). If the change crosses services, it needs an HTTP/MQ contract, not an import.
2. Read the nearest existing domain package and its `*_test.go` before writing code.
3. Schema change? Read [Database Guidelines](./database-guidelines.md#schema-changes) first — the goose target is not set up yet.
4. New dependency rule? Extend `backend/services/architecture_test.go` in the same change.
5. Run `cd backend && GOCACHE=/tmp/inkwords-go-build go test ./...` before declaring done.

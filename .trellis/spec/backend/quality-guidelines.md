# Quality Guidelines

> Tests, configuration, code-size limits, and the commands that define "done" for backend changes.

## Test style (as-is)

- **Assertions:** 70 of 79 test files use testify, mostly `require`. Plain `t.Fatalf` appears in `projectcourse/repository_test.go` and `architecture_test.go`. New tests should use `require`.
- **Fakes are hand-written** structs (`type fakeXxx` / `stubXxx`) that implement the consumer-side interface. There is no gomock or testify/mock. Don't add a mocking library.
- **Database tests** use in-memory sqlite plus `AutoMigrate` in the test itself:
  - `sqlite.Open(":memory:")`
  - `sqlite.Open("file:<name>?mode=memory&cache=shared")`

  Postgres-only behaviour, such as JSONB operators or the partial index, is not covered this way. Say so when it matters.
- **Naming** is split between `TestType_Behavior` and `TestBehaviorSentence`. Within a file, follow the existing style.
- **Table-driven tests** are used where there are many input variants. Only 5 files use them today; don't force them.

Trusted examples to copy:

| Test | Shows |
|------|-------|
| `backend/services/core-api/domain/projectcourse/repository_test.go` | CAS conflicts, owner isolation |
| `backend/services/core-api/domain/task/generation_result_repository_test.go` | Transactional result persistence |
| `backend/services/review-service/domain/review/handler_test.go` | Handler error mapping and public messages |
| `backend/services/*/transport/http/v1/routes_test.go` | Route registration |
| `backend/integration/pipeline_integration_test.go` | Cross-service contracts (`//go:build integration`) |

## What a change must test

- **New contract** (DTO, MQ payload, `shared/kernel` type): validation tests plus failure tests.
- **Code that writes revisions or approvals:** a test proving that locked, approved, or manual content can't be overwritten (CAS conflict path).
- **Handler:** status and body for each mapped sentinel, plus the generic 500.
- **New dependency rule:** an assertion in `services/architecture_test.go`.

## Coverage

`backend/scripts/check_coverage.sh` sets a 35% floor for the whole backend and 50% for `llm-stream/app/generation`.

## Configuration (as-is)

Each `services/<svc>/cmd/main.go` calls `godotenv.Load()` in `init()`. Values are read with `os.Getenv` at the point of use. There is no config struct; bootstraps carry local `envOrDefault` / `firstNonEmpty` helpers. When you add a variable, add it to `backend/.env.example`, plus `.env.e2e.example` and `docker-compose*.yml` if e2e or compose needs it, in the same change.

Known config debt:
- `pkg/jwt` falls back to `"default_inkwords_secret_key"` when `JWT_SECRET` is unset.
- When `DEV_AUTH_USER_ID` is set and there is no auth header, auth is bypassed.

Don't extend either fallback.

## Code size and comments

- When a file approaches 500 lines, consider splitting it before adding another responsibility. Never grow a file past 800 lines. Largest production files today: `llm-stream/app/generation/decomposition_quality.go` (623), `decomposition_series.go` (621), `generator_service.go` (501), `review-service/domain/review/session_service.go` (473), `llm-stream/domain/stream/task_consumer.go` (460), `shared/platform/llm/deepseek.go` (452). The largest test file, `core-api/domain/blog/service_test.go`, has 809 lines.
- Public identifiers need a useful Godoc comment.
- Comments explain non-obvious constraints and *why*, often prefixed `Why:` and written in Chinese (see `review/handler.go` `handleServiceError`). Don't write line-by-line "what" comments.
- Use dependency injection at application boundaries: constructors take interfaces (`NewHandler(service)`, `NewService(repo)`), and bootstrap wires them.

## Validation commands

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./...
```

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test -tags integration ./integration/...
```

```bash
cd backend && gofmt -l .
```

```bash
git diff --check
```

Real-provider, real-network, Docker sandbox, and destructive migration checks are opt-in. Never report them as passed when only mocks or contract tests ran.

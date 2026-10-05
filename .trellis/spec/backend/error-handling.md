# Error Handling

> How errors are defined, wrapped, mapped to HTTP, and surfaced to the user.

## Error types (as-is)

There is no shared error-code type. Each domain package defines **sentinel errors** with `errors.New`:

- `core-api/domain/projectcourse`: `ErrNotFound`, `ErrVersionConflict`, `ErrBlueprintImmutable`
- `core-api/domain/task`: `ErrTaskNotFound`, `ErrTaskAccessDenied`
- `review-service/domain/review/service_types.go`: Chinese public sentinels such as `errReviewSessionNotFound = errors.New("复习会话不存在")`

**Rules for new code:**

- Define sentinels in the domain package that owns the failure. Keep them unexported unless another package needs `errors.Is` on them.
- When a sentinel's message is shown to users, write it in Chinese (the review-service pattern).
- Wrap internal errors with their cause: `fmt.Errorf("创建任务失败: %w", err)`. Always use `%w`, never `%v`, when the caller may need `errors.Is` or `errors.As`. Today about 45% of `fmt.Errorf` calls drop `%w`; don't add more.

## Mapping errors in handlers

Handlers map domain errors to HTTP status codes in one `switch` with `errors.Is`. Services never pick HTTP status codes. The reference implementation is `handleServiceError` in `backend/services/review-service/domain/review/handler.go`:

```go
switch {
case errors.Is(err, errReviewSessionNotFound):
    h.writeError(c, http.StatusNotFound, err.Error())   // known sentinel: its Chinese message is public
case errors.Is(err, errReviewSessionClosed):
    h.writeError(c, http.StatusConflict, err.Error())
default:
    // Infrastructure failures may contain credentials, internal URLs, or transport details.
    h.writeError(c, http.StatusInternalServerError, "复习服务暂时不可用")
}
```

Rules:

- Only a **known sentinel's** message may reach the client.
- The `default` branch must return a stable, generic Chinese message. Never send `err.Error()` from an infrastructure, provider, or database error.
- Map CAS failures (`ErrVersionConflict`) to `409 Conflict` so the frontend can reload and retry. See `projectcourse/handler.go`.
- Test the mapping. See `review-service/domain/review/handler_test.go`, which asserts that the 500 body is `"复习服务暂时不可用"`.

## HTTP response shapes

**As-is:** the codebase uses four response shapes:

| Shape | Used by |
|-------|---------|
| `{"code": <http status>, "message": "...", "data": ...}` | auth, user, blog, review, parser, export, `shared/kernel/httpx/auth.go` |
| `{"error": "english text"}` | `core-api/domain/task/handler.go` (`writeServiceError`), `llm-stream/domain/stream/handler*.go` |
| `{"code": 0, "data": ...}` on success, `{"code", "message"}` on error | `core-api/domain/projectcourse/handler.go` |
| SSE `error` event with an English string | `externalStreamErrorMessage` in `llm-stream/domain/stream/handler.go` |

The frontend `requestEnvelope` helper (`frontend/src/services/apiClient.ts`) expects `{code: 200, message, data}` and reads `message`, then `error`, as the error text.

**Rules for new code:**

- New endpoints in a domain that already has a shape keep that domain's shape. Do not mix shapes inside one handler file.
- New domains use `{"code": <http status>, "message": "<中文>", "data": ...}`. It is the majority shape and the one `requestEnvelope` expects.
- User-visible messages are Chinese. Identifiers and codes stay as they are.

## Known debt (do not copy)

- 15 call sites send `err.Error()` straight to the client, including from non-sentinel errors: `core-api/domain/auth/handler.go` (5), `export-service/domain/export/handler.go` (4), `core-api/domain/project/handler.go` (3), `parser-service/domain/parse/handler.go`, `shared/kernel/httpx/auth.go`, `shared/kernel/httpx/health.go`.
- `task` and `llm-stream` handlers return English `{"error": ...}` bodies.
- Persisted task errors can contain raw provider text (see [Logging Guidelines](./logging-guidelines.md#known-debt-do-not-copy)).

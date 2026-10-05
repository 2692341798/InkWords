# Logging Guidelines

> What logs exist, which logger to use, and what must never be logged.

## Loggers in use (as-is)

- **Request logging:** `shared/kernel/httpx/request_logger.go` uses `log/slog` to emit JSON with `service`, `request_id`, `path`, `method`, `status`, and `latency_ms`. Every service bootstrap installs it as `r.Use(gin.Recovery(), httpx.RequestID(), httpx.RequestLogger("<svc>"))`. See `core-api/app/bootstrap/bootstrap.go`.
- **Request ID:** `httpx.RequestID()` reads or creates `X-Request-ID`.
- **Application logs:** mostly stdlib `log.Printf` / `log.Println` / `log.Fatal` (about 79 call sites). `slog` is used outside the request logger only in `llm-stream/domain/stream/handler_events.go`: `slog.Error("stream operation failed", "error", err)`.
- There is no zap or logrus. Do not add one.

## Rules for new code

- Prefer `log/slog` with key/value attributes in new code. Include `request_id` / `task_id` when you have them. `log.Printf` is acceptable inside a file that already uses it consistently.
- Use `log.Fatal` only in `cmd/main.go` startup.
- Log the cause on the server side **before** returning a generic public error, so the generic 500 is still diagnosable. See [Error Handling](./error-handling.md#mapping-errors-in-handlers).
- Record LLM call provenance: provider/model, input/output/cache tokens, latency, retries, and cost provenance. If the provider doesn't return a field, record it as unknown, not zero (`AGENTS.md` §6).

## Never log, persist, or return

- API keys, JWTs, `Authorization` headers, OAuth tokens.
- Raw provider error bodies. Redact them before user-visible or persistent logging.
- Imported source content in bulk. Prompt-injection text is kept only as quoted source data, never acted on.

There is no shared redaction helper yet. If your change touches one of these paths, add a small redaction function next to the adapter, with a test that proves the secret does not appear in the output. Don't extend the leaking patterns below.

Partial sanitizers you can reuse: `sanitizeUserID`, `shared/platform/llm/output_sanitize.go`, and the generic review 500 message.

## Known debt (do not copy)

- `shared/platform/llm/deepseek.go`: `APIError.Error()` returns `"API request failed with status %d: %s"` with the full provider `Body`.
- `llm-stream/domain/stream/task_consumer.go` calls `MarkFailed(ctx, id, err.Error())`. That persists raw error text into `error_message` and `job_task_events`, which core-api's task SSE then streams to the client.
- `shared/kernel/httpx/auth.go` logs the raw `Authorization` header when its format is invalid.

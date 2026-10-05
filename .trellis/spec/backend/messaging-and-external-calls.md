# Messaging & External Calls

> RabbitMQ task flow, SSE streaming, LLM calls, and the rules for anything that leaves the process.

## Task flow (as-is)

1. **core-api creates the job task.** `core-api/domain/task/service.go` `createTask` deduplicates by idempotency key, then publishes through `core-api/infra/mq/publisher.go` (`PublishWithContext`, persistent JSON). If publishing fails, the task is marked failed.
2. **A worker consumes the message.** Consumers: `llm-stream/cmd/delivery.go`, `parser-service/domain/parse/task_consumer.go`, `export-service/domain/export/consumer.go`, `course-runner/domain/verification/consumer.go`.
3. **The worker writes task state.** It updates `job_tasks` / `job_task_events` in the shared core DB.
4. **The client reads progress.** core-api `StreamTask` (`core-api/domain/task/handler.go`) polls `job_task_events` every 500 ms and streams them as SSE.

## Message envelope and routing

- **Envelope:** `shared/platform/rabbitmq/message.go` defines `{task_id, kind, user_id, payload json.RawMessage}`.
- **Exchange:** a topic exchange, `inkwords.events`.
- **Routing keys:** `generation.requested`, `parse.requested`, `export.requested`, `course.verification.requested`. The keys themselves are unversioned. The only versioned field is `result_version: 1` inside project-course results (`core-api/domain/projectcourse/repository.go`).
- **Contract test:** `backend/integration/pipeline_integration_test.go` (`//go:build integration`) covers the parser → MQ envelope → export contracts.

**Rules for new contracts:**

- Cross-service contracts must be versioned. Give a new message kind or payload a version field, and validate it on the consumer side before use.
- Put the payload type in `shared/kernel/` when more than one service decodes it.
- Consumers must be idempotent by task, stage, and input hash (see [Database Guidelines](./database-guidelines.md#idempotency)).
- Ack malformed or undecodable messages; don't requeue them. That is what existing consumers already do.

## Consumer acknowledgement

**As-is:** consumers ack manually. On a handler error they call `Nack(false, true)` and requeue, with no retry cap, no dead-letter queue, and no `Qos` prefetch. That lets a poison message loop forever. **Do not copy this in new consumers.** Bound retries using `job_tasks.retry_count`, then mark the task as terminally failed with a redacted Chinese message and ack the message.

## SSE streaming

- **llm-stream:** `sseStreamBody` (`llm-stream/domain/stream/handler_events.go`) uses `c.Stream`, emits `chunk`, `error`, `done` and `ping` (10 s keepalive), and drains channels when the client disconnects.
- **Server timeouts:** `httpx.NewServer` sets WriteTimeout to 0 so streams are not cut off. Read 15 s, ReadHeader 10 s, Idle 60 s, and graceful shutdown is 15 s through `ShutdownOnContextDone`.
- **The frontend never auto-retries streams** (`frontend/src/services/sse.ts`). That makes server-side stream handlers safe to keep non-idempotent, but every stream must end with a `done` or `error` event.

## LLM calls

**As-is:**

- The client lives in `shared/platform/llm/deepseek.go` (`DeepSeekClient`). Business code depends on small local interfaces over `platformllm.Message` / `ChatOptions`, such as `jsonGenerator` in `llm-stream/app/generation/prompt_service.go`.
- There is no `GenerationPort` and no `TaskModelPolicy` yet. Model names (`"deepseek-v4-flash"`, `"deepseek-v4-pro"`, `"deepseek-chat"`) are string literals spread over about 15 sites, e.g. `llm-stream/app/generation/generator_service.go`, `decomposition_quality.go`, `decomposition_series.go`, `core-api/app/projectanalysis/service.go`.
- `NewDeepSeekClient` uses `&http.Client{}` with no timeout. `IsRetryableError` exists but has no callers.
- Rate and concurrency limits: `rate.NewLimiter` (`LLM_API_RPM_LIMIT`) in `projectanalysis/service.go`, and semaphores in `decomposition_series.go` and `git_fetcher_github.go`.

**Rules for new code** (`AGENTS.md` §5–6, §9):

- Depend on a small consumer-side interface (the `jsonGenerator` pattern), never on provider request structs.
- Don't add new model-name literals. Read the model from settings/env (`DEEPSEEK_MODEL`, `DEEPSEEK_REVIEW_MODEL`) or from a single policy location. If you need a task-specific model, introduce or extend one policy table and don't scatter strings.
- Every external call takes a `context.Context` with a timeout, uses bounded retries (`IsRetryableError` is the intended predicate), and respects the existing rate and concurrency limiters.
- Ask for structured output for persisted contracts, and validate it with the `shared/kernel` `Validate()` methods before use. Claims must reference existing `EvidenceRef` IDs from the supplied EvidencePack.
- Cache keys include contract version, snapshot hashes, blueprint/revision, audience, provider/model/options, and the evidence hash. A cache hit is valid only when all of them match.
- Send only task-relevant evidence, never the whole corpus by default.

## Fetching imported sources

Revalidate HTTPS, allowed domains, DNS results, redirects, size, type, and crawl budget on every fetch. Never run an imported target repository. Generated teaching artifacts run only through the approved manifest and the sandbox, and sandbox execution fails closed (`course-runner`).

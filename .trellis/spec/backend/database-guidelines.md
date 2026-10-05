# Database Guidelines

> GORM + PostgreSQL (sqlite in tests). How repositories, schema, concurrency, and idempotency work.

## Databases

- **Core DB** (`DATABASE_URL`): shared by `core-api`, `llm-stream`, `parser-service`, `export-service`, `course-runner`. Workers write `job_tasks` / `job_task_events` directly.
- **Review DB** (`REVIEW_DATABASE_URL`): owned by `review-service`. Created by `backend/db/init/00-create-review-db.sql`.

## Schema changes

**As-is:** there are no SQL migrations and no goose dependency. Schema is created by GORM `AutoMigrate` at every service startup: `postgres.InitCore` / `InitReview` → `internal/infra/db/db.go` `autoMigrateCore` / `autoMigrateReview`. One raw Postgres-only partial expression index is created there too (`ensureProjectCourseResultIndex` on `job_task_events`).

**Rules for new code** (`AGENTS.md` §7):

- New production schema changes must be versioned goose SQL migrations. Goose is not installed yet, so the first migration means adding a dependency and a migration runner. Stop and get explicit user approval for that before doing it. Do not quietly add another model to `autoMigrateCore`.
- AutoMigrate is allowed only in tests and in explicitly documented transition paths.
- Put query-critical identity, status, relation, hash, and due-date data in columns. Use JSONB only for flexible documents.
- Any new or changed index needs the target query, PostgreSQL `EXPLAIN` evidence, and a note on write and storage cost.
- Destructive or irreversible migrations need a backup/restore plan and explicit user approval before they run.

## Repository pattern

Follow `backend/services/core-api/domain/projectcourse/repository.go`:

- A `Repository` interface in the domain package, plus a `GormRepository{db *gorm.DB}` built by `NewGormRepository(db)`.
- A compile-time check: `var _ Repository = (*GormRepository)(nil)`.
- `r.db.WithContext(ctx)` on every query.
- Scope every user-owned query by owner: `Where("id = ? AND user_id = ?", ...)`.
- Parameterized `Where` clauses only. Never build SQL by string concatenation.
- Return domain sentinel errors (`ErrNotFound`, `ErrVersionConflict`) rather than raw `gorm.ErrRecordNotFound`.

## Compare-and-swap (CAS) for revisions and approvals

Approved, locked, and manually edited content must never be overwritten without an explicit apply step and a CAS check. The local pattern is a conditional `Updates` followed by a `RowsAffected` check:

```go
result := r.db.WithContext(ctx).Model(&ProjectCourse{}).
    Where("id = ? AND user_id = ? AND blueprint_version = ? AND status NOT IN ?",
        courseID, userID, update.ExpectedVersion, immutableStatuses).
    Updates(map[string]any{ /* ... */ "blueprint_version": update.ExpectedVersion + 1 })
if result.Error != nil { return result.Error }
if result.RowsAffected == 0 { return ErrVersionConflict }
```

References: `UpdateBlueprintCAS`, `Approve`, `PersistProjectCourseResult`, `PersistProjectCourseGenerationResult` in `projectcourse/repository.go`. The HTTP side requires `expected_version` (`blueprintUpdateRequest` in `projectcourse/handler.go`). Claim-style locking: `ClaimResultPersistence` in `core-api/domain/task/repository.go` (`result_persisted_at IS NULL AND (started_at IS NULL OR started_at < stale)`).

## Transactions

Use `db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })` when several writes must succeed or fail together. References: `core-api/domain/task/generation_result_repository.go`, `core-api/domain/blog/persistence.go`, `llm-stream/domain/stream/blog_persistence.go`.

## Idempotency

Consumers and result persistence must be idempotent by task, stage, and input hash.

- **Task creation:** `createTask` in `core-api/domain/task/service.go` looks up `FindByIdempotencyKey(requestedBy, taskType, key)` before calling `Create`.
- **Stage results:** `CourseCheckpoint.InputHash` must have a `sha256:` prefix (`shared/kernel/projectcourse/checkpoint.go`). `FindCompletedProjectCourseResult` in `llm-stream/domain/stream/task_store.go` finds results by `course_id` / `stage` / `input_hash`.

**Known debt:** the idempotency-key column has a plain index, not a unique one, so concurrent creates can race. New idempotent writes should rely on a unique constraint or CAS, not on lookup-then-insert alone.

## JSONB

Use `gorm.io/datatypes.JSON` with `gorm:"type:jsonb;not null;default:'{}'"`. Examples: `BlueprintJSON`, `CoverageJSON`, `QualityReportJSON` in `core-api/domain/projectcourse/model.go`, and the models in `review-service/domain/review/model.go`. Validate the decoded document against its `shared/kernel` contract (`Validate()`) before you persist it.

## Known debt (do not copy)

- AutoMigrate runs in production on every bootstrap, and a global `db.DB` lives in `internal/infra/db`.
- Job-task and review models are duplicated across packages (see [Directory Structure](./directory-structure.md#known-debt-do-not-copy)). If you change a column, update every copy and its tests, or consolidate the copies in a separately scoped change.

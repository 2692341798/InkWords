-- +goose Up
-- InkWords migration role: review
-- Why: mastery history is owned by review-service and must not be created by runtime AutoMigrate.
CREATE TABLE mastery_objectives (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL,
    chapter_id TEXT NOT NULL,
    title TEXT NOT NULL CHECK (length(trim(title)) > 0),
    behavior TEXT NOT NULL CHECK (length(trim(behavior)) > 0),
    required_skills JSONB NOT NULL DEFAULT '[]'::jsonb,
    rubric JSONB NOT NULL DEFAULT '[]'::jsonb,
    key_points JSONB NOT NULL DEFAULT '[]'::jsonb,
    prerequisites JSONB NOT NULL DEFAULT '[]'::jsonb,
    evidence_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_mastery_objectives_workspace ON mastery_objectives (workspace_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_mastery_objectives_chapter ON mastery_objectives (chapter_id) WHERE deleted_at IS NULL;

CREATE TABLE mastery_attempts (
    id UUID PRIMARY KEY,
    objective_id UUID NOT NULL REFERENCES mastery_objectives (id) ON DELETE RESTRICT,
    skill VARCHAR(32) NOT NULL,
    correct BOOLEAN NOT NULL,
    independent BOOLEAN NOT NULL,
    hint_count INTEGER NOT NULL CHECK (hint_count >= 0),
    took_millis BIGINT NOT NULL CHECK (took_millis >= 0),
    confidence INTEGER NOT NULL CHECK (confidence BETWEEN 1 AND 5),
    error_kinds JSONB NOT NULL DEFAULT '[]'::jsonb,
    attempted_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_mastery_attempts_objective_time ON mastery_attempts (objective_id, attempted_at, created_at);

CREATE TABLE mastery_schedules (
    objective_id UUID PRIMARY KEY REFERENCES mastery_objectives (id) ON DELETE RESTRICT,
    next_skill VARCHAR(32) NOT NULL,
    due_at TIMESTAMPTZ NOT NULL,
    reason TEXT NOT NULL,
    algorithm_version VARCHAR(32) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_mastery_schedules_due ON mastery_schedules (due_at, objective_id);

-- +goose Down
-- Why: append-only attempts are learning evidence; remove only before any data exists.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM mastery_attempts) OR EXISTS (SELECT 1 FROM mastery_objectives) THEN
        RAISE EXCEPTION 'refusing to roll back mastery review migration after data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd
DROP TABLE mastery_schedules;
DROP TABLE mastery_attempts;
DROP TABLE mastery_objectives;

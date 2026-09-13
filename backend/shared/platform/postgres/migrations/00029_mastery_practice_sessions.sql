-- +goose Up
-- InkWords migration role: review
CREATE TABLE mastery_practice_sessions (
    id UUID PRIMARY KEY,
    objective_id UUID NOT NULL REFERENCES mastery_objectives(id) ON DELETE RESTRICT,
    skill TEXT NOT NULL CHECK (skill IN ('explain','complete','reproduce','transfer','diagnose','retain')),
    practice_task_id TEXT NOT NULL CHECK (char_length(practice_task_id) BETWEEN 1 AND 100),
    practice_content_hash TEXT NOT NULL CHECK (practice_content_hash ~ '^sha256:[0-9a-f]{64}$'),
    started_at TIMESTAMPTZ NOT NULL,
    submitted_at TIMESTAMPTZ,
    attempt_id UUID REFERENCES mastery_attempts(id) ON DELETE RESTRICT,
    submission_hash TEXT NOT NULL DEFAULT '',
    submission_result JSONB,
    CHECK (COALESCE((
      (submitted_at IS NULL AND attempt_id IS NULL AND submission_hash = '' AND submission_result IS NULL)
      OR (submitted_at >= started_at AND attempt_id IS NOT NULL
          AND submission_hash ~ '^sha256:[0-9a-f]{64}$' AND submission_result IS NOT NULL)
    ), FALSE))
);
-- The partial unique index makes repeated opens converge on one unfinished attempt.
CREATE UNIQUE INDEX ux_mastery_open_practice_session ON mastery_practice_sessions(objective_id,skill) WHERE submitted_at IS NULL;
-- Includes completed sessions when querying known help contact for one objective.
CREATE INDEX idx_mastery_sessions_objective ON mastery_practice_sessions(objective_id,id);
CREATE TABLE mastery_practice_help (
    session_id UUID NOT NULL REFERENCES mastery_practice_sessions(id) ON DELETE RESTRICT,
    kind TEXT NOT NULL,
    level INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY(session_id,kind,level),
    CHECK ((kind = 'hint' AND level BETWEEN 1 AND 3) OR (kind = 'answer' AND level = 0))
);
ALTER TABLE mastery_attempts ADD COLUMN practice_session_id UUID REFERENCES mastery_practice_sessions(id) ON DELETE RESTRICT;
CREATE UNIQUE INDEX ux_mastery_attempt_session ON mastery_attempts(practice_session_id) WHERE practice_session_id IS NOT NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM mastery_practice_sessions)
        OR EXISTS (SELECT 1 FROM mastery_attempts WHERE practice_session_id IS NOT NULL) THEN
        RAISE EXCEPTION 'refusing to remove practice sessions or disclosure evidence; retain schema or restore backup';
    END IF;
END
$$;
-- +goose StatementEnd
DROP INDEX ux_mastery_attempt_session;
ALTER TABLE mastery_attempts DROP COLUMN practice_session_id;
DROP TABLE mastery_practice_help;
DROP TABLE mastery_practice_sessions;

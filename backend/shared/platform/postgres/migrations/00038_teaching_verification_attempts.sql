-- +goose Up
-- InkWords migration role: core
CREATE TABLE textbook_verification_series (
    artifact_id UUID PRIMARY KEY REFERENCES textbook_code_artifacts(id) ON DELETE RESTRICT,
    workspace_id UUID NOT NULL REFERENCES local_workspaces(id) ON DELETE RESTRICT
);
ALTER TABLE job_tasks
    ADD COLUMN verification_artifact_id UUID REFERENCES textbook_code_artifacts(id) ON DELETE RESTRICT,
    ADD COLUMN verification_attempt INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN previous_verification_task_id UUID REFERENCES job_tasks(id) ON DELETE RESTRICT,
    ADD COLUMN verification_worker_token UUID,
    ADD COLUMN verification_worker_released_at TIMESTAMPTZ,
    ADD COLUMN verification_release_kind TEXT NOT NULL DEFAULT '';

-- Legacy minimal schemas may have no JSON/clock columns and no teaching tasks.
-- Do not invent executor exits for cancelled tasks: finished_at is cancel time.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM job_tasks WHERE task_subtype = 'textbook_teaching_artifact_verify') THEN
        EXECUTE $sql$
            UPDATE job_tasks SET
                verification_artifact_id = (payload_json ->> 'artifact_id')::uuid,
                verification_attempt = 1,
                verification_worker_token = CASE WHEN started_at IS NOT NULL THEN id END,
                verification_worker_released_at = CASE WHEN started_at IS NOT NULL AND status IN ('succeeded','failed') THEN finished_at END,
                verification_release_kind = CASE WHEN started_at IS NOT NULL AND finished_at IS NOT NULL AND status IN ('succeeded','failed') THEN 'legacy_terminal' ELSE '' END
            WHERE task_subtype = 'textbook_teaching_artifact_verify'
        $sql$;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE job_tasks ADD CONSTRAINT job_tasks_verification_attempt_shape CHECK (
    (task_subtype = 'textbook_teaching_artifact_verify' AND verification_artifact_id IS NOT NULL AND verification_attempt > 0
     AND ((verification_attempt = 1 AND previous_verification_task_id IS NULL)
       OR (verification_attempt > 1 AND previous_verification_task_id IS NOT NULL)))
    OR (task_subtype <> 'textbook_teaching_artifact_verify' AND verification_artifact_id IS NULL AND verification_attempt = 0
        AND previous_verification_task_id IS NULL AND verification_worker_token IS NULL)
);
ALTER TABLE job_tasks ADD CONSTRAINT job_tasks_verification_release_shape CHECK (
    (verification_worker_released_at IS NULL AND verification_release_kind = '')
    OR (verification_worker_released_at IS NOT NULL AND verification_worker_token IS NOT NULL
        AND verification_release_kind IN ('worker','legacy_terminal','operator_observed_exit'))
);
-- Latest-attempt lookup uses the artifact prefix and a backward index scan;
-- workspace ownership is checked separately. One small index write per attempt.
CREATE UNIQUE INDEX idx_job_tasks_verification_attempt
    ON job_tasks (verification_artifact_id, verification_attempt DESC)
    WHERE verification_artifact_id IS NOT NULL;
INSERT INTO textbook_verification_series (artifact_id, workspace_id)
    SELECT DISTINCT verification_artifact_id, workspace_id FROM job_tasks
    WHERE verification_artifact_id IS NOT NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM textbook_verification_series) THEN
        RAISE EXCEPTION 'refusing to discard verification attempts or executor exit evidence; retain schema or restore a verified backup';
    END IF;
END
$$;
-- +goose StatementEnd
DROP INDEX idx_job_tasks_verification_attempt;
ALTER TABLE job_tasks
    DROP CONSTRAINT job_tasks_verification_attempt_shape,
    DROP CONSTRAINT job_tasks_verification_release_shape,
    DROP COLUMN verification_artifact_id, DROP COLUMN verification_attempt,
    DROP COLUMN previous_verification_task_id, DROP COLUMN verification_worker_token,
    DROP COLUMN verification_worker_released_at, DROP COLUMN verification_release_kind;
DROP TABLE textbook_verification_series;

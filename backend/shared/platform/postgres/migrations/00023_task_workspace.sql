-- +goose Up
-- InkWords migration role: core
-- Why: every local task belongs to the installation workspace. Preserve
-- requested_by only as historical audit evidence, never as runtime authority.
UPDATE job_tasks AS task
SET workspace_id = owner.workspace_id
FROM local_workspace_legacy_owner AS owner
WHERE task.workspace_id IS NULL
  AND task.requested_by = owner.legacy_user_id;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM job_tasks WHERE workspace_id IS NULL) THEN
        RAISE EXCEPTION 'cannot migrate tasks to workspace identity while unmapped rows exist; repair the explicit legacy owner mapping before retrying';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE job_tasks
    DROP CONSTRAINT ck_job_tasks_legacy_owner_or_textbook_workspace,
    ALTER COLUMN workspace_id SET NOT NULL;

-- Existing ux_job_tasks_workspace_type_idempotency now covers every new task.
-- No second index is required; preserving requested_by avoids rewriting audit data.

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM job_tasks
        WHERE requested_by IS NULL
          AND task_subtype NOT IN (
              'textbook_sample_generate',
              'textbook_source_import',
              'textbook_official_web_import',
              'textbook_teaching_artifact_verify'
          )
    ) THEN
        RAISE EXCEPTION 'refusing to roll back task workspace identity after workspace-only compatibility tasks exist; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE job_tasks ALTER COLUMN workspace_id DROP NOT NULL;
ALTER TABLE job_tasks
    ADD CONSTRAINT ck_job_tasks_legacy_owner_or_textbook_workspace CHECK (
        requested_by IS NOT NULL
        OR (
            task_subtype IN (
                'textbook_sample_generate',
                'textbook_source_import',
                'textbook_official_web_import',
                'textbook_teaching_artifact_verify'
            )
            AND workspace_id IS NOT NULL
        )
    );

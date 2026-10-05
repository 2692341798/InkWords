-- +goose Up
-- InkWords migration role: core
-- Why: 教材任务属于本地 workspace；旧 requested_by 只保留为兼容队列字段，不能继续充当教材访问边界。
ALTER TABLE job_tasks
    ADD COLUMN workspace_id UUID REFERENCES local_workspaces (id) ON DELETE RESTRICT;

-- 已有教材任务由显式 legacy-owner 映射回填。无法唯一映射时迁移必须失败，
-- 不允许用任意用户或任意 workspace 猜测归属。
UPDATE job_tasks AS task
SET workspace_id = owner.workspace_id
FROM local_workspace_legacy_owner AS owner
WHERE task.requested_by = owner.legacy_user_id
  AND task.task_subtype IN (
      'textbook_sample_generate',
      'textbook_source_import',
      'textbook_official_web_import',
      'textbook_teaching_artifact_verify'
  );

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM job_tasks
        WHERE task_subtype IN (
            'textbook_sample_generate',
            'textbook_source_import',
            'textbook_official_web_import',
            'textbook_teaching_artifact_verify'
        )
          AND workspace_id IS NULL
    ) THEN
        RAISE EXCEPTION 'cannot migrate textbook tasks without an explicit local workspace owner mapping';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE job_tasks
    ADD CONSTRAINT ck_job_tasks_textbook_workspace CHECK (
        task_subtype NOT IN (
            'textbook_sample_generate',
            'textbook_source_import',
            'textbook_official_web_import',
            'textbook_teaching_artifact_verify'
        )
        OR workspace_id IS NOT NULL
    );

-- Target query: workspace-owned task creation reuses one frozen input and the
-- local task route reads that same identity. Cost: one partial B-tree entry per
-- keyed textbook task; legacy user-owned tasks do not enter the index.
CREATE UNIQUE INDEX ux_job_tasks_workspace_type_idempotency
    ON job_tasks (workspace_id, task_type, idempotency_key)
    WHERE workspace_id IS NOT NULL AND idempotency_key IS NOT NULL AND idempotency_key <> '';

-- +goose Down
-- Why: removing workspace ownership from any task would restore the legacy user bridge as authority.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM job_tasks WHERE workspace_id IS NOT NULL) THEN
        RAISE EXCEPTION 'refusing to roll back textbook task workspace identity after workspace-owned tasks exist; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP INDEX ux_job_tasks_workspace_type_idempotency;
ALTER TABLE job_tasks
    DROP CONSTRAINT ck_job_tasks_textbook_workspace,
    DROP COLUMN workspace_id;

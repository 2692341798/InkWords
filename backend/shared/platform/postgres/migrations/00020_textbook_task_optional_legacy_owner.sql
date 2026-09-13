-- +goose Up
-- InkWords migration role: core
-- Why: 新教材任务只属于 local workspace，不应为了填充 requested_by 而创建非登录 bridge user。
ALTER TABLE job_tasks
    ALTER COLUMN requested_by DROP NOT NULL;

-- 历史教材任务保留 requested_by 作为审计信息；所有非教材任务仍必须有 legacy user owner。
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

-- No new index: textbook identity and idempotency queries continue to use
-- ux_job_tasks_workspace_type_idempotency from migration 00019.

-- +goose Down
-- Why: restoring NOT NULL is safe only while no ownerless textbook task exists.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM job_tasks WHERE requested_by IS NULL) THEN
        RAISE EXCEPTION 'refusing to restore required legacy task owners after ownerless textbook tasks exist; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE job_tasks
    DROP CONSTRAINT ck_job_tasks_legacy_owner_or_textbook_workspace,
    ALTER COLUMN requested_by SET NOT NULL;

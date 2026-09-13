-- +goose Up
-- InkWords migration role: core
-- Why: worker retry must not append a second candidate revision for the same immutable generation task.
ALTER TABLE chapter_revisions
    ADD COLUMN generation_task_id UUID;

CREATE UNIQUE INDEX ux_chapter_revisions_generation_task
    ON chapter_revisions (generation_task_id)
    WHERE generation_task_id IS NOT NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM chapter_revisions WHERE generation_task_id IS NOT NULL) THEN
        RAISE EXCEPTION 'refusing to roll back generation task idempotency migration after task-linked candidates exist; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP INDEX ux_chapter_revisions_generation_task;
ALTER TABLE chapter_revisions DROP COLUMN generation_task_id;

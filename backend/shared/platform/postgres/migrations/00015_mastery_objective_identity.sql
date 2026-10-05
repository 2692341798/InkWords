-- +goose Up
-- InkWords migration role: review
-- Why: a frozen approved revision can be requested by more than one browser
-- tab. The active workspace/chapter identity is the idempotency boundary.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM mastery_objectives
        WHERE deleted_at IS NULL
        GROUP BY workspace_id, chapter_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'cannot add mastery objective identity: active duplicates exist; back up and resolve them before retrying migration';
    END IF;
END
$$;
-- +goose StatementEnd
CREATE UNIQUE INDEX uq_mastery_objectives_workspace_chapter_active
    ON mastery_objectives (workspace_id, chapter_id)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS uq_mastery_objectives_workspace_chapter_active;

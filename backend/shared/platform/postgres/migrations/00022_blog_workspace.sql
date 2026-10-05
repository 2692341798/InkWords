-- +goose Up
-- InkWords migration role: core
-- Why: local V1 owns all blogs through one installation workspace. Preserve
-- legacy user_id values as historical evidence while new rows use workspace_id.
ALTER TABLE blogs ADD COLUMN workspace_id UUID;

UPDATE blogs AS blog
SET workspace_id = owner.workspace_id
FROM local_workspace_legacy_owner AS owner
WHERE blog.user_id = owner.legacy_user_id
  AND blog.workspace_id IS NULL;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM blogs WHERE workspace_id IS NULL) THEN
        RAISE EXCEPTION 'cannot migrate blogs to workspace identity while unmapped rows exist; back up and repair the legacy owner mapping before retrying';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE blogs
    ADD CONSTRAINT fk_blogs_workspace
    FOREIGN KEY (workspace_id) REFERENCES local_workspaces (id) ON DELETE RESTRICT;
ALTER TABLE blogs ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE blogs ALTER COLUMN user_id DROP NOT NULL;

CREATE INDEX idx_blogs_workspace_parent_chapter
    ON blogs (workspace_id, parent_id, chapter_sort)
    WHERE deleted_at IS NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM blogs WHERE user_id IS NULL) THEN
        RAISE EXCEPTION 'refusing to roll back blog workspace identity after workspace-only data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP INDEX IF EXISTS idx_blogs_workspace_parent_chapter;
ALTER TABLE blogs DROP CONSTRAINT IF EXISTS fk_blogs_workspace;
ALTER TABLE blogs ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE blogs DROP COLUMN workspace_id;

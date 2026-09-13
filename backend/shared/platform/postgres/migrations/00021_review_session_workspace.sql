-- +goose Up
-- InkWords migration role: review
-- Why: local V1 has one installation workspace. Keep legacy user_id as historical
-- evidence while allowing all new review sessions to use workspace_id only.
ALTER TABLE review_sessions
    ADD COLUMN workspace_id UUID;

ALTER TABLE review_sessions
    ALTER COLUMN user_id DROP NOT NULL;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM review_sessions
        WHERE workspace_id IS NULL AND user_id IS NULL
    ) THEN
        RAISE EXCEPTION 'cannot add review workspace identity while ownerless sessions exist; back up and resolve them before retrying migration';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE review_sessions
    ADD CONSTRAINT ck_review_sessions_workspace_or_legacy_owner
    CHECK (workspace_id IS NOT NULL OR user_id IS NOT NULL);

CREATE INDEX idx_review_sessions_workspace_note_created
    ON review_sessions (workspace_id, note_path, created_at DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
-- Why: once sessions are workspace-owned, reverting would discard their only identity.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM review_sessions WHERE workspace_id IS NOT NULL) THEN
        RAISE EXCEPTION 'refusing to roll back review workspace identity after workspace-owned data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP INDEX IF EXISTS idx_review_sessions_workspace_note_created;
ALTER TABLE review_sessions DROP CONSTRAINT IF EXISTS ck_review_sessions_workspace_or_legacy_owner;
ALTER TABLE review_sessions ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE review_sessions DROP COLUMN workspace_id;

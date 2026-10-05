-- +goose Up
-- InkWords migration role: review
-- Why: every persisted review session is now owned by the installation workspace;
-- retaining user_id would preserve a retired account authority in the review schema.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM review_sessions WHERE workspace_id IS NULL) THEN
        RAISE EXCEPTION 'refusing review legacy identity cleanup while sessions lack workspace ownership';
    END IF;
END
$$;
-- +goose StatementEnd

DROP INDEX IF EXISTS idx_review_sessions_user_note_created;
ALTER TABLE review_sessions
    DROP CONSTRAINT IF EXISTS ck_review_sessions_workspace_or_legacy_owner,
    ALTER COLUMN workspace_id SET NOT NULL,
    DROP COLUMN IF EXISTS user_id;

-- +goose Down
-- Why: removed owner values cannot be reconstructed without inventing identity.
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'review legacy identity cleanup is irreversible; restore the verified local backup instead';
END
$$;
-- +goose StatementEnd

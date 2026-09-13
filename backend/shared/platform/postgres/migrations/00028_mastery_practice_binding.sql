-- +goose Up
-- InkWords migration role: review
-- Immutable projections are read models, never an independently editable book.
-- No new index: all accesses use the existing objective primary key.
ALTER TABLE mastery_objectives
    ADD COLUMN practice_revision_id UUID,
    ADD COLUMN practice_content_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN practice_projection JSONB,
    ADD CONSTRAINT ck_mastery_practice_basis CHECK (COALESCE((
        (practice_revision_id IS NULL AND practice_content_hash = '' AND practice_projection IS NULL)
        OR (practice_revision_id IS NOT NULL
            AND practice_content_hash ~ '^sha256:[0-9a-f]{64}$'
            AND practice_projection IS NOT NULL
            AND practice_projection->>'format' = 'inkwords.learning-projection.v2'
            AND practice_projection->>'revision_id' = practice_revision_id::text
            AND practice_projection->>'content_hash' = practice_content_hash
            AND chapter_id = 'approved-revision:' || (practice_projection->>'chapter_id') || ':' || practice_revision_id::text)
    ), FALSE));
ALTER TABLE mastery_attempts
    ADD COLUMN practice_task_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN practice_content_hash TEXT NOT NULL DEFAULT '',
    ADD CONSTRAINT ck_mastery_attempt_practice CHECK (
        (practice_task_id = '' AND practice_content_hash = '')
        OR (char_length(practice_task_id) BETWEEN 1 AND 100
            AND practice_content_hash ~ '^sha256:[0-9a-f]{64}$')
    );

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM mastery_objectives WHERE practice_revision_id IS NOT NULL)
        OR EXISTS (SELECT 1 FROM mastery_attempts WHERE practice_task_id <> '') THEN
        RAISE EXCEPTION 'refusing to remove frozen mastery practice evidence; retain schema or restore backup';
    END IF;
END
$$;
-- +goose StatementEnd
ALTER TABLE mastery_attempts DROP CONSTRAINT ck_mastery_attempt_practice,
    DROP COLUMN practice_task_id, DROP COLUMN practice_content_hash;
ALTER TABLE mastery_objectives DROP CONSTRAINT ck_mastery_practice_basis,
    DROP COLUMN practice_revision_id, DROP COLUMN practice_content_hash, DROP COLUMN practice_projection;

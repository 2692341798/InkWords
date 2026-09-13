-- +goose Up
-- InkWords migration role: core
-- Why: failed sample-generation tasks must remain discoverable from their chapter
-- after browser-local state is lost. The relation belongs in a column so recovery
-- does not depend on scanning or interpreting JSONB at read time.
ALTER TABLE job_tasks
    ADD COLUMN textbook_chapter_id UUID REFERENCES textbook_chapters (id) ON DELETE RESTRICT;

-- Some supported legacy databases have the pre-JSON task table. They cannot
-- contain a recoverable sample task; allow that empty shape to upgrade, but
-- fail closed if it claims to contain one.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'job_tasks'
          AND column_name = 'payload_json'
    ) THEN
        EXECUTE $sql$
            UPDATE job_tasks
            SET textbook_chapter_id = NULLIF(trim(payload_json ->> 'chapter_id'), '')::uuid
            WHERE task_subtype = 'textbook_sample_generate'
        $sql$;
    ELSIF EXISTS (SELECT 1 FROM job_tasks WHERE task_subtype = 'textbook_sample_generate') THEN
        RAISE EXCEPTION 'cannot recover textbook sample task chapter identity without payload_json';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE job_tasks
    ADD CONSTRAINT job_tasks_textbook_sample_chapter_check
        CHECK (task_subtype <> 'textbook_sample_generate' OR textbook_chapter_id IS NOT NULL) NOT VALID;

ALTER TABLE job_tasks VALIDATE CONSTRAINT job_tasks_textbook_sample_chapter_check;

-- Target query: recover the newest sample-generation task for one local
-- workspace and chapter, ordered deterministically across equal timestamps.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'job_tasks'
          AND column_name = 'created_at'
    ) THEN
        CREATE INDEX idx_job_tasks_textbook_chapter_created
            ON job_tasks (workspace_id, textbook_chapter_id, created_at DESC, id DESC)
            WHERE task_subtype = 'textbook_sample_generate';
    ELSIF EXISTS (SELECT 1 FROM job_tasks WHERE task_subtype = 'textbook_sample_generate') THEN
        RAISE EXCEPTION 'cannot index textbook sample task recovery without created_at';
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- The column is a derived relation; the immutable payload remains authoritative,
-- so dropping this recovery projection does not discard task input or results.
DROP INDEX IF EXISTS idx_job_tasks_textbook_chapter_created;

ALTER TABLE job_tasks
    DROP CONSTRAINT job_tasks_textbook_sample_chapter_check,
    DROP COLUMN textbook_chapter_id;

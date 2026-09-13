-- +goose Up
-- InkWords migration role: core
-- Explicitly delegated AI evidence must never enter the human-review table.
CREATE TABLE textbook_delegated_publication_reviews (
    id UUID PRIMARY KEY,
    build_id UUID NOT NULL REFERENCES textbook_book_builds (id) ON DELETE RESTRICT,
    stage VARCHAR(32) NOT NULL CHECK (stage IN ('developmental', 'technical', 'self_study', 'consistency', 'copy_editing', 'layout', 'rights', 'reader_trial')),
    revision INTEGER NOT NULL CHECK (revision > 0),
    verdict VARCHAR(32) NOT NULL CHECK (verdict IN ('pass', 'needs_revision', 'not_assessed')),
    manifest_hash TEXT NOT NULL CHECK (manifest_hash ~ '^sha256:[0-9a-f]{64}$'),
    input_hash TEXT NOT NULL CHECK (input_hash ~ '^sha256:[0-9a-f]{64}$'),
    document_json JSONB NOT NULL CHECK (COALESCE((
        document_json->>'contract_version' = 'inkwords.delegated-publication-review.v1'
        AND document_json->>'reviewer_kind' = 'delegated_ai'
        AND document_json->>'id' = id::text
        AND document_json->>'build_id' = build_id::text
        AND document_json->>'stage' = stage
        AND document_json->>'verdict' = verdict
        AND document_json->>'manifest_hash' = manifest_hash
        AND (document_json->>'revision')::integer = revision
    ), FALSE)),
    completed_at TIMESTAMPTZ NOT NULL,
    UNIQUE (build_id, stage, revision)
);
-- Query: one build's review history ordered by stage/revision; latest stage
-- lookup uses the same unique B-tree. Cost: one PK and one composite-index
-- write per append; no second overlapping index or mutable latest pointer.

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM textbook_delegated_publication_reviews) THEN
        RAISE EXCEPTION 'refusing to discard delegated editorial evidence; retain migration and roll back application or restore a verified backup';
    END IF;
END
$$;
-- +goose StatementEnd
DROP TABLE textbook_delegated_publication_reviews;

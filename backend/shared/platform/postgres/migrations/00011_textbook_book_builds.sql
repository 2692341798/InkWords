-- +goose Up
-- InkWords migration role: core
-- Why: publication outputs must identify an immutable set of approved chapter
-- revisions and their provenance instead of following mutable project pointers.
CREATE TABLE textbook_book_builds (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES textbook_projects (id) ON DELETE RESTRICT,
    book_contract_revision_id UUID NOT NULL REFERENCES book_contract_revisions (id) ON DELETE RESTRICT,
    style_sheet_revision_id UUID NOT NULL REFERENCES style_sheet_revisions (id) ON DELETE RESTRICT,
    approved_revision_ids JSONB NOT NULL,
    manifest_json JSONB NOT NULL,
    manifest_hash TEXT NOT NULL CHECK (manifest_hash LIKE 'sha256:%'),
    status VARCHAR(32) NOT NULL CHECK (status IN ('draft', 'ready_for_review', 'publication_candidate', 'blocked')),
    blockers_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, manifest_hash)
);

CREATE INDEX idx_textbook_book_builds_project_created
    ON textbook_book_builds (project_id, created_at DESC);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM textbook_book_builds) THEN
        RAISE EXCEPTION 'refusing to roll back book builds after data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd
DROP TABLE textbook_book_builds;

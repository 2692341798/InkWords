-- +goose Up
-- InkWords migration role: core
CREATE TABLE textbook_rights_amendments (
    id UUID PRIMARY KEY,
    build_id UUID NOT NULL REFERENCES textbook_book_builds(id) ON DELETE RESTRICT,
    base_item_id UUID NOT NULL REFERENCES textbook_rights_items(id) ON DELETE RESTRICT,
    previous_amendment_id UUID REFERENCES textbook_rights_amendments(id) ON DELETE RESTRICT,
    revision INTEGER NOT NULL CHECK (revision > 0),
    manifest_hash TEXT NOT NULL CHECK (manifest_hash ~ '^sha256:[0-9a-f]{64}$'),
    input_hash TEXT NOT NULL CHECK (input_hash ~ '^sha256:[0-9a-f]{64}$'),
    document_json JSONB NOT NULL CHECK (COALESCE((
        document_json->>'contract_version' = 'inkwords.rights-amendment.v1'
        AND document_json->>'id' = id::text
        AND document_json->>'build_id' = build_id::text
        AND document_json->>'base_item_id' = base_item_id::text
        AND document_json->>'previous_amendment_id' = COALESCE(previous_amendment_id::text, '')
        AND document_json->>'manifest_hash' = manifest_hash
        AND (document_json->>'revision')::integer = revision
    ), FALSE)),
    completed_at TIMESTAMPTZ NOT NULL,
    CHECK ((revision = 1) = (previous_amendment_id IS NULL)),
    UNIQUE (build_id, base_item_id, revision),
    UNIQUE (previous_amendment_id)
);
-- Target query: one build's history ordered by base_item_id, revision.
-- The composite unique B-tree also prevents a second root/fork for a work.
-- Cost per append: PK + composite uniqueness + predecessor uniqueness; no
-- mutable current pointer or additional overlapping build index.

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM textbook_rights_amendments) THEN
        RAISE EXCEPTION 'refusing to discard rights evidence; retain migration and roll back application or restore a verified backup';
    END IF;
END $$;
-- +goose StatementEnd
DROP TABLE textbook_rights_amendments;

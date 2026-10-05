-- +goose Up
-- InkWords migration role: core
-- Why: 教材证据必须回到不可变快照中的具体文档与片段，不能只保留一段合并文本。
CREATE TABLE source_documents (
    id TEXT PRIMARY KEY,
    snapshot_id UUID NOT NULL REFERENCES source_snapshots (id) ON DELETE RESTRICT,
    canonical_locator TEXT NOT NULL CHECK (length(trim(canonical_locator)) > 0),
    title TEXT NOT NULL CHECK (length(trim(title)) > 0),
    media_type TEXT NOT NULL CHECK (length(trim(media_type)) > 0),
    content_hash TEXT NOT NULL CHECK (content_hash LIKE 'sha256:%'),
    parent_id TEXT REFERENCES source_documents (id) ON DELETE RESTRICT,
    artifact_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX ux_source_documents_snapshot_locator_hash
    ON source_documents (snapshot_id, canonical_locator, content_hash);
CREATE INDEX idx_source_documents_snapshot_created
    ON source_documents (snapshot_id, created_at);

CREATE TABLE source_chunks (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES source_documents (id) ON DELETE RESTRICT,
    ordinal INTEGER NOT NULL CHECK (ordinal > 0),
    heading_path JSONB NOT NULL DEFAULT '[]'::jsonb,
    locator JSONB NOT NULL,
    paragraph INTEGER CHECK (paragraph IS NULL OR paragraph > 0),
    code_language TEXT,
    start_byte INTEGER NOT NULL DEFAULT 0 CHECK (start_byte >= 0),
    end_byte INTEGER NOT NULL DEFAULT 0 CHECK (end_byte = 0 OR end_byte > start_byte),
    text_hash TEXT NOT NULL CHECK (text_hash LIKE 'sha256:%'),
    search_text TEXT NOT NULL CHECK (length(trim(search_text)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (document_id, ordinal)
);

CREATE INDEX idx_source_chunks_document_ordinal ON source_chunks (document_id, ordinal);
CREATE INDEX idx_source_chunks_text_hash ON source_chunks (text_hash);

-- +goose Down
-- Why: 已持久化的来源证据是教材可审计性的依据；有数据时仅能从备份恢复。
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM source_chunks)
        OR EXISTS (SELECT 1 FROM source_documents) THEN
        RAISE EXCEPTION 'refusing to roll back source document migration after data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP TABLE source_chunks;
DROP TABLE source_documents;

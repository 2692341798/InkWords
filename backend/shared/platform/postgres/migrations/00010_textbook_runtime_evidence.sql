-- +goose Up
-- InkWords migration role: core
-- Why: 运行结果、截图和录屏必须可追溯到不可变教材修订与教学工件，不能以临时文件或模型叙述替代真实证据。
CREATE TABLE textbook_code_artifacts (
    id UUID PRIMARY KEY,
    revision_id UUID NOT NULL REFERENCES chapter_revisions (id) ON DELETE RESTRICT,
    kind VARCHAR(32) NOT NULL,
    language VARCHAR(64) NOT NULL,
    entrypoint TEXT,
    source_tree_path TEXT,
    source_ref TEXT,
    manifest_json JSONB NOT NULL,
    manifest_hash TEXT NOT NULL CHECK (manifest_hash LIKE 'sha256:%'),
    artifact_hash TEXT NOT NULL CHECK (artifact_hash LIKE 'sha256:%'),
    limitations_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    status VARCHAR(32) NOT NULL CHECK (status IN ('draft', 'unverified', 'verified', 'blocked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (revision_id, artifact_hash)
);

CREATE INDEX idx_textbook_code_artifacts_revision
    ON textbook_code_artifacts (revision_id, created_at DESC);

CREATE TABLE textbook_runtime_evidence (
    id UUID PRIMARY KEY,
    revision_id UUID NOT NULL REFERENCES chapter_revisions (id) ON DELETE RESTRICT,
    code_artifact_id UUID NOT NULL REFERENCES textbook_code_artifacts (id) ON DELETE RESTRICT,
    code_artifact_hash TEXT NOT NULL CHECK (code_artifact_hash LIKE 'sha256:%'),
    input_hash TEXT NOT NULL CHECK (input_hash LIKE 'sha256:%'),
    kind VARCHAR(32) NOT NULL CHECK (kind IN ('terminal_output', 'browser_page', 'ide_capture', 'image')),
    status VARCHAR(32) NOT NULL CHECK (status IN ('draft', 'unverified', 'verified', 'blocked')),
    command_manifest_hash TEXT,
    runner_image_digest TEXT,
    toolchain_version TEXT,
    tool_name TEXT,
    tool_version TEXT,
    sampling_conditions_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    structured_output TEXT,
    raw_evidence_ref TEXT,
    output_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    captured_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    stale_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (expires_at IS NULL OR captured_at IS NULL OR expires_at >= captured_at),
    CHECK (
        status <> 'verified' OR (
            command_manifest_hash IS NOT NULL
            AND runner_image_digest IS NOT NULL
            AND toolchain_version IS NOT NULL
            AND raw_evidence_ref IS NOT NULL
            AND captured_at IS NOT NULL
        )
    ),
    CHECK (status <> 'blocked' OR stale_reason IS NOT NULL)
);

CREATE INDEX idx_textbook_runtime_evidence_revision_current
    ON textbook_runtime_evidence (revision_id, status, expires_at DESC);
CREATE INDEX idx_textbook_runtime_evidence_artifact
    ON textbook_runtime_evidence (code_artifact_id, created_at DESC);

CREATE TABLE textbook_manuscript_assets (
    id UUID PRIMARY KEY,
    revision_id UUID NOT NULL REFERENCES chapter_revisions (id) ON DELETE RESTRICT,
    evidence_id UUID NOT NULL REFERENCES textbook_runtime_evidence (id) ON DELETE RESTRICT,
    stable_ref TEXT NOT NULL UNIQUE,
    kind VARCHAR(32) NOT NULL CHECK (kind IN ('screenshot', 'diagram', 'recording')),
    content_hash TEXT NOT NULL CHECK (content_hash LIKE 'sha256:%'),
    alt_text TEXT NOT NULL CHECK (length(trim(alt_text)) > 0),
    source TEXT NOT NULL CHECK (length(trim(source)) > 0),
    generation_method TEXT NOT NULL CHECK (length(trim(generation_method)) > 0),
    rights_status VARCHAR(32) NOT NULL CHECK (rights_status IN ('pending', 'ready', 'blocked')),
    status VARCHAR(32) NOT NULL CHECK (status IN ('draft', 'unverified', 'verified', 'blocked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_textbook_manuscript_assets_revision
    ON textbook_manuscript_assets (revision_id, created_at DESC);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM textbook_code_artifacts)
       OR EXISTS (SELECT 1 FROM textbook_runtime_evidence)
       OR EXISTS (SELECT 1 FROM textbook_manuscript_assets) THEN
        RAISE EXCEPTION 'refusing to roll back textbook runtime evidence after data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd
DROP TABLE textbook_manuscript_assets;
DROP TABLE textbook_runtime_evidence;
DROP TABLE textbook_code_artifacts;

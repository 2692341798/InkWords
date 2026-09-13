-- +goose Up
-- InkWords migration role: core
-- Why: 教材母稿、资料与修订是 core-api 的权威事实，不能再混入旧 ProjectCourse JSON。
CREATE TABLE textbook_projects (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES local_workspaces (id) ON DELETE RESTRICT,
    title TEXT NOT NULL CHECK (length(trim(title)) > 0),
    audience_level VARCHAR(32) NOT NULL CHECK (audience_level IN ('foundation', 'programming', 'stack_familiar')),
    status VARCHAR(32) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'candidate', 'approved', 'needs_evidence', 'verification_failed', 'archived')),
    primary_source_id UUID,
    approved_blueprint_revision_id UUID,
    revision_version INTEGER NOT NULL DEFAULT 1 CHECK (revision_version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_textbook_projects_workspace_status_created
    ON textbook_projects (workspace_id, status, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE textbook_sources (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES textbook_projects (id) ON DELETE RESTRICT,
    kind VARCHAR(32) NOT NULL CHECK (kind IN ('git_repository', 'official_web', 'pdf', 'docx', 'markdown', 'text', 'zip')),
    role VARCHAR(32) NOT NULL CHECK (role IN ('primary', 'official_supporting', 'user_reference')),
    locator TEXT NOT NULL CHECK (length(trim(locator)) > 0),
    official_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    license_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CHECK (role <> 'official_supporting' OR official_confirmed)
);

CREATE UNIQUE INDEX ux_textbook_sources_one_primary_per_project
    ON textbook_sources (project_id)
    WHERE role = 'primary' AND deleted_at IS NULL;
CREATE INDEX idx_textbook_sources_project_role
    ON textbook_sources (project_id, role, created_at)
    WHERE deleted_at IS NULL;

CREATE TABLE source_snapshots (
    id UUID PRIMARY KEY,
    source_id UUID NOT NULL REFERENCES textbook_sources (id) ON DELETE RESTRICT,
    resolved_version TEXT NOT NULL,
    content_hash CHAR(64) NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(32) NOT NULL CHECK (status IN ('pending', 'captured', 'failed', 'partial')),
    limits_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_source_snapshots_source_captured
    ON source_snapshots (source_id, captured_at DESC);

CREATE TABLE book_contract_revisions (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES textbook_projects (id) ON DELETE RESTRICT,
    revision_number INTEGER NOT NULL CHECK (revision_number > 0),
    document_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    content_hash CHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN ('draft', 'candidate', 'approved')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (project_id, revision_number)
);

CREATE TABLE style_sheet_revisions (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES textbook_projects (id) ON DELETE RESTRICT,
    revision_number INTEGER NOT NULL CHECK (revision_number > 0),
    document_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    content_hash CHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN ('draft', 'candidate', 'approved')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (project_id, revision_number)
);

CREATE TABLE blueprint_revisions (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES textbook_projects (id) ON DELETE RESTRICT,
    revision_number INTEGER NOT NULL CHECK (revision_number > 0),
    document_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    content_hash CHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN ('draft', 'candidate', 'approved')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (project_id, revision_number)
);

CREATE INDEX idx_blueprint_revisions_project_status_number
    ON blueprint_revisions (project_id, status, revision_number DESC);

CREATE TABLE textbook_chapters (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES textbook_projects (id) ON DELETE RESTRICT,
    sort_order INTEGER NOT NULL CHECK (sort_order > 0),
    title TEXT NOT NULL CHECK (length(trim(title)) > 0),
    chapter_profile VARCHAR(32) NOT NULL CHECK (chapter_profile IN ('concept', 'hands_on', 'source_walkthrough', 'project_iteration', 'troubleshooting', 'integration_review', 'reference')),
    status VARCHAR(32) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'candidate', 'approved', 'needs_evidence', 'verification_failed', 'archived')),
    current_revision_id UUID,
    approved_revision_id UUID,
    revision_version INTEGER NOT NULL DEFAULT 0 CHECK (revision_version >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX ux_textbook_chapters_project_sort
    ON textbook_chapters (project_id, sort_order)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_textbook_chapters_project_status_sort
    ON textbook_chapters (project_id, status, sort_order)
    WHERE deleted_at IS NULL;

CREATE TABLE chapter_revisions (
    id UUID PRIMARY KEY,
    chapter_id UUID NOT NULL REFERENCES textbook_chapters (id) ON DELETE RESTRICT,
    revision_number INTEGER NOT NULL CHECK (revision_number > 0),
    kind VARCHAR(32) NOT NULL CHECK (kind IN ('draft', 'candidate', 'approved')),
    markdown TEXT NOT NULL,
    document_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    content_hash CHAR(64) NOT NULL,
    created_by VARCHAR(32) NOT NULL CHECK (created_by IN ('manual', 'generation')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (chapter_id, revision_number)
);

CREATE INDEX idx_chapter_revisions_chapter_number
    ON chapter_revisions (chapter_id, revision_number DESC);

ALTER TABLE textbook_projects
    ADD CONSTRAINT fk_textbook_projects_primary_source
    FOREIGN KEY (primary_source_id) REFERENCES textbook_sources (id) ON DELETE RESTRICT;
ALTER TABLE textbook_projects
    ADD CONSTRAINT fk_textbook_projects_approved_blueprint_revision
    FOREIGN KEY (approved_blueprint_revision_id) REFERENCES blueprint_revisions (id) ON DELETE RESTRICT;
ALTER TABLE textbook_chapters
    ADD CONSTRAINT fk_textbook_chapters_current_revision
    FOREIGN KEY (current_revision_id) REFERENCES chapter_revisions (id) ON DELETE RESTRICT;
ALTER TABLE textbook_chapters
    ADD CONSTRAINT fk_textbook_chapters_approved_revision
    FOREIGN KEY (approved_revision_id) REFERENCES chapter_revisions (id) ON DELETE RESTRICT;

CREATE TABLE chapter_locks (
    chapter_id UUID PRIMARY KEY REFERENCES textbook_chapters (id) ON DELETE RESTRICT,
    owner_id UUID NOT NULL,
    version INTEGER NOT NULL CHECK (version > 0),
    lease_expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_chapter_locks_lease_expires_at ON chapter_locks (lease_expires_at);

-- +goose Down
-- Why: 第一批教材数据表一旦有数据就不能用 Down 丢弃；必须从备份恢复。
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM textbook_projects)
        OR EXISTS (SELECT 1 FROM textbook_sources)
        OR EXISTS (SELECT 1 FROM source_snapshots)
        OR EXISTS (SELECT 1 FROM book_contract_revisions)
        OR EXISTS (SELECT 1 FROM style_sheet_revisions)
        OR EXISTS (SELECT 1 FROM blueprint_revisions)
        OR EXISTS (SELECT 1 FROM textbook_chapters)
        OR EXISTS (SELECT 1 FROM chapter_revisions)
        OR EXISTS (SELECT 1 FROM chapter_locks) THEN
        RAISE EXCEPTION 'refusing to roll back textbook core migration after data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP TABLE chapter_locks;
ALTER TABLE textbook_chapters DROP CONSTRAINT fk_textbook_chapters_approved_revision;
ALTER TABLE textbook_chapters DROP CONSTRAINT fk_textbook_chapters_current_revision;
ALTER TABLE textbook_projects DROP CONSTRAINT fk_textbook_projects_approved_blueprint_revision;
ALTER TABLE textbook_projects DROP CONSTRAINT fk_textbook_projects_primary_source;
DROP TABLE chapter_revisions;
DROP TABLE textbook_chapters;
DROP TABLE blueprint_revisions;
DROP TABLE style_sheet_revisions;
DROP TABLE book_contract_revisions;
DROP TABLE source_snapshots;
DROP TABLE textbook_sources;
DROP TABLE textbook_projects;

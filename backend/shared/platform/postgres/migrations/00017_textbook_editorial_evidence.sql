-- +goose Up
-- InkWords migration role: core
-- Why: 出版候选必须依赖与不可变 BookBuild 绑定的人工审校和逐作品权利证据，不能依赖浏览器勾选状态。
CREATE TABLE textbook_rights_items (
    id UUID PRIMARY KEY,
    build_id UUID NOT NULL REFERENCES textbook_book_builds (id) ON DELETE RESTRICT,
    project_id UUID NOT NULL REFERENCES textbook_projects (id) ON DELETE RESTRICT,
    subject_ref TEXT NOT NULL CHECK (length(trim(subject_ref)) > 0),
    work_type VARCHAR(32) NOT NULL CHECK (work_type IN ('prose', 'code', 'image', 'screenshot', 'font', 'trademark', 'data')),
    rights_basis TEXT NOT NULL CHECK (length(trim(rights_basis)) > 0),
    allowed_use TEXT NOT NULL CHECK (length(trim(allowed_use)) > 0),
    attribution TEXT NOT NULL CHECK (length(trim(attribution)) > 0),
    publication_status VARCHAR(32) NOT NULL CHECK (publication_status IN ('pending', 'ready', 'blocked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (build_id, subject_ref)
);

-- Target query: list one frozen build's immutable rights ledger in audit order.
-- Cost: one bounded B-tree write per rights item; rows are append-only and low volume.
CREATE INDEX idx_textbook_rights_items_build_created
    ON textbook_rights_items (build_id, created_at, id);

CREATE TABLE textbook_publication_reviews (
    id UUID PRIMARY KEY,
    build_id UUID NOT NULL REFERENCES textbook_book_builds (id) ON DELETE RESTRICT,
    stage VARCHAR(32) NOT NULL CHECK (stage IN ('developmental', 'technical', 'self_study', 'consistency', 'copy_editing', 'layout', 'rights', 'reader_trial')),
    reviewer TEXT NOT NULL CHECK (length(trim(reviewer)) > 0),
    notes TEXT NOT NULL CHECK (char_length(trim(notes)) BETWEEN 8 AND 4000),
    automated BOOLEAN NOT NULL DEFAULT FALSE CHECK (automated = FALSE),
    completed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (build_id, stage)
);

-- Target query: list the eight stage records for one build in completion order.
-- Cost: one bounded B-tree write per stage; at most eight rows exist per build.
CREATE INDEX idx_textbook_publication_reviews_build_completed
    ON textbook_publication_reviews (build_id, completed_at, id);

-- +goose Down
-- Why: 权利与人工审校记录是不可替代的出版证据；存在数据时只能从备份恢复。
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM textbook_rights_items)
       OR EXISTS (SELECT 1 FROM textbook_publication_reviews) THEN
        RAISE EXCEPTION 'refusing to roll back textbook editorial evidence after data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP TABLE textbook_publication_reviews;
DROP TABLE textbook_rights_items;

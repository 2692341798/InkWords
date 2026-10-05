-- +goose Up
-- InkWords migration role: core
-- Why: 人工驳回是教材候选稿审阅证据，必须独立于可变 UI 状态并绑定被审阅的不可变内容。
CREATE TABLE textbook_candidate_reviews (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES textbook_projects (id) ON DELETE RESTRICT,
    chapter_id UUID NOT NULL REFERENCES textbook_chapters (id) ON DELETE RESTRICT,
    candidate_revision_id UUID NOT NULL REFERENCES chapter_revisions (id) ON DELETE RESTRICT,
    reviewer_workspace_id UUID NOT NULL REFERENCES local_workspaces (id) ON DELETE RESTRICT,
    decision VARCHAR(32) NOT NULL CHECK (decision = 'rejected'),
    reason TEXT NOT NULL CHECK (char_length(trim(reason)) BETWEEN 8 AND 2000),
    candidate_content_hash CHAR(64) NOT NULL,
    quality_contract_version TEXT NOT NULL CHECK (length(trim(quality_contract_version)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (candidate_revision_id)
);

CREATE INDEX idx_textbook_candidate_reviews_chapter_created
    ON textbook_candidate_reviews (chapter_id, created_at DESC);

-- +goose Down
-- Why: 人工驳回理由属于不可替代的审阅历史；存在记录时只能从备份恢复，不能静默丢弃。
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM textbook_candidate_reviews) THEN
        RAISE EXCEPTION 'refusing to roll back candidate reviews after data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP TABLE textbook_candidate_reviews;

-- +goose Up
-- InkWords migration role: core
-- Why: 每次教材资料检索都必须保留查询、候选、分数理由与最终选择，不能只把片段塞进生成提示后失去可解释性。
CREATE TABLE source_retrieval_runs (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES textbook_projects (id) ON DELETE RESTRICT,
    query TEXT NOT NULL CHECK (length(trim(query)) BETWEEN 2 AND 1024),
    candidates_json JSONB NOT NULL,
    selected_json JSONB NOT NULL,
    input_hash TEXT NOT NULL CHECK (input_hash LIKE 'sha256:%'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ck_source_retrieval_runs_candidates_array CHECK (jsonb_typeof(candidates_json) = 'array'),
    CONSTRAINT ck_source_retrieval_runs_selected_array CHECK (jsonb_typeof(selected_json) = 'array')
);

CREATE UNIQUE INDEX ux_source_retrieval_runs_project_input_hash
    ON source_retrieval_runs (project_id, input_hash);
CREATE INDEX idx_source_retrieval_runs_project_created
    ON source_retrieval_runs (project_id, created_at DESC);

-- +goose Down
-- Why: 已记录的检索理由是已生成稿件的可追溯证据；有数据时只能从备份恢复，不能静默删除。
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM source_retrieval_runs) THEN
        RAISE EXCEPTION 'refusing to roll back source retrieval migration after data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP TABLE source_retrieval_runs;

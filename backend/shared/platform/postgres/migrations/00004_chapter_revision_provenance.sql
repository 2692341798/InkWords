-- +goose Up
-- InkWords migration role: core
-- Why: 候选教材必须保留其合同、证据、提示和模型用量谱系；否则人工批准后无法复核其生成条件。
ALTER TABLE chapter_revisions
    ADD COLUMN parent_revision_id UUID REFERENCES chapter_revisions (id) ON DELETE RESTRICT,
    ADD COLUMN book_contract_revision_id UUID REFERENCES book_contract_revisions (id) ON DELETE RESTRICT,
    ADD COLUMN style_sheet_revision_id UUID REFERENCES style_sheet_revisions (id) ON DELETE RESTRICT,
    ADD COLUMN evidence_pack_hash TEXT,
    ADD COLUMN prompt_hash TEXT,
    ADD COLUMN provider_name TEXT,
    ADD COLUMN model_name TEXT,
    ADD COLUMN provider_usage_json JSONB,
    ADD COLUMN quality_report_json JSONB;

ALTER TABLE chapter_revisions
    ADD CONSTRAINT ck_generated_candidate_provenance
    CHECK (
        kind <> 'candidate' OR created_by <> 'generation' OR (
            parent_revision_id IS NOT NULL
            AND book_contract_revision_id IS NOT NULL
            AND style_sheet_revision_id IS NOT NULL
            AND evidence_pack_hash LIKE 'sha256:%'
            AND prompt_hash LIKE 'sha256:%'
            AND length(trim(provider_name)) > 0
            AND length(trim(model_name)) > 0
            AND provider_usage_json IS NOT NULL
            AND quality_report_json IS NOT NULL
        )
    );

CREATE INDEX idx_chapter_revisions_contract_provenance
    ON chapter_revisions (book_contract_revision_id, style_sheet_revision_id)
    WHERE book_contract_revision_id IS NOT NULL;

-- +goose Down
-- Why: 已生成的候选稿依赖这些字段进行审计；有该类数据时只能从备份恢复。
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM chapter_revisions
        WHERE parent_revision_id IS NOT NULL
           OR book_contract_revision_id IS NOT NULL
           OR style_sheet_revision_id IS NOT NULL
           OR evidence_pack_hash IS NOT NULL
           OR prompt_hash IS NOT NULL
           OR provider_usage_json IS NOT NULL
           OR quality_report_json IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'refusing to roll back chapter revision provenance migration after data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP INDEX idx_chapter_revisions_contract_provenance;
ALTER TABLE chapter_revisions DROP CONSTRAINT ck_generated_candidate_provenance;
ALTER TABLE chapter_revisions
    DROP COLUMN quality_report_json,
    DROP COLUMN provider_usage_json,
    DROP COLUMN model_name,
    DROP COLUMN provider_name,
    DROP COLUMN prompt_hash,
    DROP COLUMN evidence_pack_hash,
    DROP COLUMN style_sheet_revision_id,
    DROP COLUMN book_contract_revision_id,
    DROP COLUMN parent_revision_id;

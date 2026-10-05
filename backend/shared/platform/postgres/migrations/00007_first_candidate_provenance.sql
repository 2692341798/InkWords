-- +goose Up
-- InkWords migration role: core
-- Why: 首次样章没有上一份正文修订；它仍必须绑定当前契约、蓝图、证据包和生成记录，且只能作为章节的第一份修订。
ALTER TABLE chapter_revisions DROP CONSTRAINT ck_generated_candidate_provenance;

ALTER TABLE chapter_revisions
    ADD CONSTRAINT ck_generated_candidate_provenance
    CHECK (
        kind <> 'candidate' OR created_by <> 'generation' OR (
            (parent_revision_id IS NOT NULL OR revision_number = 1)
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

-- +goose Down
-- Why: 旧约束不能表示无父修订的首个候选；存在该数据时应从备份恢复，而不是假造父修订。
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM chapter_revisions
        WHERE kind = 'candidate'
          AND created_by = 'generation'
          AND parent_revision_id IS NULL
    ) THEN
        RAISE EXCEPTION 'refusing to roll back first candidate provenance migration after initial generated candidates exist; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE chapter_revisions DROP CONSTRAINT ck_generated_candidate_provenance;
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

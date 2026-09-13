-- +goose Up
-- InkWords migration role: core
-- Why: 生成必须只依赖项目当前人工批准的 BookContract 与 StyleSheet，而不是从最新草稿中猜测规则。
ALTER TABLE textbook_projects
    ADD COLUMN approved_book_contract_revision_id UUID REFERENCES book_contract_revisions (id) ON DELETE RESTRICT,
    ADD COLUMN approved_style_sheet_revision_id UUID REFERENCES style_sheet_revisions (id) ON DELETE RESTRICT;

CREATE INDEX idx_textbook_projects_generation_contracts
    ON textbook_projects (approved_book_contract_revision_id, approved_style_sheet_revision_id)
    WHERE approved_book_contract_revision_id IS NOT NULL;

-- +goose Down
-- Why: 已批准合同定义了可复现候选稿的生成边界；存在引用时必须从备份恢复。
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM textbook_projects
        WHERE approved_book_contract_revision_id IS NOT NULL
           OR approved_style_sheet_revision_id IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'refusing to roll back generation contract approval migration after data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP INDEX idx_textbook_projects_generation_contracts;
ALTER TABLE textbook_projects
    DROP COLUMN approved_style_sheet_revision_id,
    DROP COLUMN approved_book_contract_revision_id;

-- +goose Up
-- +goose StatementBegin
-- Why: screenshots are exceptional evidence. Persist the observation purpose
-- so exports and reviews can distinguish a necessary visual claim from text
-- that should have stayed structured.
ALTER TABLE textbook_manuscript_assets
    ADD COLUMN visual_purpose VARCHAR(32) NOT NULL DEFAULT 'legacy_unclassified'
    CHECK (visual_purpose IN ('layout', 'memory_map', 'call_stack', 'network_flow', 'rendered_ui', 'legacy_unclassified'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM textbook_manuscript_assets WHERE visual_purpose <> 'legacy_unclassified') THEN
        RAISE EXCEPTION 'refusing to drop visual evidence purposes after classified captures exist; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd
ALTER TABLE textbook_manuscript_assets DROP COLUMN visual_purpose;

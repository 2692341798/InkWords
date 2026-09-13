-- +goose Up
-- InkWords migration role: core
-- Why: a generated chapter is auditable only when it names the approved blueprint that selected its chapter order and evidence mapping.
ALTER TABLE chapter_revisions
    ADD COLUMN blueprint_revision_id UUID REFERENCES blueprint_revisions (id) ON DELETE RESTRICT;

ALTER TABLE chapter_revisions
    ADD CONSTRAINT ck_generated_candidate_blueprint_provenance
    CHECK (
        kind <> 'candidate' OR created_by <> 'generation' OR blueprint_revision_id IS NOT NULL
    );

CREATE INDEX idx_chapter_revisions_blueprint_provenance
    ON chapter_revisions (blueprint_revision_id)
    WHERE blueprint_revision_id IS NOT NULL;

-- +goose Down
-- Why: removing the blueprint link from a generated candidate would make an approved sample unauditable.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM chapter_revisions WHERE blueprint_revision_id IS NOT NULL) THEN
        RAISE EXCEPTION 'refusing to roll back chapter blueprint provenance migration after data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP INDEX idx_chapter_revisions_blueprint_provenance;
ALTER TABLE chapter_revisions DROP CONSTRAINT ck_generated_candidate_blueprint_provenance;
ALTER TABLE chapter_revisions DROP COLUMN blueprint_revision_id;

-- +goose Up
-- InkWords migration role: core
-- Preserve legacy text/timestamps. No verdict or score is inferred for old rows.
ALTER TABLE textbook_publication_reviews
    ADD COLUMN contract_version TEXT,
    ADD COLUMN manifest_hash TEXT,
    ADD COLUMN revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    ADD COLUMN reviewer_kind TEXT,
    ADD COLUMN verdict TEXT,
    ADD COLUMN score INTEGER,
    ADD COLUMN scope TEXT,
    ADD COLUMN evidence_refs JSONB,
    ADD COLUMN hard_failures JSONB,
    ADD COLUMN input_hash TEXT,
    ADD CONSTRAINT human_review_decision_v2 CHECK (
        (contract_version IS NULL AND manifest_hash IS NULL AND reviewer_kind IS NULL
         AND verdict IS NULL AND score IS NULL AND scope IS NULL
         AND evidence_refs IS NULL AND hard_failures IS NULL AND input_hash IS NULL AND revision = 1)
        OR COALESCE((contract_version = 'inkwords.human-publication-review.v2'
         AND reviewer_kind = 'human' AND manifest_hash ~ '^sha256:[0-9a-f]{64}$'
         AND input_hash ~ '^sha256:[0-9a-f]{64}$'
         AND char_length(trim(scope)) BETWEEN 8 AND 1000 AND score BETWEEN 0 AND 4
         AND jsonb_typeof(evidence_refs) = 'array' AND jsonb_typeof(hard_failures) = 'array'
         AND jsonb_array_length(evidence_refs) <= 32 AND jsonb_array_length(hard_failures) <= 32
         AND ((verdict = 'pass' AND score >= 3 AND jsonb_array_length(evidence_refs) > 0 AND jsonb_array_length(hard_failures) = 0)
           OR (verdict = 'needs_revision' AND jsonb_array_length(hard_failures) > 0)
           OR (verdict = 'not_assessed' AND score = 0 AND jsonb_array_length(hard_failures) = 0))), FALSE)
    ),
    ADD CONSTRAINT human_review_build_stage_revision UNIQUE (build_id, stage, revision);
ALTER TABLE textbook_publication_reviews DROP CONSTRAINT textbook_publication_reviews_build_id_stage_key;
-- The replacement B-tree supports latest-revision and stage-history reads.
-- One index write per append; old completion-order index remains for exports.

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM textbook_publication_reviews WHERE contract_version IS NOT NULL OR revision <> 1) THEN
        RAISE EXCEPTION 'refusing to discard explicit human review evidence; retain schema or restore a verified backup';
    END IF;
END
$$;
-- +goose StatementEnd
ALTER TABLE textbook_publication_reviews
    ADD CONSTRAINT textbook_publication_reviews_build_id_stage_key UNIQUE (build_id, stage),
    DROP CONSTRAINT human_review_build_stage_revision,
    DROP CONSTRAINT human_review_decision_v2,
    DROP COLUMN contract_version, DROP COLUMN manifest_hash, DROP COLUMN revision,
    DROP COLUMN reviewer_kind, DROP COLUMN verdict, DROP COLUMN score, DROP COLUMN scope,
    DROP COLUMN evidence_refs, DROP COLUMN hard_failures, DROP COLUMN input_hash;

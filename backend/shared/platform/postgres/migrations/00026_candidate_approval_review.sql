-- +goose Up
-- InkWords migration role: core
-- Why: 样章批准必须保存人工量表，不能把“应用候选稿”或自动门禁冒充人工审阅。
ALTER TABLE textbook_candidate_reviews
    DROP CONSTRAINT IF EXISTS textbook_candidate_reviews_decision_check;

ALTER TABLE textbook_candidate_reviews
    ADD CONSTRAINT textbook_candidate_reviews_decision_check
        CHECK (decision IN ('approved', 'rejected')),
    ADD COLUMN human_review_json JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
-- Why: 已有人工批准证据时不允许降级丢失；必须使用迁移前备份恢复。
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM textbook_candidate_reviews
        WHERE decision = 'approved' OR human_review_json <> '{}'::jsonb
    ) THEN
        RAISE EXCEPTION 'refusing to roll back candidate approval reviews after evidence exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE textbook_candidate_reviews
    DROP COLUMN human_review_json,
    DROP CONSTRAINT textbook_candidate_reviews_decision_check;

ALTER TABLE textbook_candidate_reviews
    ADD CONSTRAINT textbook_candidate_reviews_decision_check
        CHECK (decision = 'rejected');

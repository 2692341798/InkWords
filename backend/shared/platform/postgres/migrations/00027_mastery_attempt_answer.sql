-- +goose Up
-- InkWords migration role: review
-- Keep learner text beside its append-only self-assessment. Historical attempts
-- remain empty: a boolean outcome cannot reconstruct an actual answer.
ALTER TABLE mastery_attempts
    ADD COLUMN answer TEXT NOT NULL DEFAULT '' CHECK (char_length(answer) <= 20000);

-- +goose Down
-- Refuse to discard submitted learner evidence during a rollback.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM mastery_attempts WHERE answer <> '') THEN
        RAISE EXCEPTION 'refusing to roll back mastery answers after evidence exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd
ALTER TABLE mastery_attempts DROP COLUMN answer;

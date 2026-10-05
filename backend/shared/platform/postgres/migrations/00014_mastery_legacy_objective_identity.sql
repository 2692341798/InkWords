-- +goose Up
-- InkWords migration role: review
-- The adapter is idempotent per local workspace and immutable legacy note path.
CREATE UNIQUE INDEX uq_mastery_legacy_note_objective
ON mastery_objectives (workspace_id, chapter_id)
WHERE deleted_at IS NULL AND chapter_id LIKE 'legacy-note:%';

-- +goose Down
DROP INDEX uq_mastery_legacy_note_objective;

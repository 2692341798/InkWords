-- +goose Up
-- InkWords migration role: review
ALTER TABLE mastery_attempts ADD COLUMN learner_artifact_hash TEXT NOT NULL DEFAULT ''
 CHECK(learner_artifact_hash='' OR learner_artifact_hash ~ '^sha256:[0-9a-f]{64}$');
CREATE TABLE mastery_learner_artifacts (
 attempt_id UUID PRIMARY KEY REFERENCES mastery_attempts(id) ON DELETE RESTRICT,
 objective_id UUID NOT NULL REFERENCES mastery_objectives(id) ON DELETE RESTRICT,
 workspace_id UUID NOT NULL,
 snapshot_hash TEXT NOT NULL CHECK(snapshot_hash ~ '^sha256:[0-9a-f]{64}$'),
 snapshot_json JSONB NOT NULL,
 CHECK(COALESCE(jsonb_typeof(snapshot_json)='object'
  AND snapshot_json->>'format'='inkwords.learner-artifact.v1'
  AND snapshot_json->>'attempt_id'=attempt_id::text
  AND snapshot_json->>'objective_id'=objective_id::text
  AND snapshot_json->>'workspace_id'=workspace_id::text
  AND snapshot_json->>'snapshot_hash'=snapshot_hash
  AND jsonb_typeof(snapshot_json->'files')='array',FALSE))
);
-- Reads use the saved attempt primary key; no source-text index is needed.

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM mastery_learner_artifacts) OR EXISTS(SELECT 1 FROM mastery_attempts WHERE learner_artifact_hash<>'') THEN
  RAISE EXCEPTION 'refusing to remove learner code snapshots; retain schema or restore backup';
 END IF;
END $$;
-- +goose StatementEnd
DROP TABLE mastery_learner_artifacts;
ALTER TABLE mastery_attempts DROP COLUMN learner_artifact_hash;

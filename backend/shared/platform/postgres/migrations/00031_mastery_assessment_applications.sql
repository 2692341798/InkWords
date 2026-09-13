-- +goose Up
-- InkWords migration role: review
CREATE TABLE mastery_assessment_applications (
 id UUID PRIMARY KEY,
 objective_id UUID NOT NULL REFERENCES mastery_objectives(id) ON DELETE RESTRICT,
 attempt_id UUID NOT NULL REFERENCES mastery_attempts(id) ON DELETE RESTRICT,
 job_id UUID NOT NULL REFERENCES mastery_assessment_jobs(id) ON DELETE RESTRICT,
 sequence_no INTEGER NOT NULL CHECK(sequence_no>0),
 application_json JSONB NOT NULL,
 UNIQUE(objective_id,sequence_no),
 CHECK(COALESCE(jsonb_typeof(application_json)='object'
  AND application_json->>'id'=id::text
  AND application_json->>'objective_id'=objective_id::text
  AND application_json->>'attempt_id'=attempt_id::text
  AND application_json->>'job_id'=job_id::text
  AND application_json->>'sequence_no'=sequence_no::text
  AND application_json->>'algorithm_version'='fsrs-v4-assessment-v1'
  AND application_json->>'feedback_hash' ~ '^sha256:[0-9a-f]{64}$'
  AND application_json->>'input_hash' ~ '^sha256:[0-9a-f]{64}$',FALSE))
);
CREATE INDEX idx_mastery_assessment_application_attempt ON mastery_assessment_applications(objective_id,attempt_id,sequence_no DESC);

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM mastery_assessment_applications) THEN
  RAISE EXCEPTION 'refusing to remove applied assessment evidence; retain schema or restore backup';
 END IF;
END $$;
-- +goose StatementEnd
DROP TABLE mastery_assessment_applications;

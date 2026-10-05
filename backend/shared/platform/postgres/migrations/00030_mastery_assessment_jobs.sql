-- +goose Up
-- InkWords migration role: review
CREATE TABLE mastery_assessment_jobs (
 id UUID PRIMARY KEY,
 workspace_id UUID NOT NULL,
 objective_id UUID NOT NULL REFERENCES mastery_objectives(id) ON DELETE RESTRICT,
 attempt_id UUID NOT NULL REFERENCES mastery_attempts(id) ON DELETE RESTRICT,
 request_id UUID NOT NULL,
 retry_of UUID REFERENCES mastery_assessment_jobs(id) ON DELETE RESTRICT,
 status VARCHAR(16) NOT NULL CHECK (status IN ('running','succeeded','failed','cancelled','interrupted')),
 input_hash TEXT NOT NULL CHECK (input_hash ~ '^sha256:[0-9a-f]{64}$'),
 request_hash TEXT NOT NULL CHECK (request_hash ~ '^sha256:[0-9a-f]{64}$'),
 input_json JSONB NOT NULL,
 preview_json JSONB NOT NULL,
 result_json JSONB,
 error_code VARCHAR(64) NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL,
 completed_at TIMESTAMPTZ,
 CHECK (COALESCE(jsonb_typeof(input_json)='object' AND preview_json->>'input_hash'=input_hash AND preview_json->>'request_hash'=request_hash,FALSE)),
 CHECK (COALESCE((status='running' AND completed_at IS NULL AND result_json IS NULL) OR (status<>'running' AND completed_at>=created_at),FALSE)),
 CHECK (status<>'succeeded' OR COALESCE(jsonb_typeof(result_json->'feedback')='object' AND result_json->>'origin'='automated',FALSE))
);
CREATE UNIQUE INDEX ux_mastery_assessment_request ON mastery_assessment_jobs(workspace_id,request_id);
-- Local V1 permits one in-flight grading call across the installation.
CREATE UNIQUE INDEX ux_mastery_assessment_running ON mastery_assessment_jobs((TRUE)) WHERE status='running';
CREATE INDEX idx_mastery_assessment_attempt ON mastery_assessment_jobs(workspace_id,objective_id,attempt_id,created_at DESC,id DESC);
CREATE TABLE mastery_assessment_corrections (
 id UUID PRIMARY KEY,
 job_id UUID NOT NULL REFERENCES mastery_assessment_jobs(id) ON DELETE RESTRICT,
 sequence_no INTEGER NOT NULL CHECK (sequence_no>0),
 correction_json JSONB NOT NULL CHECK (jsonb_typeof(correction_json)='object'),
 UNIQUE(job_id,sequence_no)
);

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM mastery_assessment_jobs) THEN
   RAISE EXCEPTION 'refusing to remove assessment jobs or corrections; retain schema or restore backup';
 END IF;
END $$;
-- +goose StatementEnd
DROP TABLE mastery_assessment_corrections;
DROP TABLE mastery_assessment_jobs;

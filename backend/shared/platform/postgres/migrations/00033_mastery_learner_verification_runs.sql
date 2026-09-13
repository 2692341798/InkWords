-- +goose Up
-- InkWords migration role: review
CREATE TABLE mastery_learner_verification_runs (
 id UUID PRIMARY KEY,
 workspace_id UUID NOT NULL,
 objective_id UUID NOT NULL REFERENCES mastery_objectives(id) ON DELETE RESTRICT,
 attempt_id UUID NOT NULL REFERENCES mastery_attempts(id) ON DELETE RESTRICT,
 request_id UUID NOT NULL,
 retry_of UUID REFERENCES mastery_learner_verification_runs(id) ON DELETE RESTRICT,
 status VARCHAR(24) NOT NULL CHECK(status IN ('queued','running','passed','failed','timed_out','cancelled','unavailable','runner_error','interrupted')),
 input_hash TEXT NOT NULL CHECK(input_hash ~ '^sha256:[0-9a-f]{64}$'),
 claim_token_hash TEXT NOT NULL CHECK(claim_token_hash ~ '^sha256:[0-9a-f]{64}$'),
 input_json JSONB NOT NULL,
 preview_json JSONB NOT NULL,
 report_json JSONB,
 error_code VARCHAR(64) NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL,
 started_at TIMESTAMPTZ,
 completed_at TIMESTAMPTZ,
 CHECK(COALESCE(jsonb_typeof(input_json)='object'
   AND input_json->>'format'='inkwords.learner-verification-input.v1'
   AND input_json->'plan'->>'input_hash'=input_hash
   AND preview_json->>'input_hash'=input_hash,FALSE)),
 CHECK(COALESCE((status='queued' AND started_at IS NULL AND completed_at IS NULL AND report_json IS NULL)
   OR (status='running' AND started_at IS NOT NULL AND completed_at IS NULL AND report_json IS NULL)
   OR (status IN ('passed','failed','timed_out','unavailable','runner_error') AND completed_at>=created_at AND jsonb_typeof(report_json)='object' AND report_json->>'status'=status)
   OR (status IN ('cancelled','interrupted') AND completed_at>=created_at),FALSE))
);
CREATE UNIQUE INDEX ux_mastery_learner_verification_request ON mastery_learner_verification_runs(workspace_id,request_id);
CREATE UNIQUE INDEX ux_mastery_learner_verification_active ON mastery_learner_verification_runs((TRUE)) WHERE status IN ('queued','running');
CREATE INDEX idx_mastery_learner_verification_attempt ON mastery_learner_verification_runs(workspace_id,objective_id,attempt_id,created_at DESC,id DESC);

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM mastery_learner_verification_runs) THEN
  RAISE EXCEPTION 'refusing to remove learner verification evidence; retain schema or restore backup';
 END IF;
END $$;
-- +goose StatementEnd
DROP TABLE mastery_learner_verification_runs;

-- +goose Up
-- InkWords migration role: core
-- Why: all local runtime ownership has moved to the installation workspace and
-- the verified backup is now the sole recovery path for retired account data.
-- This migration intentionally avoids CASCADE so any unretired dependency fails closed.
-- +goose StatementBegin
DO $$
DECLARE
    workspace_count BIGINT;
    legacy_rows_exist BOOLEAN;
BEGIN
    SELECT COUNT(*) INTO workspace_count FROM local_workspaces;
    IF workspace_count > 1 THEN
        RAISE EXCEPTION 'refusing legacy identity cleanup with more than one local workspace';
    END IF;
    IF EXISTS (SELECT 1 FROM job_tasks WHERE workspace_id IS NULL) THEN
        RAISE EXCEPTION 'refusing legacy identity cleanup while tasks lack workspace ownership';
    END IF;
    IF EXISTS (SELECT 1 FROM blogs WHERE workspace_id IS NULL) THEN
        RAISE EXCEPTION 'refusing legacy identity cleanup while blogs lack workspace ownership';
    END IF;
    IF (EXISTS (SELECT 1 FROM job_tasks) OR EXISTS (SELECT 1 FROM blogs))
       AND workspace_count <> 1 THEN
        RAISE EXCEPTION 'refusing legacy identity cleanup without exactly one workspace for existing data';
    END IF;
    IF to_regclass('public.o_auth_tokens') IS NOT NULL THEN
        EXECUTE 'SELECT EXISTS (SELECT 1 FROM o_auth_tokens)' INTO legacy_rows_exist;
        IF legacy_rows_exist THEN
            RAISE EXCEPTION 'refusing legacy identity cleanup while OAuth token rows exist';
        END IF;
    END IF;
    IF to_regclass('public.project_courses') IS NOT NULL THEN
        EXECUTE 'SELECT EXISTS (SELECT 1 FROM project_courses)' INTO legacy_rows_exist;
        IF legacy_rows_exist THEN
            RAISE EXCEPTION 'refusing legacy identity cleanup while ProjectCourse rows exist';
        END IF;
    END IF;
    IF to_regclass('public.user_prompt_settings') IS NOT NULL THEN
        EXECUTE 'SELECT EXISTS (SELECT 1 FROM user_prompt_settings)' INTO legacy_rows_exist;
        IF legacy_rows_exist THEN
            RAISE EXCEPTION 'refusing legacy identity cleanup while user prompt setting rows exist';
        END IF;
    END IF;
END
$$;
-- +goose StatementEnd

DROP INDEX IF EXISTS idx_user_parent_chapter;
DROP INDEX IF EXISTS idx_job_tasks_requested_by;

ALTER TABLE blogs DROP COLUMN IF EXISTS user_id;
ALTER TABLE job_tasks DROP COLUMN IF EXISTS requested_by;

DROP TABLE IF EXISTS o_auth_tokens;
DROP TABLE IF EXISTS project_courses;
DROP TABLE IF EXISTS user_prompt_settings;
DROP TABLE local_workspace_legacy_owner;
DROP TABLE users;

-- +goose Down
-- Why: account rows and historical owner fields were intentionally removed only
-- after a verified backup/restore drill. Reconstructing them would invent data.
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'legacy identity cleanup is irreversible; restore the verified local backup instead';
END
$$;
-- +goose StatementEnd

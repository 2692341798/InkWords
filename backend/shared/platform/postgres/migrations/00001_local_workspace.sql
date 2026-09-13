-- +goose Up
-- Why: 新教材数据需要稳定的本地工作区身份；旧 user_id 数据在过渡期不被重写。
CREATE TABLE local_workspaces (
    id UUID PRIMARY KEY,
    installation_key TEXT NOT NULL UNIQUE CHECK (installation_key = 'local-default'),
    display_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE local_workspace_legacy_owner (
    workspace_id UUID PRIMARY KEY REFERENCES local_workspaces (id) ON DELETE RESTRICT,
    legacy_user_id UUID NOT NULL UNIQUE REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
-- Why: 仅在工作区及映射尚未承载任何数据时允许回退；有数据时必须从备份恢复。
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM local_workspace_legacy_owner)
        OR EXISTS (SELECT 1 FROM local_workspaces) THEN
        RAISE EXCEPTION 'refusing to roll back local workspace migration after data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP TABLE local_workspace_legacy_owner;
DROP TABLE local_workspaces;

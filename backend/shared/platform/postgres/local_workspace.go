package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	localWorkspaceInstallationKey = "local-default"
	localWorkspaceDisplayName     = "本地工作区"
)

// LocalWorkspace is the stable identity of this local installation.
type LocalWorkspace struct {
	ID uuid.UUID
}

// NewLocalWorkspaceResolver returns a resolver suitable for httpx.LocalWorkspaceContext.
// It creates or reads only the installation workspace. Legacy user identity is
// deliberately not part of textbook request admission.
func NewLocalWorkspaceResolver(database *gorm.DB) func(context.Context) (uuid.UUID, error) {
	return func(ctx context.Context) (uuid.UUID, error) {
		return EnsureLocalWorkspaceIdentity(ctx, database)
	}
}

// EnsureLocalWorkspaceIdentity returns the stable installation workspace
// without resolving, creating, or mapping a compatibility user.
func EnsureLocalWorkspaceIdentity(ctx context.Context, database *gorm.DB) (uuid.UUID, error) {
	if database == nil {
		return uuid.Nil, errors.New("local workspace database is nil")
	}
	workspaceID, err := upsertLocalWorkspace(database.WithContext(ctx))
	if err != nil {
		return uuid.Nil, fmt.Errorf("ensure local workspace identity: %w", err)
	}
	return workspaceID, nil
}

// EnsureLocalWorkspace retains the structured helper used by repository tests
// while resolving only the installation workspace. It never creates a user or
// a legacy-owner mapping.
func EnsureLocalWorkspace(ctx context.Context, database *gorm.DB) (LocalWorkspace, error) {
	workspaceID, err := EnsureLocalWorkspaceIdentity(ctx, database)
	if err != nil {
		return LocalWorkspace{}, err
	}
	return LocalWorkspace{ID: workspaceID}, nil
}

func upsertLocalWorkspace(tx *gorm.DB) (uuid.UUID, error) {
	var workspaceID uuid.UUID
	row := tx.Raw(`
INSERT INTO local_workspaces (id, installation_key, display_name)
VALUES ($1, $2, $3)
ON CONFLICT (installation_key) DO UPDATE SET installation_key = EXCLUDED.installation_key
RETURNING id`, uuid.New(), localWorkspaceInstallationKey, localWorkspaceDisplayName).Row()
	if err := row.Scan(&workspaceID); err != nil {
		return uuid.Nil, fmt.Errorf("upsert local workspace: %w", err)
	}
	return workspaceID, nil
}

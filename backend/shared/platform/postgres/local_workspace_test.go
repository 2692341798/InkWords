package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnsureLocalWorkspaceCreatesOneStableWorkspaceWithoutLegacyOwner(t *testing.T) {
	database := newLocalWorkspaceTestDatabase(t)

	first, err := EnsureLocalWorkspace(t.Context(), database)
	require.NoError(t, err)
	second, err := EnsureLocalWorkspace(t.Context(), database)
	require.NoError(t, err)
	require.Equal(t, first, second)

	var workspaceCount, mappingCount, userCount int64
	require.NoError(t, database.Table("local_workspaces").Count(&workspaceCount).Error)
	require.NoError(t, database.Table("local_workspace_legacy_owner").Count(&mappingCount).Error)
	require.NoError(t, database.Table("users").Count(&userCount).Error)
	require.EqualValues(t, 1, workspaceCount)
	require.Zero(t, mappingCount)
	require.Zero(t, userCount)
}

func TestNewLocalWorkspaceResolverUsesDatabaseIdentity(t *testing.T) {
	database := newLocalWorkspaceTestDatabase(t)
	resolver := NewLocalWorkspaceResolver(database)

	workspaceID, err := resolver(context.Background())
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, workspaceID)

	var workspaceCount, mappingCount, userCount int64
	require.NoError(t, database.Table("local_workspaces").Count(&workspaceCount).Error)
	require.NoError(t, database.Table("local_workspace_legacy_owner").Count(&mappingCount).Error)
	require.NoError(t, database.Table("users").Count(&userCount).Error)
	require.EqualValues(t, 1, workspaceCount)
	require.Zero(t, mappingCount)
	require.Zero(t, userCount)
}

func newLocalWorkspaceTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:workspace-%s?mode=memory&cache=shared", uuid.NewString())), &gorm.Config{})
	require.NoError(t, err)

	for _, statement := range []string{
		`CREATE TABLE users (
            id TEXT PRIMARY KEY,
            username TEXT NOT NULL,
            email TEXT NOT NULL UNIQUE,
            password_hash TEXT NOT NULL,
            created_at DATETIME NOT NULL,
            updated_at DATETIME NOT NULL,
            deleted_at DATETIME
        )`,
		`CREATE TABLE local_workspaces (
            id TEXT PRIMARY KEY,
            installation_key TEXT NOT NULL UNIQUE,
            display_name TEXT NOT NULL,
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE local_workspace_legacy_owner (
            workspace_id TEXT PRIMARY KEY,
            legacy_user_id TEXT NOT NULL UNIQUE,
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        )`,
	} {
		require.NoError(t, database.Exec(statement).Error)
	}
	return database
}

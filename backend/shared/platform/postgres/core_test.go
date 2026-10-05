package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"

	"inkwords-backend/shared/platform/postgres/migrations"
)

func TestInitCoreRunsEmbeddedMigrationsAfterLegacyAutoMigrate(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(
		ctx,
		"postgres:14-alpine",
		postgrescontainer.WithDatabase("inkwords_core_init_test"),
		postgrescontainer.WithUsername("inkwords"),
		postgrescontainer.WithPassword("inkwords-test-password"),
		testcontainers.WithAdditionalWaitStrategy(
			wait.ForSQL("5432/tcp", "pgx", coreInitPostgresURL).WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	database, err := InitCore(dsn)
	require.NoError(t, err)

	require.False(t, relationExists(t, ctx, database, "users"))
	require.False(t, relationExists(t, ctx, database, "local_workspace_legacy_owner"))
	require.False(t, relationExists(t, ctx, database, "o_auth_tokens"))
	require.False(t, relationExists(t, ctx, database, "project_courses"))
	require.False(t, relationExists(t, ctx, database, "user_prompt_settings"))
	require.True(t, relationExists(t, ctx, database, "local_workspaces"))
	require.True(t, relationExists(t, ctx, database, migrations.CoreVersionTable))
	require.False(t, columnExists(t, ctx, database, "blogs", "user_id"))
	require.False(t, columnExists(t, ctx, database, "job_tasks", "requested_by"))

	workspace, err := EnsureLocalWorkspace(ctx, database)
	require.NoError(t, err)
	secondWorkspace, err := EnsureLocalWorkspace(ctx, database)
	require.NoError(t, err)
	require.Equal(t, workspace, secondWorkspace, "the local identity must be stable across requests")

	var workspaceCount int64
	require.NoError(t, database.WithContext(ctx).Table("local_workspaces").Count(&workspaceCount).Error)
	require.EqualValues(t, 1, workspaceCount)

	database, err = InitCore(dsn)
	require.NoError(t, err, "restarting a service must not reapply the migration")
	require.False(t, relationExists(t, ctx, database, "users"), "AutoMigrate must not recreate retired identity tables")
	require.False(t, columnExists(t, ctx, database, "blogs", "user_id"))
	require.False(t, columnExists(t, ctx, database, "job_tasks", "requested_by"))
}

func TestInitReviewRemovesLegacyOwnerAndDoesNotRestoreItOnRestart(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(
		ctx,
		"postgres:14-alpine",
		postgrescontainer.WithDatabase("inkwords_review_init_test"),
		postgrescontainer.WithUsername("inkwords"),
		postgrescontainer.WithPassword("inkwords-test-password"),
		testcontainers.WithAdditionalWaitStrategy(
			wait.ForSQL("5432/tcp", "pgx", reviewInitPostgresURL).WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	database, err := InitReview(dsn)
	require.NoError(t, err)
	require.True(t, relationExists(t, ctx, database, migrations.ReviewVersionTable))
	require.False(t, columnExists(t, ctx, database, "review_sessions", "user_id"))
	require.True(t, columnNotNull(t, ctx, database, "review_sessions", "workspace_id"))

	database, err = InitReview(dsn)
	require.NoError(t, err)
	require.False(t, columnExists(t, ctx, database, "review_sessions", "user_id"))
}

func relationExists(t *testing.T, ctx context.Context, database *gorm.DB, name string) bool {
	t.Helper()
	var relation *string
	result := database.WithContext(ctx).Raw("SELECT to_regclass('public.' || ?)", name).Scan(&relation)
	require.NoError(t, result.Error)
	return relation != nil
}

func columnExists(t *testing.T, ctx context.Context, database *gorm.DB, tableName, columnName string) bool {
	t.Helper()
	var exists bool
	result := database.WithContext(ctx).Raw(`
SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = ? AND column_name = ?
)`, tableName, columnName).Scan(&exists)
	require.NoError(t, result.Error)
	return exists
}

func columnNotNull(t *testing.T, ctx context.Context, database *gorm.DB, tableName, columnName string) bool {
	t.Helper()
	var nullable string
	result := database.WithContext(ctx).Raw(`
SELECT is_nullable FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = ? AND column_name = ?
`, tableName, columnName).Scan(&nullable)
	require.NoError(t, result.Error)
	return nullable == "NO"
}

func coreInitPostgresURL(host string, port network.Port) string {
	return fmt.Sprintf(
		"postgres://inkwords:inkwords-test-password@%s:%s/inkwords_core_init_test?sslmode=disable",
		host,
		port.Port(),
	)
}

func reviewInitPostgresURL(host string, port network.Port) string {
	return fmt.Sprintf(
		"postgres://inkwords:inkwords-test-password@%s:%s/inkwords_review_init_test?sslmode=disable",
		host,
		port.Port(),
	)
}

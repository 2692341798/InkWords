package db

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

func TestAutoMigrate_RegistersReviewTables(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = autoMigrateReview(testDB)
	require.NoError(t, err)
	require.True(t, testDB.Migrator().HasTable(&model.ReviewSession{}))
	require.True(t, testDB.Migrator().HasTable(&model.ReviewTurn{}))
	require.True(t, testDB.Migrator().HasColumn(&model.ReviewSession{}, "phase"))
	require.True(t, testDB.Migrator().HasColumn(&model.ReviewSession{}, "reading_completed_at"))
	require.True(t, testDB.Migrator().HasColumn("review_sessions", "user_id"))
}

func TestAutoMigrateCore_UsesMinimalLegacyOwnerProjection(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = autoMigrateCore(testDB)
	require.NoError(t, err)
	require.True(t, testDB.Migrator().HasTable("users"))
	require.True(t, testDB.Migrator().HasColumn("blogs", "user_id"))
	require.True(t, testDB.Migrator().HasColumn("job_tasks", "requested_by"))
	require.False(t, testDB.Migrator().HasTable("user_prompt_settings"))
	require.False(t, testDB.Migrator().HasTable("o_auth_tokens"))
	for _, retiredColumn := range []string{
		"github_id",
		"wechat_open_id",
		"avatar_url",
		"subscription_tier",
		"tokens_used",
		"token_limit",
		"failed_login_attempts",
		"locked_until",
	} {
		require.False(t, testDB.Migrator().HasColumn("users", retiredColumn), retiredColumn)
	}
}

func TestAutoMigrateCore_DoesNotRestoreLegacySchemaAfterCleanup(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, recordAppliedMigration(testDB, coreVersionTable, legacyCoreCleanupVersion))

	require.NoError(t, autoMigrateCore(testDB))
	require.False(t, testDB.Migrator().HasTable("users"))
	require.False(t, testDB.Migrator().HasColumn("blogs", "user_id"))
	require.False(t, testDB.Migrator().HasColumn("job_tasks", "requested_by"))
}

func TestAutoMigrateReview_DoesNotRestoreLegacyOwnerAfterCleanup(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, recordAppliedMigration(testDB, reviewVersionTable, legacyReviewCleanupVersion))

	require.NoError(t, autoMigrateReview(testDB))
	require.False(t, testDB.Migrator().HasColumn("review_sessions", "user_id"))
}

func TestAutoMigrateCore_PreservesHistoricalAccountColumns(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.Exec(`
CREATE TABLE users (
    id uuid PRIMARY KEY,
    username varchar(255) NOT NULL,
    email varchar(255) NOT NULL,
    password_hash varchar(255),
    tokens_used integer DEFAULT 0,
    created_at datetime,
    updated_at datetime,
    deleted_at datetime
)`).Error)
	require.NoError(t, testDB.Exec(`CREATE TABLE user_prompt_settings (user_id uuid PRIMARY KEY, overrides text)`).Error)
	require.NoError(t, testDB.Exec(`CREATE TABLE o_auth_tokens (id uuid PRIMARY KEY, user_id uuid NOT NULL)`).Error)

	err = autoMigrateCore(testDB)
	require.NoError(t, err)
	require.True(t, testDB.Migrator().HasColumn("users", "tokens_used"))
	require.True(t, testDB.Migrator().HasTable("user_prompt_settings"))
	require.True(t, testDB.Migrator().HasTable("o_auth_tokens"))
}

func recordAppliedMigration(database *gorm.DB, tableName string, version int64) error {
	if err := database.Exec("CREATE TABLE " + tableName + " (version_id integer NOT NULL, is_applied boolean NOT NULL)").Error; err != nil {
		return err
	}
	return database.Exec("INSERT INTO "+tableName+" (version_id, is_applied) VALUES (?, ?)", version, true).Error
}

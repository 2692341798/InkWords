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

	err = autoMigrate(testDB)
	require.NoError(t, err)
	require.True(t, testDB.Migrator().HasTable(&model.ReviewSession{}))
	require.True(t, testDB.Migrator().HasTable(&model.ReviewTurn{}))
}

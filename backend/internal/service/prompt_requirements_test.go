package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
	"inkwords-backend/internal/prompt"
)

func TestPromptRequirementsService_Resolve_DefaultFallback(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserPromptSettings{}))

	svc := NewPromptRequirementsService(db)
	uid := uuid.New()

	got, err := svc.Resolve(context.Background(), uid, prompt.ArticleStyleGeneral)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestPromptRequirementsService_Resolve_OverrideWins(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserPromptSettings{}))

	uid := uuid.New()
	require.NoError(t, db.Create(&model.UserPromptSettings{
		UserID:    uid,
		Overrides: datatypes.JSON([]byte(`{"general":"CUSTOM"}`)),
	}).Error)

	svc := NewPromptRequirementsService(db)
	got, err := svc.Resolve(context.Background(), uid, prompt.ArticleStyleGeneral)
	require.NoError(t, err)
	require.Equal(t, "CUSTOM", got)
}

func TestPromptRequirementsService_Resolve_EmptyOverrideMeansReset(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserPromptSettings{}))

	uid := uuid.New()
	require.NoError(t, db.Create(&model.UserPromptSettings{
		UserID:    uid,
		Overrides: datatypes.JSON([]byte(`{"general":""}`)),
	}).Error)

	svc := NewPromptRequirementsService(db)
	got, err := svc.Resolve(context.Background(), uid, prompt.ArticleStyleGeneral)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	require.NotEqual(t, "", got)
}


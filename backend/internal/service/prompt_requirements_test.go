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

func TestPromptRequirementsService_Resolve_UsesScenarioAndStyleDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserPromptSettings{}))

	svc := NewPromptRequirementsService(db)
	uid := uuid.New()

	got, err := svc.Resolve(
		context.Background(),
		uid,
		prompt.ScenarioModeBeginnerWalkthrough,
		prompt.ArticleStyleBeginnerTutorial,
	)
	require.NoError(t, err)
	require.Contains(t, got, "零基础或初学者")
	require.Contains(t, got, "每一步都给出明确操作步骤")
}

func TestPromptRequirementsService_Resolve_FallsBackForInvalidScenario(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserPromptSettings{}))

	svc := NewPromptRequirementsService(db)
	uid := uuid.New()

	got, err := svc.Resolve(
		context.Background(),
		uid,
		prompt.ScenarioMode("bad"),
		prompt.ArticleStyleGeneral,
	)
	require.NoError(t, err)
	require.Contains(t, got, "高质量技术博客")
	require.NotContains(t, got, "bad")
}

func TestPromptRequirementsService_Resolve_StillHonorsUserOverride(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserPromptSettings{}))

	uid := uuid.New()
	require.NoError(t, db.Create(&model.UserPromptSettings{
		UserID:    uid,
		Overrides: datatypes.JSON([]byte(`{"beginner_tutorial":"CUSTOM STYLE"}`)),
	}).Error)

	svc := NewPromptRequirementsService(db)
	got, err := svc.Resolve(
		context.Background(),
		uid,
		prompt.ScenarioModeBeginnerWalkthrough,
		prompt.ArticleStyleBeginnerTutorial,
	)
	require.NoError(t, err)
	require.Contains(t, got, "零基础或初学者")
	require.Contains(t, got, "CUSTOM STYLE")
}

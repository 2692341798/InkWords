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
	require.Contains(t, got, "电子书或长文本解读场景")
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

func TestPromptRequirementsService_Resolve_AvoidsTechnicalBlogDefaultForEbookInterpretation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserPromptSettings{}))

	svc := NewPromptRequirementsService(db)
	uid := uuid.New()

	got, err := svc.Resolve(
		context.Background(),
		uid,
		prompt.ScenarioModeEbookInterpretation,
		prompt.ArticleStyleGeneral,
	)
	require.NoError(t, err)
	require.Contains(t, got, "电子书或长文本解读场景")
	require.NotContains(t, got, "高质量技术博客")
	require.NotContains(t, got, "可独立复现")
	// 经典文本导读约束：不引导模型做现代应用映射
	require.NotContains(t, got, "现实映射")
	require.NotContains(t, got, "现代应用")
	// 经典文本导读约束：引导模型聚焦原文自身结构与逐章解读
	require.Contains(t, got, "逐章")
	require.Contains(t, got, "篇章结构")
	require.Contains(t, got, "原文摘录")
	require.Contains(t, got, "历史背景")
}

func TestPromptRequirementsService_ResolveWithProfile_PrependsProfileRequirements(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserPromptSettings{}))

	svc := NewPromptRequirementsService(db)
	uid := uuid.New()
	profile := prompt.ResolvePromptProfileKey("classic_text_interpretation", prompt.ScenarioModeEbookInterpretation)

	got, err := svc.ResolveWithProfile(
		context.Background(),
		uid,
		prompt.ScenarioModeEbookInterpretation,
		prompt.ArticleStyleGeneral,
		profile,
	)
	require.NoError(t, err)
	require.Contains(t, got, profile.GenerateRequirements)
	require.Contains(t, got, "电子书或长文本解读场景")
	require.NotContains(t, got, "高质量技术博客")
	require.NotContains(t, got, "可独立复现")
}

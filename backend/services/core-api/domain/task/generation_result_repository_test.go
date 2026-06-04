package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

func TestGormGenerationResultRepository_PersistSingleGenerationResult(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	blogID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	require.NoError(t, testDB.Create(&model.User{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:         blogID,
		UserID:     userID,
		Title:      "旧标题",
		Content:    "旧内容",
		SourceType: "file",
		Status:     0,
	}).Error)

	repo := NewGormGenerationResultRepository(testDB)
	result := map[string]any{
		"result_version":   1,
		"task_type":        "generation",
		"task_subtype":     "generate_single",
		"persistence_mode": "task_only",
		"final_status":     "succeeded",
		"usage": map[string]any{
			"estimated_tokens": 24,
		},
		"payload": map[string]any{
			"blog_id":     blogID.String(),
			"title":       "文件解析生成的博客",
			"content":     "# 标题",
			"source_type": "file",
			"word_count":  float64(3),
			"tech_stacks": []any{"Go", "Docker"},
		},
	}

	taskID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	require.NoError(t, repo.PersistGenerationResult(context.Background(), taskID, result))
	require.NoError(t, repo.AccumulateTokens(context.Background(), taskID, result))

	var blog model.Blog
	require.NoError(t, testDB.First(&blog, "id = ?", blogID).Error)
	require.Equal(t, "文件解析生成的博客", blog.Title)
	require.Equal(t, "# 标题", blog.Content)
	require.Equal(t, 3, blog.WordCount)
	require.Equal(t, int16(1), blog.Status)
	require.JSONEq(t, `["Go","Docker"]`, string(blog.TechStacks))

	var user model.User
	require.NoError(t, testDB.First(&user, "id = ?", userID).Error)
	require.Equal(t, 24, user.TokensUsed)
}

func TestGormGenerationResultRepository_PersistContinueResult(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	blogID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	require.NoError(t, testDB.Create(&model.User{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:         blogID,
		UserID:     userID,
		Title:      "旧标题",
		Content:    "旧内容",
		SourceType: "file",
		Status:     1,
	}).Error)

	repo := NewGormGenerationResultRepository(testDB)
	result := map[string]any{
		"result_version":   1,
		"task_type":        "generation",
		"task_subtype":     "continue",
		"persistence_mode": "task_only",
		"final_status":     "succeeded",
		"usage": map[string]any{
			"estimated_tokens": 8,
		},
		"payload": map[string]any{
			"blog_id":          blogID.String(),
			"appended_content": "追加内容",
			"final_content":    "旧内容追加内容",
		},
	}

	taskID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	require.NoError(t, repo.PersistGenerationResult(context.Background(), taskID, result))

	var blog model.Blog
	require.NoError(t, testDB.First(&blog, "id = ?", blogID).Error)
	require.Equal(t, "旧内容追加内容", blog.Content)
}

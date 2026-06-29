package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGormGenerationResultRepository_PersistSingleGenerationResult(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&userRecord{}, &blogRecord{}))

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	blogID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	require.NoError(t, testDB.Create(&userRecord{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&blogRecord{
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
			"estimated_tokens":         24,
			"prompt_tokens":            10,
			"completion_tokens":        7,
			"prompt_cache_hit_tokens":  6,
			"prompt_cache_miss_tokens": 4,
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

	var blog blogRecord
	require.NoError(t, testDB.First(&blog, "id = ?", blogID).Error)
	require.Equal(t, "文件解析生成的博客", blog.Title)
	require.Equal(t, "# 标题", blog.Content)
	require.Equal(t, 3, blog.WordCount)
	require.Equal(t, int16(1), blog.Status)
	require.JSONEq(t, `["Go","Docker"]`, string(blog.TechStacks))

	var user userRecord
	require.NoError(t, testDB.First(&user, "id = ?", userID).Error)
	require.Equal(t, 17, user.TokensUsed)
}

func TestGormGenerationResultRepository_AccumulateTokensFallsBackToEstimatedTokens(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&userRecord{}, &blogRecord{}))

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	blogID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	require.NoError(t, testDB.Create(&userRecord{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&blogRecord{
		ID:         blogID,
		UserID:     userID,
		Title:      "标题",
		Content:    "内容",
		SourceType: "file",
		Status:     1,
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
			"blog_id": blogID.String(),
		},
	}

	taskID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	require.NoError(t, repo.AccumulateTokens(context.Background(), taskID, result))

	var user userRecord
	require.NoError(t, testDB.First(&user, "id = ?", userID).Error)
	require.Equal(t, 24, user.TokensUsed)
}

func TestGormGenerationResultRepository_AccumulateTokensFallsBackToTaskOwnerWithoutBlogID(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&userRecord{}, &JobTask{}))

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	taskID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	require.NoError(t, testDB.Create(&userRecord{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&JobTask{
		ID:          taskID,
		TaskType:    "generation",
		TaskSubtype: "generate_single",
		Status:      JobTaskStatusSucceeded,
		RequestedBy: userID,
	}).Error)

	repo := NewGormGenerationResultRepository(testDB)
	result := map[string]any{
		"result_version":   1,
		"task_type":        "generation",
		"task_subtype":     "generate_single",
		"persistence_mode": "task_only",
		"final_status":     "succeeded",
		"usage": map[string]any{
			"prompt_tokens":     11,
			"completion_tokens": 9,
		},
		"payload": map[string]any{},
	}

	require.NoError(t, repo.AccumulateTokens(context.Background(), taskID, result))

	var user userRecord
	require.NoError(t, testDB.First(&user, "id = ?", userID).Error)
	require.Equal(t, 20, user.TokensUsed)
}

func TestGormGenerationResultRepository_PersistContinueResult(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&userRecord{}, &blogRecord{}))

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	blogID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	require.NoError(t, testDB.Create(&userRecord{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&blogRecord{
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

	var blog blogRecord
	require.NoError(t, testDB.First(&blog, "id = ?", blogID).Error)
	require.Equal(t, "旧内容追加内容", blog.Content)
}

func TestGormGenerationResultRepository_PersistGenerateSeriesResult(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&userRecord{}, &blogRecord{}))

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	parentID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	childID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	require.NoError(t, testDB.Create(&userRecord{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&blogRecord{
		ID:         parentID,
		UserID:     userID,
		Title:      "旧系列标题",
		Content:    "旧导读",
		SourceType: "file",
		IsSeries:   true,
		Status:     0,
	}).Error)
	require.NoError(t, testDB.Create(&blogRecord{
		ID:          childID,
		UserID:      userID,
		ParentID:    &parentID,
		ChapterSort: 1,
		Title:       "旧章节标题",
		Content:     "旧章节内容",
		SourceType:  "file",
		Status:      0,
	}).Error)

	repo := NewGormGenerationResultRepository(testDB)
	result := map[string]any{
		"result_version":   1,
		"task_type":        "generation",
		"task_subtype":     "generate_series",
		"persistence_mode": "task_only",
		"final_status":     "succeeded",
		"usage": map[string]any{
			"estimated_tokens":         64,
			"prompt_tokens":            30,
			"completion_tokens":        12,
			"prompt_cache_hit_tokens":  25,
			"prompt_cache_miss_tokens": 5,
		},
		"payload": map[string]any{
			"parent_blog": map[string]any{
				"blog_id": parentID.String(),
				"title":   "Go 源码解析系列",
				"content": "导读正文",
			},
			"chapters": []any{
				map[string]any{
					"blog_id":       childID.String(),
					"chapter_sort":  float64(1),
					"title":         "第 1 章",
					"content":       "章节正文",
					"word_count":    float64(4),
					"tech_stacks":   []any{"Go"},
					"status":        "succeeded",
					"error_message": "",
				},
			},
		},
	}

	taskID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	require.NoError(t, repo.PersistGenerationResult(context.Background(), taskID, result))

	var parent blogRecord
	require.NoError(t, testDB.First(&parent, "id = ?", parentID).Error)
	require.Equal(t, "Go 源码解析系列", parent.Title)
	require.Equal(t, "导读正文", parent.Content)
	require.Equal(t, int16(1), parent.Status)

	var child blogRecord
	require.NoError(t, testDB.First(&child, "id = ?", childID).Error)
	require.Equal(t, "第 1 章", child.Title)
	require.Equal(t, "章节正文", child.Content)
	require.Equal(t, 4, child.WordCount)
	require.Equal(t, int16(1), child.Status)
	require.JSONEq(t, `["Go"]`, string(child.TechStacks))

	require.NoError(t, repo.AccumulateTokens(context.Background(), taskID, result))
	var user userRecord
	require.NoError(t, testDB.First(&user, "id = ?", userID).Error)
	require.Equal(t, 42, user.TokensUsed)
}

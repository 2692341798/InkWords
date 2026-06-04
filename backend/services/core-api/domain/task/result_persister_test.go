package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeBlogRepository struct {
	persisted bool
}

func (r *fakeBlogRepository) PersistGenerationResult(context.Context, uuid.UUID, map[string]any) error {
	r.persisted = true
	return nil
}

type fakeUsageRepository struct {
	accumulated bool
}

func (r *fakeUsageRepository) AccumulateTokens(context.Context, uuid.UUID, map[string]any) error {
	r.accumulated = true
	return nil
}

func TestResultPersister_PersistsGenerationResultToBlogRepository(t *testing.T) {
	repo := &fakeBlogRepository{}
	persister := NewResultPersister(repo, nil)

	err := persister.PersistGenerationResult(context.Background(), uuid.New(), map[string]any{"content": "# 内容"})
	require.NoError(t, err)
	require.True(t, repo.persisted)
}

func TestResultPersister_PersistsSingleGenerationResult(t *testing.T) {
	repo := &fakeBlogRepository{}
	usage := &fakeUsageRepository{}
	persister := NewResultPersister(repo, usage)

	taskID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
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
			"blog_id":     "33333333-3333-3333-3333-333333333333",
			"title":       "文件解析生成的博客",
			"content":     "# 标题",
			"source_type": "file",
			"word_count":  float64(3),
			"tech_stacks": []any{"Go", "Docker"},
		},
	}

	require.NoError(t, persister.PersistGenerationResult(context.Background(), taskID, result))
	require.True(t, repo.persisted)
	require.True(t, usage.accumulated)
}

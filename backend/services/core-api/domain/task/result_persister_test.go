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

type fakeTextbookSampleRepository struct {
	persisted bool
}

type fakeTextbookSourceImportRepository struct {
	persisted bool
}

func (r *fakeTextbookSampleRepository) PersistTextbookSampleResult(context.Context, uuid.UUID, map[string]any) error {
	r.persisted = true
	return nil
}

func (r *fakeTextbookSourceImportRepository) PersistTextbookSourceImportResult(context.Context, uuid.UUID, map[string]any) error {
	r.persisted = true
	return nil
}

func TestResultPersister_PersistsGenerationResultToBlogRepository(t *testing.T) {
	repo := &fakeBlogRepository{}
	persister := NewResultPersister(repo)

	err := persister.PersistGenerationResult(context.Background(), uuid.New(), map[string]any{"content": "# 内容"})
	require.NoError(t, err)
	require.True(t, repo.persisted)
}

func TestResultPersister_PersistsSingleGenerationResult(t *testing.T) {
	repo := &fakeBlogRepository{}
	persister := NewResultPersister(repo)

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
}

func TestResultPersisterRoutesTextbookSamplesAwayFromLegacyBlogWrites(t *testing.T) {
	blog := &fakeBlogRepository{}
	textbook := &fakeTextbookSampleRepository{}
	persister := NewResultPersister(blog).WithTextbookSampleRepository(textbook)

	require.NoError(t, persister.PersistGenerationResult(context.Background(), uuid.New(), map[string]any{"task_subtype": "textbook_sample_generate"}))
	require.True(t, textbook.persisted)
	require.False(t, blog.persisted)
}

func TestResultPersisterRoutesTypedSourceImportAwayFromLegacyBlogWrites(t *testing.T) {
	blog := &fakeBlogRepository{}
	imports := &fakeTextbookSourceImportRepository{}
	persister := NewResultPersister(blog).WithTextbookSourceImportRepository(imports)

	require.NoError(t, persister.PersistParseResult(context.Background(), uuid.New(), map[string]any{"task_subtype": "textbook_source_import"}))
	require.True(t, imports.persisted)
	require.False(t, blog.persisted)
}

func TestResultPersisterRoutesOfficialWebImportToTheCoreOwnedSourcePersister(t *testing.T) {
	imports := &fakeTextbookSourceImportRepository{}
	persister := NewResultPersister(nil).WithTextbookSourceImportRepository(imports)

	require.NoError(t, persister.PersistParseResult(context.Background(), uuid.New(), map[string]any{"task_subtype": "textbook_official_web_import"}))
	require.True(t, imports.persisted)
}

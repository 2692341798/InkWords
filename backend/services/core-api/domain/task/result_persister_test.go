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

func TestResultPersister_PersistsGenerationResultToBlogRepository(t *testing.T) {
	repo := &fakeBlogRepository{}
	persister := NewResultPersister(repo, nil)

	err := persister.PersistGenerationResult(context.Background(), uuid.New(), map[string]any{"content": "# 内容"})
	require.NoError(t, err)
	require.True(t, repo.persisted)
}

package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeProjectCourseResultRepository struct {
	called bool
	result map[string]any
}

func (r *fakeProjectCourseResultRepository) PersistProjectCourseResult(_ context.Context, result map[string]any) error {
	r.called = true
	r.result = result
	return nil
}

func TestResultPersisterRoutesProjectCourseAnalysisToCourseRepository(t *testing.T) {
	repository := &fakeProjectCourseResultRepository{}
	persister := NewResultPersister(nil, nil, repository)
	result := map[string]any{"task_subtype": ProjectCourseAnalyzeTaskSubtype, "course_id": uuid.NewString()}
	require.NoError(t, persister.PersistGenerationResult(context.Background(), uuid.New(), result))
	require.True(t, repository.called)
	require.Equal(t, result, repository.result)
}

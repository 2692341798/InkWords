package stream

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

// Historical queued messages must fail before entering any surviving generator.
func TestTaskConsumerRejectsRetiredProjectCourseKinds(t *testing.T) {
	for _, kind := range []string{"project_course_analyze", "project_course_generate"} {
		t.Run(kind, func(t *testing.T) {
			workspaceID := uuid.New()
			tasks := &fakeTaskService{}
			streams := &fakeStreamService{generateFunc: func(context.Context, uuid.UUID, GenerateRequest, chan<- string, chan<- error) {
				t.Fatal("retired task reached generation")
			}}
			consumer := NewTaskConsumer(tasks, streams)
			err := consumer.HandleGenerationRequested(context.Background(), sharedrabbitmq.GenerationRequestedMessage{
				TaskID: uuid.New(), Kind: kind, WorkspaceID: &workspaceID, Payload: []byte(`{"course_id":"course-1"}`),
			})
			require.NoError(t, err)
			require.Equal(t, TaskStatusFailed, tasks.lastStatus)
			require.Contains(t, tasks.lastErrorMessage, "unsupported generation kind")
			require.False(t, tasks.markRunningCalled)
			require.Empty(t, tasks.lastResult)
			require.Empty(t, tasks.appendEvents)
		})
	}
}

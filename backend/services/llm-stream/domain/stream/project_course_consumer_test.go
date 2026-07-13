package stream

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

type fakeProjectCourseRunner struct{}

func (fakeProjectCourseRunner) Run(context.Context, sharedrabbitmq.GenerationRequestedMessage) ([]byte, error) {
	return []byte(`{"status":"awaiting_approval"}`), nil
}

type cancellableProjectCourseRunner struct{}

func (cancellableProjectCourseRunner) Run(ctx context.Context, _ sharedrabbitmq.GenerationRequestedMessage) ([]byte, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestTaskConsumerRoutesProjectCourseToDedicatedRunner(t *testing.T) {
	tasks := &fakeTaskService{}
	consumer := NewTaskConsumer(tasks, &fakeStreamService{}, fakeProjectCourseRunner{})
	err := consumer.HandleGenerationRequested(context.Background(), sharedrabbitmq.GenerationRequestedMessage{TaskID: uuid.New(), Kind: "project_course_analyze", UserID: uuid.New(), Payload: []byte(`{"course_id":"course-1"}`)})
	require.NoError(t, err)
	require.Equal(t, TaskStatusSucceeded, tasks.lastStatus)
	require.JSONEq(t, `{"status":"awaiting_approval"}`, string(tasks.lastResult))
	require.Len(t, tasks.appendEvents, 2)
	require.Contains(t, string(tasks.appendEvents[0].Payload), `"stage":"analysis"`)
	require.Contains(t, string(tasks.appendEvents[0].Payload), `"blueprint_version":1`)
	require.Contains(t, string(tasks.appendEvents[1].Payload), `"checkpoint":"result_ready"`)
	require.Contains(t, string(tasks.appendEvents[1].Payload), `"completed":true`)
	require.Contains(t, string(tasks.appendEvents[1].Payload), `"output_hash":"sha256:`)
}

func TestTaskConsumerFailsProjectCourseWhenRunnerIsNotConfigured(t *testing.T) {
	tasks := &fakeTaskService{}
	consumer := NewTaskConsumer(tasks, &fakeStreamService{})
	err := consumer.HandleGenerationRequested(context.Background(), sharedrabbitmq.GenerationRequestedMessage{TaskID: uuid.New(), Kind: "project_course_generate", UserID: uuid.New(), Payload: []byte(`{"course_id":"course-1"}`)})
	require.NoError(t, err)
	require.Equal(t, TaskStatusFailed, tasks.lastStatus)
	require.Contains(t, tasks.lastErrorMessage, "worker is not configured")
}

func TestTaskConsumerCancelsProjectCourseRunnerWithoutMarkingFailure(t *testing.T) {
	tasks := &fakeTaskService{cancelAfterNCalls: 2}
	consumer := NewTaskConsumer(tasks, &fakeStreamService{}, cancellableProjectCourseRunner{})
	consumer.cancellationPollInterval = 1 * time.Millisecond
	err := consumer.HandleGenerationRequested(context.Background(), sharedrabbitmq.GenerationRequestedMessage{TaskID: uuid.New(), Kind: "project_course_generate", UserID: uuid.New(), Payload: []byte(`{"course_id":"course-1"}`)})
	require.NoError(t, err)
	require.NotEqual(t, TaskStatusFailed, tasks.lastStatus)
}

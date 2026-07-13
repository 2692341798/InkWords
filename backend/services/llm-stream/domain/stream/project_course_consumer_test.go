package stream

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

type fakeProjectCourseRunner struct{}

func (fakeProjectCourseRunner) Run(context.Context, sharedrabbitmq.GenerationRequestedMessage) ([]byte, error) {
	return []byte(`{"status":"awaiting_approval"}`), nil
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

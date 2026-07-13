package verification

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

type fakeTaskStore struct {
	running, succeeded, failed bool
	result                     []byte
	message                    string
}

func (s *fakeTaskStore) MarkRunning(context.Context, uuid.UUID) error { s.running = true; return nil }
func (s *fakeTaskStore) MarkSucceeded(_ context.Context, _ uuid.UUID, result []byte) error {
	s.succeeded = true
	s.result = result
	return nil
}
func (s *fakeTaskStore) MarkFailed(_ context.Context, _ uuid.UUID, message string) error {
	s.failed = true
	s.message = message
	return nil
}
func (s *fakeTaskStore) IsCancelled(context.Context, uuid.UUID) (bool, error) { return false, nil }

type fakeResolver struct{ request RunRequest }

func (r fakeResolver) Resolve(context.Context, VerificationPayload) (RunRequest, error) {
	return r.request, nil
}

type fakeVerifier struct{}

func (fakeVerifier) Verify(context.Context, RunRequest) Report {
	return Report{Passed: true, CheckpointID: "checkpoint-01"}
}

func TestConsumerResolvesStoredArtifactAndPersistsStructuredReport(t *testing.T) {
	tasks := &fakeTaskStore{}
	consumer := NewConsumer(tasks, fakeResolver{}, fakeVerifier{})
	payload, err := json.Marshal(VerificationPayload{CourseID: uuid.New(), BlueprintVersion: 1, ChapterID: "chapter-1", ArtifactToken: "artifact-1"})
	require.NoError(t, err)
	err = consumer.HandleVerificationRequested(context.Background(), sharedrabbitmq.VerificationRequestedMessage{TaskID: uuid.New(), Kind: TaskSubtype, Payload: payload})
	require.NoError(t, err)
	require.True(t, tasks.running)
	require.True(t, tasks.succeeded)
	require.False(t, tasks.failed)
	var report Report
	require.NoError(t, json.Unmarshal(tasks.result, &report))
	require.True(t, report.Passed)
}

func TestConsumerRejectsArbitraryPayloadAndUnsupportedKind(t *testing.T) {
	tasks := &fakeTaskStore{}
	consumer := NewConsumer(tasks, fakeResolver{}, fakeVerifier{})
	err := consumer.HandleVerificationRequested(context.Background(), sharedrabbitmq.VerificationRequestedMessage{TaskID: uuid.New(), Kind: TaskSubtype, Payload: []byte(`{"root_dir":"/tmp/host"}`)})
	require.NoError(t, err)
	require.True(t, tasks.failed)
	require.Contains(t, tasks.message, "invalid verification payload")
	tasks.failed = false
	err = consumer.HandleVerificationRequested(context.Background(), sharedrabbitmq.VerificationRequestedMessage{TaskID: uuid.New(), Kind: "shell", Payload: []byte(`{}`)})
	require.NoError(t, err)
	require.True(t, tasks.failed)
	require.Contains(t, tasks.message, "unsupported")
}

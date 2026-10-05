package textbookartifact

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	task "inkwords-backend/services/core-api/domain/task"
	textbook "inkwords-backend/services/core-api/domain/textbook"
	shared "inkwords-backend/shared/kernel/textbook"
	"strings"
	"testing"
)

type attemptAdapter struct {
	recordingVerificationTaskCreator
	rows    []task.JobTask
	creates int
	last    task.CreateVerificationAttemptInput
	err     error
	status  task.JobTaskStatus
}

func (a *attemptAdapter) ListVerificationAttempts(context.Context, uuid.UUID, uuid.UUID) ([]task.JobTask, error) {
	return a.rows, a.err
}
func (a *attemptAdapter) CreateVerificationAttempt(_ context.Context, input task.CreateVerificationAttemptInput) (task.JobTask, error) {
	a.creates++
	a.last = input
	status := a.status
	if status == "" {
		status = task.JobTaskStatusQueued
	}
	return task.JobTask{ID: input.RequestID, Status: status, VerificationAttempt: 2}, a.err
}

func TestVerificationAttemptsResolveFrozenInputsAndReadHistoryWithoutRestart(t *testing.T) {
	workspace, chapter, artifact, revision := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	hash := "sha256:" + strings.Repeat("a", 64)
	adapter := &attemptAdapter{}
	service := NewVerificationTaskService(fakeChapterWorkspaceReader{workspace: &textbook.ChapterWorkspace{CodeArtifacts: []textbook.CodeArtifactRow{{ID: artifact, RevisionID: revision, Kind: shared.CodeArtifactTeachingImplementation, ArtifactHash: hash, ManifestHash: hash}}}}, adapter, true)
	payload := shared.ArtifactVerificationRequest{ArtifactID: artifact.String(), RevisionID: revision.String(), ArtifactHash: hash, ManifestHash: hash}
	raw, _ := json.Marshal(payload)
	old := uuid.New()
	adapter.rows = []task.JobTask{{ID: old, Status: task.JobTaskStatusCancelled, PayloadJSON: raw, VerificationAttempt: 1}}
	lookup, err := service.GetTextbookVerificationTask(t.Context(), workspace, chapter, artifact)
	require.NoError(t, err)
	require.True(t, lookup.CanStartNew)
	require.Len(t, lookup.History, 1)
	restored, err := service.CreateTextbookVerificationTask(t.Context(), workspace, chapter, artifact)
	require.NoError(t, err)
	require.Equal(t, old, restored.ID)
	require.Zero(t, adapter.creates)
	input := textbook.VerificationAttemptInput{RequestID: uuid.New(), ExpectedPreviousTaskID: &old}
	created, err := service.CreateVerificationAttempt(t.Context(), workspace, chapter, artifact, input)
	require.NoError(t, err)
	require.Equal(t, input.RequestID, created.ID)
	require.Equal(t, payload, adapter.last.Payload)
	require.Equal(t, workspace, adapter.last.WorkspaceID)
	require.Equal(t, &old, adapter.last.ExpectedPreviousTaskID)
	_, err = service.CreateVerificationAttempt(t.Context(), workspace, chapter, uuid.New(), input)
	require.ErrorIs(t, err, textbook.ErrNotFound)
	require.Equal(t, 1, adapter.creates)
	adapter.status = task.JobTaskStatusSucceeded
	replayed, err := service.CreateVerificationAttempt(t.Context(), workspace, chapter, artifact, input)
	require.NoError(t, err)
	require.False(t, replayed.CanStartNew, "POST replay must not advertise an older terminal task as the latest predecessor")
	adapter.err = task.ErrVerificationExecutionActive
	_, err = service.CreateVerificationAttempt(t.Context(), workspace, chapter, artifact, input)
	require.ErrorIs(t, err, textbook.ErrVersionConflict)
	adapter.err = nil
	service.enabled = false
	lookup, err = service.GetTextbookVerificationTask(t.Context(), workspace, chapter, artifact)
	require.NoError(t, err)
	require.False(t, lookup.CanStartNew)
	_, err = service.CreateVerificationAttempt(t.Context(), workspace, chapter, artifact, input)
	require.ErrorIs(t, err, textbook.ErrInvalidState)
	adapter.rows[0].PayloadJSON = []byte(`{"artifact_id":"changed"}`)
	_, err = service.GetTextbookVerificationTask(t.Context(), workspace, chapter, artifact)
	require.ErrorIs(t, err, textbook.ErrInvalidState)
}

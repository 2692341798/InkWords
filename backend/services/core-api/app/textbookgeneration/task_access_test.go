package textbookgeneration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	coretask "inkwords-backend/services/core-api/domain/task"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type fakeTaskAccess struct {
	task               coretask.JobTask
	requestedWorkspace uuid.UUID
	retried            bool
}

func (access *fakeTaskAccess) GetTextbookTask(_ context.Context, _ uuid.UUID, workspaceID uuid.UUID) (coretask.JobTask, error) {
	access.requestedWorkspace = workspaceID
	return access.task, nil
}

func (access *fakeTaskAccess) RetryTextbookTask(_ context.Context, _ uuid.UUID, workspaceID uuid.UUID) (coretask.JobTask, error) {
	access.requestedWorkspace = workspaceID
	access.retried = true
	retried := access.task
	retried.Status = coretask.JobTaskStatusQueued
	retried.RetryCount++
	return retried, nil
}

func TestTaskAccessUsesWorkspaceIdentityForTextbookTask(t *testing.T) {
	workspaceID := uuid.New()
	payload := textbookGenerationPayload()
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	access := &fakeTaskAccess{task: coretask.JobTask{ID: uuid.New(), TaskType: "generation", TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, Status: coretask.JobTaskStatusFailed, PayloadJSON: encoded}}
	service := NewTaskAccessService(access)

	snapshot, err := service.Get(t.Context(), workspaceID, access.task.ID)
	require.NoError(t, err)
	require.Equal(t, access.task.ID, snapshot.ID)
	require.Equal(t, workspaceID, access.requestedWorkspace)
	require.NotNil(t, snapshot.RetryConfirmation)
	require.Equal(t, payload.InputHash, snapshot.RetryConfirmation.InputHash)
	require.Equal(t, sharedgeneration.ConservativeTokenEstimate(string(encoded))+sharedtextbook.SamplePreflightInstructionReserveTokens, snapshot.RetryConfirmation.EstimatedInputTokens)
	require.True(t, snapshot.RetryConfirmation.RequiresConfirmation)

	_, err = service.Retry(t.Context(), workspaceID, access.task.ID, "")
	require.ErrorIs(t, err, ErrRetryConfirmationRequired)
	require.False(t, access.retried)

	retried, err := service.Retry(t.Context(), workspaceID, access.task.ID, payload.InputHash)
	require.NoError(t, err)
	require.True(t, access.retried)
	require.Equal(t, coretask.JobTaskStatusQueued, retried.Status)
}

func TestTaskAccessRejectsLegacyTaskEvenForBridgeOwner(t *testing.T) {
	workspaceID := uuid.New()
	access := &fakeTaskAccess{task: coretask.JobTask{ID: uuid.New(), TaskType: "generation", TaskSubtype: "project_course_generate", Status: coretask.JobTaskStatusFailed}}
	service := NewTaskAccessService(access)

	_, err := service.Get(t.Context(), workspaceID, access.task.ID)
	require.ErrorIs(t, err, ErrTaskOutsideTextbookScope)

	_, err = service.Retry(t.Context(), workspaceID, access.task.ID, "")
	require.ErrorIs(t, err, ErrTaskOutsideTextbookScope)
	require.False(t, access.retried)
}

package textbookartifact

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type fakeChapterWorkspaceReader struct {
	workspace *textbookdomain.ChapterWorkspace
	err       error
}

func (reader fakeChapterWorkspaceReader) GetChapterWorkspace(context.Context, uuid.UUID, uuid.UUID) (*textbookdomain.ChapterWorkspace, error) {
	return reader.workspace, reader.err
}

type recordingVerificationTaskCreator struct {
	input coretask.CreateTextbookVerificationTaskInput
}

func (creator *recordingVerificationTaskCreator) CreateTextbookVerificationTask(_ context.Context, input coretask.CreateTextbookVerificationTaskInput) (coretask.JobTask, error) {
	creator.input = input
	return coretask.JobTask{ID: uuid.New(), Status: coretask.JobTaskStatusQueued}, nil
}

func TestVerificationTaskServiceResolvesOnlyAChapterOwnedTeachingArtifact(t *testing.T) {
	workspaceID, chapterID, artifactID, revisionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	creator := &recordingVerificationTaskCreator{}
	service := NewVerificationTaskService(fakeChapterWorkspaceReader{workspace: &textbookdomain.ChapterWorkspace{CodeArtifacts: []textbookdomain.CodeArtifactRow{{ID: artifactID, RevisionID: revisionID, Kind: sharedtextbook.CodeArtifactTeachingImplementation, ArtifactHash: "sha256:" + strings.Repeat("a", 64), ManifestHash: "sha256:manifest"}}}}, creator, true)

	// The fake hash is not parsed by admission; its value is frozen and later
	// rechecked by the runner against the content-addressed store.
	queued, err := service.CreateTextbookVerificationTask(context.Background(), workspaceID, chapterID, artifactID)
	require.NoError(t, err)
	require.Equal(t, string(coretask.JobTaskStatusQueued), queued.Status)
	require.Equal(t, VerificationTaskSubtype, creator.input.TaskSubtype)
	require.Equal(t, workspaceID, creator.input.WorkspaceID)
	var payload VerificationTaskPayload
	require.NoError(t, json.Unmarshal(creator.input.Payload, &payload))
	require.Equal(t, artifactID.String(), payload.ArtifactID)
	require.Equal(t, revisionID.String(), payload.RevisionID)

	_, err = service.CreateTextbookVerificationTask(context.Background(), workspaceID, chapterID, uuid.New())
	require.ErrorIs(t, err, textbookdomain.ErrNotFound)
}

func TestVerificationTaskServiceRejectsQueueingWhenTheIsolatedRunnerIsDisabled(t *testing.T) {
	service := NewVerificationTaskService(fakeChapterWorkspaceReader{}, &recordingVerificationTaskCreator{}, false)
	_, err := service.CreateTextbookVerificationTask(context.Background(), uuid.New(), uuid.New(), uuid.New())
	require.Error(t, err)
}

package textbookartifact

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"strings"
	"testing"
)

type lookupVerificationTasks struct {
	recordingVerificationTaskCreator
	item      *coretask.JobTask
	calls     int
	workspace uuid.UUID
	key       string
}

func (f *lookupVerificationTasks) FindTextbookVerificationTask(_ context.Context, workspace uuid.UUID, key string) (*coretask.JobTask, error) {
	f.calls++
	f.workspace = workspace
	f.key = key
	return f.item, nil
}

func TestVerificationLookupRejectsMissingOwnershipAndMismatchedFrozenInputs(t *testing.T) {
	workspace, chapter, artifactID, revision := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	artifact := textbookdomain.CodeArtifactRow{ID: artifactID, RevisionID: revision, Kind: sharedtextbook.CodeArtifactTeachingImplementation, ArtifactHash: "sha256:" + strings.Repeat("a", 64), ManifestHash: "sha256:" + strings.Repeat("b", 64)}
	reader := &lookupVerificationTasks{}
	service := NewVerificationTaskService(fakeChapterWorkspaceReader{workspace: &textbookdomain.ChapterWorkspace{CodeArtifacts: []textbookdomain.CodeArtifactRow{artifact}}}, reader, false)
	found, err := service.GetTextbookVerificationTask(t.Context(), workspace, chapter, artifactID)
	require.NoError(t, err)
	require.Nil(t, found)
	require.Equal(t, workspace, reader.workspace)
	require.Equal(t, "textbook-verify:"+artifact.ID.String()+":"+artifact.ManifestHash, reader.key)
	_, err = service.GetTextbookVerificationTask(t.Context(), workspace, chapter, uuid.New())
	require.ErrorIs(t, err, textbookdomain.ErrNotFound)
	require.Equal(t, 1, reader.calls)
	frozen := sharedtextbook.ArtifactVerificationRequest{ArtifactID: artifactID.String(), RevisionID: revision.String(), ArtifactHash: artifact.ArtifactHash, ManifestHash: artifact.ManifestHash}
	raw, err := json.Marshal(frozen)
	require.NoError(t, err)
	reader.item = &coretask.JobTask{ID: uuid.New(), Status: coretask.JobTaskStatusRunning, PayloadJSON: raw}
	found, err = service.GetTextbookVerificationTask(t.Context(), workspace, chapter, artifactID)
	require.NoError(t, err)
	require.Equal(t, reader.item.ID, found.ID)
	frozen.RevisionID = uuid.NewString()
	reader.item.PayloadJSON, _ = json.Marshal(frozen)
	_, err = service.GetTextbookVerificationTask(t.Context(), workspace, chapter, artifactID)
	require.ErrorIs(t, err, textbookdomain.ErrInvalidState)
	require.Equal(t, uuid.Nil, reader.input.WorkspaceID, "lookup must not enqueue")
}

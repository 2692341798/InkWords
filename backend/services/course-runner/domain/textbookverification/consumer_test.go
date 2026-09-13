package textbookverification

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

type recordingTextbookTaskStore struct {
	running, succeeded bool
	failed             string
	workspaceMismatch  bool
}

func (store *recordingTextbookTaskStore) ClaimVerificationWorker(context.Context, uuid.UUID) (uuid.UUID, error) {
	store.running = true
	return uuid.New(), nil
}
func (*recordingTextbookTaskStore) ReleaseVerificationWorker(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (store *recordingTextbookTaskStore) MarkSucceeded(context.Context, uuid.UUID, []byte) error {
	store.succeeded = true
	return nil
}
func (store *recordingTextbookTaskStore) MarkFailed(_ context.Context, _ uuid.UUID, message string) error {
	store.failed = message
	return nil
}
func (*recordingTextbookTaskStore) IsCancelled(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}
func (store *recordingTextbookTaskStore) TextbookWorkspaceMatches(context.Context, uuid.UUID, uuid.UUID, string) (bool, error) {
	return !store.workspaceMismatch, nil
}

type fakeTextbookResolver struct{ request RunRequest }

func (resolver fakeTextbookResolver) Resolve(context.Context, VerificationPayload) (RunRequest, error) {
	return resolver.request, nil
}

type recordingEvidencePersister struct{ report Report }

func (persister *recordingEvidencePersister) PersistVerificationReport(_ context.Context, _ uuid.UUID, _ VerificationPayload, report Report) error {
	persister.report = report
	return nil
}

func TestConsumerPersistsUnverifiedReportInsteadOfClaimingSuccess(t *testing.T) {
	manifest := ArtifactManifest{Format: ArtifactManifestFormat, ArtifactID: uuid.NewString(), RevisionID: uuid.NewString(), ArtifactHash: "sha256:" + strings.Repeat("a", 64), BookContractHash: "sha256:" + strings.Repeat("c", 64), StyleSheetHash: "sha256:" + strings.Repeat("d", 64), Language: "go", ToolchainVersion: "go1.25.4", Commands: []CommandTemplate{{Kind: "go_test"}}}
	payload := VerificationPayload{ArtifactID: manifest.ArtifactID, RevisionID: manifest.RevisionID, ArtifactHash: manifest.ArtifactHash, ManifestHash: "sha256:" + strings.Repeat("b", 64)}
	rawPayload, err := json.Marshal(payload)
	require.NoError(t, err)
	tasks := &recordingTextbookTaskStore{}
	evidence := &recordingEvidencePersister{}
	consumer := NewConsumer(tasks, fakeTextbookResolver{request: RunRequest{Manifest: manifest, RootDir: t.TempDir()}}, Runner{}, evidence)
	workspaceID := uuid.New()
	require.NoError(t, consumer.HandleVerificationRequested(context.Background(), sharedrabbitmq.TextbookVerificationRequestedMessage{TaskID: uuid.New(), Kind: TaskSubtype, WorkspaceID: &workspaceID, Payload: rawPayload}))
	require.True(t, tasks.running)
	require.True(t, tasks.succeeded, "the verification task completed, even though its evidence was not verified")
	require.Empty(t, tasks.failed)
	require.Equal(t, sharedtextbook.ArtifactStatusUnverified, evidence.report.Status)
	require.Contains(t, evidence.report.Reason, "not configured")
}

func TestConsumerRejectsUntrustedPayloadBeforeResolution(t *testing.T) {
	tasks := &recordingTextbookTaskStore{}
	consumer := NewConsumer(tasks, fakeTextbookResolver{}, Runner{}, &recordingEvidencePersister{})
	workspaceID := uuid.New()
	require.NoError(t, consumer.HandleVerificationRequested(context.Background(), sharedrabbitmq.TextbookVerificationRequestedMessage{TaskID: uuid.New(), Kind: TaskSubtype, WorkspaceID: &workspaceID, Payload: []byte(`{"artifact_id":"/tmp/host"}`)}))
	require.Equal(t, "invalid textbook verification payload", tasks.failed)
}

func TestConsumerRejectsMissingOrMismatchedWorkspaceBeforeResolution(t *testing.T) {
	workspaceID := uuid.New()
	payload := VerificationPayload{ArtifactID: uuid.NewString(), RevisionID: uuid.NewString(), ArtifactHash: "sha256:" + strings.Repeat("a", 64), ManifestHash: "sha256:" + strings.Repeat("b", 64)}
	rawPayload, err := json.Marshal(payload)
	require.NoError(t, err)
	for _, test := range []struct {
		name      string
		workspace *uuid.UUID
		mismatch  bool
	}{
		{name: "missing"},
		{name: "mismatched", workspace: &workspaceID, mismatch: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			tasks := &recordingTextbookTaskStore{workspaceMismatch: test.mismatch}
			resolver := &countingTextbookResolver{}
			consumer := NewConsumer(tasks, resolver, Runner{}, &recordingEvidencePersister{})
			require.NoError(t, consumer.HandleVerificationRequested(t.Context(), sharedrabbitmq.TextbookVerificationRequestedMessage{TaskID: uuid.New(), Kind: TaskSubtype, WorkspaceID: test.workspace, Payload: rawPayload}))
			require.False(t, tasks.running)
			require.NotEmpty(t, tasks.failed)
			require.Zero(t, resolver.calls)
		})
	}
}

type countingTextbookResolver struct{ calls int }

func (resolver *countingTextbookResolver) Resolve(context.Context, VerificationPayload) (RunRequest, error) {
	resolver.calls++
	return RunRequest{}, nil
}

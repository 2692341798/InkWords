package task

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	shared "inkwords-backend/shared/kernel/textbook"
	"strings"
	"testing"
)

type uncertainVerificationPublisher struct {
	verificationRetryPublisher
	fail bool
}

func (p *uncertainVerificationPublisher) PublishTextbookVerificationRequested(ctx context.Context, m TextbookVerificationRequestedMessage) error {
	if p.fail {
		p.fail = false
		return errors.New("broker response unavailable")
	}
	return p.verificationRetryPublisher.PublishTextbookVerificationRequested(ctx, m)
}

func TestVerificationAttemptExplicitRetryRecoversPublishWithoutReplacingTask(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &VerificationSeries{}))
	repo := NewGormRepository(db)
	publisher := &uncertainVerificationPublisher{fail: true}
	service := NewService(repo, publisher, nil)
	input := CreateVerificationAttemptInput{RequestID: uuid.New(), WorkspaceID: uuid.New(), Payload: shared.ArtifactVerificationRequest{ArtifactID: uuid.NewString(), RevisionID: uuid.NewString(), ArtifactHash: "sha256:" + strings.Repeat("a", 64), ManifestHash: "sha256:" + strings.Repeat("b", 64)}}
	first, err := service.CreateVerificationAttempt(t.Context(), input)
	require.Error(t, err)
	require.Equal(t, input.RequestID, first.ID)
	same, err := service.CreateVerificationAttempt(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, first.ID, same.ID)
	require.Len(t, publisher.messages, 1)
	_, err = repo.ClaimVerificationWorker(t.Context(), first.ID)
	require.NoError(t, err)
	same, err = service.CreateVerificationAttempt(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, first.ID, same.ID)
	require.Len(t, publisher.messages, 1)
	history, err := service.ListVerificationAttempts(t.Context(), input.WorkspaceID, uuid.MustParse(input.Payload.ArtifactID))
	require.NoError(t, err)
	require.Len(t, history, 1)
}

func TestVerificationAttemptsPreserveCancelledRunAndWaitForWorkerRelease(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &VerificationSeries{}))
	repo := NewGormRepository(db)
	workspace, artifact, revision := uuid.New(), uuid.New(), uuid.New()
	input := CreateVerificationAttemptInput{RequestID: uuid.New(), WorkspaceID: workspace, Payload: shared.ArtifactVerificationRequest{ArtifactID: artifact.String(), RevisionID: revision.String(), ArtifactHash: "sha256:" + strings.Repeat("a", 64), ManifestHash: "sha256:" + strings.Repeat("b", 64)}}
	first, created, err := repo.CreateVerificationAttempt(t.Context(), input)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, 1, first.VerificationAttempt)
	same, created, err := repo.CreateVerificationAttempt(t.Context(), input)
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, first.ID, same.ID)
	token, err := repo.ClaimVerificationWorker(t.Context(), first.ID)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, token)
	require.NoError(t, repo.UpdateStatus(t.Context(), first.ID, JobTaskStatusCancelled, ""))
	next := input
	next.RequestID = uuid.New()
	next.ExpectedPreviousTaskID = &first.ID
	_, _, err = repo.CreateVerificationAttempt(t.Context(), next)
	require.ErrorIs(t, err, ErrVerificationExecutionActive)
	require.Error(t, repo.ReleaseVerificationWorker(t.Context(), first.ID, uuid.New()))
	require.NoError(t, repo.ReleaseVerificationWorker(t.Context(), first.ID, token))
	second, created, err := repo.CreateVerificationAttempt(t.Context(), next)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, 2, second.VerificationAttempt)
	preserved, err := repo.GetByID(t.Context(), first.ID)
	require.NoError(t, err)
	require.Equal(t, JobTaskStatusCancelled, preserved.Status)
	require.NotNil(t, preserved.VerificationWorkerReleasedAt)
	_, err = NewService(repo, nil, nil).retryResolvedTask(t.Context(), *preserved)
	require.ErrorIs(t, err, ErrTaskNotRetryable)
	conflict := next
	conflict.RequestID = uuid.New()
	_, _, err = repo.CreateVerificationAttempt(t.Context(), conflict)
	require.ErrorIs(t, err, ErrVerificationAttemptConflict)
	conflict = input
	conflict.WorkspaceID = uuid.New()
	_, _, err = repo.CreateVerificationAttempt(t.Context(), conflict)
	require.Error(t, err)
	changed := next
	changed.Payload.ManifestHash = "sha256:" + strings.Repeat("c", 64)
	_, _, err = repo.CreateVerificationAttempt(t.Context(), changed)
	require.ErrorIs(t, err, ErrVerificationAttemptConflict)
	history, err := repo.ListVerificationAttempts(t.Context(), workspace, artifact)
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Equal(t, second.ID, history[0].ID)
}

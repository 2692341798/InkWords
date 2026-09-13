package task

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"strings"
	"testing"
)

type verificationRetryPublisher struct {
	retryPublisher
	messages []TextbookVerificationRequestedMessage
}

func (p *verificationRetryPublisher) PublishTextbookVerificationRequested(_ context.Context, m TextbookVerificationRequestedMessage) error {
	p.messages = append(p.messages, m)
	return nil
}

func TestVerificationRetryKeepsExactManifestAndNeverCallsGeneration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &JobTaskEvent{}))
	workspace := uuid.New()
	raw, err := json.Marshal(sharedtextbook.ArtifactVerificationRequest{ArtifactID: uuid.NewString(), RevisionID: uuid.NewString(), ArtifactHash: "sha256:" + strings.Repeat("a", 64), ManifestHash: "sha256:" + strings.Repeat("b", 64)})
	require.NoError(t, err)
	task := JobTask{TaskType: taskTypeVerification, TaskSubtype: sharedtextbook.TextbookTeachingArtifactVerifyTaskSubtype, WorkspaceID: &workspace, Status: JobTaskStatusFailed, PayloadJSON: raw, ResultJSON: []byte(`{}`)}
	require.NoError(t, db.Create(&task).Error)
	publisher := &verificationRetryPublisher{}
	service := NewService(NewGormRepository(db), publisher, nil)
	_, err = service.RetryTextbookTask(t.Context(), task.ID, uuid.New())
	require.ErrorIs(t, err, ErrTaskAccessDenied)
	retried, err := service.RetryTextbookTask(t.Context(), task.ID, workspace)
	require.NoError(t, err)
	require.Equal(t, task.ID, retried.ID)
	require.Equal(t, 1, retried.RetryCount)
	require.Len(t, publisher.messages, 1)
	require.Equal(t, string(raw), string(publisher.messages[0].Payload))
	require.Equal(t, workspace, *publisher.messages[0].WorkspaceID)
	require.Empty(t, publisher.generation)
	require.Empty(t, publisher.textbookGeneration)
	_, err = service.RetryTextbookTask(t.Context(), task.ID, workspace)
	require.ErrorIs(t, err, ErrTaskNotRetryable)
	require.NoError(t, db.Model(&task).Updates(map[string]any{"status": "failed", "payload_json": "{}"}).Error)
	_, err = service.RetryTextbookTask(t.Context(), task.ID, workspace)
	require.ErrorIs(t, err, ErrTaskNotRetryable)
	require.Len(t, publisher.messages, 1)
}

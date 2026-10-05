package task

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"testing"
)

func TestVerificationLookupIsReadOnlyAndWorkspaceScoped(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &JobTaskEvent{}))
	workspace := uuid.New()
	item := JobTask{TaskType: taskTypeVerification, TaskSubtype: sharedtextbook.TextbookTeachingArtifactVerifyTaskSubtype, WorkspaceID: &workspace, Status: JobTaskStatusQueued, IdempotencyKey: "textbook-verify:artifact:manifest", PayloadJSON: []byte(`{}`)}
	require.NoError(t, db.Create(&item).Error)
	service := NewService(NewGormRepository(db), nil, nil)
	found, err := service.FindTextbookVerificationTask(t.Context(), workspace, item.IdempotencyKey)
	require.NoError(t, err)
	require.Equal(t, item.ID, found.ID)
	missing, err := service.FindTextbookVerificationTask(t.Context(), uuid.New(), item.IdempotencyKey)
	require.NoError(t, err)
	require.Nil(t, missing)
	_, err = service.FindTextbookVerificationTask(t.Context(), uuid.Nil, item.IdempotencyKey)
	require.ErrorIs(t, err, ErrTaskAccessDenied)
	var count int64
	require.NoError(t, db.Model(&JobTask{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

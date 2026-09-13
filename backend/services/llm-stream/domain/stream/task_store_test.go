package stream

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGormTaskStoreRedactsCredentialLikeFailureBeforePersistence(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&jobTask{}, &jobTaskEvent{}))
	taskID := uuid.New()
	require.NoError(t, db.Create(&jobTask{ID: taskID, Status: TaskStatusRunning}).Error)

	store := NewGormTaskStore(db)
	require.NoError(t, store.MarkFailed(context.Background(), taskID, "provider response: Authorization: Bearer test-secret"))

	var task jobTask
	require.NoError(t, db.Where("id = ?", taskID).First(&task).Error)
	require.Equal(t, "generation task failed; provider diagnostic redacted", task.ErrorMessage)
	require.NotContains(t, task.ErrorMessage, "test-secret")
	var event jobTaskEvent
	require.NoError(t, db.Where("task_id = ?", taskID).First(&event).Error)
	require.NotContains(t, string(event.Payload), "test-secret")
}

func TestGormTaskStorePersistsSafeStructuredFailureResult(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&jobTask{}, &jobTaskEvent{}))
	taskID := uuid.New()
	require.NoError(t, db.Create(&jobTask{ID: taskID, Status: TaskStatusRunning}).Error)
	failure := []byte(`{"result_version":1,"final_status":"failed","provider_name":"deepseek","provider_usage_json":{"known":true,"input_tokens":101},"candidate_persisted":false}`)

	store := NewGormTaskStore(db)
	require.NoError(t, store.MarkFailedWithResult(context.Background(), taskID, "generated candidate failed quality gates", failure))

	var task jobTask
	require.NoError(t, db.Where("id = ?", taskID).First(&task).Error)
	require.Equal(t, TaskStatusFailed, task.Status)
	require.JSONEq(t, string(failure), string(task.ResultJSON))
	var event jobTaskEvent
	require.NoError(t, db.Where("task_id = ?", taskID).First(&event).Error)
	var eventPayload struct {
		Failure json.RawMessage `json:"failure"`
	}
	require.NoError(t, json.Unmarshal(event.Payload, &eventPayload))
	require.JSONEq(t, string(failure), string(eventPayload.Failure))
	require.NotContains(t, string(event.Payload), "markdown")
}

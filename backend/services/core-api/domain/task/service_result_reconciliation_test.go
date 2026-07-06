package task

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type failingTaskPublisher struct{}

func (failingTaskPublisher) PublishGenerationRequested(context.Context, GenerationRequestedMessage) error {
	return errors.New("broker unavailable")
}

func (failingTaskPublisher) PublishParseRequested(context.Context, ParseRequestedMessage) error {
	return errors.New("broker unavailable")
}

func (failingTaskPublisher) PublishExportRequested(context.Context, ExportRequestedMessage) error {
	return errors.New("broker unavailable")
}

type countingResultPersister struct {
	calls int
}

func (p *countingResultPersister) PersistGenerationResult(context.Context, uuid.UUID, map[string]any) error {
	p.calls++
	return nil
}

func TestGetTaskReconcilesWorkerResultExactlyOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &JobTaskEvent{}))

	ownerID := uuid.New()
	task := JobTask{
		TaskType: taskTypeGeneration, TaskSubtype: "generate_series", Status: JobTaskStatusSucceeded,
		RequestedBy: ownerID, ResultJSON: []byte(`{"task_type":"generation","task_subtype":"generate_series"}`),
	}
	require.NoError(t, db.Create(&task).Error)

	persister := &countingResultPersister{}
	service := NewService(NewGormRepository(db), nil, persister)
	_, err = service.GetTask(context.Background(), task.ID, ownerID)
	require.NoError(t, err)
	_, err = service.GetTask(context.Background(), task.ID, ownerID)
	require.NoError(t, err)
	require.Equal(t, 1, persister.calls)

	var stored JobTask
	require.NoError(t, db.First(&stored, "id = ?", task.ID).Error)
	require.NotNil(t, stored.ResultPersistedAt)
	require.Nil(t, stored.ResultPersistenceStartedAt)
}

func TestCreateTaskMarksStoredTaskFailedWhenPublishFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &JobTaskEvent{}))

	ownerID := uuid.New()
	service := NewService(NewGormRepository(db), failingTaskPublisher{}, nil)
	_, err = service.CreateParseTask(context.Background(), CreateParseTaskInput{
		RequestedBy: ownerID, TaskSubtype: "parse_file", IdempotencyKey: "parse:test", Payload: []byte(`{"filename":"test.md"}`),
	})
	require.ErrorContains(t, err, "发布任务消息失败")

	var stored JobTask
	require.NoError(t, db.First(&stored).Error)
	require.Equal(t, JobTaskStatusFailed, stored.Status)
	require.Equal(t, "发布任务消息失败", stored.ErrorMessage)
	require.NotNil(t, stored.FinishedAt)
}

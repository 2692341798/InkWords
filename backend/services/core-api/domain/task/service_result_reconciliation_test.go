package task

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
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
	calls      int
	parseCalls int
}

func (p *countingResultPersister) PersistGenerationResult(context.Context, uuid.UUID, map[string]any) error {
	p.calls++
	return nil
}

func (p *countingResultPersister) PersistParseResult(context.Context, uuid.UUID, map[string]any) error {
	p.parseCalls++
	return nil
}

func TestGetTaskReconcilesWorkerResultExactlyOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &JobTaskEvent{}))

	workspaceID := uuid.New()
	task := JobTask{
		TaskType: taskTypeGeneration, TaskSubtype: "generate_series", Status: JobTaskStatusSucceeded,
		WorkspaceID: &workspaceID, ResultJSON: []byte(`{"task_type":"generation","task_subtype":"generate_series"}`),
	}
	require.NoError(t, db.Create(&task).Error)

	persister := &countingResultPersister{}
	service := NewService(NewGormRepository(db), nil, persister)
	_, err = service.GetTask(context.Background(), task.ID, workspaceID)
	require.NoError(t, err)
	_, err = service.GetTask(context.Background(), task.ID, workspaceID)
	require.NoError(t, err)
	require.Equal(t, 1, persister.calls)

	var stored JobTask
	require.NoError(t, db.First(&stored, "id = ?", task.ID).Error)
	require.NotNil(t, stored.ResultPersistedAt)
	require.Nil(t, stored.ResultPersistenceStartedAt)
}

func TestGetTaskReconcilesTypedSourceImportExactlyOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &JobTaskEvent{}))

	workspaceID := uuid.New()
	task := JobTask{
		TaskType: taskTypeParse, TaskSubtype: sharedtextbook.TextbookSourceImportTaskSubtype, Status: JobTaskStatusSucceeded,
		WorkspaceID: &workspaceID, ResultJSON: []byte(`{"task_subtype":"textbook_source_import"}`),
	}
	require.NoError(t, db.Create(&task).Error)

	persister := &countingResultPersister{}
	service := NewService(NewGormRepository(db), nil, persister)
	_, err = service.GetTextbookTask(context.Background(), task.ID, workspaceID)
	require.NoError(t, err)
	_, err = service.GetTextbookTask(context.Background(), task.ID, workspaceID)
	require.NoError(t, err)
	require.Equal(t, 1, persister.parseCalls)
	require.Zero(t, persister.calls)
}

func TestGetTaskReconcilesOfficialWebImportExactlyOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &JobTaskEvent{}))

	workspaceID := uuid.New()
	task := JobTask{TaskType: taskTypeParse, TaskSubtype: sharedtextbook.TextbookOfficialWebImportTaskSubtype, Status: JobTaskStatusSucceeded, WorkspaceID: &workspaceID, ResultJSON: []byte(`{"task_subtype":"textbook_official_web_import"}`)}
	require.NoError(t, db.Create(&task).Error)
	persister := &countingResultPersister{}
	service := NewService(NewGormRepository(db), nil, persister)
	_, err = service.GetTextbookTask(context.Background(), task.ID, workspaceID)
	require.NoError(t, err)
	_, err = service.GetTextbookTask(context.Background(), task.ID, workspaceID)
	require.NoError(t, err)
	require.Equal(t, 1, persister.parseCalls)
}

func TestCreateTaskMarksStoredTaskFailedWhenPublishFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &JobTaskEvent{}))

	workspaceID := uuid.New()
	service := NewService(NewGormRepository(db), failingTaskPublisher{}, nil)
	_, err = service.CreateParseTask(context.Background(), CreateParseTaskInput{
		WorkspaceID: workspaceID, TaskSubtype: "parse_file", IdempotencyKey: "parse:test", Payload: []byte(`{"filename":"test.md"}`),
	})
	require.ErrorContains(t, err, "发布任务消息失败")

	var stored JobTask
	require.NoError(t, db.First(&stored).Error)
	require.Equal(t, JobTaskStatusFailed, stored.Status)
	require.Equal(t, "发布任务消息失败", stored.ErrorMessage)
	require.NotNil(t, stored.FinishedAt)
}

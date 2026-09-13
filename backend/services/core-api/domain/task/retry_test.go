package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type retryPublisher struct {
	generation         []GenerationRequestedMessage
	parse              []ParseRequestedMessage
	textbookGeneration []TextbookGenerationRequestedMessage
	textbookParse      []TextbookParseRequestedMessage
}

func (p *retryPublisher) PublishGenerationRequested(_ context.Context, message GenerationRequestedMessage) error {
	p.generation = append(p.generation, message)
	return nil
}
func (p *retryPublisher) PublishParseRequested(_ context.Context, message ParseRequestedMessage) error {
	p.parse = append(p.parse, message)
	return nil
}
func (p *retryPublisher) PublishTextbookGenerationRequested(_ context.Context, message TextbookGenerationRequestedMessage) error {
	p.textbookGeneration = append(p.textbookGeneration, message)
	return nil
}
func (p *retryPublisher) PublishTextbookParseRequested(_ context.Context, message TextbookParseRequestedMessage) error {
	p.textbookParse = append(p.textbookParse, message)
	return nil
}
func (*retryPublisher) PublishExportRequested(context.Context, ExportRequestedMessage) error {
	return nil
}

func TestRetryGenerationTaskReusesExactFailedTaskInput(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &JobTaskEvent{}))
	workspaceID := uuid.New()
	task := JobTask{TaskType: taskTypeGeneration, TaskSubtype: "generate_single", Status: JobTaskStatusFailed, WorkspaceID: &workspaceID, PayloadJSON: datatypes.JSON([]byte(`{"frozen":"input"}`)), ResultJSON: datatypes.JSON([]byte(`{"discard":"failed"}`)), ErrorMessage: "provider unavailable"}
	require.NoError(t, db.Create(&task).Error)
	publisher := &retryPublisher{}
	service := NewService(NewGormRepository(db), publisher, nil)

	retried, err := service.RetryGenerationTask(context.Background(), task.ID, workspaceID)
	require.NoError(t, err)
	require.Equal(t, task.ID, retried.ID)
	require.Equal(t, JobTaskStatusQueued, retried.Status)
	require.Equal(t, 1, retried.RetryCount)
	require.Empty(t, retried.ErrorMessage)
	require.JSONEq(t, `{}`, string(retried.ResultJSON))
	require.Len(t, publisher.generation, 1)
	require.Equal(t, task.ID, publisher.generation[0].TaskID)
	require.Equal(t, task.PayloadJSON, datatypes.JSON(publisher.generation[0].Payload))

	_, err = service.RetryGenerationTask(context.Background(), task.ID, workspaceID)
	require.ErrorIs(t, err, ErrTaskNotRetryable)
}

func TestRetryGenerationTaskRequeuesFailedParseWithFrozenPayload(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &JobTaskEvent{}))
	workspaceID := uuid.New()
	task := JobTask{TaskType: taskTypeParse, TaskSubtype: "parse_file", Status: JobTaskStatusFailed, WorkspaceID: &workspaceID, PayloadJSON: datatypes.JSON([]byte(`{"snapshot":"frozen"}`)), ResultJSON: datatypes.JSON([]byte(`{}`)), ErrorMessage: "broker unavailable"}
	require.NoError(t, db.Create(&task).Error)
	publisher := &retryPublisher{}
	service := NewService(NewGormRepository(db), publisher, nil)

	retried, err := service.RetryGenerationTask(context.Background(), task.ID, workspaceID)
	require.NoError(t, err)
	require.Equal(t, JobTaskStatusQueued, retried.Status)
	require.Equal(t, 1, retried.RetryCount)
	require.Len(t, publisher.parse, 1)
	require.Equal(t, task.ID, publisher.parse[0].TaskID)
	require.Equal(t, task.PayloadJSON, datatypes.JSON(publisher.parse[0].Payload))
}

func TestTextbookTaskAccessAndRetryAreScopedByWorkspace(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &JobTaskEvent{}))
	workspaceID := uuid.New()
	task := JobTask{TaskType: taskTypeGeneration, TaskSubtype: "textbook_sample_generate", Status: JobTaskStatusFailed, WorkspaceID: &workspaceID, PayloadJSON: datatypes.JSON([]byte(`{}`)), ResultJSON: datatypes.JSON([]byte(`{}`))}
	require.NoError(t, db.Create(&task).Error)
	publisher := &retryPublisher{}
	service := NewService(NewGormRepository(db), publisher, nil)

	_, err = service.GetTextbookTask(context.Background(), task.ID, uuid.New())
	require.ErrorIs(t, err, ErrTaskAccessDenied)
	loaded, err := service.GetTextbookTask(context.Background(), task.ID, workspaceID)
	require.NoError(t, err)
	require.Equal(t, task.ID, loaded.ID)
	retried, err := service.RetryTextbookTask(context.Background(), task.ID, workspaceID)
	require.NoError(t, err)
	require.Equal(t, JobTaskStatusQueued, retried.Status)
	require.Len(t, publisher.textbookGeneration, 1)
	require.Equal(t, workspaceID, *publisher.textbookGeneration[0].WorkspaceID)
	require.Empty(t, publisher.generation)
}

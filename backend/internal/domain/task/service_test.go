package task

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"inkwords-backend/internal/model"
)

func TestService_CreateGenerationTask_ReusesIdempotencyKey(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, nil)

	req := CreateGenerationTaskInput{
		RequestedBy:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		TaskSubtype:    "generate_series",
		IdempotencyKey: "series:abc",
		Payload:        []byte(`{"series_title":"Go源码入门"}`),
	}

	first, err := service.CreateGenerationTask(context.Background(), req)
	require.NoError(t, err)

	second, err := service.CreateGenerationTask(context.Background(), req)
	require.NoError(t, err)

	require.Equal(t, first.ID, second.ID)
	require.Equal(t, model.JobTaskStatusQueued, second.Status)
}

func TestService_AppendEvent_UpdatesStreamingState(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, nil)

	task := repo.seedTask(model.JobTask{
		ID:     uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Status: model.JobTaskStatusRunning,
	})

	err := service.AppendEvent(context.Background(), task.ID, AppendEventInput{
		EventType: "chunk",
		Status:    model.JobTaskStatusStreaming,
		Payload:   []byte(`{"status":"streaming","chapter_sort":1,"content":"hello"}`),
	})
	require.NoError(t, err)

	stored, err := repo.GetByID(context.Background(), task.ID)
	require.NoError(t, err)
	require.Equal(t, model.JobTaskStatusStreaming, stored.Status)
	require.Len(t, repo.eventsByTaskID[task.ID], 1)
	require.Equal(t, "chunk", repo.eventsByTaskID[task.ID][0].EventType)
}

func TestService_CancelTask_MarksCancelled(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, nil)

	task := repo.seedTask(model.JobTask{
		ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		RequestedBy: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		Status:      model.JobTaskStatusQueued,
	})

	err := service.CancelTask(context.Background(), task.ID, task.RequestedBy)
	require.NoError(t, err)

	stored, err := repo.GetByID(context.Background(), task.ID)
	require.NoError(t, err)
	require.Equal(t, model.JobTaskStatusCancelled, stored.Status)
}

type fakeRepository struct {
	mu             sync.Mutex
	tasksByID      map[uuid.UUID]model.JobTask
	eventsByTaskID map[uuid.UUID][]model.JobTaskEvent
	nextEventID    uint64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		tasksByID:      make(map[uuid.UUID]model.JobTask),
		eventsByTaskID: make(map[uuid.UUID][]model.JobTaskEvent),
		nextEventID:    1,
	}
}

func (r *fakeRepository) seedTask(task model.JobTask) model.JobTask {
	r.mu.Lock()
	defer r.mu.Unlock()

	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	now := time.Now().UTC()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = now
	}
	r.tasksByID[task.ID] = task
	return task
}

func (r *fakeRepository) FindByIdempotencyKey(_ context.Context, requestedBy uuid.UUID, taskType, key string) (*model.JobTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, task := range r.tasksByID {
		if task.RequestedBy == requestedBy && task.TaskType == taskType && task.IdempotencyKey == key {
			cloned := task
			return &cloned, nil
		}
	}
	return nil, nil
}

func (r *fakeRepository) Create(_ context.Context, task *model.JobTask) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	now := time.Now().UTC()
	task.CreatedAt = now
	task.UpdatedAt = now
	r.tasksByID[task.ID] = *task
	return nil
}

func (r *fakeRepository) GetByID(_ context.Context, taskID uuid.UUID) (*model.JobTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasksByID[taskID]
	if !ok {
		return nil, ErrTaskNotFound
	}
	cloned := task
	return &cloned, nil
}

func (r *fakeRepository) UpdateStatus(_ context.Context, taskID uuid.UUID, status model.JobTaskStatus, errorMessage string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasksByID[taskID]
	if !ok {
		return ErrTaskNotFound
	}
	task.Status = status
	task.ErrorMessage = errorMessage
	task.UpdatedAt = time.Now().UTC()
	if status == model.JobTaskStatusRunning && task.StartedAt == nil {
		now := time.Now().UTC()
		task.StartedAt = &now
	}
	if isTerminalStatus(status) {
		now := time.Now().UTC()
		task.FinishedAt = &now
	}
	r.tasksByID[taskID] = task
	return nil
}

func (r *fakeRepository) UpdateResult(_ context.Context, taskID uuid.UUID, result datatypes.JSON) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasksByID[taskID]
	if !ok {
		return ErrTaskNotFound
	}
	task.ResultJSON = append(datatypes.JSON(nil), result...)
	task.UpdatedAt = time.Now().UTC()
	r.tasksByID[taskID] = task
	return nil
}

func (r *fakeRepository) AppendEvent(_ context.Context, event *model.JobTaskEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tasksByID[event.TaskID]; !ok {
		return ErrTaskNotFound
	}
	event.ID = r.nextEventID
	r.nextEventID++
	event.CreatedAt = time.Now().UTC()
	r.eventsByTaskID[event.TaskID] = append(r.eventsByTaskID[event.TaskID], *event)
	return nil
}

func (r *fakeRepository) ListEventsAfter(_ context.Context, taskID uuid.UUID, afterID uint64, limit int) ([]model.JobTaskEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	events := r.eventsByTaskID[taskID]
	filtered := make([]model.JobTaskEvent, 0, len(events))
	for _, event := range events {
		if event.ID > afterID {
			filtered = append(filtered, event)
		}
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

type fakePublisher struct {
	mu            sync.Mutex
	messages      []GenerationRequestedMessage
	parseMessages []ParseRequestedMessage
	exportMessages []ExportRequestedMessage
	err           error
	parseErr      error
	exportErr     error
}

func (p *fakePublisher) PublishGenerationRequested(_ context.Context, message GenerationRequestedMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.err != nil {
		return p.err
	}
	p.messages = append(p.messages, message)
	return nil
}

func (p *fakePublisher) PublishParseRequested(_ context.Context, message ParseRequestedMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.parseErr != nil {
		return p.parseErr
	}
	p.parseMessages = append(p.parseMessages, message)
	return nil
}

func (p *fakePublisher) PublishExportRequested(_ context.Context, message ExportRequestedMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.exportErr != nil {
		return p.exportErr
	}
	p.exportMessages = append(p.exportMessages, message)
	return nil
}

var _ Repository = (*fakeRepository)(nil)
var _ Publisher = (*fakePublisher)(nil)

func TestService_CreateGenerationTask_PublishesMessage(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{}
	service := NewService(repo, publisher)

	task, err := service.CreateGenerationTask(context.Background(), CreateGenerationTaskInput{
		RequestedBy:    uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		TaskSubtype:    "generate_single",
		IdempotencyKey: "single:1",
		Payload:        []byte(`{"source_content":"hello"}`),
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, task.ID)
	require.Len(t, publisher.messages, 1)
	require.Equal(t, task.ID, publisher.messages[0].TaskID)
}

func TestService_CreateParseTask_PublishesMessage(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{}
	service := NewService(repo, publisher)

	task, err := service.CreateParseTask(context.Background(), CreateParseTaskInput{
		RequestedBy:    uuid.MustParse("77777777-7777-7777-7777-777777777777"),
		TaskSubtype:    "parse_archive",
		IdempotencyKey: "parse:archive:1",
		Payload:        []byte(`{"filename":"courseware.zip","size_bytes":123}`),
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, task.ID)
	require.Equal(t, "parse", task.TaskType)
	require.Len(t, publisher.parseMessages, 1)
	require.Equal(t, task.ID, publisher.parseMessages[0].TaskID)
	require.Equal(t, "parse_archive", publisher.parseMessages[0].Kind)
}

func TestService_CreateExportTask_PublishesMessage(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{}
	service := NewService(repo, publisher)

	task, err := service.CreateExportTask(context.Background(), CreateExportTaskInput{
		RequestedBy:    uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		TaskSubtype:    ExportTaskSubtypePDF,
		IdempotencyKey: "export-pdf:series-1",
		Payload:        []byte(`{"blog_id":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}`),
	})
	require.NoError(t, err)
	require.Equal(t, taskTypeExport, task.TaskType)
	require.Len(t, publisher.exportMessages, 1)
	require.Equal(t, task.ID, publisher.exportMessages[0].TaskID)
	require.Equal(t, ExportTaskSubtypePDF, publisher.exportMessages[0].Kind)
}

func TestService_CreateGenerationTask_PropagatesPublishError(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{err: errors.New("publish failed")}
	service := NewService(repo, publisher)

	_, err := service.CreateGenerationTask(context.Background(), CreateGenerationTaskInput{
		RequestedBy:    uuid.MustParse("66666666-6666-6666-6666-666666666666"),
		TaskSubtype:    "generate_single",
		IdempotencyKey: "single:2",
		Payload:        []byte(`{"source_content":"hello"}`),
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "publish failed")
}

func TestService_CreateParseTask_PropagatesPublishError(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{parseErr: errors.New("publish parse failed")}
	service := NewService(repo, publisher)

	_, err := service.CreateParseTask(context.Background(), CreateParseTaskInput{
		RequestedBy:    uuid.MustParse("88888888-8888-8888-8888-888888888888"),
		TaskSubtype:    "parse_file",
		IdempotencyKey: "parse:file:2",
		Payload:        []byte(`{"filename":"lesson.md"}`),
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "publish parse failed")
}

func TestService_ListStreamEvents_ReturnsDoneWhenTaskFinished(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, nil)
	task := repo.seedTask(model.JobTask{
		ID:     uuid.MustParse("77777777-7777-7777-7777-777777777777"),
		Status: model.JobTaskStatusSucceeded,
	})
	require.NoError(t, repo.AppendEvent(context.Background(), &model.JobTaskEvent{
		TaskID:    task.ID,
		EventType: "done",
		Status:    model.JobTaskStatusSucceeded,
		Payload:   datatypes.JSON([]byte(`{"done":true}`)),
	}))

	events, done, err := service.ListStreamEvents(context.Background(), task.ID, 0)
	require.NoError(t, err)
	require.True(t, done)
	require.Len(t, events, 1)
}

func TestService_MarkRunning_UpdatesTaskStatus(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, nil)
	task := repo.seedTask(model.JobTask{
		ID:     uuid.MustParse("88888888-8888-8888-8888-888888888888"),
		Status: model.JobTaskStatusQueued,
	})

	err := service.MarkRunning(context.Background(), task.ID)
	require.NoError(t, err)

	stored, err := repo.GetByID(context.Background(), task.ID)
	require.NoError(t, err)
	require.Equal(t, model.JobTaskStatusRunning, stored.Status)
	require.NotNil(t, stored.StartedAt)
}

func TestService_MarkFailed_AppendsErrorEventAndStoresMessage(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, nil)
	task := repo.seedTask(model.JobTask{
		ID:     uuid.MustParse("99999999-9999-9999-9999-999999999999"),
		Status: model.JobTaskStatusRunning,
	})

	err := service.MarkFailed(context.Background(), task.ID, "generation failed")
	require.NoError(t, err)

	stored, err := repo.GetByID(context.Background(), task.ID)
	require.NoError(t, err)
	require.Equal(t, model.JobTaskStatusFailed, stored.Status)
	require.Equal(t, "generation failed", stored.ErrorMessage)
	require.Len(t, repo.eventsByTaskID[task.ID], 1)
	require.Equal(t, "error", repo.eventsByTaskID[task.ID][0].EventType)
	require.Contains(t, string(repo.eventsByTaskID[task.ID][0].Payload), `"generation failed"`)
}

func TestService_MarkSucceeded_PersistsResult(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, nil)
	task := repo.seedTask(model.JobTask{
		ID:     uuid.MustParse("12121212-1212-1212-1212-121212121212"),
		Status: model.JobTaskStatusStreaming,
	})

	err := service.MarkSucceeded(context.Background(), task.ID, []byte(`{"done":true}`))
	require.NoError(t, err)

	stored, err := repo.GetByID(context.Background(), task.ID)
	require.NoError(t, err)
	require.Equal(t, model.JobTaskStatusSucceeded, stored.Status)
	require.JSONEq(t, `{"done":true}`, string(stored.ResultJSON))
}

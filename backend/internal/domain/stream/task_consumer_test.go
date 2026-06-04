package stream

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/mq"
	"inkwords-backend/internal/model"
)

func TestTaskConsumer_RunGenerateSingle_AppendsChunkAndCompletes(t *testing.T) {
	taskService := &fakeTaskService{}
	streamService := &fakeStreamService{
		generateFunc: func(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error) {
			require.Equal(t, "hello world", req.SourceContent)
			chunkChan <- "hello"
			close(chunkChan)
			close(errChan)
		},
		buildGenerateSingleTaskResultFunc: func(ctx context.Context, req GenerateRequest, content string) ([]byte, error) {
			require.Equal(t, "hello", content)
			return []byte(`{"result_version":1,"task_type":"generation","task_subtype":"generate_single","persistence_mode":"task_only","final_status":"succeeded","usage":{"estimated_tokens":6},"payload":{"title":"A","content":"hello","source_type":"file","word_count":1,"tech_stacks":["Go"]}}`), nil
		},
	}

	consumer := NewTaskConsumer(taskService, streamService)
	message := mq.GenerationRequestedMessage{
		TaskID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Kind:   "generate_single",
		UserID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Payload: json.RawMessage(`{
			"source_type":"file",
			"source_content":"hello world",
			"scenario_mode":"ebook_interpretation"
		}`),
	}

	err := consumer.HandleGenerationRequested(context.Background(), message)
	require.NoError(t, err)
	require.True(t, taskService.markRunningCalled)
	require.Equal(t, model.JobTaskStatusSucceeded, taskService.lastStatus)
	require.Len(t, taskService.appendedPayloads, 1)
	require.Contains(t, string(taskService.appendedPayloads[0]), `"hello"`)
	require.JSONEq(t, `{"result_version":1,"task_type":"generation","task_subtype":"generate_single","persistence_mode":"task_only","final_status":"succeeded","usage":{"estimated_tokens":6},"payload":{"title":"A","content":"hello","source_type":"file","word_count":1,"tech_stacks":["Go"]}}`, string(taskService.lastResult))
}

func TestTaskConsumer_InvalidPayload_MarksTaskFailed(t *testing.T) {
	taskService := &fakeTaskService{}
	consumer := NewTaskConsumer(taskService, &fakeStreamService{})

	err := consumer.HandleGenerationRequested(context.Background(), mq.GenerationRequestedMessage{
		TaskID:  uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		Kind:    "generate_single",
		UserID:  uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		Payload: json.RawMessage(`{"source_content":`),
	})

	require.NoError(t, err)
	require.Equal(t, model.JobTaskStatusFailed, taskService.lastStatus)
	require.Equal(t, "invalid generation payload", taskService.lastErrorMessage)
}

func TestTaskConsumer_RunContinue_UsesLegacyContinuationServiceBehindTaskEnvelope(t *testing.T) {
	taskService := &fakeTaskService{}
	streamService := &fakeStreamService{
		continueFunc: func(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error) {
			require.Equal(t, uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"), blogID)
			chunkChan <- "续写片段"
			close(chunkChan)
			close(errChan)
		},
		buildContinueTaskResultFunc: func(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, appendedContent string) ([]byte, error) {
			require.Equal(t, uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"), blogID)
			require.Equal(t, "续写片段", appendedContent)
			return []byte(`{"result_version":1,"task_type":"generation","task_subtype":"continue","persistence_mode":"task_only","final_status":"succeeded","usage":{"estimated_tokens":8},"payload":{"blog_id":"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee","appended_content":"续写片段","final_content":"旧内容续写片段"}}`), nil
		},
	}

	consumer := NewTaskConsumer(taskService, streamService)
	message := mq.GenerationRequestedMessage{
		TaskID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab"),
		Kind:   "continue",
		UserID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Payload: json.RawMessage(`{
			"blog_id":"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
		}`),
	}

	err := consumer.HandleGenerationRequested(context.Background(), message)
	require.NoError(t, err)
	require.Equal(t, model.JobTaskStatusSucceeded, taskService.lastStatus)
	require.Len(t, taskService.appendedPayloads, 1)
	require.Contains(t, string(taskService.appendedPayloads[0]), `"续写片段"`)
	require.JSONEq(t, `{"result_version":1,"task_type":"generation","task_subtype":"continue","persistence_mode":"task_only","final_status":"succeeded","usage":{"estimated_tokens":8},"payload":{"blog_id":"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee","appended_content":"续写片段","final_content":"旧内容续写片段"}}`, string(taskService.lastResult))
}

func TestTaskConsumer_RunPolish_UsesLegacyPolishServiceBehindTaskEnvelope(t *testing.T) {
	taskService := &fakeTaskService{}
	streamService := &fakeStreamService{
		polishFunc: func(ctx context.Context, req PolishRequest, chunkChan chan<- string, errChan chan<- error) {
			require.Equal(t, "旧标题", req.Title)
			require.Equal(t, "原正文", req.Content)
			chunkChan <- "润色片段"
			close(chunkChan)
			close(errChan)
		},
	}

	consumer := NewTaskConsumer(taskService, streamService)
	message := mq.GenerationRequestedMessage{
		TaskID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaac"),
		Kind:   "polish",
		UserID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Payload: json.RawMessage(`{
			"blog_id":"ffffffff-ffff-ffff-ffff-ffffffffffff",
			"title":"旧标题",
			"content":"原正文"
		}`),
	}

	err := consumer.HandleGenerationRequested(context.Background(), message)
	require.NoError(t, err)
	require.Equal(t, model.JobTaskStatusSucceeded, taskService.lastStatus)
	require.Len(t, taskService.appendedPayloads, 1)
	require.Contains(t, string(taskService.appendedPayloads[0]), `"润色片段"`)
}

func TestHandleGenerationRequested_MarkSucceededWithStructuredResult(t *testing.T) {
	tasks := &fakeTaskService{}
	streams := &fakeStreamService{
		generateFunc: func(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error) {
			chunkChan <- "B"
			close(chunkChan)
			close(errChan)
		},
		buildGenerateSingleTaskResultFunc: func(ctx context.Context, req GenerateRequest, content string) ([]byte, error) {
			require.Equal(t, "file", req.SourceType)
			require.Equal(t, "B", content)
			return []byte(`{"result_version":1,"task_type":"generation","task_subtype":"generate_single","persistence_mode":"task_only","final_status":"succeeded","usage":{"estimated_tokens":6},"payload":{"title":"A","content":"B","source_type":"file","word_count":1,"tech_stacks":["Go"]}}`), nil
		},
	}
	consumer := NewTaskConsumer(tasks, streams)

	err := consumer.HandleGenerationRequested(context.Background(), mq.GenerationRequestedMessage{
		TaskID:  uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Kind:    "generate_single",
		UserID:  uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Payload: []byte(`{"source_type":"file","source_content":"hello"}`),
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"result_version":1,"task_type":"generation","task_subtype":"generate_single","persistence_mode":"task_only","final_status":"succeeded","usage":{"estimated_tokens":6},"payload":{"title":"A","content":"B","source_type":"file","word_count":1,"tech_stacks":["Go"]}}`, string(tasks.lastResult))
}

type fakeTaskService struct {
	markRunningCalled bool
	appendCalls       []taskdomain.AppendEventInput
	appendedPayloads  [][]byte
	lastStatus        model.JobTaskStatus
	lastResult        []byte
	lastErrorMessage  string
	cancelled         bool
}

func (f *fakeTaskService) MarkRunning(_ context.Context, _ uuid.UUID) error {
	f.markRunningCalled = true
	f.lastStatus = model.JobTaskStatusRunning
	return nil
}

func (f *fakeTaskService) AppendEvent(_ context.Context, _ uuid.UUID, input taskdomain.AppendEventInput) error {
	f.appendCalls = append(f.appendCalls, input)
	f.appendedPayloads = append(f.appendedPayloads, append([]byte(nil), input.Payload...))
	f.lastStatus = input.Status
	return nil
}

func (f *fakeTaskService) MarkSucceeded(_ context.Context, _ uuid.UUID, result []byte) error {
	f.lastStatus = model.JobTaskStatusSucceeded
	f.lastResult = append([]byte(nil), result...)
	return nil
}

func (f *fakeTaskService) MarkFailed(_ context.Context, _ uuid.UUID, message string) error {
	f.lastStatus = model.JobTaskStatusFailed
	f.lastErrorMessage = message
	return nil
}

func (f *fakeTaskService) IsCancelled(_ context.Context, _ uuid.UUID) (bool, error) {
	return f.cancelled, nil
}

type fakeStreamService struct {
	generateFunc                      func(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error)
	continueFunc                      func(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error)
	polishFunc                        func(ctx context.Context, req PolishRequest, chunkChan chan<- string, errChan chan<- error)
	buildGenerateSingleTaskResultFunc func(ctx context.Context, req GenerateRequest, content string) ([]byte, error)
	buildContinueTaskResultFunc       func(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, appendedContent string) ([]byte, error)
}

func (f *fakeStreamService) Generate(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error) {
	if f.generateFunc != nil {
		f.generateFunc(ctx, userID, req, chunkChan, errChan)
		return
	}
	close(chunkChan)
	close(errChan)
}

func (f *fakeStreamService) Continue(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error) {
	if f.continueFunc != nil {
		f.continueFunc(ctx, userID, blogID, chunkChan, errChan)
		return
	}
	close(chunkChan)
	close(errChan)
}

func (f *fakeStreamService) Polish(ctx context.Context, req PolishRequest, chunkChan chan<- string, errChan chan<- error) {
	if f.polishFunc != nil {
		f.polishFunc(ctx, req, chunkChan, errChan)
		return
	}
	close(chunkChan)
	close(errChan)
}

func (f *fakeStreamService) BuildGenerateSingleTaskResult(ctx context.Context, req GenerateRequest, content string) ([]byte, error) {
	if f.buildGenerateSingleTaskResultFunc != nil {
		return f.buildGenerateSingleTaskResultFunc(ctx, req, content)
	}
	return nil, nil
}

func (f *fakeStreamService) BuildContinueTaskResult(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, appendedContent string) ([]byte, error) {
	if f.buildContinueTaskResultFunc != nil {
		return f.buildContinueTaskResultFunc(ctx, userID, blogID, appendedContent)
	}
	return nil, nil
}

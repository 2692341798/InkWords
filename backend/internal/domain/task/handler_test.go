package task

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"inkwords-backend/internal/model"
)

func TestHandler_CreateGenerationTask_ReturnsAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &fakeTaskHandlerService{
		createTaskResult: model.JobTask{
			ID:     uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			Status: model.JobTaskStatusQueued,
		},
	}
	handler := NewHandler(service)

	router := gin.New()
	router.POST("/api/v1/tasks/generation", func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		handler.CreateGenerationTask(c)
	})

	body := `{"kind":"generate_single","payload":{"source_content":"hello","scenario_mode":"ebook_interpretation"},"idempotency_key":"gen:1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/generation", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusAccepted, resp.Code)
	require.Contains(t, resp.Body.String(), `"task_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"`)
	require.Contains(t, resp.Body.String(), `"/api/v1/tasks/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/stream"`)
	require.Equal(t, "generate_single", service.lastCreateInput.TaskSubtype)
}

func TestHandler_GetTask_ReturnsTaskSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &fakeTaskHandlerService{
		getTaskResult: model.JobTask{
			ID:          uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			TaskType:    "generation",
			TaskSubtype: "generate_series",
			Status:      model.JobTaskStatusStreaming,
			ResultJSON:  datatypes.JSON([]byte(`{"done":false}`)),
		},
	}
	handler := NewHandler(service)

	router := gin.New()
	router.GET("/api/v1/tasks/:id", func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		handler.GetTask(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Contains(t, resp.Body.String(), `"task_subtype":"generate_series"`)
	require.Contains(t, resp.Body.String(), `"status":"streaming"`)
}

func TestHandler_CancelTask_ReturnsAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &fakeTaskHandlerService{}
	handler := NewHandler(service)

	router := gin.New()
	router.POST("/api/v1/tasks/:id/cancel", func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		handler.CancelTask(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/cccccccc-cccc-cccc-cccc-cccccccccccc/cancel", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusAccepted, resp.Code)
	require.Equal(t, uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"), service.lastCancelledTaskID)
}

func TestHandler_StreamTask_StreamsEventsUntilDone(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &fakeTaskHandlerService{
		getTaskResult: model.JobTask{
			ID:          uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
			RequestedBy: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Status:      model.JobTaskStatusStreaming,
		},
		streamEvents: [][]model.JobTaskEvent{
			{
				{
					ID:        1,
					TaskID:    uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
					EventType: "chunk",
					Status:    model.JobTaskStatusStreaming,
					Payload:   datatypes.JSON([]byte(`{"content":"hello"}`)),
				},
			},
		},
		streamDoneAfterCall: 2,
	}
	handler := NewHandler(service)
	handler.pollInterval = time.Millisecond

	router := gin.New()
	router.GET("/api/v1/tasks/:id/stream", func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		handler.StreamTask(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/dddddddd-dddd-dddd-dddd-dddddddddddd/stream", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Contains(t, resp.Body.String(), "event:chunk")
	require.Contains(t, resp.Body.String(), "event:done")
	require.Equal(t, uint64(1), service.lastAfterID)
	require.Equal(t, 2, service.streamCalls)
}

type fakeTaskHandlerService struct {
	createTaskResult    model.JobTask
	getTaskResult       model.JobTask
	streamEvents        [][]model.JobTaskEvent
	streamDoneAfterCall int

	lastCreateInput     CreateGenerationTaskInput
	lastGetTaskID       uuid.UUID
	lastCancelledTaskID uuid.UUID
	lastAfterID         uint64
	streamCalls         int
}

func (f *fakeTaskHandlerService) CreateGenerationTask(_ context.Context, input CreateGenerationTaskInput) (model.JobTask, error) {
	f.lastCreateInput = input
	return f.createTaskResult, nil
}

func (f *fakeTaskHandlerService) GetTask(_ context.Context, taskID uuid.UUID, _ uuid.UUID) (model.JobTask, error) {
	f.lastGetTaskID = taskID
	return f.getTaskResult, nil
}

func (f *fakeTaskHandlerService) CancelTask(_ context.Context, taskID uuid.UUID, _ uuid.UUID) error {
	f.lastCancelledTaskID = taskID
	return nil
}

func (f *fakeTaskHandlerService) ListStreamEvents(_ context.Context, _ uuid.UUID, afterID uint64) ([]model.JobTaskEvent, bool, error) {
	f.lastAfterID = afterID
	f.streamCalls++
	index := f.streamCalls - 1
	if index < len(f.streamEvents) {
		return f.streamEvents[index], f.streamDoneAfterCall == f.streamCalls, nil
	}
	return nil, f.streamDoneAfterCall > 0 && f.streamCalls >= f.streamDoneAfterCall, nil
}

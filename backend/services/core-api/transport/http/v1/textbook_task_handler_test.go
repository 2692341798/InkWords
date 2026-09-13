package v1

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/services/core-api/app/textbookgeneration"
	coretask "inkwords-backend/services/core-api/domain/task"
	"inkwords-backend/shared/kernel/httpx"
)

type fakeTextbookTaskAccess struct {
	workspaceID        uuid.UUID
	taskID             uuid.UUID
	snapshot           textbookgeneration.TaskSnapshot
	err                error
	confirmedInputHash string
}

func (access *fakeTextbookTaskAccess) Get(_ context.Context, workspaceID, taskID uuid.UUID) (textbookgeneration.TaskSnapshot, error) {
	access.workspaceID = workspaceID
	access.taskID = taskID
	return access.snapshot, access.err
}

func (access *fakeTextbookTaskAccess) Retry(_ context.Context, workspaceID, taskID uuid.UUID, confirmedInputHash string) (textbookgeneration.TaskSnapshot, error) {
	access.workspaceID = workspaceID
	access.taskID = taskID
	access.confirmedInputHash = confirmedInputHash
	return access.snapshot, access.err
}

func TestTextbookTaskHandlerForwardsExplicitGenerationRetryConfirmation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceID, taskID := uuid.New(), uuid.New()
	access := &fakeTextbookTaskAccess{snapshot: textbookgeneration.TaskSnapshot{ID: taskID, TaskType: "generation", TaskSubtype: "textbook_sample_generate", Status: coretask.JobTaskStatusQueued}}
	handler := NewTextbookTaskHandler(access)
	router := gin.New()
	router.POST("/tasks/:taskID/retry", httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspaceID, nil }), handler.RetryTask)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/tasks/"+taskID.String()+"/retry", bytes.NewBufferString(`{"confirmed_input_hash":"sha256:confirmed"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.Equal(t, "sha256:confirmed", access.confirmedInputHash)
}

func TestTextbookTaskHandlerUsesLocalWorkspaceIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceID := uuid.New()
	taskID := uuid.New()
	access := &fakeTextbookTaskAccess{snapshot: textbookgeneration.TaskSnapshot{ID: taskID, TaskType: "generation", TaskSubtype: "textbook_sample_generate", Status: coretask.JobTaskStatusRunning}}
	handler := NewTextbookTaskHandler(access)
	router := gin.New()
	router.GET("/tasks/:taskID", httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspaceID, nil }), handler.GetTask)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/tasks/"+taskID.String(), nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, workspaceID, access.workspaceID)
	require.Equal(t, taskID, access.taskID)
	require.JSONEq(t, `{"id":"`+taskID.String()+`","task_type":"generation","task_subtype":"textbook_sample_generate","status":"running","retry_count":0,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}`, recorder.Body.String())
}

func TestTextbookTaskHandlerDoesNotExposeLegacyTaskScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceID := uuid.New()
	taskID := uuid.New()
	access := &fakeTextbookTaskAccess{err: textbookgeneration.ErrTaskOutsideTextbookScope}
	handler := NewTextbookTaskHandler(access)
	router := gin.New()
	router.POST("/tasks/:taskID/retry", httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspaceID, nil }), handler.RetryTask)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/tasks/"+taskID.String()+"/retry", nil))

	require.Equal(t, http.StatusForbidden, recorder.Code)
}

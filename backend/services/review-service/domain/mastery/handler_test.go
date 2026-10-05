package mastery

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/shared/kernel/httpx"
)

type conflictingAttemptService struct {
	applicationService
	err   error
	input Attempt
}

func (service *conflictingAttemptService) RecordAttempt(_ context.Context, _, _ uuid.UUID, input Attempt) (DueTask, error) {
	service.input = input
	return DueTask{}, service.err
}

func TestAttemptHandlerPreservesBindingAndExplainsConflicts(t *testing.T) {
	for _, example := range []struct {
		err     error
		message string
	}{{ErrPracticeNotDue, "延迟复习时间"}, {ErrPracticeMismatch, "批准题目不匹配"}, {ErrMasteryHistoryChanged, "学习记录已更新"}, {ErrPracticeSessionClosed, "已记录"}} {
		service := &conflictingAttemptService{err: example.err}
		handler := NewHandler(service)
		router := gin.New()
		router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return uuid.New(), nil }))
		router.POST("/objectives/:id/attempts", handler.RecordAttempt)
		request := httptest.NewRequest(http.MethodPost, "/objectives/"+uuid.New().String()+"/attempts", bytes.NewBufferString(`{"skill":"retain","answer":"我的回答","practice_task_id":"task-retain","practice_content_hash":"sha256:fixture","confidence":3,"attempted_at":"2026-09-01T00:00:00Z"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusConflict, response.Code)
		require.Contains(t, response.Body.String(), example.message)
		require.Equal(t, "task-retain", service.input.PracticeTaskID)
		require.Equal(t, "sha256:fixture", service.input.PracticeContentHash)
	}
}

func TestHandlerCreatesObjectiveRecordsAttemptAndShowsDueOnlyForOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newMemoryStore()
	service := NewService(store)
	userID := uuid.New()
	handler := NewHandler(service)
	handler.now = func() time.Time { return time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC) }
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return userID, nil }))
	router.POST("/objectives", handler.CreateObjective)
	router.POST("/objectives/:id/attempts", handler.RecordAttempt)
	router.GET("/objectives/:id", handler.Workspace)
	router.GET("/due", handler.Due)

	create := httptest.NewRequest(http.MethodPost, "/objectives", bytes.NewBufferString(`{"chapter_id":"chapter-1","title":"解释路由登记","behavior":"独立说明路由如何登记","skills":["explain","transfer","diagnose","retain"],"rubric":["说明方法与路径"],"key_points":["路由树"],"evidence_refs":["evidence:gin-routing"]}`))
	create.Header.Set("Content-Type", "application/json")
	created := httptest.NewRecorder()
	router.ServeHTTP(created, create)
	require.Equal(t, http.StatusOK, created.Code)
	var createPayload struct {
		Data Objective `json:"data"`
	}
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &createPayload))
	require.NotEqual(t, uuid.Nil, createPayload.Data.ID)

	attempt := httptest.NewRequest(http.MethodPost, "/objectives/"+createPayload.Data.ID.String()+"/attempts", bytes.NewBufferString(`{"skill":"explain","answer":"先按方法选择路由树，再查找路径。","correct":false,"independent":false,"hint_count":1,"took_millis":1000,"confidence":2,"error_kinds":["missing_mechanism"],"attempted_at":"2026-01-01T09:00:00Z"}`))
	attempt.Header.Set("Content-Type", "application/json")
	attempted := httptest.NewRecorder()
	router.ServeHTTP(attempted, attempt)
	require.Equal(t, http.StatusOK, attempted.Code)
	var attemptPayload struct {
		Data struct {
			Skill  Skill  `json:"skill"`
			DueAt  string `json:"due_at"`
			Reason string `json:"reason"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(attempted.Body.Bytes(), &attemptPayload))
	require.Equal(t, Explain, attemptPayload.Data.Skill)
	require.NotEmpty(t, attemptPayload.Data.DueAt)
	require.NotEmpty(t, attemptPayload.Data.Reason)

	due := httptest.NewRequest(http.MethodGet, "/due", nil)
	dueResponse := httptest.NewRecorder()
	router.ServeHTTP(dueResponse, due)
	require.Equal(t, http.StatusOK, dueResponse.Code)
	var duePayload struct {
		Data struct {
			Tasks []DueObjective `json:"tasks"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(dueResponse.Body.Bytes(), &duePayload))
	require.Len(t, duePayload.Data.Tasks, 1)
	require.Equal(t, createPayload.Data.ID, duePayload.Data.Tasks[0].ObjectiveID)

	workspaceResponse := httptest.NewRecorder()
	router.ServeHTTP(workspaceResponse, httptest.NewRequest(http.MethodGet, "/objectives/"+createPayload.Data.ID.String(), nil))
	require.Equal(t, http.StatusOK, workspaceResponse.Code)
	require.Contains(t, workspaceResponse.Body.String(), "先按方法选择路由树，再查找路径。")
	userID = uuid.New()
	forbidden := httptest.NewRecorder()
	router.ServeHTTP(forbidden, httptest.NewRequest(http.MethodGet, "/objectives/"+createPayload.Data.ID.String(), nil))
	require.Equal(t, http.StatusNotFound, forbidden.Code)
	require.NotContains(t, forbidden.Body.String(), "先按方法")
}

func TestHandlerSerializesAnEmptyDueQueueAsAnArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(NewService(newMemoryStore()))
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return uuid.New(), nil }))
	router.GET("/due", handler.Due)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/due", nil))
	require.Equal(t, http.StatusOK, response.Code)

	var payload struct {
		Data struct {
			Tasks json.RawMessage `json:"tasks"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	require.JSONEq(t, "[]", string(payload.Data.Tasks))
}

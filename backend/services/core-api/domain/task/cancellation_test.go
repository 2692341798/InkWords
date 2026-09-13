package task

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"inkwords-backend/shared/kernel/httpx"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCancellationCannotOverwriteCompletionOrBeResurrectedByWorker(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}))
	repo := NewGormRepository(db)
	for _, status := range []JobTaskStatus{JobTaskStatusSucceeded, JobTaskStatusFailed, JobTaskStatusCancelled} {
		row := JobTask{ID: uuid.New(), TaskType: "verification", TaskSubtype: "textbook_teaching_artifact_verify", Status: status, ResultJSON: []byte(`{"original":true}`)}
		require.NoError(t, db.Create(&row).Error)
		require.NoError(t, repo.UpdateStatus(t.Context(), row.ID, JobTaskStatusCancelled, ""))
		if status == JobTaskStatusCancelled {
			require.NoError(t, repo.UpdateStatus(t.Context(), row.ID, JobTaskStatusRunning, ""))
			require.NoError(t, repo.UpdateResult(t.Context(), row.ID, []byte(`{"late":true}`)))
			require.NoError(t, repo.UpdateStatus(t.Context(), row.ID, JobTaskStatusSucceeded, ""))
		}
		after, err := repo.GetByID(t.Context(), row.ID)
		require.NoError(t, err)
		require.Equal(t, status, after.Status)
		require.JSONEq(t, `{"original":true}`, string(after.ResultJSON))
	}
	require.ErrorIs(t, repo.UpdateStatus(t.Context(), uuid.New(), JobTaskStatusCancelled, ""), ErrTaskNotFound)
}

func TestCancellationResponseReportsActualTerminalState(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &JobTaskEvent{}))
	workspace := uuid.New()
	service := NewService(NewGormRepository(db), &verificationRetryPublisher{}, nil)
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspace, nil }))
	router.POST("/tasks/:id/cancel", NewHandler(service, "").CancelTask)
	for _, status := range []JobTaskStatus{JobTaskStatusRunning, JobTaskStatusSucceeded} {
		task := JobTask{ID: uuid.New(), WorkspaceID: &workspace, TaskType: "verification", TaskSubtype: "textbook_teaching_artifact_verify", Status: status}
		require.NoError(t, db.Create(&task).Error)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/tasks/"+task.ID.String()+"/cancel", nil))
		require.Equal(t, http.StatusAccepted, recorder.Code)
		var response struct {
			Status JobTaskStatus `json:"status"`
		}
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		if status == JobTaskStatusRunning {
			require.Equal(t, JobTaskStatusCancelled, response.Status)
		} else {
			require.Equal(t, JobTaskStatusSucceeded, response.Status)
		}
	}
}

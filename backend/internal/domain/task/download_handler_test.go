package task

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"inkwords-backend/internal/model"
)

func TestHandler_DownloadTask_ServesPDF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	taskID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	token := "exp_pdf_" + taskID.String()
	require.NoError(t, os.WriteFile(filepath.Join(dir, token+".pdf"), []byte("pdf"), 0o644))

	service := &fakeTaskHandlerService{
		getTaskResult: model.JobTask{
			ID:       taskID,
			TaskType: taskTypeExport,
			Status:   model.JobTaskStatusSucceeded,
			ResultJSON: datatypes.JSON([]byte(`{
				"file_token":"` + token + `",
				"filename":"系列标题.pdf",
				"content_type":"application/pdf",
				"expires_at":"2099-06-03T13:00:00Z"
			}`)),
		},
	}
	handler := NewHandler(service, dir)

	router := gin.New()
	router.GET("/api/v1/tasks/:id/download", func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		handler.DownloadTask(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID.String()+"/download", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "application/pdf", resp.Header().Get("Content-Type"))
	require.Contains(t, resp.Header().Get("Content-Disposition"), "系列标题.pdf")
	require.Equal(t, "pdf", resp.Body.String())
}

func TestHandler_DownloadTask_RejectsUnfinishedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &fakeTaskHandlerService{
		getTaskResult: model.JobTask{
			ID:       uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			TaskType: taskTypeExport,
			Status:   model.JobTaskStatusRunning,
		},
	}
	handler := NewHandler(service, t.TempDir())

	router := gin.New()
	router.GET("/api/v1/tasks/:id/download", func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		handler.DownloadTask(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb/download", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusConflict, resp.Code)
}

func TestHandler_DownloadTask_ReturnsNotFoundWhenArtifactMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	taskID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	service := &fakeTaskHandlerService{
		getTaskResult: model.JobTask{
			ID:       taskID,
			TaskType: taskTypeExport,
			Status:   model.JobTaskStatusSucceeded,
			ResultJSON: datatypes.JSON([]byte(`{
				"file_token":"exp_pdf_` + taskID.String() + `",
				"filename":"缺失文件.pdf",
				"content_type":"application/pdf",
				"expires_at":"2099-06-03T13:00:00Z"
			}`)),
		},
	}
	handler := NewHandler(service, t.TempDir())

	router := gin.New()
	router.GET("/api/v1/tasks/:id/download", func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		handler.DownloadTask(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID.String()+"/download", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
}

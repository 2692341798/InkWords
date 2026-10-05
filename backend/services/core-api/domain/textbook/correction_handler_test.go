package textbook

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/shared/kernel/httpx"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeCorrectionCreator struct {
	fakeSampleTaskCreator
	calls     int
	workspace uuid.UUID
	source    uuid.UUID
}

func (f *fakeCorrectionCreator) CreateSampleCorrectionTask(_ context.Context, w, p, c, s uuid.UUID, edit sharedtextbook.SampleCorrectionInput) (SampleGenerationTask, error) {
	f.calls++
	f.workspace = w
	f.source = s
	return SampleGenerationTask{ID: uuid.New(), Status: "queued"}, nil
}
func TestCorrectionHandlerRejectsClientResultsAndUsesLocalIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspace, source := uuid.New(), uuid.New()
	creator := &fakeCorrectionCreator{}
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspace, nil }))
	router.POST("/projects/:projectID/chapters/:chapterID/correct", NewHandler(NewService(nil), creator).CorrectSample)
	path := "/projects/" + uuid.NewString() + "/chapters/" + uuid.NewString() + "/correct"
	for _, body := range []string{`{"original_task_id":"` + source.String() + `","edit":{},"quality_report_json":{"passed":true}}`, `{"edit":{"workspace_id":"spoof"}}`, `{} {}`} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		require.Equal(t, http.StatusBadRequest, response.Code)
	}
	require.Zero(t, creator.calls)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"original_task_id":"`+source.String()+`","edit":{"reason":"explicit local correction"}}`)))
	require.Equal(t, http.StatusAccepted, response.Code)
	require.Equal(t, workspace, creator.workspace)
	require.Equal(t, source, creator.source)
	require.Equal(t, 1, creator.calls)
}

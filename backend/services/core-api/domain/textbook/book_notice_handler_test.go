package textbook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/shared/kernel/httpx"
)

func TestBookBuildRejectsClientRevisionSelectionAndUnknownNoticeFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return uuid.New(), nil }))
	router.POST("/projects/:projectID/book-builds", NewHandler(NewService(nil)).CreateBookBuild)
	for _, body := range []string{
		`{"approved_revision_ids":["00000000-0000-4000-8000-000000000001"]}`,
		`{"publication_notices":[{"publication_status":"ready"}]}`,
		`{"publication_notices":[]} {"publication_status":"ready"}`,
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/projects/"+uuid.NewString()+"/book-builds", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Contains(t, recorder.Body.String(), "INVALID_STATE")
	}
}

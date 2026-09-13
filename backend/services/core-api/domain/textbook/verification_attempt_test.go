package textbook

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/shared/kernel/httpx"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type attemptHandlerRecorder struct {
	calls int
	input VerificationAttemptInput
}

func (r *attemptHandlerRecorder) CreateTextbookVerificationTask(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (TextbookVerificationTask, error) {
	r.calls++
	return TextbookVerificationTask{ID: uuid.New(), Status: "cancelled"}, nil
}
func (r *attemptHandlerRecorder) CreateVerificationAttempt(_ context.Context, _, _, _ uuid.UUID, input VerificationAttemptInput) (TextbookVerificationTask, error) {
	r.calls++
	r.input = input
	return TextbookVerificationTask{ID: input.RequestID, Status: "queued"}, nil
}

func TestVerificationAttemptHandlerRejectsExecutableInputAndPreservesRequestIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := &attemptHandlerRecorder{}
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return uuid.New(), nil }))
	router.POST("/chapters/:chapterID/artifacts/:artifactID/verify", NewHandler(nil).WithVerificationTaskCreator(recorder).CreateArtifactVerification)
	path := "/chapters/" + uuid.NewString() + "/artifacts/" + uuid.NewString() + "/verify"
	request, previous := uuid.New(), uuid.New()
	for _, body := range []string{`{"path":"/tmp/untrusted"}`, `{"command":"go test"}`, `{"env":{}}`, `{"request_id":"bad"}`, `{"expected_previous_task_id":"` + previous.String() + `"}`, `{} {}`, `{"request_id":`} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		require.Equal(t, http.StatusBadRequest, response.Code, body)
	}
	require.Zero(t, recorder.calls)
	body := `{"request_id":"` + request.String() + `","expected_previous_task_id":"` + previous.String() + `"}`
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
	require.Equal(t, http.StatusAccepted, response.Code)
	require.Equal(t, request, recorder.input.RequestID)
	require.Equal(t, &previous, recorder.input.ExpectedPreviousTaskID)
	for _, body := range []string{"", `{}`} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		require.Equal(t, http.StatusAccepted, response.Code)
		require.Contains(t, response.Body.String(), "cancelled")
	}
}

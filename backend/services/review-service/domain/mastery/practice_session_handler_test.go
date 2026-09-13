package mastery

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"inkwords-backend/shared/kernel/httpx"
)

type practiceHTTPService struct {
	applicationService
	view      PracticeSessionView
	err       error
	workspace uuid.UUID
	kind      string
	level     int
}

func (s *practiceHTTPService) BeginPracticeSession(_ context.Context, workspaceID, _ uuid.UUID, _ Skill) (PracticeSessionView, error) {
	s.workspace = workspaceID
	return s.view, s.err
}
func (s *practiceHTTPService) LoadPracticeSession(_ context.Context, workspaceID, _, _ uuid.UUID) (PracticeSessionView, error) {
	s.workspace = workspaceID
	return s.view, s.err
}
func (s *practiceHTTPService) RevealPracticeHelp(_ context.Context, workspaceID, _, _ uuid.UUID, kind string, level int) (PracticeSessionView, error) {
	s.workspace, s.kind, s.level = workspaceID, kind, level
	return s.view, s.err
}

func TestPracticeHTTPRoutesUseLocalIdentityAndHideInternalSubmission(t *testing.T) {
	workspaceID, objectiveID, sessionID := uuid.New(), uuid.New(), uuid.New()
	service := &practiceHTTPService{view: PracticeSessionView{PracticeSession: PracticeSession{ID: sessionID, ObjectiveID: objectiveID, SubmissionHash: "private-submission-hash"}, HintsShown: 1, HintCount: 1}}
	handler := NewHandler(service)
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspaceID, nil }))
	router.POST("/objectives/:id/practice-sessions", handler.BeginPracticeSession)
	router.GET("/objectives/:id/practice-sessions/:sessionID", handler.LoadPracticeSession)
	router.POST("/objectives/:id/practice-sessions/:sessionID/help", handler.RevealPracticeHelp)
	base := "/objectives/" + objectiveID.String() + "/practice-sessions"
	for _, example := range []struct {
		method, path, body string
		err                error
		status             int
	}{
		{http.MethodPost, base, `{"skill":"explain","workspace_id":"spoofed"}`, nil, 200},
		{http.MethodGet, base + "/" + sessionID.String(), "", nil, 200},
		{http.MethodPost, base + "/" + sessionID.String() + "/help", `{"kind":"hint","level":1}`, nil, 200},
		{http.MethodGet, base + "/invalid", "", nil, 400},
		{http.MethodGet, base + "/" + sessionID.String(), "", gorm.ErrRecordNotFound, 404},
		{http.MethodPost, base + "/" + sessionID.String() + "/help", `{"kind":"answer","level":0}`, ErrPracticeSessionClosed, 409},
	} {
		service.err = example.err
		request := httptest.NewRequest(example.method, example.path, bytes.NewBufferString(example.body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, example.status, response.Code)
		require.NotContains(t, response.Body.String(), "private-submission-hash")
	}
	require.Equal(t, workspaceID, service.workspace)
	require.Equal(t, "answer", service.kind)
	require.Zero(t, service.level)
}

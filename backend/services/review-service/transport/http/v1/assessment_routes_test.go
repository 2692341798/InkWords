package v1

import (
	"bytes"
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	app "inkwords-backend/services/review-service/app/masteryassessment"
	"inkwords-backend/shared/kernel/httpx"
)

type assessmentHTTPStub struct {
	assessmentApplication
	owner uuid.UUID
	calls int
	err   error
}

func (s *assessmentHTTPStub) Preview(_ context.Context, owner, _, _ uuid.UUID) (app.Preview, error) {
	s.owner = owner
	return app.Preview{Model: "test-model"}, s.err
}
func (s *assessmentHTTPStub) Start(_ context.Context, owner, _, _ uuid.UUID, _ app.StartInput) (app.Job, error) {
	s.owner = owner
	s.calls++
	return app.Job{Status: "running"}, s.err
}
func (s *assessmentHTTPStub) Read(_ context.Context, owner, _, _ uuid.UUID) (app.Job, error) {
	s.owner = owner
	return app.Job{}, s.err
}
func (s *assessmentHTTPStub) Correct(_ context.Context, owner, _, _ uuid.UUID, _ app.CorrectionInput) (app.Job, error) {
	s.owner = owner
	s.calls++
	return app.Job{}, s.err
}
func (s *assessmentHTTPStub) Apply(_ context.Context, owner, _, _ uuid.UUID, _ app.ApplyInput) (app.Job, error) {
	s.owner = owner
	s.calls++
	return app.Job{}, s.err
}

func TestAssessmentRoutesRequireLocalIdentityAndRejectInjectedFields(t *testing.T) {
	owner, objective, attempt := uuid.New(), uuid.New(), uuid.New()
	currentOwner := owner
	service := &assessmentHTTPStub{}
	router := gin.New()
	RegisterAssessmentRoutes(router, httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return currentOwner, nil }), service)
	prefix := "/api/v1/mastery/objectives/" + objective.String()
	request := func(method, path, body string) (int, string) {
		req := httptest.NewRequest(method, prefix+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response.Code, response.Body.String()
	}
	status, body := request("GET", "/attempts/"+attempt.String()+"/assessment-preview", "")
	require.Equal(t, 200, status)
	require.Contains(t, body, "test-model")
	require.Equal(t, owner, service.owner)
	status, _ = request("POST", "/attempts/"+attempt.String()+"/assessments", `{"answer":"replace saved answer"}`)
	require.Equal(t, 400, status)
	require.Zero(t, service.calls)
	status, _ = request("POST", "/assessments/"+uuid.New().String()+"/apply", `{"correct":true,"due_at":"2099-01-01"}`)
	require.Equal(t, 400, status)
	require.Zero(t, service.calls)
	status, _ = request("POST", "/assessments/"+uuid.New().String()+"/corrections", `{"reviewer_id":"spoofed"}`)
	require.Equal(t, 400, status)
	require.Zero(t, service.calls)
	status, _ = request("POST", "/attempts/"+attempt.String()+"/assessments", `{"request_id":"`+uuid.New().String()+`","expected_input_hash":"sha256:input","expected_request_hash":"sha256:request"}`)
	require.Equal(t, 200, status)
	require.Equal(t, 1, service.calls)
	service.err = app.ErrNotFound
	status, _ = request("GET", "/assessments/"+uuid.New().String(), "")
	require.Equal(t, 404, status)
	service.err = errors.New("private provider response")
	status, body = request("GET", "/assessments/"+uuid.New().String(), "")
	require.Equal(t, 400, status)
	require.NotContains(t, body, "private")
	currentOwner = uuid.Nil
	status, _ = request("GET", "/attempts/"+attempt.String()+"/assessment-preview", "")
	require.Equal(t, 503, status)
	require.Len(t, router.Routes(), 7)
}

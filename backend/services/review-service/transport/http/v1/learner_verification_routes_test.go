package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	app "inkwords-backend/services/review-service/app/masteryverification"
	"inkwords-backend/shared/kernel/httpx"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type fakeLearnerVerificationApp struct {
	preview  app.Preview
	calls    int
	starts   int
	resolves int
}

func (service *fakeLearnerVerificationApp) Preview(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (app.Preview, error) {
	service.calls++
	return service.preview, nil
}
func (service *fakeLearnerVerificationApp) Start(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, app.StartInput) (app.Job, error) {
	service.starts++
	return app.Job{}, nil
}
func (service *fakeLearnerVerificationApp) Latest(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*app.Job, error) {
	return nil, nil
}
func (service *fakeLearnerVerificationApp) Read(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (app.Job, error) {
	return app.Job{}, nil
}
func (service *fakeLearnerVerificationApp) Cancel(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (app.Job, error) {
	return app.Job{}, nil
}
func (service *fakeLearnerVerificationApp) Resolve(context.Context, sharedtextbook.LearnerVerificationReference) (sharedtextbook.LearnerVerificationInput, error) {
	service.resolves++
	return sharedtextbook.LearnerVerificationInput{}, app.ErrNotFound
}

func TestLearnerVerificationPreviewIsReadOnlyAndWorkspaceScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	service := &fakeLearnerVerificationApp{preview: app.Preview{RequiresExplicit: true, Capability: sharedtextbook.LearnerVerificationCapability{Format: sharedtextbook.LearnerVerificationCapabilityFormat, Accepted: true, Profile: sharedtextbook.LearnerGoTestProfile, Reason: "disabled"}}}
	RegisterLearnerVerificationRoutes(router, httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return uuid.New(), nil }), service)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/mastery/objectives/"+uuid.NewString()+"/attempts/"+uuid.NewString()+"/verification-preview", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), `"requires_explicit_start":true`)
	require.Contains(t, response.Body.String(), `"available":false`)
	require.Equal(t, 1, service.calls)

	bad := httptest.NewRecorder()
	router.ServeHTTP(bad, httptest.NewRequest(http.MethodGet, "/api/v1/mastery/objectives/not-an-id/attempts/nope/verification-preview", nil))
	require.Equal(t, 400, bad.Code)
	require.Equal(t, 1, service.calls)
}

func TestLearnerVerificationStartRejectsInjectedExecutionFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	owner, objective, attempt := uuid.New(), uuid.New(), uuid.New()
	service := &fakeLearnerVerificationApp{}
	router := gin.New()
	RegisterLearnerVerificationRoutes(router, httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return owner, nil }), service)
	path := "/api/v1/mastery/objectives/" + objective.String() + "/attempts/" + attempt.String() + "/verifications"
	bad := httptest.NewRecorder()
	router.ServeHTTP(bad, httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"request_id":"`+uuid.NewString()+`","expected_input_hash":"sha256:`+strings.Repeat("a", 64)+`","command":"go test"}`)))
	require.Equal(t, 400, bad.Code)
	require.Zero(t, service.starts)

	body, err := json.Marshal(app.StartInput{RequestID: uuid.New(), ExpectedInputHash: "sha256:" + strings.Repeat("a", 64)})
	require.NoError(t, err)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body)))
	require.Equal(t, 200, response.Code)
	require.Equal(t, 1, service.starts)

	internal := httptest.NewRecorder()
	router.ServeHTTP(internal, httptest.NewRequest(http.MethodPost, "/internal/v1/learner-verification/resolve", bytes.NewBufferString(`{"format":"inkwords.learner-verification-reference.v1"}`)))
	require.Equal(t, 400, internal.Code)
	require.Zero(t, service.resolves)
}

package textbook

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/shared/kernel/httpx"
	"net/http"
	"net/http/httptest"
	"testing"
)

type verificationLookupStub struct {
	creates   int
	reads     int
	workspace uuid.UUID
	task      *TextbookVerificationTask
}

func (s *verificationLookupStub) CreateTextbookVerificationTask(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (TextbookVerificationTask, error) {
	s.creates++
	return TextbookVerificationTask{}, nil
}
func (s *verificationLookupStub) GetTextbookVerificationTask(_ context.Context, w, _, _ uuid.UUID) (*TextbookVerificationTask, error) {
	s.reads++
	s.workspace = w
	return s.task, nil
}
func TestGetArtifactVerificationDoesNotCreateATask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspace := uuid.New()
	stub := &verificationLookupStub{}
	h := NewHandler(NewService(nil), nil).WithVerificationTaskCreator(stub)
	r := gin.New()
	r.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspace, nil }))
	r.GET("/chapters/:chapterID/artifacts/:artifactID", h.GetArtifactVerification)
	path := "/chapters/" + uuid.NewString() + "/artifacts/" + uuid.NewString()
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	require.Equal(t, 200, response.Code)
	require.JSONEq(t, `{"code":0,"data":null}`, response.Body.String())
	require.Equal(t, workspace, stub.workspace)
	require.Equal(t, 1, stub.reads)
	require.Zero(t, stub.creates)
	bad := httptest.NewRecorder()
	r.ServeHTTP(bad, httptest.NewRequest(http.MethodGet, "/chapters/invalid/artifacts/invalid", nil))
	require.Equal(t, 400, bad.Code)
	require.Equal(t, 1, stub.reads)
}

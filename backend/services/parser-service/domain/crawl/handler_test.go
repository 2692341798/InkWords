package crawl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubCrawlService struct {
	policy     Policy
	checkpoint string
	err        error
}

func (service *stubCrawlService) Resume(_ context.Context, checkpointID string) (Manifest, error) {
	service.checkpoint = checkpointID
	return Manifest{CheckpointID: checkpointID, Status: "complete"}, service.err
}

func (service *stubCrawlService) Crawl(_ context.Context, policy Policy) (Manifest, error) {
	service.policy = policy
	return Manifest{EntryURL: policy.EntryURL, Status: "complete"}, service.err
}

func TestHandlerCrawlUsesSafeDefaultsAndAllowsNarrowerPathBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &stubCrawlService{}
	router := gin.New()
	router.POST("/crawl", NewHandler(service).Crawl)

	request := httptest.NewRequest(http.MethodPost, "/crawl", strings.NewReader(`{"entry_url":"https://docs.example.test/guide/","allowed_path_prefixes":["/guide"]}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "https://docs.example.test/guide/", service.policy.EntryURL)
	require.Equal(t, []string{"docs.example.test"}, service.policy.AllowedHosts)
	require.Equal(t, []string{"/guide"}, service.policy.AllowedPathPrefixes)
	require.Equal(t, 200, service.policy.MaxPages)
}

func TestHandlerCrawlRejectsUnsafeEntryBeforeCallingService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &stubCrawlService{}
	router := gin.New()
	router.POST("/crawl", NewHandler(service).Crawl)

	request := httptest.NewRequest(http.MethodPost, "/crawl", strings.NewReader(`{"entry_url":"file:///etc/passwd"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Empty(t, service.policy.EntryURL)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	require.Equal(t, "INVALID_CRAWL_POLICY", payload["code"])
}

func TestHandlerCrawlResumesPersistedCheckpointWithoutWideningPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &stubCrawlService{}
	router := gin.New()
	router.POST("/crawl", NewHandler(service).Crawl)

	request := httptest.NewRequest(http.MethodPost, "/crawl", strings.NewReader(`{"checkpoint_id":"0123456789abcdef0123456789abcdef"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "0123456789abcdef0123456789abcdef", service.checkpoint)
	require.Empty(t, service.policy.EntryURL)
}

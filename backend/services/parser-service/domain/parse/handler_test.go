package parse

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	parserinfra "inkwords-backend/shared/platform/parser"
)

func TestHandler_Parse_ReturnsArchiveSummaryForZip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(&stubParseService{
		parseResult: ParseResult{
			SourceContent: "merged content",
			ArchiveSummary: &parserinfra.ArchiveSummary{
				TotalFiles: 3,
				KeptFiles:  2,
			},
		},
	}, &stubQuotaChecker{})

	response := performMultipartParseRequest(t, handler, "courseware.zip", []byte("fake zip"))
	require.Equal(t, http.StatusOK, response.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	assert.Equal(t, float64(http.StatusOK), payload["code"])
	assert.Equal(t, "success", payload["message"])

	data, ok := payload["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "merged content", data["source_content"])

	summary, ok := data["archive_summary"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(3), summary["total_files"])
	assert.Equal(t, float64(2), summary["kept_files"])
}

func TestHandler_Parse_OmitsArchiveSummaryForNormalFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(&stubParseService{
		parseResult: ParseResult{SourceContent: "plain content"},
	}, &stubQuotaChecker{})

	response := performMultipartParseRequest(t, handler, "lesson.md", []byte("# title"))
	require.Equal(t, http.StatusOK, response.Code)
	assert.NotContains(t, response.Body.String(), "archive_summary")
}

func TestHandler_Parse_RejectsWhenQuotaExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	quotaChecker := &stubQuotaChecker{err: errors.New("quota exceeded")}
	parseService := &stubParseService{}
	handler := NewHandler(parseService, quotaChecker)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("file", "lesson.md")
	require.NoError(t, err)
	_, err = fileWriter.Write([]byte("# title"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/api/v1/project/parse", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		c.Next()
	})
	engine.POST("/api/v1/project/parse", handler.Parse)
	engine.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusPaymentRequired, recorder.Code)
	assert.True(t, quotaChecker.called)
	assert.False(t, parseService.called)
}

type stubParseService struct {
	parseResult ParseResult
	parseErr    error
	called      bool
}

func (s *stubParseService) Parse(io.Reader, string) (ParseResult, error) {
	s.called = true
	return s.parseResult, s.parseErr
}

type stubQuotaChecker struct {
	err    error
	called bool
}

func (s *stubQuotaChecker) CheckQuota(uuid.UUID) error {
	s.called = true
	return s.err
}

func performMultipartParseRequest(t *testing.T, handler *Handler, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = fileWriter.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/api/v1/project/parse", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	engine := gin.New()
	engine.POST("/api/v1/project/parse", handler.Parse)
	engine.ServeHTTP(recorder, request)

	return recorder
}

package project

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inkwords-backend/internal/infra/parser"
)

func TestHandler_Parse_ReturnsArchiveSummaryForZip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(&stubProjectService{
		parseResult: ParseResult{
			SourceContent: "merged content",
			ArchiveSummary: &parser.ArchiveSummary{
				TotalFiles: 3,
				KeptFiles:  2,
			},
		},
	})

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

	handler := NewHandler(&stubProjectService{
		parseResult: ParseResult{SourceContent: "plain content"},
	})

	response := performMultipartParseRequest(t, handler, "lesson.md", []byte("# title"))
	require.Equal(t, http.StatusOK, response.Code)
	assert.NotContains(t, response.Body.String(), "archive_summary")
}

type stubProjectService struct {
	parseResult ParseResult
	parseErr    error
}

func (s *stubProjectService) CheckQuota(uuid.UUID) error {
	return nil
}

func (s *stubProjectService) ScanProjectModules(context.Context, string) ([]ModuleCard, error) {
	return nil, nil
}

func (s *stubProjectService) Analyze(context.Context, string, string) (OutlineResult, string, string, error) {
	return OutlineResult{}, "", "", nil
}

func (s *stubProjectService) Parse(io.Reader, string) (ParseResult, error) {
	return s.parseResult, s.parseErr
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

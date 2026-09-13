package textbook

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/shared/kernel/httpx"
)

func TestTextbookHandlerUsesStableErrorCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{ErrNotFound, http.StatusNotFound, "TEXTBOOK_NOT_FOUND"},
		{ErrVersionConflict, http.StatusConflict, "TEXTBOOK_VERSION_CONFLICT"},
		{ErrPrimarySourceExists, http.StatusConflict, "PRIMARY_SOURCE_EXISTS"},
		{ErrRevisionLocked, http.StatusConflict, "REVISION_LOCKED"},
		{ErrClaimCoverageIncomplete, http.StatusUnprocessableEntity, "CLAIM_COVERAGE_INCOMPLETE"},
		{ErrEvidenceBudgetExceeded, http.StatusUnprocessableEntity, "EVIDENCE_BUDGET_EXCEEDED"},
		{ErrInvalidState, http.StatusBadRequest, "INVALID_STATE"},
	} {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		NewHandler(NewService(nil)).writeError(context, tc.err)
		require.Equal(t, tc.status, recorder.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
		require.Equal(t, tc.code, body["code"])
		require.NotEmpty(t, body["message"])
	}
}

func TestSourceEvidenceQueryRejectsEmptyOrOversizedSelection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return uuid.New(), nil }))
	router.GET("/projects/:projectID/evidence", NewHandler(NewService(nil)).ListSourceEvidence)
	for _, query := range []string{"chunk_id=", strings.Repeat("chunk_id=a&", 201)} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/projects/"+uuid.NewString()+"/evidence?"+query, nil))
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	}
}

func TestAppendRevisionRejectsClientCandidateAndUnknownFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceID := uuid.New()
	chapterID := uuid.New()
	handler := NewHandler(NewService(nil))
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspaceID, nil }))
	router.POST("/chapters/:chapterID/revisions", handler.AppendRevision)

	for _, body := range []string{
		`{"expected_version":0,"kind":"candidate","markdown":"x","document_json":{},"content_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
		`{"expected_version":0,"kind":"draft","unexpected":true}`,
	} {
		request := httptest.NewRequest(http.MethodPost, "/chapters/"+chapterID.String()+"/revisions", bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		var response map[string]any
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		require.Equal(t, "INVALID_STATE", response["code"])
	}
}

type fakeSampleTaskCreator struct {
	workspaceID        uuid.UUID
	projectID          uuid.UUID
	chapterID          uuid.UUID
	confirmedInputHash string
	createCalls        int
}

type fakeSourceImportTaskCreator struct {
	workspaceID     uuid.UUID
	projectID       uuid.UUID
	sourceID        uuid.UUID
	filename        string
	content         string
	resolvedVersion string
}

func (f *fakeSourceImportTaskCreator) CreateSourceImportTask(_ context.Context, workspaceID, projectID, sourceID uuid.UUID, filename string, content io.Reader, resolvedVersion string) (SourceImportTask, error) {
	raw, _ := io.ReadAll(content)
	f.workspaceID, f.projectID, f.sourceID, f.filename, f.content, f.resolvedVersion = workspaceID, projectID, sourceID, filename, string(raw), resolvedVersion
	return SourceImportTask{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222"), Status: "queued"}, nil
}

func (f *fakeSampleTaskCreator) PrepareSampleGeneration(_ context.Context, workspaceID, projectID, chapterID uuid.UUID) (SampleGenerationPreflight, error) {
	f.workspaceID, f.projectID, f.chapterID = workspaceID, projectID, chapterID
	return SampleGenerationPreflight{InputHash: "sha256:" + strings.Repeat("a", 64), WithinBudget: true, RequiresConfirmation: true}, nil
}

func (f *fakeSampleTaskCreator) CreateSampleGenerationTask(_ context.Context, workspaceID, projectID, chapterID uuid.UUID, confirmedInputHash string) (SampleGenerationTask, error) {
	f.workspaceID, f.projectID, f.chapterID = workspaceID, projectID, chapterID
	f.confirmedInputHash = confirmedInputHash
	f.createCalls++
	return SampleGenerationTask{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Status: "queued"}, nil
}

func TestSampleGenerationPreflightIsReadOnlyAndUsesServerSideIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceID, projectID, chapterID := uuid.New(), uuid.New(), uuid.New()
	creator := &fakeSampleTaskCreator{}
	handler := NewHandler(NewService(nil), creator)
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspaceID, nil }))
	router.GET("/projects/:projectID/chapters/:chapterID/generation-preflight", handler.GetSampleGenerationPreflight)

	request := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String()+"/chapters/"+chapterID.String()+"/generation-preflight", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, workspaceID, creator.workspaceID)
	require.Equal(t, projectID, creator.projectID)
	require.Equal(t, chapterID, creator.chapterID)
	require.Zero(t, creator.createCalls)
}

func TestGenerateSampleUsesServerSideChapterAndProjectIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceID, projectID, chapterID := uuid.New(), uuid.New(), uuid.New()
	creator := &fakeSampleTaskCreator{}
	handler := NewHandler(NewService(nil), creator)
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspaceID, nil }))
	router.POST("/projects/:projectID/chapters/:chapterID/generate-sample", handler.GenerateSample)

	confirmedInputHash := "sha256:" + strings.Repeat("a", 64)
	request := httptest.NewRequest(http.MethodPost, "/projects/"+projectID.String()+"/chapters/"+chapterID.String()+"/generate-sample", bytes.NewBufferString(`{"confirmed_input_hash":"`+confirmedInputHash+`"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.Equal(t, workspaceID, creator.workspaceID)
	require.Equal(t, projectID, creator.projectID)
	require.Equal(t, chapterID, creator.chapterID)
	require.Equal(t, confirmedInputHash, creator.confirmedInputHash)
	var response struct {
		Code int `json:"code"`
		Data struct {
			TaskID string `json:"task_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.Equal(t, "11111111-1111-1111-1111-111111111111", response.Data.TaskID)
}

func TestCreateSourceImportUsesServerSideSourceAndProjectIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceID, projectID, sourceID := uuid.New(), uuid.New(), uuid.New()
	creator := &fakeSourceImportTaskCreator{}
	handler := NewHandler(NewService(nil)).WithSourceImportCreator(creator)
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspaceID, nil }))
	router.POST("/projects/:projectID/source-imports", handler.CreateSourceImport)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("source_id", sourceID.String()))
	file, err := writer.CreateFormFile("file", "intro.md")
	require.NoError(t, err)
	_, err = file.Write([]byte("# 入门"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	request := httptest.NewRequest(http.MethodPost, "/projects/"+projectID.String()+"/source-imports", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.Equal(t, workspaceID, creator.workspaceID)
	require.Equal(t, projectID, creator.projectID)
	require.Equal(t, sourceID, creator.sourceID)
	require.Equal(t, "intro.md", creator.filename)
	require.Equal(t, "# 入门", creator.content)
}

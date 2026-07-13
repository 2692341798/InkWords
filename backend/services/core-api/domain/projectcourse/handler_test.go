package projectcourse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	coretask "inkwords-backend/services/core-api/domain/task"
)

type fakeCourseRepository struct{ created *ProjectCourse }

func (r *fakeCourseRepository) Create(_ context.Context, course *ProjectCourse) error {
	r.created = course
	return nil
}
func (r *fakeCourseRepository) GetByID(context.Context, uuid.UUID, uuid.UUID) (*ProjectCourse, error) {
	return r.created, nil
}
func (r *fakeCourseRepository) UpdateBlueprintCAS(context.Context, uuid.UUID, uuid.UUID, BlueprintUpdate) error {
	return nil
}
func (r *fakeCourseRepository) Approve(context.Context, uuid.UUID, uuid.UUID, int) error { return nil }

type fakeAnalyzeTaskCreator struct {
	input coretask.CreateProjectCourseTaskInput
}

func (f *fakeAnalyzeTaskCreator) CreateProjectCourseAnalyzeTask(_ context.Context, input coretask.CreateProjectCourseTaskInput) (coretask.JobTask, error) {
	f.input = input
	return coretask.JobTask{ID: uuid.New(), Status: coretask.JobTaskStatusQueued}, nil
}

func (f *fakeAnalyzeTaskCreator) CreateProjectCourseGenerateTask(_ context.Context, input coretask.CreateProjectCourseTaskInput) (coretask.JobTask, error) {
	f.input = input
	return coretask.JobTask{ID: uuid.New(), Status: coretask.JobTaskStatusQueued}, nil
}

func (f *fakeAnalyzeTaskCreator) CreateProjectCoursePackageTask(_ context.Context, input coretask.CreateProjectCourseTaskInput) (coretask.JobTask, error) {
	f.input = input
	return coretask.JobTask{ID: uuid.New(), Status: coretask.JobTaskStatusQueued}, nil
}

func TestCreateStartsAnalyzeTaskWithoutClientSuppliedSHA(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repository := &fakeCourseRepository{}
	tasks := &fakeAnalyzeTaskCreator{}
	handler := NewHandler(NewService(repository), tasks)
	router := gin.New()
	router.POST("/project-courses", func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		handler.Create(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/project-courses", strings.NewReader(`{"repository_url":"https://github.com/example/repo","requested_ref":"main","audience_level":"programming"}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	require.Equal(t, http.StatusAccepted, response.Code)
	require.Equal(t, StatusAnalyzing, repository.created.Status)
	require.Empty(t, repository.created.ResolvedCommitSHA)
	var body struct {
		Data struct {
			Course ProjectCourse `json:"course"`
			TaskID uuid.UUID     `json:"task_id"`
			Status string        `json:"status"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Equal(t, repository.created.ID, body.Data.Course.ID)
	require.NotEqual(t, uuid.Nil, body.Data.TaskID)
	require.Equal(t, string(coretask.JobTaskStatusQueued), string(body.Data.Status))

	var payload map[string]any
	require.NoError(t, json.Unmarshal(tasks.input.Payload, &payload))
	require.Equal(t, repository.created.ID.String(), payload["course_id"])
}

func TestCreateRejectsClientSuppliedResolvedSHA(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(NewService(&fakeCourseRepository{}), &fakeAnalyzeTaskCreator{})
	router := gin.New()
	router.POST("/project-courses", func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		handler.Create(c)
	})
	req := httptest.NewRequest(http.MethodPost, "/project-courses", strings.NewReader(`{"repository_url":"https://github.com/example/repo","requested_ref":"main","resolved_commit_sha":"0123456789abcdef0123456789abcdef01234567","audience_level":"programming"}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestReportsAreReadOnlyAndScopedToTheAuthenticatedCourse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	owner := uuid.New()
	repository := &fakeCourseRepository{created: &ProjectCourse{ID: uuid.New(), UserID: owner, CoverageJSON: []byte(`{"modules":[]}`), QualityReportJSON: []byte(`{"status":"completed"}`)}}
	handler := NewHandler(NewService(repository))
	router := gin.New()
	router.GET("/project-courses/:id/coverage", func(c *gin.Context) { c.Set("user_id", owner); handler.Coverage(c) })
	router.GET("/project-courses/:id/quality-report", func(c *gin.Context) { c.Set("user_id", owner); handler.QualityReport(c) })
	for _, path := range []string{"/project-courses/" + repository.created.ID.String() + "/coverage", "/project-courses/" + repository.created.ID.String() + "/quality-report"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, response.Code)
	}
}

package projectcourse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	coretask "inkwords-backend/services/core-api/domain/task"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

// Handler 是 ProjectCourse 的 HTTP 适配层；不允许客户端提交事实、证据或覆盖矩阵。
type analyzeTaskCreator interface {
	CreateProjectCourseAnalyzeTask(context.Context, coretask.CreateProjectCourseTaskInput) (coretask.JobTask, error)
	CreateProjectCourseGenerateTask(context.Context, coretask.CreateProjectCourseTaskInput) (coretask.JobTask, error)
	CreateProjectCoursePackageTask(context.Context, coretask.CreateProjectCourseTaskInput) (coretask.JobTask, error)
}

type Handler struct {
	service     *Service
	taskCreator analyzeTaskCreator
}

func NewHandler(service *Service, taskCreators ...analyzeTaskCreator) *Handler {
	var taskCreator analyzeTaskCreator
	if len(taskCreators) > 0 {
		taskCreator = taskCreators[0]
	}
	return &Handler{service: service, taskCreator: taskCreator}
}

type createRequest struct {
	RepositoryURL string `json:"repository_url" binding:"required"`
	RequestedRef  string `json:"requested_ref" binding:"required"`
	AudienceLevel string `json:"audience_level" binding:"required"`
}

type blueprintUpdateRequest struct {
	ExpectedVersion int                  `json:"expected_version" binding:"required,min=1"`
	Chapters        []chapterUpdateInput `json:"chapters" binding:"required"`
}

type chapterUpdateInput struct {
	ChapterID string `json:"chapter_id" binding:"required"`
	Title     string `json:"title" binding:"required"`
	Sort      int    `json:"sort"`
	Enabled   bool   `json:"enabled"`
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := requestUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req createRequest
	if err := decodeStrictJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid project course request")
		return
	}
	if req.RepositoryURL == "" || req.RequestedRef == "" || req.AudienceLevel == "" {
		writeError(c, http.StatusBadRequest, "required project course fields are missing")
		return
	}
	if h.taskCreator == nil {
		writeError(c, http.StatusServiceUnavailable, "project course analyzer is unavailable")
		return
	}
	course, err := h.service.Create(c.Request.Context(), CreateInput{UserID: userID, RepositoryURL: req.RepositoryURL, RequestedRef: req.RequestedRef, AudienceLevel: req.AudienceLevel})
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := json.Marshal(gin.H{"course_id": course.ID, "repository_url": course.RepositoryURL, "requested_ref": course.RequestedRef, "audience_level": course.AudienceLevel})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "failed to create project course task")
		return
	}
	task, err := h.taskCreator.CreateProjectCourseAnalyzeTask(c.Request.Context(), coretask.CreateProjectCourseTaskInput{RequestedBy: userID, IdempotencyKey: "project-course:analyze:" + course.ID.String(), Payload: payload})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "failed to create project course task")
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"code": 0, "data": gin.H{"course": course, "task_id": task.ID, "status": task.Status}})
}

func (h *Handler) Get(c *gin.Context) {
	userID, courseID, ok := h.ids(c)
	if !ok {
		return
	}
	course, err := h.service.Get(c.Request.Context(), userID, courseID)
	if err != nil {
		h.writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": course})
}

func (h *Handler) Coverage(c *gin.Context) {
	h.getReport(c, true)
}

func (h *Handler) QualityReport(c *gin.Context) {
	h.getReport(c, false)
}

func (h *Handler) getReport(c *gin.Context, coverage bool) {
	userID, courseID, ok := h.ids(c)
	if !ok {
		return
	}
	course, err := h.service.Get(c.Request.Context(), userID, courseID)
	if err != nil {
		h.writeDomainError(c, err)
		return
	}
	if coverage {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": json.RawMessage(course.CoverageJSON)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": json.RawMessage(course.QualityReportJSON)})
}

func (h *Handler) UpdateBlueprint(c *gin.Context) {
	userID, courseID, ok := h.ids(c)
	if !ok {
		return
	}
	var req blueprintUpdateRequest
	if err := decodeStrictJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, "only chapter title, sort and enabled may be updated")
		return
	}
	if req.ExpectedVersion < 1 || req.Chapters == nil {
		writeError(c, http.StatusBadRequest, "expected_version and chapters are required")
		return
	}
	updates := make([]ChapterUpdate, 0, len(req.Chapters))
	for _, chapter := range req.Chapters {
		updates = append(updates, ChapterUpdate(chapter))
	}
	err := h.service.UpdateBlueprint(c.Request.Context(), userID, courseID, BlueprintUpdate{ExpectedVersion: req.ExpectedVersion, ChapterUpdates: updates})
	if err != nil {
		h.writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"blueprint_version": req.ExpectedVersion + 1}})
}

func (h *Handler) PreviewBlueprint(c *gin.Context) {
	userID, courseID, ok := h.ids(c)
	if !ok {
		return
	}
	var req blueprintUpdateRequest
	if err := decodeStrictJSON(c, &req); err != nil || req.ExpectedVersion < 1 || req.Chapters == nil {
		writeError(c, http.StatusBadRequest, "expected_version and chapters are required")
		return
	}
	updates := make([]ChapterUpdate, 0, len(req.Chapters))
	for _, chapter := range req.Chapters {
		updates = append(updates, ChapterUpdate(chapter))
	}
	preview, err := h.service.PreviewBlueprint(c.Request.Context(), userID, courseID, BlueprintUpdate{ExpectedVersion: req.ExpectedVersion, ChapterUpdates: updates})
	if err != nil {
		h.writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": preview})
}

func (h *Handler) Approve(c *gin.Context) {
	userID, courseID, ok := h.ids(c)
	if !ok {
		return
	}
	if h.taskCreator == nil {
		writeError(c, http.StatusServiceUnavailable, "project course generator is unavailable")
		return
	}
	var req struct {
		ExpectedVersion int `json:"expected_version" binding:"required,min=1"`
	}
	if err := decodeStrictJSON(c, &req); err != nil || req.ExpectedVersion < 1 {
		writeError(c, http.StatusBadRequest, "expected_version is required")
		return
	}
	if err := h.service.Approve(c.Request.Context(), userID, courseID, req.ExpectedVersion); err != nil {
		h.writeDomainError(c, err)
		return
	}
	course, err := h.service.Get(c.Request.Context(), userID, courseID)
	if err != nil {
		h.writeDomainError(c, err)
		return
	}
	payload, err := json.Marshal(gin.H{
		"course_id":           course.ID,
		"repository_url":      course.RepositoryURL,
		"resolved_commit_sha": course.ResolvedCommitSHA,
		"blueprint":           json.RawMessage(course.BlueprintJSON),
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "failed to create project course generation task")
		return
	}
	task, err := h.taskCreator.CreateProjectCourseGenerateTask(c.Request.Context(), coretask.CreateProjectCourseTaskInput{RequestedBy: userID, IdempotencyKey: "project-course:generate:" + course.ID.String() + ":" + fmt.Sprint(course.BlueprintVersion), Payload: payload})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "failed to create project course generation task")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"status": StatusApproved, "task_id": task.ID}})
}

func (h *Handler) Package(c *gin.Context) {
	userID, courseID, ok := h.ids(c)
	if !ok {
		return
	}
	if h.taskCreator == nil {
		writeError(c, http.StatusServiceUnavailable, "project course package worker is unavailable")
		return
	}
	course, err := h.service.Get(c.Request.Context(), userID, courseID)
	if err != nil {
		h.writeDomainError(c, err)
		return
	}
	if course.Status != StatusCompleted {
		writeError(c, http.StatusConflict, "course package requires a completed course")
		return
	}
	var report struct {
		Chapters []struct {
			ChapterID string `json:"chapter_id"`
			Status    string `json:"status"`
			Document  struct {
				Title string                    `json:"title"`
				Lab   *sharedkernel.LabManifest `json:"lab"`
			} `json:"document"`
		} `json:"chapters"`
	}
	if err := json.Unmarshal(course.QualityReportJSON, &report); err != nil {
		writeError(c, http.StatusConflict, "course package report is unavailable")
		return
	}
	artifacts := make([]map[string]any, 0)
	for _, chapter := range report.Chapters {
		if chapter.Status != "succeeded" || chapter.Document.Lab == nil {
			continue
		}
		artifacts = append(artifacts, map[string]any{"chapter_id": chapter.ChapterID, "title": chapter.Document.Title, "manifest": chapter.Document.Lab})
	}
	if len(artifacts) == 0 {
		writeError(c, http.StatusConflict, "no verified course lab artifacts are available")
		return
	}
	payload, err := json.Marshal(map[string]any{"package": map[string]any{
		"course_id": course.ID, "blueprint_version": course.BlueprintVersion, "repository_url": course.RepositoryURL, "commit_sha": course.ResolvedCommitSHA,
		"artifacts": artifacts, "coverage": json.RawMessage(course.CoverageJSON), "verification": map[string]any{"passed": true},
	}})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "failed to create course package task")
		return
	}
	task, err := h.taskCreator.CreateProjectCoursePackageTask(c.Request.Context(), coretask.CreateProjectCourseTaskInput{RequestedBy: userID, IdempotencyKey: fmt.Sprintf("project-course:package:%s:%d", course.ID, course.BlueprintVersion), Payload: payload})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "failed to create course package task")
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"code": 0, "data": gin.H{"task_id": task.ID, "status": task.Status}})
}

func (h *Handler) ids(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := requestUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, uuid.Nil, false
	}
	courseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid project course id")
		return uuid.Nil, uuid.Nil, false
	}
	return userID, courseID, true
}

func (h *Handler) writeDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(c, http.StatusNotFound, "project course not found")
	case errors.Is(err, ErrVersionConflict):
		writeError(c, http.StatusConflict, "blueprint version conflict")
	case errors.Is(err, ErrBlueprintImmutable):
		writeError(c, http.StatusConflict, "approved blueprint is immutable")
	default:
		writeError(c, http.StatusInternalServerError, "project course operation failed")
	}
}

func requestUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	userID, ok := value.(uuid.UUID)
	return userID, ok && userID != uuid.Nil
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"code": status, "message": message})
}

func decodeStrictJSON(c *gin.Context, target any) error {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

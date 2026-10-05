package mastery

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"inkwords-backend/shared/kernel/httpx"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type applicationService interface {
	CreateObjective(context.Context, uuid.UUID, ObjectiveInput) (Objective, error)
	RecordAttempt(context.Context, uuid.UUID, uuid.UUID, Attempt) (DueTask, error)
	Due(context.Context, uuid.UUID, time.Time, int) ([]DueObjective, error)
	Workspace(context.Context, uuid.UUID, uuid.UUID) (ObjectiveWorkspace, error)
}

// Handler is the local HTTP boundary for mastery tasks. The browser only asks
// for due work when it is open; this surface does not schedule notifications.
type Handler struct {
	service applicationService
	now     func() time.Time
}

func NewHandler(service applicationService) *Handler {
	return &Handler{service: service, now: time.Now}
}

func (handler *Handler) CreateObjective(c *gin.Context) {
	workspaceID, ok := handler.workspaceID(c)
	if !ok {
		return
	}
	var request struct {
		ChapterID     string   `json:"chapter_id"`
		Title         string   `json:"title"`
		Behavior      string   `json:"behavior"`
		Skills        []Skill  `json:"skills"`
		Rubric        []string `json:"rubric"`
		KeyPoints     []string `json:"key_points"`
		Prerequisites []string `json:"prerequisites"`
		EvidenceRefs  []string `json:"evidence_refs"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		handler.writeError(c, http.StatusBadRequest, "无效的掌握目标")
		return
	}
	objective, err := handler.service.CreateObjective(c.Request.Context(), workspaceID, ObjectiveInput{ChapterID: request.ChapterID, Title: request.Title, Behavior: request.Behavior, Skills: request.Skills, Rubric: request.Rubric, KeyPoints: request.KeyPoints, Prerequisites: request.Prerequisites, EvidenceRefs: request.EvidenceRefs})
	if err != nil {
		handler.writeError(c, http.StatusBadRequest, "无法创建掌握目标")
		return
	}
	handler.writeSuccess(c, objective)
}

func (handler *Handler) RecordAttempt(c *gin.Context) {
	workspaceID, ok := handler.workspaceID(c)
	if !ok {
		return
	}
	objectiveID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		handler.writeError(c, http.StatusBadRequest, "无效的掌握目标 ID")
		return
	}
	var request struct {
		Skill               Skill                            `json:"skill"`
		CodeFiles           []sharedtextbook.LearnerCodeFile `json:"code_files"`
		Answer              string                           `json:"answer"`
		PracticeTaskID      string                           `json:"practice_task_id"`
		PracticeContentHash string                           `json:"practice_content_hash"`
		PracticeSessionID   string                           `json:"practice_session_id"`
		Correct             bool                             `json:"correct"`
		Independent         bool                             `json:"independent"`
		HintCount           int                              `json:"hint_count"`
		TookMillis          int64                            `json:"took_millis"`
		Confidence          int                              `json:"confidence"`
		ErrorKinds          []string                         `json:"error_kinds"`
		AttemptedAt         string                           `json:"attempted_at"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2*1024*1024)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(new(any)) != io.EOF {
		handler.writeError(c, http.StatusBadRequest, "无效的学习表现")
		return
	}
	attemptedAt, err := time.Parse(time.RFC3339, request.AttemptedAt)
	if err != nil {
		handler.writeError(c, http.StatusBadRequest, "学习表现必须包含 RFC3339 时间")
		return
	}
	sessionID := uuid.Nil
	if request.PracticeSessionID != "" {
		sessionID, err = uuid.Parse(request.PracticeSessionID)
		if err != nil {
			handler.writeError(c, http.StatusBadRequest, "无效的练习会话 ID")
			return
		}
	}
	due, err := handler.service.RecordAttempt(c.Request.Context(), workspaceID, objectiveID, Attempt{Skill: request.Skill, Answer: request.Answer, LearnerFiles: request.CodeFiles, PracticeTaskID: request.PracticeTaskID, PracticeSessionID: sessionID, PracticeContentHash: request.PracticeContentHash, Correct: request.Correct, Independent: request.Independent, HintCount: request.HintCount, Took: time.Duration(request.TookMillis) * time.Millisecond, Confidence: request.Confidence, ErrorKinds: request.ErrorKinds, At: attemptedAt})
	if err != nil {
		switch {
		case errors.Is(err, ErrPracticeSessionClosed):
			handler.writeError(c, http.StatusConflict, "本次练习已记录且内容不同，不能覆盖原作答；请重新打开查看记录。")
			return
		case errors.Is(err, ErrPracticeNotDue):
			handler.writeError(c, http.StatusConflict, "尚未达到本题的延迟复习时间，请按到期队列安排继续。")
			return
		case errors.Is(err, ErrMasteryHistoryChanged):
			handler.writeError(c, http.StatusConflict, "学习记录已更新，请保留作答并重试。")
			return
		case errors.Is(err, ErrPracticeMismatch):
			handler.writeError(c, http.StatusConflict, "作答与批准题目不匹配，请重新读取学习目标。")
			return
		}
		handler.writeError(c, http.StatusBadRequest, "无法记录学习表现")
		return
	}
	handler.writeSuccess(c, due)
}

func (handler *Handler) Due(c *gin.Context) {
	workspaceID, ok := handler.workspaceID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	tasks, err := handler.service.Due(c.Request.Context(), workspaceID, handler.now().UTC(), limit)
	if err != nil {
		handler.writeError(c, http.StatusInternalServerError, "无法读取到期任务")
		return
	}
	if tasks == nil {
		tasks = []DueObjective{}
	}
	handler.writeSuccess(c, gin.H{"tasks": tasks})
}

// Workspace returns only an objective owned by the current local workspace.
func (handler *Handler) Workspace(c *gin.Context) {
	workspaceID, ok := handler.workspaceID(c)
	if !ok {
		return
	}
	objectiveID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		handler.writeError(c, http.StatusBadRequest, "无效的掌握目标 ID")
		return
	}
	workspace, err := handler.service.Workspace(c.Request.Context(), workspaceID, objectiveID)
	if err != nil {
		handler.writeError(c, http.StatusNotFound, "学习目标不可用")
		return
	}
	handler.writeSuccess(c, workspace)
}

func (handler *Handler) workspaceID(c *gin.Context) (uuid.UUID, bool) {
	workspaceID, err := httpx.GetLocalWorkspaceID(c)
	if err != nil || workspaceID == uuid.Nil {
		handler.writeError(c, http.StatusServiceUnavailable, "本地工作区尚未准备完成")
		return uuid.Nil, false
	}
	return workspaceID, true
}

func (handler *Handler) writeSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "success", "data": data})
}
func (handler *Handler) writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"code": status, "message": message, "data": nil})
}

package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"inkwords-backend/internal/model"
)

const defaultTaskStreamPollInterval = 500 * time.Millisecond

type taskService interface {
	CreateGenerationTask(ctx context.Context, input CreateGenerationTaskInput) (model.JobTask, error)
	CreateParseTask(ctx context.Context, input CreateParseTaskInput) (model.JobTask, error)
	CreateExportTask(ctx context.Context, input CreateExportTaskInput) (model.JobTask, error)
	GetTask(ctx context.Context, taskID uuid.UUID, requestedBy uuid.UUID) (model.JobTask, error)
	CancelTask(ctx context.Context, taskID uuid.UUID, requestedBy uuid.UUID) error
	ListStreamEvents(ctx context.Context, taskID uuid.UUID, afterID uint64) ([]model.JobTaskEvent, bool, error)
}

// Handler 提供 generation task 的 HTTP 适配层。
type Handler struct {
	service            taskService
	exportArtifactsDir string
	pollInterval       time.Duration
}

// NewHandler 通过依赖注入组装任务 HTTP Handler。
func NewHandler(service taskService, exportArtifactsDir string) *Handler {
	return &Handler{
		service:            service,
		exportArtifactsDir: exportArtifactsDir,
		pollInterval:       defaultTaskStreamPollInterval,
	}
}

// CreateGenerationTask 接收前端任务创建请求，并返回可订阅的任务地址。
func (h *Handler) CreateGenerationTask(c *gin.Context) {
	h.createTask(c, func(userID uuid.UUID, req CreateGenerationTaskRequest) (model.JobTask, error) {
		return h.service.CreateGenerationTask(c.Request.Context(), CreateGenerationTaskInput{
			RequestedBy:    userID,
			TaskSubtype:    req.Kind,
			IdempotencyKey: req.IdempotencyKey,
			Payload:        []byte(req.Payload),
		})
	})
}

// CreateParseTask 接收前端解析任务创建请求，并返回可订阅的任务地址。
func (h *Handler) CreateParseTask(c *gin.Context) {
	h.createTask(c, func(userID uuid.UUID, req CreateGenerationTaskRequest) (model.JobTask, error) {
		return h.service.CreateParseTask(c.Request.Context(), CreateParseTaskInput{
			RequestedBy:    userID,
			TaskSubtype:    req.Kind,
			IdempotencyKey: req.IdempotencyKey,
			Payload:        []byte(req.Payload),
		})
	})
}

// CreateExportTask 接收前端导出任务创建请求，并返回可订阅的任务地址。
func (h *Handler) CreateExportTask(c *gin.Context) {
	h.createTask(c, func(userID uuid.UUID, req CreateGenerationTaskRequest) (model.JobTask, error) {
		return h.service.CreateExportTask(c.Request.Context(), CreateExportTaskInput{
			RequestedBy:    userID,
			TaskSubtype:    req.Kind,
			IdempotencyKey: req.IdempotencyKey,
			Payload:        []byte(req.Payload),
		})
	})
}

func (h *Handler) createTask(
	c *gin.Context,
	create func(userID uuid.UUID, req CreateGenerationTaskRequest) (model.JobTask, error),
) {
	var req CreateGenerationTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	task, err := create(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	c.JSON(http.StatusAccepted, CreateGenerationTaskResponse{
		TaskID:    task.ID,
		Status:    task.Status,
		StreamURL: "/api/v1/tasks/" + task.ID.String() + "/stream",
	})
}

// GetTask 返回任务当前状态快照，便于前端轮询查询。
func (h *Handler) GetTask(c *gin.Context) {
	taskID, ok := h.parseTaskID(c)
	if !ok {
		return
	}
	userID, ok := h.getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	task, err := h.service.GetTask(c.Request.Context(), taskID, userID)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildTaskResponse(task))
}

// CancelTask 标记任务为取消态，并把终态事件写入事件流。
func (h *Handler) CancelTask(c *gin.Context) {
	taskID, ok := h.parseTaskID(c)
	if !ok {
		return
	}
	userID, ok := h.getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.service.CancelTask(c.Request.Context(), taskID, userID); err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"task_id": taskID,
		"status":  model.JobTaskStatusCancelled,
	})
}

// StreamTask 通过轮询事件表输出 SSE，直到任务进入终态或客户端断开。
func (h *Handler) StreamTask(c *gin.Context) {
	taskID, ok := h.parseTaskID(c)
	if !ok {
		return
	}
	userID, ok := h.getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if _, err := h.service.GetTask(c.Request.Context(), taskID, userID); err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(h.pollInterval)
	defer ticker.Stop()

	var afterID uint64
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			events, done, err := h.service.ListStreamEvents(c.Request.Context(), taskID, afterID)
			if err != nil {
				writeTaskStreamEvent(c, "error", "task stream failed")
				return
			}
			for _, event := range events {
				afterID = event.ID
				writeTaskStreamEvent(c, event.EventType, string(event.Payload))
			}
			if done {
				writeTaskStreamEvent(c, "done", "[DONE]")
				return
			}
		}
	}
}

func (h *Handler) getUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	userID, ok := value.(uuid.UUID)
	return userID, ok
}

func (h *Handler) parseTaskID(c *gin.Context) (uuid.UUID, bool) {
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return uuid.Nil, false
	}
	return taskID, true
}

func (h *Handler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrTaskNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
	case errors.Is(err, ErrTaskAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "task access denied"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "task request failed"})
	}
}

func buildTaskResponse(task model.JobTask) TaskResponse {
	return TaskResponse{
		ID:           task.ID,
		TaskType:     task.TaskType,
		TaskSubtype:  task.TaskSubtype,
		Status:       task.Status,
		Result:       cloneJSON(task.ResultJSON),
		ErrorMessage: task.ErrorMessage,
		RetryCount:   task.RetryCount,
		StartedAt:    task.StartedAt,
		FinishedAt:   task.FinishedAt,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
}

func cloneJSON(payload []byte) json.RawMessage {
	if len(payload) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), payload...)
}

// 为什么手写 SSE：Gin 默认会把 string 作为字符串字面量编码，任务事件这里需要原样输出 JSON 片段，
// 否则前端会拿到被额外转义的 payload。
func writeTaskStreamEvent(c *gin.Context, eventType string, payload string) {
	_, _ = fmt.Fprintf(c.Writer, "event:%s\n", eventType)
	_, _ = fmt.Fprintf(c.Writer, "data:%s\n\n", payload)
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

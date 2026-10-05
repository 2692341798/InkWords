package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"inkwords-backend/services/core-api/app/textbookgeneration"
	coretask "inkwords-backend/services/core-api/domain/task"
	"inkwords-backend/shared/kernel/httpx"
)

type textbookTaskAccess interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (textbookgeneration.TaskSnapshot, error)
	Retry(context.Context, uuid.UUID, uuid.UUID, string) (textbookgeneration.TaskSnapshot, error)
}

// TextbookTaskHandler exposes only workspace-scoped textbook task snapshots.
// It deliberately does not accept an authenticated user ID from the request.
type TextbookTaskHandler struct {
	service textbookTaskAccess
}

func NewTextbookTaskHandler(service textbookTaskAccess) *TextbookTaskHandler {
	return &TextbookTaskHandler{service: service}
}

func (handler *TextbookTaskHandler) GetTask(c *gin.Context) {
	workspaceID, taskID, ok := handler.identities(c)
	if !ok {
		return
	}
	snapshot, err := handler.service.Get(c.Request.Context(), workspaceID, taskID)
	if err != nil {
		handler.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

func (handler *TextbookTaskHandler) RetryTask(c *gin.Context) {
	workspaceID, taskID, ok := handler.identities(c)
	if !ok {
		return
	}
	var request struct {
		ConfirmedInputHash string `json:"confirmed_input_hash"`
	}
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "重试确认格式无效"})
			return
		}
	}
	snapshot, err := handler.service.Retry(c.Request.Context(), workspaceID, taskID, request.ConfirmedInputHash)
	if err != nil {
		handler.writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, snapshot)
}

func (handler *TextbookTaskHandler) identities(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	if handler == nil || handler.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "本地教材任务服务尚未准备完成"})
		return uuid.Nil, uuid.Nil, false
	}
	workspaceID, err := httpx.GetLocalWorkspaceID(c)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "本地工作区尚未准备完成"})
		return uuid.Nil, uuid.Nil, false
	}
	taskID, err := uuid.Parse(c.Param("taskID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务 ID 无效"})
		return uuid.Nil, uuid.Nil, false
	}
	return workspaceID, taskID, true
}

func (handler *TextbookTaskHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, coretask.ErrTaskNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "教材任务不存在"})
	case errors.Is(err, coretask.ErrTaskAccessDenied), errors.Is(err, textbookgeneration.ErrTaskOutsideTextbookScope):
		c.JSON(http.StatusForbidden, gin.H{"error": "该任务不属于本地教材工作区"})
	case errors.Is(err, coretask.ErrTaskNotRetryable):
		c.JSON(http.StatusConflict, gin.H{"error": "该教材任务当前不可重试"})
	case errors.Is(err, textbookgeneration.ErrRetryConfirmationRequired):
		c.JSON(http.StatusConflict, gin.H{"error": "请先确认该失败任务的冻结输入，再重试一次模型调用"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "教材任务请求失败"})
	}
}

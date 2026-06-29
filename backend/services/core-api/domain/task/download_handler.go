package task

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// DownloadTask 根据任务结果中的文件令牌提供受控下载，并在成功发送后删除落地文件。
func (h *Handler) DownloadTask(c *gin.Context) {
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
	if task.TaskType != taskTypeExport {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task is not downloadable"})
		return
	}
	if task.Status != JobTaskStatusSucceeded {
		c.JSON(http.StatusConflict, gin.H{"error": "task is not finished"})
		return
	}

	var result ExportTaskResult
	if err := json.Unmarshal(task.ResultJSON, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid task result"})
		return
	}
	if result.FileToken == "" || result.ContentType == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid task result"})
		return
	}
	if !result.ExpiresAt.IsZero() && time.Now().UTC().After(result.ExpiresAt) {
		c.JSON(http.StatusNotFound, gin.H{"error": "download expired"})
		return
	}

	filePath := filepath.Join(h.exportArtifactsDir, result.FileToken+".pdf")
	//nolint:gosec
	file, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "download artifact missing"})
		return
	}
	defer func() { _ = file.Close() }()
	defer func() { _ = os.Remove(filePath) }()

	c.Header("Content-Type", result.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", result.Filename))
	c.Status(http.StatusOK)
	if err := copyDownload(c.Writer, file); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
}

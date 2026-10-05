package stream

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) GenerateBlogStreamHandler(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	req.SourceType = resolveAnalyzeSourceType(req)
	req.ScenarioMode = string(normalizeScenarioMode(req.ScenarioMode, req.SourceType))

	chunkChan, errChan := newGenerateStreamChannels()

	ctx := c.Request.Context()

	workspaceID := h.getWorkspaceID(c)
	if workspaceID == uuid.Nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "local workspace unavailable"})
		return
	}

	go h.service.Generate(ctx, workspaceID, req, chunkChan, errChan)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	sseStreamBody(c, chunkChan, &errChan, streamOperationGenerate)
}

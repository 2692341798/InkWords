package stream

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) GenerateBlogStreamHandler(c *gin.Context) {
	if !h.maybeCheckQuota(c) {
		return
	}

	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	req.SourceType = resolveAnalyzeSourceType(req)
	req.ScenarioMode = string(normalizeScenarioMode(req.ScenarioMode, req.SourceType))

	chunkChan, errChan := newGenerateStreamChannels()

	ctx := c.Request.Context()

	userID := h.getUserID(c)
	if userID == uuid.Nil {
		userID = uuid.New()
	}

	go h.service.Generate(ctx, userID, req, chunkChan, errChan)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	sseStreamBody(c, chunkChan, &errChan, streamOperationGenerate)
}

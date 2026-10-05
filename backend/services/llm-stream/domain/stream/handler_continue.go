package stream

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) ContinueBlogStreamHandler(c *gin.Context) {
	blogIDStr := c.Param("id")
	blogID, err := uuid.Parse(blogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid blog id"})
		return
	}

	workspaceID := h.getWorkspaceID(c)
	if workspaceID == uuid.Nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "local workspace unavailable"})
		return
	}

	chunkChan, errChan := newGenerateStreamChannels()

	ctx := c.Request.Context()

	go h.service.Continue(ctx, workspaceID, blogID, chunkChan, errChan)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	sseStreamBody(c, chunkChan, &errChan, streamOperationContinue)
}

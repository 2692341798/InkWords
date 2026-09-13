package stream

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) PolishBlogStreamHandler(c *gin.Context) {
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

	if err := h.blogRepo.Exists(c.Request.Context(), workspaceID, blogID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "blog not found"})
		return
	}

	var req PolishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	chunkChan, errChan := newGenerateStreamChannels()

	ctx := c.Request.Context()
	go h.service.Polish(ctx, req, chunkChan, errChan)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	sseStreamBody(c, chunkChan, &errChan, streamOperationPolish)
}

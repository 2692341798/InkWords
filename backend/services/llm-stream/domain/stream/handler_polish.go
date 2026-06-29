package stream

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) PolishBlogStreamHandler(c *gin.Context) {
	if !h.maybeCheckQuota(c) {
		return
	}

	blogIDStr := c.Param("id")
	blogID, err := uuid.Parse(blogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid blog id"})
		return
	}

	userID := h.getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.blogRepo.Exists(c.Request.Context(), userID, blogID); err != nil {
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

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			go func() {
				chunkOpen, errOpen := true, true
				for chunkOpen || errOpen {
					select {
					case _, ok := <-chunkChan:
						if !ok {
							chunkOpen = false
						}
					case _, ok := <-errChan:
						if !ok {
							errOpen = false
						}
					}
				}
			}()
			return false
		case err, ok := <-errChan:
			if ok && err != nil {
				writeStreamEvent(c, w, "error", externalStreamErrorMessage(streamOperationPolish, err))
				return false
			}
			if !ok {
				errChan = nil
			}
			return true
		case chunk, ok := <-chunkChan:
			if !ok {
				writeStreamEvent(c, w, "done", "[DONE]")
				return false
			}
			writeStreamEvent(c, w, "chunk", chunk)
			return true
		case <-time.After(10 * time.Second):
			writeStreamEvent(c, w, "ping", "keepalive")
			return true
		}
	})
}

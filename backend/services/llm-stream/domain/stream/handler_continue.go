package stream

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) ContinueBlogStreamHandler(c *gin.Context) {
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

	chunkChan, errChan := newGenerateStreamChannels()

	ctx := c.Request.Context()

	go h.service.Continue(ctx, userID, blogID, chunkChan, errChan)

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
				writeStreamEvent(c, w, "error", externalStreamErrorMessage(streamOperationContinue, err))
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

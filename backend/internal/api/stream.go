package api

import (
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"inkwords-backend/internal/service"
)

// StreamAPI handles SSE streaming requests
type StreamAPI struct {
	generatorService *service.GeneratorService
}

// NewStreamAPI creates a new StreamAPI instance
func NewStreamAPI() *StreamAPI {
	return &StreamAPI{
		generatorService: service.NewGeneratorService(),
	}
}

// GenerateRequest represents the request body for generating a blog
type GenerateRequest struct {
	SourceContent string `json:"source_content" binding:"required"`
}

// GenerateBlogStreamHandler handles the /api/v1/stream/generate endpoint
func (api *StreamAPI) GenerateBlogStreamHandler(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	chunkChan := make(chan string)
	errChan := make(chan error)

	ctx := c.Request.Context()

	// Start generation in a goroutine
	go api.generatorService.GenerateBlogStream(ctx, req.SourceContent, chunkChan, errChan)

	// Set headers for SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			// Client disconnected
			return false
		case err, ok := <-errChan:
			if ok && err != nil {
				c.SSEvent("error", err.Error())
				return false
			}
			if !ok {
				errChan = nil // disable this case
			}
			return true
		case chunk, ok := <-chunkChan:
			if !ok {
				// Stream finished
				c.SSEvent("done", "[DONE]")
				return false
			}
			c.SSEvent("chunk", chunk)
			return true
		}
	})
}

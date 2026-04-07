package api

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"inkwords-backend/internal/service"
)

// StreamAPI handles SSE streaming requests
type StreamAPI struct {
	generatorService     *service.GeneratorService
	decompositionService *service.DecompositionService
}

// NewStreamAPI creates a new StreamAPI instance
func NewStreamAPI() *StreamAPI {
	return &StreamAPI{
		generatorService:     service.NewGeneratorService(),
		decompositionService: service.NewDecompositionService(),
	}
}

// GenerateRequest represents the request body for generating a blog
type GenerateRequest struct {
	SourceContent string            `json:"source_content"`
	SourceType    string            `json:"source_type"`
	Outline       []service.Chapter `json:"outline"`      // Optional outline for series generation
	GitURL        string            `json:"git_url"`      // For analyze stream
	SeriesTitle   string            `json:"series_title"` // Series title for parent blog
	ParentID      string            `json:"parent_id"`    // Optional parent ID for resuming series
}

// AnalyzeStreamHandler handles the /api/v1/stream/analyze endpoint
func (api *StreamAPI) AnalyzeStreamHandler(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.GitURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "git_url is required"})
		return
	}

	progressChan := make(chan string)
	errChan := make(chan error)

	bgCtx := context.WithoutCancel(c.Request.Context())
	ctx := c.Request.Context()

	// We use a WaitGroup to ensure the goroutine finishes before we return from the handler
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		api.decompositionService.AnalyzeStream(bgCtx, req.GitURL, progressChan, errChan)
	}()

	// Set headers for SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	// Read from channels until done or error
	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			// Client disconnected or request timed out
			// Drain channels in background so the generation task doesn't block
			go func() {
				for {
					select {
					case <-progressChan:
					case err, ok := <-errChan:
						if !ok || err != nil {
							return
						}
					}
				}
			}()
			return false
		case err, ok := <-errChan:
			if ok && err != nil {
				c.SSEvent("error", err.Error())
				return false
			}
			if !ok {
				errChan = nil
			}
			return true
		case msg, ok := <-progressChan:
			if !ok {
				c.SSEvent("done", "[DONE]")
				return false
			}
			c.SSEvent("chunk", msg)
			return true
		case <-time.After(15 * time.Second):
			// Keep-alive ping to prevent proxy timeout (e.g. Vite proxy times out after 120s)
			c.SSEvent("ping", "keepalive")
			return true
		}
	})

	// Wait for the goroutine to finish before exiting the handler
	// so we don't leak goroutines or write to closed channels.
	// But note: Gin's c.Stream blocks until it returns false,
	// so the context might already be done here.
	// We wait in a separate goroutine so we don't block the request handler forever
	// if something goes wrong, but we allow it to clean up.
	go wg.Wait()
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

	// Retrieve UserID from context if available, otherwise create a dummy one for testing
	var userID uuid.UUID
	if v, exists := c.Get("user_id"); exists {
		if id, ok := v.(uuid.UUID); ok {
			userID = id
		}
	}
	if userID == uuid.Nil {
		// Fallback to a dummy UUID if auth middleware is not applied on this route yet
		userID = uuid.New()
	}

	// Start generation in a goroutine
	if len(req.Outline) > 0 {
		// Series Generation
		var parentID uuid.UUID
		if req.ParentID != "" {
			parsedID, err := uuid.Parse(req.ParentID)
			if err == nil {
				parentID = parsedID
			}
		}
		if parentID == uuid.Nil {
			parentID = uuid.New()
		}
		go api.decompositionService.GenerateSeries(ctx, userID, parentID, req.SeriesTitle, req.Outline, req.SourceContent, req.SourceType, req.GitURL, chunkChan, errChan)
	} else {
		// Single blog stream
		go api.generatorService.GenerateBlogStream(ctx, userID, req.SourceContent, req.SourceType, chunkChan, errChan)
	}

	// Set headers for SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			// Client disconnected
			// Drain channels in background so the generation task doesn't block
			go func() {
				for {
					select {
					case <-chunkChan:
					case err, ok := <-errChan:
						if !ok || err != nil {
							return
						}
					}
				}
			}()
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
		case <-time.After(15 * time.Second):
			// Keep-alive ping
			c.SSEvent("ping", "keepalive")
			return true
		}
	})
}

// ContinueBlogStreamHandler handles the /api/v1/blogs/:id/continue endpoint
func (api *StreamAPI) ContinueBlogStreamHandler(c *gin.Context) {
	blogIDStr := c.Param("id")
	blogID, err := uuid.Parse(blogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blog ID"})
		return
	}

	var userID uuid.UUID
	if v, exists := c.Get("user_id"); exists {
		if id, ok := v.(uuid.UUID); ok {
			userID = id
		}
	}
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	chunkChan := make(chan string)
	errChan := make(chan error)

	bgCtx := context.WithoutCancel(c.Request.Context())
	ctx := c.Request.Context()

	go api.decompositionService.ContinueGeneration(bgCtx, userID, blogID, chunkChan, errChan)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			// Client disconnected
			go func() {
				for {
					select {
					case <-chunkChan:
					case err, ok := <-errChan:
						if !ok || err != nil {
							return
						}
					}
				}
			}()
			return false
		case err, ok := <-errChan:
			if ok && err != nil {
				c.SSEvent("error", err.Error())
				return false
			}
			if !ok {
				errChan = nil
			}
			return true
		case chunk, ok := <-chunkChan:
			if !ok {
				c.SSEvent("done", "[DONE]")
				return false
			}
			c.SSEvent("chunk", chunk)
			return true
		case <-time.After(15 * time.Second):
			// Keep-alive ping
			c.SSEvent("ping", "keepalive")
			return true
		}
	})
}

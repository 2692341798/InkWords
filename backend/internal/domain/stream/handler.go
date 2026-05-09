package stream

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service  *Service
	blogRepo BlogReadable
}

func NewHandler(service *Service, blogRepo BlogReadable) *Handler {
	return &Handler{service: service, blogRepo: blogRepo}
}

func (h *Handler) getUserID(c *gin.Context) uuid.UUID {
	if v, exists := c.Get("user_id"); exists {
		if id, ok := v.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

func (h *Handler) maybeCheckQuota(c *gin.Context) bool {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			if err := h.service.CheckQuota(uid); err != nil {
				c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
				return false
			}
		}
	}
	return true
}

func (h *Handler) GenerateBlogStreamHandler(c *gin.Context) {
	if !h.maybeCheckQuota(c) {
		return
	}

	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	chunkChan := make(chan string)
	errChan := make(chan error)

	ctx := c.Request.Context()

	userID := h.getUserID(c)
	if userID == uuid.Nil {
		userID = uuid.New()
	}

	go h.service.Generate(ctx, userID, req, chunkChan, errChan)

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
		case <-time.After(10 * time.Second):
			c.SSEvent("ping", "keepalive")
			return true
		}
	})
}

func (h *Handler) ContinueBlogStreamHandler(c *gin.Context) {
	if !h.maybeCheckQuota(c) {
		return
	}

	blogIDStr := c.Param("id")
	blogID, err := uuid.Parse(blogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blog ID"})
		return
	}

	userID := h.getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	chunkChan := make(chan string)
	errChan := make(chan error)

	bgCtx := context.WithoutCancel(c.Request.Context())
	ctx := c.Request.Context()

	go h.service.Continue(bgCtx, userID, blogID, chunkChan, errChan)

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
		case <-time.After(10 * time.Second):
			c.SSEvent("ping", "keepalive")
			return true
		}
	})
}

func (h *Handler) PolishBlogStreamHandler(c *gin.Context) {
	if !h.maybeCheckQuota(c) {
		return
	}

	blogIDStr := c.Param("id")
	blogID, err := uuid.Parse(blogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blog ID"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	chunkChan := make(chan string)
	errChan := make(chan error)

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
		case <-time.After(10 * time.Second):
			c.SSEvent("ping", "keepalive")
			return true
		}
	})
}

func (h *Handler) AnalyzeStreamHandler(c *gin.Context) {
	if !h.maybeCheckQuota(c) {
		return
	}

	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.SourceType != "file" && req.GitURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "git_url is required for git source type"})
		return
	}

	progressChan := make(chan string)
	errChan := make(chan error)

	bgCtx := context.WithoutCancel(c.Request.Context())
	ctx := c.Request.Context()

	var wg sync.WaitGroup
	wg.Add(1)

	userID := h.getUserID(c)
	if userID == uuid.Nil {
		userID = uuid.New()
	}

	go func() {
		defer wg.Done()
		h.service.AnalyzeStream(bgCtx, userID, req, progressChan, errChan)
	}()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	w := c.Writer
	w.Flush()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			go func() {
				progressOpen, errOpen := true, true
				for progressOpen || errOpen {
					select {
					case _, ok := <-progressChan:
						if !ok {
							progressOpen = false
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
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return true
		case <-time.After(10 * time.Second):
			c.SSEvent("ping", "keepalive")
			return true
		}
	})

	go wg.Wait()
}

func (h *Handler) ScanStreamHandler(c *gin.Context) {
	var req struct {
		GitURL string `json:"git_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	progressChan := make(chan string)
	errChan := make(chan error)
	resultChan := make(chan []ModuleCard)

	bgCtx := context.WithoutCancel(c.Request.Context())
	ctx := c.Request.Context()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		modules, err := h.service.ScanProjectModules(bgCtx, req.GitURL, progressChan)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- modules
	}()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	w := c.Writer
	w.Flush()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			go func() {
				for len(progressChan) > 0 {
					<-progressChan
				}
				for len(errChan) > 0 {
					<-errChan
				}
				for len(resultChan) > 0 {
					<-resultChan
				}
			}()
			return false
		case err, ok := <-errChan:
			if ok && err != nil {
				c.SSEvent("error", err.Error())
				return false
			}
			return true
		case msg, ok := <-progressChan:
			if !ok {
				return false
			}
			c.SSEvent("progress", msg)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return true
		case modules, ok := <-resultChan:
			if ok {
				c.SSEvent("result", modules)
				c.SSEvent("done", "[DONE]")
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
			return false
		case <-time.After(10 * time.Second):
			c.SSEvent("ping", "keepalive")
			return true
		}
	})

	go wg.Wait()
}

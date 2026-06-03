package stream

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"inkwords-backend/internal/prompt"
)

type streamOperation string

const (
	streamOperationGenerate streamOperation = "generate"
	streamOperationContinue streamOperation = "continue"
	streamOperationPolish   streamOperation = "polish"
	streamOperationAnalyze  streamOperation = "analyze"
	streamOperationScan     streamOperation = "scan"
	streamChannelBufferSize                 = 128
)

func newGenerateStreamChannels() (chan string, chan error) {
	return make(chan string, streamChannelBufferSize), make(chan error, 1)
}

func writeStreamEvent(c *gin.Context, w io.Writer, event string, payload interface{}) {
	c.SSEvent(event, payload)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

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
				c.JSON(http.StatusPaymentRequired, gin.H{"error": "quota exceeded"})
				return false
			}
		}
	}
	return true
}

func externalStreamErrorMessage(operation streamOperation, err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "request canceled"
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(err.Error()), "blog not found") {
		return "blog not found"
	}

	switch operation {
	case streamOperationContinue:
		return "blog continuation failed"
	case streamOperationPolish:
		return "blog polish failed"
	case streamOperationAnalyze:
		return "blog analysis failed"
	case streamOperationScan:
		return "project scan failed"
	default:
		return "blog generation failed"
	}
}

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
				writeStreamEvent(c, w, "error", externalStreamErrorMessage(streamOperationGenerate, err))
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

func (h *Handler) AnalyzeStreamHandler(c *gin.Context) {
	if !h.maybeCheckQuota(c) {
		return
	}

	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Why: 老前端或缓存中的静态资源可能没显式传 source_type，但文件上传链路仍会带 source_content。
	// 这里在后端做一次兼容推断，避免把文档解析误判成 git 分析。
	req.SourceType = resolveAnalyzeSourceType(req)
	req.ScenarioMode = string(normalizeScenarioMode(req.ScenarioMode, req.SourceType))

	if req.SourceType != "file" && req.GitURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "git_url is required for git source type"})
		return
	}

	progressChan := make(chan string, streamChannelBufferSize)
	errChan := make(chan error, 1)

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
				writeStreamEvent(c, w, "error", externalStreamErrorMessage(streamOperationAnalyze, err))
				return false
			}
			if !ok {
				errChan = nil
			}
			return true
		case msg, ok := <-progressChan:
			if !ok {
				writeStreamEvent(c, w, "done", "[DONE]")
				return false
			}
			writeStreamEvent(c, w, "chunk", msg)
			return true
		case <-time.After(10 * time.Second):
			writeStreamEvent(c, w, "ping", "keepalive")
			return true
		}
	})

	go wg.Wait()
}

func resolveAnalyzeSourceType(req GenerateRequest) string {
	if req.SourceType != "" {
		return req.SourceType
	}
	if req.SourceContent != "" && req.GitURL == "" {
		return "file"
	}
	return "git"
}

func normalizeScenarioMode(raw string, sourceType string) prompt.ScenarioMode {
	mode := prompt.ScenarioMode(raw)
	if mode.IsValid() {
		return mode
	}
	return prompt.DefaultScenarioModeForSource(sourceType)
}

func (h *Handler) ScanStreamHandler(c *gin.Context) {
	var req struct {
		GitURL string `json:"git_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	progressChan := make(chan string, streamChannelBufferSize)
	errChan := make(chan error, 1)
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
				writeStreamEvent(c, w, "error", externalStreamErrorMessage(streamOperationScan, err))
				return false
			}
			return true
		case msg, ok := <-progressChan:
			if !ok {
				return false
			}
			writeStreamEvent(c, w, "progress", msg)
			return true
		case modules, ok := <-resultChan:
			if ok {
				writeStreamEvent(c, w, "result", modules)
				writeStreamEvent(c, w, "done", "[DONE]")
			}
			return false
		case <-time.After(10 * time.Second):
			writeStreamEvent(c, w, "ping", "keepalive")
			return true
		}
	})

	go wg.Wait()
}

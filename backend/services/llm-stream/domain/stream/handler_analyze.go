package stream

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"inkwords-backend/shared/kernel/prompt"
)

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

	ctx := c.Request.Context()

	var wg sync.WaitGroup
	wg.Add(1)

	userID := h.getUserID(c)
	if userID == uuid.Nil {
		userID = uuid.New()
	}

	go func() {
		defer wg.Done()
		h.service.AnalyzeStream(ctx, userID, req, progressChan, errChan)
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

	ctx := c.Request.Context()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		modules, err := h.service.ScanProjectModules(ctx, req.GitURL, progressChan)
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

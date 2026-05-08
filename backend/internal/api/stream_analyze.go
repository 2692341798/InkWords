package api

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (api *StreamAPI) AnalyzeStreamHandler(c *gin.Context) {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			if err := api.userService.CheckQuota(uid); err != nil {
				c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
				return
			}
		}
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

	var userID uuid.UUID
	if v, exists := c.Get("user_id"); exists {
		if id, ok := v.(uuid.UUID); ok {
			userID = id
		}
	}
	if userID == uuid.Nil {
		userID = uuid.New()
	}

	go func() {
		defer wg.Done()
		if req.SourceType == "file" {
			api.decompositionService.AnalyzeFileStream(bgCtx, userID, req.SourceContent, progressChan, errChan)
		} else {
			api.decompositionService.AnalyzeStream(bgCtx, userID, req.GitURL, req.SelectedModules, progressChan, errChan)
		}
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


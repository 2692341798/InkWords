package api

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"inkwords-backend/internal/service"
)

func (api *StreamAPI) ScanStreamHandler(c *gin.Context) {
	var req struct {
		GitURL string `json:"git_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	progressChan := make(chan string)
	errChan := make(chan error)
	resultChan := make(chan []service.ModuleCard)

	bgCtx := context.WithoutCancel(c.Request.Context())
	ctx := c.Request.Context()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		modules, err := api.decompositionService.ScanProjectModulesWithProgress(bgCtx, req.GitURL, progressChan)
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


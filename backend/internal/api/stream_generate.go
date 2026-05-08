package api

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (api *StreamAPI) GenerateBlogStreamHandler(c *gin.Context) {
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

	chunkChan := make(chan string)
	errChan := make(chan error)

	ctx := c.Request.Context()

	var userID uuid.UUID
	if v, exists := c.Get("user_id"); exists {
		if id, ok := v.(uuid.UUID); ok {
			userID = id
		}
	}
	if userID == uuid.Nil {
		userID = uuid.New()
	}

	if len(req.Outline) > 0 {
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
		go api.generatorService.GenerateBlogStream(ctx, userID, req.SourceContent, req.SourceType, chunkChan, errChan)
	}

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


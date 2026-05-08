package api

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"inkwords-backend/internal/db"
	"inkwords-backend/internal/model"
)

func (api *StreamAPI) PolishBlogStreamHandler(c *gin.Context) {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			if err := api.userService.CheckQuota(uid); err != nil {
				c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
				return
			}
		}
	}

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

	var blog model.Blog
	if err := db.DB.WithContext(c.Request.Context()).Select("id").First(&blog, "id = ? AND user_id = ?", blogID, userID).Error; err != nil {
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
	go api.generatorService.GeneratePolishDraftStream(ctx, req.Title, req.Content, chunkChan, errChan)

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


package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"inkwords-backend/internal/service"
)

func (a *BlogAPI) CreateDraftBlog(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    http.StatusUnauthorized,
			"message": "未授权的访问",
			"data":    nil,
		})
		return
	}

	uid, ok := userIDStr.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "用户 ID 类型错误",
			"data":    nil,
		})
		return
	}

	draft, err := a.blogService.CreateDraftBlog(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	node := &service.BlogNode{
		ID:          draft.ID,
		Title:       draft.Title,
		Content:     draft.Content,
		SourceType:  draft.SourceType,
		Status:      draft.Status,
		ChapterSort: draft.ChapterSort,
		ParentID:    draft.ParentID,
		CreatedAt:   draft.CreatedAt,
		UpdatedAt:   draft.UpdatedAt,
		Children:    []*service.BlogNode{},
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data":    node,
	})
}


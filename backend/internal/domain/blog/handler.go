package blog

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LegacyExporter 定义 Phase 1 过渡期复用旧 service 导出能力的接口。
type LegacyExporter interface {
	ExportSeriesToObsidian(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) error
	ExportToObsidian(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) error
	ExportSeriesToPDF(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) (pdfPath string, filename string, err error)
}

// Handler 提供 Blog 领域的 HTTP 适配层。
type Handler struct {
	service        *Service
	legacyExporter LegacyExporter
}

// NewHandler 创建 Blog Handler。
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// NewHandlerWithLegacy 创建 Blog Handler，并注入兼容层导出能力（Phase 1 过渡使用）。
func NewHandlerWithLegacy(service *Service, legacyExporter LegacyExporter) *Handler {
	return &Handler{service: service, legacyExporter: legacyExporter}
}

func (h *Handler) GetUserBlogs(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "unauthorized", "data": nil})
		return
	}

	uid, ok := userIDStr.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "internal server error", "data": nil})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	blogs, err := h.service.GetUserBlogs(c.Request.Context(), uid, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "failed to load blogs", "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "success", "data": blogs})
}

func (h *Handler) CreateDraftBlog(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "unauthorized", "data": nil})
		return
	}

	uid, ok := userIDStr.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "internal server error", "data": nil})
		return
	}

	draft, err := h.service.CreateDraftBlog(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "failed to create draft blog", "data": nil})
		return
	}

	node := &BlogNode{
		ID:          draft.ID,
		Title:       draft.Title,
		Content:     draft.Content,
		SourceType:  draft.SourceType,
		Status:      draft.Status,
		ChapterSort: draft.ChapterSort,
		ParentID:    draft.ParentID,
		CreatedAt:   draft.CreatedAt,
		UpdatedAt:   draft.UpdatedAt,
		Children:    []*BlogNode{},
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "success", "data": node})
}

func (h *Handler) BatchDeleteBlogs(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "unauthorized", "data": nil})
		return
	}

	uid, ok := userIDStr.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "internal server error", "data": nil})
		return
	}

	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "invalid request body", "data": nil})
		return
	}

	if len(req.BlogIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "blog_ids must not be empty", "data": nil})
		return
	}

	if err := h.service.BatchDeleteBlogs(c.Request.Context(), uid, req.BlogIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "failed to delete blogs", "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "success", "data": nil})
}

func (h *Handler) UpdateBlog(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "unauthorized", "data": nil})
		return
	}

	uid, ok := userIDStr.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "internal server error", "data": nil})
		return
	}

	blogIDStr := c.Param("id")
	blogID, err := uuid.Parse(blogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "invalid blog id", "data": nil})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "invalid request body", "data": nil})
		return
	}

	if err := h.service.UpdateBlog(c.Request.Context(), blogID, uid, req); err != nil {
		if errors.Is(err, ErrBlogNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": "blog not found", "data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "failed to update blog", "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "success", "data": nil})
}

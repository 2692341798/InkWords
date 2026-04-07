package api

import (
	"archive/zip"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"inkwords-backend/internal/service"
)

type BlogAPI struct {
	blogService *service.BlogService
}

func NewBlogAPI() *BlogAPI {
	return &BlogAPI{
		blogService: service.NewBlogService(),
	}
}

// GetUserBlogs 获取当前用户的博客列表
func (a *BlogAPI) GetUserBlogs(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    http.StatusUnauthorized,
			"message": "unauthorized",
			"data":    nil,
		})
		return
	}

	uid, ok := userIDStr.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "invalid user id type",
			"data":    nil,
		})
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

	blogs, err := a.blogService.GetUserBlogs(c.Request.Context(), uid, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data":    blogs,
	})
}

// BatchDeleteBlogsRequest 批量删除请求体
type BatchDeleteBlogsRequest struct {
	BlogIDs []uuid.UUID `json:"blog_ids" binding:"required"`
}

// BatchDeleteBlogs 批量删除博客
func (a *BlogAPI) BatchDeleteBlogs(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    http.StatusUnauthorized,
			"message": "unauthorized",
			"data":    nil,
		})
		return
	}

	uid, ok := userIDStr.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "invalid user id type",
			"data":    nil,
		})
		return
	}

	var req BatchDeleteBlogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "invalid request body",
			"data":    nil,
		})
		return
	}

	if len(req.BlogIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "blog_ids cannot be empty",
			"data":    nil,
		})
		return
	}

	if err := a.blogService.BatchDeleteBlogs(c.Request.Context(), uid, req.BlogIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data":    nil,
	})
}

// UpdateBlog 更新博客内容
func (a *BlogAPI) UpdateBlog(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    http.StatusUnauthorized,
			"message": "unauthorized",
			"data":    nil,
		})
		return
	}

	uid, ok := userIDStr.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "invalid user id type",
			"data":    nil,
		})
		return
	}

	blogIDStr := c.Param("id")
	blogID, err := uuid.Parse(blogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "invalid blog id",
			"data":    nil,
		})
		return
	}

	var req service.UpdateBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "invalid request body",
			"data":    nil,
		})
		return
	}

	if err := a.blogService.UpdateBlog(c.Request.Context(), blogID, uid, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data":    nil,
	})
}

// ExportSeries 导出系列博客为 ZIP 包
func (a *BlogAPI) ExportSeries(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    http.StatusUnauthorized,
			"message": "unauthorized",
			"data":    nil,
		})
		return
	}

	uid, ok := userIDStr.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "invalid user id type",
			"data":    nil,
		})
		return
	}

	blogIDStr := c.Param("id")
	blogID, err := uuid.Parse(blogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "invalid blog id",
			"data":    nil,
		})
		return
	}

	blogs, err := a.blogService.GetSeriesBlogs(c.Request.Context(), blogID, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	if len(blogs) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    http.StatusNotFound,
			"message": "series not found",
			"data":    nil,
		})
		return
	}

	parentTitle := blogs[0].Title
	if parentTitle == "" {
		parentTitle = "series"
	}

	c.Writer.Header().Set("Content-Type", "application/zip")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", parentTitle))

	zw := zip.NewWriter(c.Writer)

	for i, blog := range blogs {
		title := blog.Title
		if title == "" {
			title = fmt.Sprintf("未命名_%d", i)
		}

		filename := ""
		if blog.ParentID == nil || *blog.ParentID == uuid.Nil {
			filename = fmt.Sprintf("%s.md", title)
		} else {
			filename = fmt.Sprintf("%02d-%s.md", blog.ChapterSort, title)
		}

		f, err := zw.Create(filename)
		if err != nil {
			continue
		}

		// 内容前面可以加上标题
		// 如果原有内容中已经包含了标题，可以考虑不再重复，但简单处理直接追加
		_, _ = f.Write([]byte(fmt.Sprintf("# %s\n\n%s", title, blog.Content)))
	}

	if err := zw.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "failed to create zip",
			"data":    nil,
		})
		return
	}
}

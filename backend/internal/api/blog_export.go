package api

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"inkwords-backend/internal/service"
)

func (a *BlogAPI) ExportSeriesToObsidian(c *gin.Context) {
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

	blogIDStr := c.Param("id")
	blogID, err := uuid.Parse(blogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "无效的博客 ID",
			"data":    nil,
		})
		return
	}

	if err := a.blogService.ExportSeriesToObsidian(c.Request.Context(), blogID, uid); err != nil {
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

func (a *BlogAPI) ExportSeries(c *gin.Context) {
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

	blogIDStr := c.Param("id")
	blogID, err := uuid.Parse(blogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "无效的博客 ID",
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
			"message": "找不到该系列博客",
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

		_, _ = f.Write([]byte(fmt.Sprintf("# %s\n\n%s", title, blog.Content)))
	}

	if err := zw.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "创建 ZIP 包失败",
			"data":    nil,
		})
		return
	}
}

func (a *BlogAPI) ExportSeriesPDF(c *gin.Context) {
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

	blogIDStr := c.Param("id")
	blogID, err := uuid.Parse(blogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "无效的博客 ID",
			"data":    nil,
		})
		return
	}

	pdfPath, filename, err := a.blogService.ExportSeriesToPDF(c.Request.Context(), blogID, uid)
	if err != nil {
		if errors.Is(err, service.ErrSeriesNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    http.StatusNotFound,
				"message": "找不到该系列博客",
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	f, err := os.Open(pdfPath)
	if err != nil {
		_ = os.Remove(pdfPath)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "读取 PDF 失败",
			"data":    nil,
		})
		return
	}
	defer f.Close()

	c.Writer.Header().Set("Content-Type", "application/pdf")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, f)
	_ = os.Remove(pdfPath)
}

func (a *BlogAPI) ExportToObsidian(c *gin.Context) {
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

	blogIDStr := c.Param("id")
	blogID, err := uuid.Parse(blogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "无效的博客 ID",
			"data":    nil,
		})
		return
	}

	if err := a.blogService.ExportToObsidian(c.Request.Context(), blogID, uid); err != nil {
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


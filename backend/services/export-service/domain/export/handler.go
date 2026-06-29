package export

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ExportSeriesToObsidian(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	blogID, ok := blogIDParam(c)
	if !ok {
		return
	}

	if err := h.service.ExportSeriesToObsidian(c.Request.Context(), blogID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "success", "data": nil})
}

func (h *Handler) ExportSeries(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	blogID, ok := blogIDParam(c)
	if !ok {
		return
	}

	blogs, err := h.service.GetSeriesBlogs(c.Request.Context(), blogID, userID)
	if err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": "找不到该系列博客", "data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": err.Error(), "data": nil})
		return
	}

	c.Writer.Header().Set("Content-Type", "application/zip")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", seriesParentTitle(blogs)))

	if err := writeSeriesZip(c.Writer, blogs); err != nil {
		log.Printf("series zip write failed: %v", err)
		return
	}
}

//nolint:gosec
func (h *Handler) ExportSeriesPDF(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	blogID, ok := blogIDParam(c)
	if !ok {
		return
	}

	pdfPath, filename, err := h.service.ExportSeriesToPDF(c.Request.Context(), blogID, userID)
	if err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": "找不到该系列博客", "data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": err.Error(), "data": nil})
		return
	}

	file, err := os.Open(pdfPath)
	if err != nil {
		_ = os.Remove(pdfPath)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "读取 PDF 失败", "data": nil})
		return
	}
	defer func() { _ = file.Close() }()

	c.Writer.Header().Set("Content-Type", "application/pdf")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Status(http.StatusOK)

	if _, err := io.Copy(c.Writer, file); err != nil {
		log.Printf("pdf copy to response failed for %s: %v", blogID, err)
	}
	if err := os.Remove(pdfPath); err != nil {
		log.Printf("pdf temp file cleanup failed for %s: %v", pdfPath, err)
	}
}

func (h *Handler) ExportToObsidian(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	blogID, ok := blogIDParam(c)
	if !ok {
		return
	}

	if err := h.service.ExportToObsidian(c.Request.Context(), blogID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "success", "data": nil})
}

// writeSeriesZip 将系列博客归档写入 io.Writer，ZIP 创建或写入失败时返回错误。
// 提取为独立可测试函数，避免归档构造逻辑与 HTTP 响应耦合。
func writeSeriesZip(w io.Writer, blogs []Blog) error {
	zipWriter := zip.NewWriter(w)
	defer func() {
		if closeErr := zipWriter.Close(); closeErr != nil {
			log.Printf("series zip close failed: %v", closeErr)
		}
	}()

	for idx, blog := range blogs {
		title := blog.Title
		if title == "" {
			title = fmt.Sprintf("未命名_%d", idx)
		}

		filename := ""
		if blog.ParentID == nil || *blog.ParentID == uuid.Nil {
			filename = fmt.Sprintf("%s.md", title)
		} else {
			filename = fmt.Sprintf("%02d-%s.md", blog.ChapterSort, title)
		}

		file, err := zipWriter.Create(filename)
		if err != nil {
			return fmt.Errorf("zip create entry %q: %w", filename, err)
		}
		if _, err := fmt.Fprintf(file, "# %s\n\n%s", title, blog.Content); err != nil {
			return fmt.Errorf("zip write entry %q: %w", filename, err)
		}
	}
	return nil
}

// seriesParentTitle 从系列博客列表中提取父级标题，用于生成下载文件名。
func seriesParentTitle(blogs []Blog) string {
	if len(blogs) == 0 {
		return "series"
	}
	if blogs[0].Title != "" {
		return blogs[0].Title
	}
	return "series"
}

func currentUserID(c *gin.Context) (uuid.UUID, bool) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "未授权的访问", "data": nil})
		return uuid.Nil, false
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "用户 ID 类型错误", "data": nil})
		return uuid.Nil, false
	}

	return userID, true
}

func blogIDParam(c *gin.Context) (uuid.UUID, bool) {
	blogID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "无效的博客 ID", "data": nil})
		return uuid.Nil, false
	}
	return blogID, true
}

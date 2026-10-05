package parse

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type parseService interface {
	Parse(io.Reader, string) (ParseResult, error)
}

// Handler exposes the parser-service HTTP endpoint for file parsing.
type Handler struct {
	service parseService
}

// NewHandler creates a parser-service HTTP handler for the local workspace.
func NewHandler(service parseService) *Handler {
	return &Handler{service: service}
}

// Parse handles the authenticated multipart upload endpoint for parser-service.
func (h *Handler) Parse(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "获取上传文件失败: " + err.Error(),
			"data":    nil,
		})
		return
	}
	defer func() { _ = file.Close() }()

	if header.Size == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "上传的文件为空",
			"data":    nil,
		})
		return
	}

	result, err := h.service.Parse(file, header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "解析文件失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	response := gin.H{
		"source_content": result.SourceContent,
	}
	if result.ArchiveSummary != nil {
		response["archive_summary"] = result.ArchiveSummary
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data":    response,
	})
}

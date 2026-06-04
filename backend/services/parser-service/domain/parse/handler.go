package parse

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type quotaChecker interface {
	CheckQuota(uuid.UUID) error
}

type parseService interface {
	Parse(io.Reader, string) (ParseResult, error)
}

// Handler exposes the parser-service HTTP endpoint for file parsing.
type Handler struct {
	service      parseService
	quotaChecker quotaChecker
}

// NewHandler creates a parser-service HTTP handler with quota enforcement support.
func NewHandler(service parseService, quotaChecker quotaChecker) *Handler {
	return &Handler{
		service:      service,
		quotaChecker: quotaChecker,
	}
}

func (h *Handler) getUserID(c *gin.Context) (uuid.UUID, bool) {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			return uid, true
		}
	}
	return uuid.Nil, false
}

// Parse handles the authenticated multipart upload endpoint for parser-service.
func (h *Handler) Parse(c *gin.Context) {
	if h.quotaChecker != nil {
		if userID, ok := h.getUserID(c); ok {
			if err := h.quotaChecker.CheckQuota(userID); err != nil {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"code":    http.StatusPaymentRequired,
					"message": err.Error(),
					"data":    nil,
				})
				return
			}
		}
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "获取上传文件失败: " + err.Error(),
			"data":    nil,
		})
		return
	}
	defer file.Close()

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

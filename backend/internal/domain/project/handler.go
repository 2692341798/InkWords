package project

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) getUserID(c *gin.Context) (uuid.UUID, bool) {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			return uid, true
		}
	}
	return uuid.Nil, false
}

func (h *Handler) ScanGithubRepo(c *gin.Context) {
	if userID, ok := h.getUserID(c); ok {
		if err := h.service.CheckQuota(userID); err != nil {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"code":    http.StatusPaymentRequired,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
	}

	var req ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "请求参数格式错误",
			"data":    nil,
		})
		return
	}

	ctx := c.Request.Context()
	modules, err := h.service.ScanProjectModules(ctx, req.GitURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "扫描仓库失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"repo_url": req.GitURL,
			"modules":  modules,
		},
	})
}

func (h *Handler) Analyze(c *gin.Context) {
	if userID, ok := h.getUserID(c); ok {
		if err := h.service.CheckQuota(userID); err != nil {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"code":    http.StatusPaymentRequired,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
	}

	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "请求参数格式错误",
			"data":    nil,
		})
		return
	}

	ctx := c.Request.Context()
	outline, content, stage, err := h.service.Analyze(ctx, req.GitURL, req.SubDir)
	if err != nil {
		if stage == "fetch" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "拉取 Git 仓库失败: " + err.Error(),
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "生成大纲失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data": gin.H{
			"outline":        outline,
			"source_content": content,
		},
	})
}

func (h *Handler) Parse(c *gin.Context) {
	if userID, ok := h.getUserID(c); ok {
		if err := h.service.CheckQuota(userID); err != nil {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"code":    http.StatusPaymentRequired,
				"message": err.Error(),
				"data":    nil,
			})
			return
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

	content, err := h.service.Parse(file, header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "解析文件失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data": gin.H{
			"source_content": content,
		},
	})
}

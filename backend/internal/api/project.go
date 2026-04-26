package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"inkwords-backend/internal/parser"
	"inkwords-backend/internal/service"
)

type ProjectAPI struct {
	decompositionService *service.DecompositionService
	gitFetcher           *parser.GitFetcher
	docParser            *parser.DocParser
	userService          *service.UserService
}

func NewProjectAPI(userService *service.UserService) *ProjectAPI {
	return &ProjectAPI{
		decompositionService: service.NewDecompositionService(),
		gitFetcher:           parser.NewGitFetcher(),
		docParser:            parser.NewDocParser(),
		userService:          userService,
	}
}

type ScanRequest struct {
	GitURL string `json:"git_url" binding:"required"`
}

// ScanGithubRepo handles the /api/v1/project/scan endpoint
func (api *ProjectAPI) ScanGithubRepo(c *gin.Context) {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			if err := api.userService.CheckQuota(uid); err != nil {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"code":    http.StatusPaymentRequired,
					"message": err.Error(),
					"data":    nil,
				})
				return
			}
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

	modules, err := api.decompositionService.ScanProjectModules(ctx, req.GitURL)
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

type AnalyzeRequest struct {
	GitURL string `json:"git_url" binding:"required"`
	SubDir string `json:"sub_dir"`
}

// Analyze handles the /api/v1/project/analyze endpoint
func (api *ProjectAPI) Analyze(c *gin.Context) {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			if err := api.userService.CheckQuota(uid); err != nil {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"code":    http.StatusPaymentRequired,
					"message": err.Error(),
					"data":    nil,
				})
				return
			}
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

	// 1. Fetch Git content
	treeContent, chunks, err := api.gitFetcher.FetchWithSubDir(req.GitURL, req.SubDir, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "拉取 Git 仓库失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// For the legacy non-streaming endpoint, we'll just concatenate the chunks
	var fullContentBuilder strings.Builder
	fullContentBuilder.WriteString(treeContent)
	fullContentBuilder.WriteString("\n=== Repository Content ===\n")
	for _, chunk := range chunks {
		fullContentBuilder.WriteString(chunk.Content)
	}
	content := fullContentBuilder.String()

	// 2. Generate Outline
	outline, err := api.decompositionService.GenerateOutline(ctx, content, nil, nil)
	if err != nil {
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

// Parse handles the /api/v1/project/parse endpoint
func (api *ProjectAPI) Parse(c *gin.Context) {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			if err := api.userService.CheckQuota(uid); err != nil {
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

	content, err := api.docParser.Parse(file, header.Filename)
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

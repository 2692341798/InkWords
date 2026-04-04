package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"inkwords-backend/internal/parser"
	"inkwords-backend/internal/service"
)

type ProjectAPI struct {
	decompositionService *service.DecompositionService
	gitFetcher           *parser.GitFetcher
}

func NewProjectAPI() *ProjectAPI {
	return &ProjectAPI{
		decompositionService: service.NewDecompositionService(),
		gitFetcher:           parser.NewGitFetcher(),
	}
}

type AnalyzeRequest struct {
	GitURL string `json:"git_url" binding:"required"`
}

// Analyze handles the /api/v1/project/analyze endpoint
func (api *ProjectAPI) Analyze(c *gin.Context) {
	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Invalid request body",
			"data":    nil,
		})
		return
	}

	ctx := c.Request.Context()

	// 1. Fetch Git content
	content, err := api.gitFetcher.Fetch(req.GitURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to fetch git repository: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 2. Generate Outline
	outline, err := api.decompositionService.GenerateOutline(ctx, content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to generate outline: " + err.Error(),
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
package api

import (
	"github.com/gin-gonic/gin"

	projectdomain "inkwords-backend/internal/domain/project"
	"inkwords-backend/internal/service"
	"inkwords-backend/shared/platform/parser"
)

type ProjectAPI struct {
	decompositionService *service.DecompositionService
	gitFetcher           *parser.GitFetcher
	docParser            *parser.DocParser
	userService          *service.UserService
	projectDomainHandler *projectdomain.Handler
}

// ScanGithubRepo handles the /api/v1/project/scan endpoint
func (api *ProjectAPI) ScanGithubRepo(c *gin.Context) {
	api.projectDomainHandler.ScanGithubRepo(c)
}

// Analyze handles the /api/v1/project/analyze endpoint
func (api *ProjectAPI) Analyze(c *gin.Context) {
	api.projectDomainHandler.Analyze(c)
}

// Parse handles the /api/v1/project/parse endpoint
func (api *ProjectAPI) Parse(c *gin.Context) {
	api.projectDomainHandler.Parse(c)
}

func NewProjectAPIWithDeps(decompositionService *service.DecompositionService, gitFetcher *parser.GitFetcher, docParser *parser.DocParser, userService *service.UserService, projectDomainHandler *projectdomain.Handler) *ProjectAPI {
	return &ProjectAPI{
		decompositionService: decompositionService,
		gitFetcher:           gitFetcher,
		docParser:            docParser,
		userService:          userService,
		projectDomainHandler: projectDomainHandler,
	}
}

package api

import (
	"github.com/gin-gonic/gin"

	projectdomain "inkwords-backend/internal/domain/project"
	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/infra/parser"
	"inkwords-backend/internal/service"
)

type ProjectAPI struct {
	decompositionService *service.DecompositionService
	gitFetcher           *parser.GitFetcher
	docParser            *parser.DocParser
	userService          *service.UserService
	projectDomainHandler *projectdomain.Handler
}

func NewProjectAPI(userService *service.UserService) *ProjectAPI {
	promptReqService := service.NewPromptRequirementsService(db.DB)
	decompositionService := service.NewDecompositionService(promptReqService)
	gitFetcher := parser.NewGitFetcher()
	docParser := parser.NewDocParser()
	projectService := projectdomain.NewService(decompositionService, gitFetcher, docParser, userService)
	return NewProjectAPIWithDeps(decompositionService, gitFetcher, docParser, userService, projectdomain.NewHandler(projectService))
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

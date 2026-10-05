package v1

import "github.com/gin-gonic/gin"

// ParseHandler describes the service-owned parse endpoint contract.
type ParseHandler interface {
	Parse(*gin.Context)
}

// CrawlHandler describes the bounded documentation-stack manifest endpoint.
type CrawlHandler interface {
	Crawl(*gin.Context)
}

// RegisterParserRoutes registers the parser-service HTTP surface without leaking shared route aggregators.
func RegisterParserRoutes(r *gin.Engine, workspaceMiddleware gin.HandlerFunc, handler ParseHandler, crawlHandler CrawlHandler) {
	if workspaceMiddleware == nil {
		panic("missing middleware: workspaceMiddleware")
	}
	v1 := r.Group("/api/v1")
	projectGroup := v1.Group("/project")
	projectGroup.Use(workspaceMiddleware)
	projectGroup.POST("/parse", handler.Parse)
	projectGroup.POST("/crawl", crawlHandler.Crawl)
}

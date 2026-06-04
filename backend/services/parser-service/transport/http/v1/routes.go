package v1

import "github.com/gin-gonic/gin"

// ParseHandler describes the service-owned parse endpoint contract.
type ParseHandler interface {
	Parse(*gin.Context)
}

// RegisterParserRoutes registers the parser-service HTTP surface without leaking shared route aggregators.
func RegisterParserRoutes(r *gin.Engine, authMiddleware gin.HandlerFunc, handler ParseHandler) {
	v1 := r.Group("/api/v1")
	projectGroup := v1.Group("/project")
	projectGroup.Use(authMiddleware)
	projectGroup.POST("/parse", handler.Parse)
}

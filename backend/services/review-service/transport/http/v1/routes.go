package v1

import (
	"github.com/gin-gonic/gin"

	reviewdomain "inkwords-backend/services/review-service/domain/review"
)

// RegisterReviewRoutes wires the review-service owned HTTP surface without pulling unrelated legacy routes.
func RegisterReviewRoutes(r *gin.Engine, workspaceMiddleware gin.HandlerFunc, handler *reviewdomain.Handler) {
	v1 := r.Group("/api/v1")
	reviewGroup := v1.Group("/review")
	reviewGroup.Use(workspaceMiddleware)
	reviewGroup.GET("/today", handler.GetTodayCard)
	reviewGroup.GET("/history", handler.GetHistory)
	reviewGroup.POST("/pick", handler.PickRandom)
	reviewGroup.GET("/notes", handler.ListNotes)
	reviewGroup.POST("/sessions", handler.CreateSession)
	reviewGroup.GET("/sessions/:id", handler.GetSession)
	reviewGroup.POST("/sessions/:id/reading-complete", handler.CompleteReading)
	reviewGroup.POST("/sessions/:id/respond", handler.Respond)
	reviewGroup.POST("/sessions/:id/hint", handler.RequestHint)
	reviewGroup.POST("/sessions/:id/finish", handler.Finish)
}

// RegisterLegacyReviewMigrationRoutes exposes the explicit conversion path only
// inside the installation's one resolved workspace.
func RegisterLegacyReviewMigrationRoutes(r *gin.Engine, workspaceMiddleware gin.HandlerFunc, handler *reviewdomain.Handler) {
	v1 := r.Group("/api/v1")
	migrationGroup := v1.Group("/review/legacy-objectives")
	migrationGroup.Use(workspaceMiddleware)
	migrationGroup.POST("", handler.MigrateLegacyNote)
}

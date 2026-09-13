package v1

import (
	"github.com/gin-gonic/gin"

	masterydomain "inkwords-backend/services/review-service/domain/mastery"
)

// RegisterMasteryRoutes exposes due work through the installation-owned local workspace.
func RegisterMasteryRoutes(r *gin.Engine, workspaceMiddleware gin.HandlerFunc, handler *masterydomain.Handler) {
	v1 := r.Group("/api/v1")
	masteryGroup := v1.Group("/mastery")
	masteryGroup.Use(workspaceMiddleware)
	masteryGroup.GET("/due", handler.Due)
	masteryGroup.POST("/objectives", handler.CreateObjective)
	masteryGroup.GET("/objectives/:id", handler.Workspace)
	masteryGroup.POST("/objectives/:id/attempts", handler.RecordAttempt)
	masteryGroup.GET("/objectives/:id/attempts/:attemptID/code", handler.LearnerArtifact)
	masteryGroup.POST("/objectives/:id/practice-sessions", handler.BeginPracticeSession)
	masteryGroup.GET("/objectives/:id/practice-sessions/:sessionID", handler.LoadPracticeSession)
	masteryGroup.POST("/objectives/:id/practice-sessions/:sessionID/help", handler.RevealPracticeHelp)
}

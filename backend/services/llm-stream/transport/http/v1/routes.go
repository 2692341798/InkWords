package v1

import "github.com/gin-gonic/gin"

// StreamHandlers defines the service-owned llm-stream HTTP surface.
type StreamHandlers struct {
	ContinueBlog gin.HandlerFunc
	PolishBlog   gin.HandlerFunc
	Scan         gin.HandlerFunc
	Analyze      gin.HandlerFunc
	Generate     gin.HandlerFunc
}

// RegisterStreamRoutes wires the rollback-compatible stream routes for llm-stream.
func RegisterStreamRoutes(r *gin.Engine, workspaceMiddleware gin.HandlerFunc, h StreamHandlers) {
	if workspaceMiddleware == nil {
		panic("missing middleware: workspaceMiddleware")
	}

	must(h.ContinueBlog, "ContinueBlog")
	must(h.PolishBlog, "PolishBlog")
	must(h.Scan, "Scan")
	must(h.Analyze, "Analyze")
	must(h.Generate, "Generate")

	v1 := r.Group("/api/v1")

	blogGroup := v1.Group("/blogs")
	blogGroup.Use(workspaceMiddleware)
	blogGroup.POST("/:id/continue", h.ContinueBlog)
	blogGroup.POST("/:id/polish", h.PolishBlog)

	streamGroup := v1.Group("/stream")
	streamGroup.Use(workspaceMiddleware)
	streamGroup.POST("/scan", h.Scan)
	streamGroup.POST("/analyze", h.Analyze)
	streamGroup.POST("/generate", h.Generate)
}

func must(handler gin.HandlerFunc, name string) {
	if handler == nil {
		panic("missing handler: " + name)
	}
}

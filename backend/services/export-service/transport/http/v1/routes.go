package v1

import (
	"github.com/gin-gonic/gin"

	transportv1api "inkwords-backend/internal/transport/http/v1/api"
)

type exportRouteHandlers struct {
	ExportSeries           gin.HandlerFunc
	ExportSeriesPDF        gin.HandlerFunc
	ExportToObsidian       gin.HandlerFunc
	ExportSeriesToObsidian gin.HandlerFunc
}

// RegisterExportRoutes wires only export-service owned endpoints onto the shared API surface.
func RegisterExportRoutes(r *gin.Engine, authMiddleware gin.HandlerFunc, blogAPI *transportv1api.BlogAPI) {
	if blogAPI == nil {
		panic("missing dependency: blogAPI")
	}

	registerExportRoutes(r, authMiddleware, exportRouteHandlers{
		ExportSeries:           blogAPI.ExportSeries,
		ExportSeriesPDF:        blogAPI.ExportSeriesPDF,
		ExportToObsidian:       blogAPI.ExportToObsidian,
		ExportSeriesToObsidian: blogAPI.ExportSeriesToObsidian,
	})
}

func registerExportRoutes(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers exportRouteHandlers) {
	if authMiddleware == nil {
		panic("missing middleware: authMiddleware")
	}
	must(handlers.ExportSeries, "Export.ExportSeries")
	must(handlers.ExportSeriesPDF, "Export.ExportSeriesPDF")
	must(handlers.ExportToObsidian, "Export.ExportToObsidian")
	must(handlers.ExportSeriesToObsidian, "Export.ExportSeriesToObsidian")

	v1 := r.Group("/api/v1")
	blogGroup := v1.Group("/blogs")
	blogGroup.Use(authMiddleware)
	blogGroup.GET("/:id/export", handlers.ExportSeries)
	blogGroup.GET("/:id/export/pdf", handlers.ExportSeriesPDF)
	blogGroup.POST("/:id/export/obsidian", handlers.ExportToObsidian)
	blogGroup.POST("/:id/export/obsidian/series", handlers.ExportSeriesToObsidian)
}

func must(fn gin.HandlerFunc, name string) {
	if fn == nil {
		panic("missing handler: " + name)
	}
}

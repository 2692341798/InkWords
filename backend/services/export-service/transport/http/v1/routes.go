package v1

import (
	"github.com/gin-gonic/gin"

	exportdomain "inkwords-backend/services/export-service/domain/export"
)

type exportRouteHandlers struct {
	ExportSeries                          gin.HandlerFunc
	ExportSeriesPDF                       gin.HandlerFunc
	ExportToObsidian                      gin.HandlerFunc
	ExportSeriesToObsidian                gin.HandlerFunc
	ExportApprovedTextbookChapterMarkdown gin.HandlerFunc
	ExportApprovedTextbookChapterZip      gin.HandlerFunc
	ExportFrozenTextbookBookMarkdown      gin.HandlerFunc
	ExportFrozenTextbookBookDOCX          gin.HandlerFunc
	ExportFrozenTextbookBookPDF           gin.HandlerFunc
	ExportFrozenTextbookBookReviewBundle  gin.HandlerFunc
}

// RegisterExportRoutes wires only export-service owned endpoints onto the shared API surface.
func RegisterExportRoutes(r *gin.Engine, workspaceMiddleware gin.HandlerFunc, handler *exportdomain.Handler) {
	if handler == nil {
		panic("missing dependency: exportHandler")
	}

	registerExportRoutes(r, workspaceMiddleware, exportRouteHandlers{
		ExportSeries:                          handler.ExportSeries,
		ExportSeriesPDF:                       handler.ExportSeriesPDF,
		ExportToObsidian:                      handler.ExportToObsidian,
		ExportSeriesToObsidian:                handler.ExportSeriesToObsidian,
		ExportApprovedTextbookChapterMarkdown: handler.ExportApprovedTextbookChapterMarkdown,
		ExportApprovedTextbookChapterZip:      handler.ExportApprovedTextbookChapterZip,
		ExportFrozenTextbookBookMarkdown:      handler.ExportFrozenTextbookBookMarkdown,
		ExportFrozenTextbookBookDOCX:          handler.ExportFrozenTextbookBookDOCX,
		ExportFrozenTextbookBookPDF:           handler.ExportFrozenTextbookBookPDF,
		ExportFrozenTextbookBookReviewBundle:  handler.ExportFrozenTextbookBookReviewBundle,
	})
}

func registerExportRoutes(r *gin.Engine, workspaceMiddleware gin.HandlerFunc, handlers exportRouteHandlers) {
	if workspaceMiddleware == nil {
		panic("missing middleware: workspaceMiddleware")
	}
	must(handlers.ExportSeries, "Export.ExportSeries")
	must(handlers.ExportSeriesPDF, "Export.ExportSeriesPDF")
	must(handlers.ExportToObsidian, "Export.ExportToObsidian")
	must(handlers.ExportSeriesToObsidian, "Export.ExportSeriesToObsidian")
	must(handlers.ExportApprovedTextbookChapterMarkdown, "Export.ExportApprovedTextbookChapterMarkdown")
	must(handlers.ExportApprovedTextbookChapterZip, "Export.ExportApprovedTextbookChapterZip")
	must(handlers.ExportFrozenTextbookBookMarkdown, "Export.ExportFrozenTextbookBookMarkdown")
	must(handlers.ExportFrozenTextbookBookDOCX, "Export.ExportFrozenTextbookBookDOCX")
	must(handlers.ExportFrozenTextbookBookPDF, "Export.ExportFrozenTextbookBookPDF")
	must(handlers.ExportFrozenTextbookBookReviewBundle, "Export.ExportFrozenTextbookBookReviewBundle")

	v1 := r.Group("/api/v1")
	blogGroup := v1.Group("/blogs")
	blogGroup.Use(workspaceMiddleware)
	blogGroup.GET("/:id/export", handlers.ExportSeries)
	blogGroup.GET("/:id/export/pdf", handlers.ExportSeriesPDF)
	blogGroup.POST("/:id/export/obsidian", handlers.ExportToObsidian)
	blogGroup.POST("/:id/export/obsidian/series", handlers.ExportSeriesToObsidian)

	textbookGroup := v1.Group("/textbook-projects")
	textbookGroup.Use(workspaceMiddleware)
	textbookGroup.GET("/chapters/:chapterID/export/markdown", handlers.ExportApprovedTextbookChapterMarkdown)
	textbookGroup.GET("/chapters/:chapterID/export/zip", handlers.ExportApprovedTextbookChapterZip)
	textbookGroup.GET("/book-builds/:buildID/export/markdown", handlers.ExportFrozenTextbookBookMarkdown)
	textbookGroup.GET("/book-builds/:buildID/export/docx", handlers.ExportFrozenTextbookBookDOCX)
	textbookGroup.GET("/book-builds/:buildID/export/pdf", handlers.ExportFrozenTextbookBookPDF)
	textbookGroup.GET("/book-builds/:buildID/export/review-bundle", handlers.ExportFrozenTextbookBookReviewBundle)
}

func must(fn gin.HandlerFunc, name string) {
	if fn == nil {
		panic("missing handler: " + name)
	}
}

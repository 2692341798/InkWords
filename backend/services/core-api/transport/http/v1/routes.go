package v1

import "github.com/gin-gonic/gin"

// TaskHandlers defines the workspace-owned asynchronous task surface.
type TaskHandlers struct {
	TaskCreateGeneration gin.HandlerFunc
	TaskCreateParse      gin.HandlerFunc
	TaskCreateExport     gin.HandlerFunc
	TaskGet              gin.HandlerFunc
	TaskRetry            gin.HandlerFunc
	TaskCancel           gin.HandlerFunc
	TaskStream           gin.HandlerFunc
	TaskDownload         gin.HandlerFunc
}

// ProjectHandlers contains the local-workspace project preparation surface.
type ProjectHandlers struct {
	ProjectScan    gin.HandlerFunc
	ProjectAnalyze gin.HandlerFunc
}

// RegisterProjectRoutes wires project preparation to the installation
// workspace. These handlers do not persist or authorize through legacy users.
func RegisterProjectRoutes(r *gin.Engine, workspaceMiddleware gin.HandlerFunc, h ProjectHandlers) {
	if workspaceMiddleware == nil {
		panic("missing middleware: workspaceMiddleware")
	}
	must(h.ProjectScan, "ProjectScan")
	must(h.ProjectAnalyze, "ProjectAnalyze")
	v1 := r.Group("/api/v1")
	projectGroup := v1.Group("/project")
	projectGroup.Use(workspaceMiddleware)
	projectGroup.POST("/scan", h.ProjectScan)
	projectGroup.POST("/analyze", h.ProjectAnalyze)
}

// BlogHandlers contains the workspace-owned compatibility blog surface.
type BlogHandlers struct {
	BlogList        gin.HandlerFunc
	BlogCreateDraft gin.HandlerFunc
	BlogBatchDelete gin.HandlerFunc
	BlogUpdate      gin.HandlerFunc
}

// RegisterBlogRoutes wires existing blog management onto the installation
// workspace without retaining an HTTP identity bridge.
func RegisterBlogRoutes(r *gin.Engine, workspaceMiddleware gin.HandlerFunc, h BlogHandlers) {
	if workspaceMiddleware == nil {
		panic("missing middleware: workspaceMiddleware")
	}
	must(h.BlogList, "BlogList")
	must(h.BlogCreateDraft, "BlogCreateDraft")
	must(h.BlogBatchDelete, "BlogBatchDelete")
	must(h.BlogUpdate, "BlogUpdate")
	v1 := r.Group("/api/v1")
	blogGroup := v1.Group("/blogs")
	blogGroup.Use(workspaceMiddleware)
	blogGroup.GET("", h.BlogList)
	blogGroup.POST("/draft", h.BlogCreateDraft)
	blogGroup.DELETE("", h.BlogBatchDelete)
	blogGroup.PUT("/:id", h.BlogUpdate)
}

// TextbookHandlers defines the local-workspace textbook surface independently
// from the legacy-owner compatibility routes.
type TextbookHandlers struct {
	TextbookGetTask                          gin.HandlerFunc
	TextbookRetryTask                        gin.HandlerFunc
	TextbookCreateProject                    gin.HandlerFunc
	TextbookListProjects                     gin.HandlerFunc
	TextbookGetProject                       gin.HandlerFunc
	TextbookGetProjectWorkspace              gin.HandlerFunc
	TextbookGetProjectProgress               gin.HandlerFunc
	TextbookCreateBookBuild                  gin.HandlerFunc
	TextbookGetEditorialWorkspace            gin.HandlerFunc
	TextbookAddRightsItem                    gin.HandlerFunc
	TextbookAppendRightsAmendment            gin.HandlerFunc
	TextbookCompletePublicationReview        gin.HandlerFunc
	TextbookRecordDelegatedPublicationReview gin.HandlerFunc
	TextbookPromoteBookBuild                 gin.HandlerFunc
	TextbookListSourceLibrary                gin.HandlerFunc
	TextbookListSourceEvidence               gin.HandlerFunc
	TextbookRetrieveSourceEvidence           gin.HandlerFunc
	TextbookGetChapterWorkspace              gin.HandlerFunc
	TextbookGetApprovedProjections           gin.HandlerFunc
	TextbookGetPracticeEvidence              gin.HandlerFunc
	TextbookAddSource                        gin.HandlerFunc
	TextbookLoadGinFixture                   gin.HandlerFunc
	TextbookCreateSourceImport               gin.HandlerFunc
	TextbookCreateOfficialWebImport          gin.HandlerFunc
	TextbookCreateChapter                    gin.HandlerFunc
	TextbookCreateBookContract               gin.HandlerFunc
	TextbookCreateStyleSheet                 gin.HandlerFunc
	TextbookCreateBlueprint                  gin.HandlerFunc
	TextbookApproveBookContract              gin.HandlerFunc
	TextbookApproveStyleSheet                gin.HandlerFunc
	TextbookApproveBlueprint                 gin.HandlerFunc
	TextbookAcquireLock                      gin.HandlerFunc
	TextbookAppendRevision                   gin.HandlerFunc
	TextbookApplyCandidate                   gin.HandlerFunc
	TextbookRejectCandidate                  gin.HandlerFunc
	TextbookGetSampleGenerationPreflight     gin.HandlerFunc
	TextbookGenerateSample                   gin.HandlerFunc
	TextbookCorrectSample                    gin.HandlerFunc
	TextbookCreateArtifactVerification       gin.HandlerFunc
	TextbookGetArtifactVerification          gin.HandlerFunc
	TextbookUploadVisualAsset                gin.HandlerFunc
}

// RegisterTaskRoutes wires all task operations to the installation workspace.
func RegisterTaskRoutes(r *gin.Engine, workspaceMiddleware gin.HandlerFunc, h TaskHandlers) {
	if workspaceMiddleware == nil {
		panic("missing middleware: workspaceMiddleware")
	}

	must(h.TaskCreateGeneration, "TaskCreateGeneration")
	must(h.TaskCreateParse, "TaskCreateParse")
	must(h.TaskCreateExport, "TaskCreateExport")
	must(h.TaskGet, "TaskGet")
	must(h.TaskRetry, "TaskRetry")
	must(h.TaskCancel, "TaskCancel")
	must(h.TaskStream, "TaskStream")
	must(h.TaskDownload, "TaskDownload")
	v1 := r.Group("/api/v1")

	taskGroup := v1.Group("/tasks")
	taskGroup.Use(workspaceMiddleware)
	taskGroup.POST("/generation", h.TaskCreateGeneration)
	taskGroup.POST("/parse", h.TaskCreateParse)
	taskGroup.POST("/export", h.TaskCreateExport)
	taskGroup.GET("/:id", h.TaskGet)
	taskGroup.POST("/:id/retry", h.TaskRetry)
	taskGroup.POST("/:id/cancel", h.TaskCancel)
	taskGroup.GET("/:id/stream", h.TaskStream)
	taskGroup.GET("/:id/download", h.TaskDownload)
}

// RegisterTextbookRoutes wires the local single-user textbook routes. It does
// not accept a legacy-owner middleware or handler, so textbook startup can
// remain independent while the compatibility surface is retired gradually.
func RegisterTextbookRoutes(r *gin.Engine, textbookMiddleware gin.HandlerFunc, h TextbookHandlers) {
	if textbookMiddleware == nil {
		panic("missing middleware: textbookMiddleware")
	}

	must(h.TextbookCreateProject, "TextbookCreateProject")
	must(h.TextbookGetTask, "TextbookGetTask")
	must(h.TextbookRetryTask, "TextbookRetryTask")
	must(h.TextbookListProjects, "TextbookListProjects")
	must(h.TextbookGetProject, "TextbookGetProject")
	must(h.TextbookGetProjectWorkspace, "TextbookGetProjectWorkspace")
	must(h.TextbookGetProjectProgress, "TextbookGetProjectProgress")
	must(h.TextbookCreateBookBuild, "TextbookCreateBookBuild")
	must(h.TextbookGetEditorialWorkspace, "TextbookGetEditorialWorkspace")
	must(h.TextbookAddRightsItem, "TextbookAddRightsItem")
	must(h.TextbookAppendRightsAmendment, "TextbookAppendRightsAmendment")
	must(h.TextbookCompletePublicationReview, "TextbookCompletePublicationReview")
	must(h.TextbookRecordDelegatedPublicationReview, "TextbookRecordDelegatedPublicationReview")
	must(h.TextbookPromoteBookBuild, "TextbookPromoteBookBuild")
	must(h.TextbookListSourceLibrary, "TextbookListSourceLibrary")
	must(h.TextbookListSourceEvidence, "TextbookListSourceEvidence")
	must(h.TextbookRetrieveSourceEvidence, "TextbookRetrieveSourceEvidence")
	must(h.TextbookGetChapterWorkspace, "TextbookGetChapterWorkspace")
	must(h.TextbookGetApprovedProjections, "TextbookGetApprovedProjections")
	must(h.TextbookGetPracticeEvidence, "TextbookGetPracticeEvidence")
	must(h.TextbookAddSource, "TextbookAddSource")
	must(h.TextbookLoadGinFixture, "TextbookLoadGinFixture")
	must(h.TextbookCreateSourceImport, "TextbookCreateSourceImport")
	must(h.TextbookCreateOfficialWebImport, "TextbookCreateOfficialWebImport")
	must(h.TextbookCreateChapter, "TextbookCreateChapter")
	must(h.TextbookCreateBookContract, "TextbookCreateBookContract")
	must(h.TextbookCreateStyleSheet, "TextbookCreateStyleSheet")
	must(h.TextbookCreateBlueprint, "TextbookCreateBlueprint")
	must(h.TextbookApproveBookContract, "TextbookApproveBookContract")
	must(h.TextbookApproveStyleSheet, "TextbookApproveStyleSheet")
	must(h.TextbookApproveBlueprint, "TextbookApproveBlueprint")
	must(h.TextbookAcquireLock, "TextbookAcquireLock")
	must(h.TextbookAppendRevision, "TextbookAppendRevision")
	must(h.TextbookApplyCandidate, "TextbookApplyCandidate")
	must(h.TextbookRejectCandidate, "TextbookRejectCandidate")
	must(h.TextbookGenerateSample, "TextbookGenerateSample")
	must(h.TextbookCorrectSample, "TextbookCorrectSample")
	must(h.TextbookGetSampleGenerationPreflight, "TextbookGetSampleGenerationPreflight")
	must(h.TextbookCreateArtifactVerification, "TextbookCreateArtifactVerification")
	must(h.TextbookGetArtifactVerification, "TextbookGetArtifactVerification")
	must(h.TextbookUploadVisualAsset, "TextbookUploadVisualAsset")

	v1 := r.Group("/api/v1")

	textbookGroup := v1.Group("/textbook-projects")
	textbookGroup.Use(textbookMiddleware)
	textbookGroup.GET("/tasks/:taskID", h.TextbookGetTask)
	textbookGroup.POST("/tasks/:taskID/retry", h.TextbookRetryTask)
	textbookGroup.GET("", h.TextbookListProjects)
	textbookGroup.POST("", h.TextbookCreateProject)
	textbookGroup.GET("/:projectID", h.TextbookGetProject)
	textbookGroup.GET("/:projectID/workspace", h.TextbookGetProjectWorkspace)
	textbookGroup.GET("/:projectID/progress", h.TextbookGetProjectProgress)
	textbookGroup.POST("/:projectID/book-builds", h.TextbookCreateBookBuild)
	textbookGroup.GET("/book-builds/:buildID/editorial", h.TextbookGetEditorialWorkspace)
	textbookGroup.POST("/book-builds/:buildID/rights", h.TextbookAddRightsItem)
	textbookGroup.POST("/book-builds/:buildID/rights-amendments", h.TextbookAppendRightsAmendment)
	textbookGroup.POST("/book-builds/:buildID/reviews", h.TextbookCompletePublicationReview)
	textbookGroup.POST("/book-builds/:buildID/delegated-reviews", h.TextbookRecordDelegatedPublicationReview)
	textbookGroup.POST("/book-builds/:buildID/publication-candidate", h.TextbookPromoteBookBuild)
	textbookGroup.GET("/:projectID/source-library", h.TextbookListSourceLibrary)
	textbookGroup.GET("/:projectID/source-evidence", h.TextbookListSourceEvidence)
	textbookGroup.POST("/:projectID/source-retrieval", h.TextbookRetrieveSourceEvidence)
	textbookGroup.GET("/chapters/:chapterID/workspace", h.TextbookGetChapterWorkspace)
	textbookGroup.GET("/chapters/:chapterID/projections", h.TextbookGetApprovedProjections)
	textbookGroup.GET("/chapters/:chapterID/practice-evidence", h.TextbookGetPracticeEvidence)
	textbookGroup.POST("/:projectID/sources", h.TextbookAddSource)
	textbookGroup.POST("/:projectID/load-gin-fixture", h.TextbookLoadGinFixture)
	textbookGroup.POST("/:projectID/source-imports", h.TextbookCreateSourceImport)
	textbookGroup.POST("/:projectID/official-web-imports", h.TextbookCreateOfficialWebImport)
	textbookGroup.POST("/:projectID/chapters", h.TextbookCreateChapter)
	textbookGroup.POST("/:projectID/book-contracts", h.TextbookCreateBookContract)
	textbookGroup.POST("/:projectID/style-sheets", h.TextbookCreateStyleSheet)
	textbookGroup.POST("/:projectID/blueprints", h.TextbookCreateBlueprint)
	textbookGroup.POST("/:projectID/book-contracts/:revisionID/approve", h.TextbookApproveBookContract)
	textbookGroup.POST("/:projectID/style-sheets/:revisionID/approve", h.TextbookApproveStyleSheet)
	textbookGroup.POST("/:projectID/blueprints/:revisionID/approve", h.TextbookApproveBlueprint)
	textbookGroup.POST("/chapters/:chapterID/lock", h.TextbookAcquireLock)
	textbookGroup.POST("/chapters/:chapterID/revisions", h.TextbookAppendRevision)
	textbookGroup.POST("/chapters/:chapterID/revisions/:revisionID/apply", h.TextbookApplyCandidate)
	textbookGroup.POST("/chapters/:chapterID/revisions/:revisionID/reject", h.TextbookRejectCandidate)
	textbookGroup.GET("/:projectID/chapters/:chapterID/generation-preflight", h.TextbookGetSampleGenerationPreflight)
	textbookGroup.POST("/:projectID/chapters/:chapterID/generate-sample", h.TextbookGenerateSample)
	textbookGroup.POST("/:projectID/chapters/:chapterID/correct-sample", h.TextbookCorrectSample)
	textbookGroup.POST("/chapters/:chapterID/code-artifacts/:artifactID/verify", h.TextbookCreateArtifactVerification)
	textbookGroup.GET("/chapters/:chapterID/code-artifacts/:artifactID/verify", h.TextbookGetArtifactVerification)
	textbookGroup.POST("/chapters/:chapterID/visual-assets", h.TextbookUploadVisualAsset)
}

func must(handler gin.HandlerFunc, name string) {
	if handler == nil {
		panic("missing handler: " + name)
	}
}

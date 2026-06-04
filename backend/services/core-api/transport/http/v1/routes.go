package v1

import "github.com/gin-gonic/gin"

// CoreHandlers defines the service-owned core-api HTTP surface.
type CoreHandlers struct {
	AuthRegister         gin.HandlerFunc
	AuthLogin            gin.HandlerFunc
	AuthBindGithub       gin.HandlerFunc
	AuthGetCaptcha       gin.HandlerFunc
	AuthOAuthRedirect    gin.HandlerFunc
	AuthOAuthCallback    gin.HandlerFunc
	UserProfile          gin.HandlerFunc
	UserUpdateProfile    gin.HandlerFunc
	UserUploadAvatar     gin.HandlerFunc
	UserStats            gin.HandlerFunc
	UserGetPromptSetting gin.HandlerFunc
	UserPutPromptSetting gin.HandlerFunc
	BlogList             gin.HandlerFunc
	BlogCreateDraft      gin.HandlerFunc
	BlogBatchDelete      gin.HandlerFunc
	BlogUpdate           gin.HandlerFunc
	ProjectScan          gin.HandlerFunc
	ProjectAnalyze       gin.HandlerFunc
	TaskCreateGeneration gin.HandlerFunc
	TaskCreateParse      gin.HandlerFunc
	TaskCreateExport     gin.HandlerFunc
	TaskGet              gin.HandlerFunc
	TaskCancel           gin.HandlerFunc
	TaskStream           gin.HandlerFunc
	TaskDownload         gin.HandlerFunc
}

// RegisterCoreRoutes wires the core-api owned routes without depending on the shared transport aggregator.
func RegisterCoreRoutes(r *gin.Engine, authMiddleware gin.HandlerFunc, h CoreHandlers) {
	if authMiddleware == nil {
		panic("missing middleware: authMiddleware")
	}

	must(h.AuthRegister, "AuthRegister")
	must(h.AuthLogin, "AuthLogin")
	must(h.AuthBindGithub, "AuthBindGithub")
	must(h.AuthGetCaptcha, "AuthGetCaptcha")
	must(h.AuthOAuthRedirect, "AuthOAuthRedirect")
	must(h.AuthOAuthCallback, "AuthOAuthCallback")
	must(h.UserProfile, "UserProfile")
	must(h.UserUpdateProfile, "UserUpdateProfile")
	must(h.UserUploadAvatar, "UserUploadAvatar")
	must(h.UserStats, "UserStats")
	must(h.UserGetPromptSetting, "UserGetPromptSetting")
	must(h.UserPutPromptSetting, "UserPutPromptSetting")
	must(h.BlogList, "BlogList")
	must(h.BlogCreateDraft, "BlogCreateDraft")
	must(h.BlogBatchDelete, "BlogBatchDelete")
	must(h.BlogUpdate, "BlogUpdate")
	must(h.ProjectScan, "ProjectScan")
	must(h.ProjectAnalyze, "ProjectAnalyze")
	must(h.TaskCreateGeneration, "TaskCreateGeneration")
	must(h.TaskCreateParse, "TaskCreateParse")
	must(h.TaskCreateExport, "TaskCreateExport")
	must(h.TaskGet, "TaskGet")
	must(h.TaskCancel, "TaskCancel")
	must(h.TaskStream, "TaskStream")
	must(h.TaskDownload, "TaskDownload")

	v1 := r.Group("/api/v1")

	authGroup := v1.Group("/auth")
	authGroup.POST("/register", h.AuthRegister)
	authGroup.POST("/login", h.AuthLogin)
	authGroup.POST("/bind-github", h.AuthBindGithub)
	authGroup.GET("/captcha", h.AuthGetCaptcha)
	authGroup.GET("/oauth/:provider", h.AuthOAuthRedirect)
	authGroup.GET("/callback/:provider", h.AuthOAuthCallback)

	userGroup := v1.Group("/user")
	userGroup.Use(authMiddleware)
	userGroup.GET("/profile", h.UserProfile)
	userGroup.PUT("/profile", h.UserUpdateProfile)
	userGroup.POST("/avatar", h.UserUploadAvatar)
	userGroup.GET("/stats", h.UserStats)
	userGroup.GET("/prompt-settings", h.UserGetPromptSetting)
	userGroup.PUT("/prompt-settings", h.UserPutPromptSetting)

	blogGroup := v1.Group("/blogs")
	blogGroup.Use(authMiddleware)
	blogGroup.GET("", h.BlogList)
	blogGroup.POST("/draft", h.BlogCreateDraft)
	blogGroup.DELETE("", h.BlogBatchDelete)
	blogGroup.PUT("/:id", h.BlogUpdate)

	projectGroup := v1.Group("/project")
	projectGroup.Use(authMiddleware)
	projectGroup.POST("/scan", h.ProjectScan)
	projectGroup.POST("/analyze", h.ProjectAnalyze)

	taskGroup := v1.Group("/tasks")
	taskGroup.Use(authMiddleware)
	taskGroup.POST("/generation", h.TaskCreateGeneration)
	taskGroup.POST("/parse", h.TaskCreateParse)
	taskGroup.POST("/export", h.TaskCreateExport)
	taskGroup.GET("/:id", h.TaskGet)
	taskGroup.POST("/:id/cancel", h.TaskCancel)
	taskGroup.GET("/:id/stream", h.TaskStream)
	taskGroup.GET("/:id/download", h.TaskDownload)
}

func must(handler gin.HandlerFunc, name string) {
	if handler == nil {
		panic("missing handler: " + name)
	}
}

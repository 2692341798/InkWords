package v1

import "github.com/gin-gonic/gin"

type AuthHandlers struct {
	Register      gin.HandlerFunc
	Login         gin.HandlerFunc
	BindGithub    gin.HandlerFunc
	GetCaptcha    gin.HandlerFunc
	OAuthRedirect gin.HandlerFunc
	OAuthCallback gin.HandlerFunc
}

type UserHandlers struct {
	GetProfile           gin.HandlerFunc
	UpdateProfile        gin.HandlerFunc
	UploadAvatar         gin.HandlerFunc
	GetUserStats         gin.HandlerFunc
	GetPromptSettings    gin.HandlerFunc
	UpdatePromptSettings gin.HandlerFunc
}

type BlogHandlers struct {
	GetUserBlogs           gin.HandlerFunc
	CreateDraftBlog        gin.HandlerFunc
	BatchDeleteBlogs       gin.HandlerFunc
	UpdateBlog             gin.HandlerFunc
	ExportSeries           gin.HandlerFunc
	ExportSeriesPDF        gin.HandlerFunc
	ExportToObsidian       gin.HandlerFunc
	ExportSeriesToObsidian gin.HandlerFunc
	ContinueBlog           gin.HandlerFunc
	PolishBlog             gin.HandlerFunc
}

type CoreBlogHandlers struct {
	GetUserBlogs     gin.HandlerFunc
	CreateDraftBlog  gin.HandlerFunc
	BatchDeleteBlogs gin.HandlerFunc
	UpdateBlog       gin.HandlerFunc
}

type TaskHandlers struct {
	CreateGeneration gin.HandlerFunc
	GetTask          gin.HandlerFunc
	CancelTask       gin.HandlerFunc
	StreamTask       gin.HandlerFunc
}

type CoreHandlers struct {
	Auth    AuthHandlers
	User    UserHandlers
	Blog    CoreBlogHandlers
	Project CoreProjectHandlers
	Task    TaskHandlers
}

type StreamBlogHandlers struct {
	ContinueBlog gin.HandlerFunc
	PolishBlog   gin.HandlerFunc
}

type StreamOnlyHandlers struct {
	Blog   StreamBlogHandlers
	Stream StreamHandlers
}

type ExportHandlers struct {
	ExportSeries           gin.HandlerFunc
	ExportSeriesPDF        gin.HandlerFunc
	ExportToObsidian       gin.HandlerFunc
	ExportSeriesToObsidian gin.HandlerFunc
}

type ParserHandlers struct {
	Parse gin.HandlerFunc
}

type ReviewOnlyHandlers struct {
	Review ReviewHandlers
}

type ProjectHandlers struct {
	ScanGithubRepo gin.HandlerFunc
	Analyze        gin.HandlerFunc
	Parse          gin.HandlerFunc
}

type CoreProjectHandlers struct {
	ScanGithubRepo gin.HandlerFunc
	Analyze        gin.HandlerFunc
}

type StreamHandlers struct {
	ScanStreamHandler     gin.HandlerFunc
	AnalyzeStreamHandler  gin.HandlerFunc
	GenerateStreamHandler gin.HandlerFunc
}

// ReviewHandlers 聚合知识漫游复习模块的所有 HTTP 入口。
type ReviewHandlers struct {
	GetTodayCard  gin.HandlerFunc
	GetHistory    gin.HandlerFunc
	PickRandom    gin.HandlerFunc
	ListNotes     gin.HandlerFunc
	CreateSession gin.HandlerFunc
	GetSession    gin.HandlerFunc
	Respond       gin.HandlerFunc
	RequestHint   gin.HandlerFunc
	Finish        gin.HandlerFunc
}

type Handlers struct {
	Auth    AuthHandlers
	User    UserHandlers
	Blog    BlogHandlers
	Project ProjectHandlers
	Stream  StreamHandlers
	Review  ReviewHandlers
}

func RegisterCore(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers CoreHandlers) {
	if authMiddleware == nil {
		panic("missing middleware: authMiddleware")
	}
	validateCoreHandlers(handlers)

	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", handlers.Auth.Register)
			authGroup.POST("/login", handlers.Auth.Login)
			authGroup.POST("/bind-github", handlers.Auth.BindGithub)
			authGroup.GET("/captcha", handlers.Auth.GetCaptcha)
			authGroup.GET("/oauth/:provider", handlers.Auth.OAuthRedirect)
			authGroup.GET("/callback/:provider", handlers.Auth.OAuthCallback)
		}

		userGroup := v1.Group("/user")
		userGroup.Use(authMiddleware)
		{
			userGroup.GET("/profile", handlers.User.GetProfile)
			userGroup.PUT("/profile", handlers.User.UpdateProfile)
			userGroup.POST("/avatar", handlers.User.UploadAvatar)
			userGroup.GET("/stats", handlers.User.GetUserStats)
			userGroup.GET("/prompt-settings", handlers.User.GetPromptSettings)
			userGroup.PUT("/prompt-settings", handlers.User.UpdatePromptSettings)
		}

		blogGroup := v1.Group("/blogs")
		blogGroup.Use(authMiddleware)
		{
			blogGroup.GET("", handlers.Blog.GetUserBlogs)
			blogGroup.POST("/draft", handlers.Blog.CreateDraftBlog)
			blogGroup.DELETE("", handlers.Blog.BatchDeleteBlogs)
			blogGroup.PUT("/:id", handlers.Blog.UpdateBlog)
		}

		projectGroup := v1.Group("/project")
		projectGroup.Use(authMiddleware)
		{
			projectGroup.POST("/scan", handlers.Project.ScanGithubRepo)
			projectGroup.POST("/analyze", handlers.Project.Analyze)
		}

		taskGroup := v1.Group("/tasks")
		taskGroup.Use(authMiddleware)
		{
			taskGroup.POST("/generation", handlers.Task.CreateGeneration)
			taskGroup.GET("/:id", handlers.Task.GetTask)
			taskGroup.POST("/:id/cancel", handlers.Task.CancelTask)
			taskGroup.GET("/:id/stream", handlers.Task.StreamTask)
		}
	}
}

func RegisterExport(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers ExportHandlers) {
	if authMiddleware == nil {
		panic("missing middleware: authMiddleware")
	}
	must(handlers.ExportSeries, "Export.ExportSeries")
	must(handlers.ExportSeriesPDF, "Export.ExportSeriesPDF")
	must(handlers.ExportToObsidian, "Export.ExportToObsidian")
	must(handlers.ExportSeriesToObsidian, "Export.ExportSeriesToObsidian")

	v1 := r.Group("/api/v1")
	{
		blogGroup := v1.Group("/blogs")
		blogGroup.Use(authMiddleware)
		{
			blogGroup.GET("/:id/export", handlers.ExportSeries)
			blogGroup.GET("/:id/export/pdf", handlers.ExportSeriesPDF)
			blogGroup.POST("/:id/export/obsidian", handlers.ExportToObsidian)
			blogGroup.POST("/:id/export/obsidian/series", handlers.ExportSeriesToObsidian)
		}
	}
}

func RegisterParser(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers ParserHandlers) {
	if authMiddleware == nil {
		panic("missing middleware: authMiddleware")
	}
	must(handlers.Parse, "Parser.Parse")

	v1 := r.Group("/api/v1")
	{
		projectGroup := v1.Group("/project")
		projectGroup.Use(authMiddleware)
		{
			projectGroup.POST("/parse", handlers.Parse)
		}
	}
}

func RegisterReview(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers ReviewOnlyHandlers) {
	if authMiddleware == nil {
		panic("missing middleware: authMiddleware")
	}
	validateReviewHandlers(handlers.Review)

	v1 := r.Group("/api/v1")
	{
		reviewGroup := v1.Group("/review")
		reviewGroup.Use(authMiddleware)
		{
			reviewGroup.GET("/today", handlers.Review.GetTodayCard)
			reviewGroup.GET("/history", handlers.Review.GetHistory)
			reviewGroup.POST("/pick", handlers.Review.PickRandom)
			reviewGroup.GET("/notes", handlers.Review.ListNotes)
			reviewGroup.POST("/sessions", handlers.Review.CreateSession)
			reviewGroup.GET("/sessions/:id", handlers.Review.GetSession)
			reviewGroup.POST("/sessions/:id/respond", handlers.Review.Respond)
			reviewGroup.POST("/sessions/:id/hint", handlers.Review.RequestHint)
			reviewGroup.POST("/sessions/:id/finish", handlers.Review.Finish)
		}
	}
}

func RegisterStream(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers StreamOnlyHandlers) {
	if authMiddleware == nil {
		panic("missing middleware: authMiddleware")
	}
	validateStreamOnlyHandlers(handlers)

	v1 := r.Group("/api/v1")
	{
		blogGroup := v1.Group("/blogs")
		blogGroup.Use(authMiddleware)
		{
			blogGroup.POST("/:id/continue", handlers.Blog.ContinueBlog)
			blogGroup.POST("/:id/polish", handlers.Blog.PolishBlog)
		}

		streamGroup := v1.Group("/stream")
		streamGroup.Use(authMiddleware)
		{
			streamGroup.POST("/scan", handlers.Stream.ScanStreamHandler)
			streamGroup.POST("/analyze", handlers.Stream.AnalyzeStreamHandler)
			streamGroup.POST("/generate", handlers.Stream.GenerateStreamHandler)
		}
	}
}

func Register(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers Handlers) {
	if authMiddleware == nil {
		panic("missing middleware: authMiddleware")
	}
	validateHandlers(handlers)

	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", handlers.Auth.Register)
			authGroup.POST("/login", handlers.Auth.Login)
			authGroup.POST("/bind-github", handlers.Auth.BindGithub)
			authGroup.GET("/captcha", handlers.Auth.GetCaptcha)
			authGroup.GET("/oauth/:provider", handlers.Auth.OAuthRedirect)
			authGroup.GET("/callback/:provider", handlers.Auth.OAuthCallback)
		}

		userGroup := v1.Group("/user")
		userGroup.Use(authMiddleware)
		{
			userGroup.GET("/profile", handlers.User.GetProfile)
			userGroup.PUT("/profile", handlers.User.UpdateProfile)
			userGroup.POST("/avatar", handlers.User.UploadAvatar)
			userGroup.GET("/stats", handlers.User.GetUserStats)
			userGroup.GET("/prompt-settings", handlers.User.GetPromptSettings)
			userGroup.PUT("/prompt-settings", handlers.User.UpdatePromptSettings)
		}

		blogGroup := v1.Group("/blogs")
		blogGroup.Use(authMiddleware)
		{
			blogGroup.GET("", handlers.Blog.GetUserBlogs)
			blogGroup.POST("/draft", handlers.Blog.CreateDraftBlog)
			blogGroup.DELETE("", handlers.Blog.BatchDeleteBlogs)
			blogGroup.PUT("/:id", handlers.Blog.UpdateBlog)
			blogGroup.GET("/:id/export", handlers.Blog.ExportSeries)
			blogGroup.GET("/:id/export/pdf", handlers.Blog.ExportSeriesPDF)
			blogGroup.POST("/:id/export/obsidian", handlers.Blog.ExportToObsidian)
			blogGroup.POST("/:id/export/obsidian/series", handlers.Blog.ExportSeriesToObsidian)
			blogGroup.POST("/:id/continue", handlers.Blog.ContinueBlog)
			blogGroup.POST("/:id/polish", handlers.Blog.PolishBlog)
		}

		projectGroup := v1.Group("/project")
		projectGroup.Use(authMiddleware)
		{
			projectGroup.POST("/scan", handlers.Project.ScanGithubRepo)
			projectGroup.POST("/analyze", handlers.Project.Analyze)
			projectGroup.POST("/parse", handlers.Project.Parse)
		}

		streamGroup := v1.Group("/stream")
		streamGroup.Use(authMiddleware)
		{
			streamGroup.POST("/scan", handlers.Stream.ScanStreamHandler)
			streamGroup.POST("/analyze", handlers.Stream.AnalyzeStreamHandler)
			streamGroup.POST("/generate", handlers.Stream.GenerateStreamHandler)
		}

		reviewGroup := v1.Group("/review")
		reviewGroup.Use(authMiddleware)
		{
			reviewGroup.GET("/today", handlers.Review.GetTodayCard)
			reviewGroup.GET("/history", handlers.Review.GetHistory)
			reviewGroup.POST("/pick", handlers.Review.PickRandom)
			reviewGroup.GET("/notes", handlers.Review.ListNotes)
			reviewGroup.POST("/sessions", handlers.Review.CreateSession)
			reviewGroup.GET("/sessions/:id", handlers.Review.GetSession)
			reviewGroup.POST("/sessions/:id/respond", handlers.Review.Respond)
			reviewGroup.POST("/sessions/:id/hint", handlers.Review.RequestHint)
			reviewGroup.POST("/sessions/:id/finish", handlers.Review.Finish)
		}
	}
}

func validateHandlers(h Handlers) {
	must(h.Auth.Register, "Auth.Register")
	must(h.Auth.Login, "Auth.Login")
	must(h.Auth.BindGithub, "Auth.BindGithub")
	must(h.Auth.GetCaptcha, "Auth.GetCaptcha")
	must(h.Auth.OAuthRedirect, "Auth.OAuthRedirect")
	must(h.Auth.OAuthCallback, "Auth.OAuthCallback")

	must(h.User.GetProfile, "User.GetProfile")
	must(h.User.UpdateProfile, "User.UpdateProfile")
	must(h.User.UploadAvatar, "User.UploadAvatar")
	must(h.User.GetUserStats, "User.GetUserStats")
	must(h.User.GetPromptSettings, "User.GetPromptSettings")
	must(h.User.UpdatePromptSettings, "User.UpdatePromptSettings")

	must(h.Blog.GetUserBlogs, "Blog.GetUserBlogs")
	must(h.Blog.CreateDraftBlog, "Blog.CreateDraftBlog")
	must(h.Blog.BatchDeleteBlogs, "Blog.BatchDeleteBlogs")
	must(h.Blog.UpdateBlog, "Blog.UpdateBlog")
	must(h.Blog.ExportSeries, "Blog.ExportSeries")
	must(h.Blog.ExportSeriesPDF, "Blog.ExportSeriesPDF")
	must(h.Blog.ExportToObsidian, "Blog.ExportToObsidian")
	must(h.Blog.ExportSeriesToObsidian, "Blog.ExportSeriesToObsidian")
	must(h.Blog.ContinueBlog, "Blog.ContinueBlog")
	must(h.Blog.PolishBlog, "Blog.PolishBlog")

	must(h.Project.ScanGithubRepo, "Project.ScanGithubRepo")
	must(h.Project.Analyze, "Project.Analyze")
	must(h.Project.Parse, "Project.Parse")

	must(h.Stream.ScanStreamHandler, "Stream.ScanStreamHandler")
	must(h.Stream.AnalyzeStreamHandler, "Stream.AnalyzeStreamHandler")
	must(h.Stream.GenerateStreamHandler, "Stream.GenerateStreamHandler")

	must(h.Review.GetTodayCard, "Review.GetTodayCard")
	must(h.Review.GetHistory, "Review.GetHistory")
	must(h.Review.PickRandom, "Review.PickRandom")
	must(h.Review.ListNotes, "Review.ListNotes")
	must(h.Review.CreateSession, "Review.CreateSession")
	must(h.Review.GetSession, "Review.GetSession")
	must(h.Review.Respond, "Review.Respond")
	must(h.Review.RequestHint, "Review.RequestHint")
	must(h.Review.Finish, "Review.Finish")
}

func validateCoreHandlers(h CoreHandlers) {
	must(h.Auth.Register, "Auth.Register")
	must(h.Auth.Login, "Auth.Login")
	must(h.Auth.BindGithub, "Auth.BindGithub")
	must(h.Auth.GetCaptcha, "Auth.GetCaptcha")
	must(h.Auth.OAuthRedirect, "Auth.OAuthRedirect")
	must(h.Auth.OAuthCallback, "Auth.OAuthCallback")

	must(h.User.GetProfile, "User.GetProfile")
	must(h.User.UpdateProfile, "User.UpdateProfile")
	must(h.User.UploadAvatar, "User.UploadAvatar")
	must(h.User.GetUserStats, "User.GetUserStats")
	must(h.User.GetPromptSettings, "User.GetPromptSettings")
	must(h.User.UpdatePromptSettings, "User.UpdatePromptSettings")

	must(h.Blog.GetUserBlogs, "Blog.GetUserBlogs")
	must(h.Blog.CreateDraftBlog, "Blog.CreateDraftBlog")
	must(h.Blog.BatchDeleteBlogs, "Blog.BatchDeleteBlogs")
	must(h.Blog.UpdateBlog, "Blog.UpdateBlog")

	must(h.Project.ScanGithubRepo, "Project.ScanGithubRepo")
	must(h.Project.Analyze, "Project.Analyze")

	must(h.Task.CreateGeneration, "Task.CreateGeneration")
	must(h.Task.GetTask, "Task.GetTask")
	must(h.Task.CancelTask, "Task.CancelTask")
	must(h.Task.StreamTask, "Task.StreamTask")
}

func validateReviewHandlers(h ReviewHandlers) {
	must(h.GetTodayCard, "Review.GetTodayCard")
	must(h.GetHistory, "Review.GetHistory")
	must(h.PickRandom, "Review.PickRandom")
	must(h.ListNotes, "Review.ListNotes")
	must(h.CreateSession, "Review.CreateSession")
	must(h.GetSession, "Review.GetSession")
	must(h.Respond, "Review.Respond")
	must(h.RequestHint, "Review.RequestHint")
	must(h.Finish, "Review.Finish")
}

func validateStreamOnlyHandlers(h StreamOnlyHandlers) {
	must(h.Blog.ContinueBlog, "Blog.ContinueBlog")
	must(h.Blog.PolishBlog, "Blog.PolishBlog")

	must(h.Stream.ScanStreamHandler, "Stream.ScanStreamHandler")
	must(h.Stream.AnalyzeStreamHandler, "Stream.AnalyzeStreamHandler")
	must(h.Stream.GenerateStreamHandler, "Stream.GenerateStreamHandler")
}

func must(fn gin.HandlerFunc, name string) {
	if fn == nil {
		panic("missing handler: " + name)
	}
}

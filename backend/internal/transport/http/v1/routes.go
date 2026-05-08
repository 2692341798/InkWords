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
	GetProfile    gin.HandlerFunc
	UpdateProfile gin.HandlerFunc
	UploadAvatar  gin.HandlerFunc
	GetUserStats  gin.HandlerFunc
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

type ProjectHandlers struct {
	ScanGithubRepo gin.HandlerFunc
	Analyze        gin.HandlerFunc
	Parse          gin.HandlerFunc
}

type StreamHandlers struct {
	ScanStreamHandler     gin.HandlerFunc
	AnalyzeStreamHandler  gin.HandlerFunc
	GenerateStreamHandler gin.HandlerFunc
}

type Handlers struct {
	Auth    AuthHandlers
	User    UserHandlers
	Blog    BlogHandlers
	Project ProjectHandlers
	Stream  StreamHandlers
}

func Register(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers Handlers) {
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
	}
}

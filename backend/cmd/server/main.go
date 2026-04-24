package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"inkwords-backend/internal/api"
	"inkwords-backend/internal/cache"
	"inkwords-backend/internal/db"
	"inkwords-backend/internal/middleware"
	"inkwords-backend/internal/service"
)

// init 初始化环境变量
func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	// 初始化数据库
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}
	if err := db.InitDB(dsn); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	// 初始化 Redis 缓存
	if err := cache.InitRedis(); err != nil {
		log.Printf("Redis initialization failed (cache will be disabled): %v", err)
	}

	// 创建一个默认的 Gin 引擎
	r := gin.Default()

	// 开放 uploads 目录以供静态资源访问
	r.Static("/uploads", "./uploads")

	// 基础路由：健康检查
	r.GET("/api/v1/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code":    200,
			"message": "pong",
			"data":    nil,
		})
	})

	// 初始化 API Handler
	authService := service.NewAuthService(db.DB)
	userService := service.NewUserService(db.DB)
	authAPI := api.NewAuthAPI(authService)
	userAPI := api.NewUserAPI(userService)
	streamAPI := api.NewStreamAPI(userService)
	projectAPI := api.NewProjectAPI(userService)
	blogAPI := api.NewBlogAPI()

	// v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 认证相关路由 (公开)
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", authAPI.Register)
			authGroup.POST("/login", authAPI.Login)
			authGroup.POST("/bind-github", authAPI.BindGithub)
			authGroup.GET("/captcha", authAPI.GetCaptcha)
			authGroup.GET("/oauth/:provider", authAPI.OAuthRedirect)
			authGroup.GET("/callback/:provider", authAPI.OAuthCallback)
		}

		// 用户相关路由 (需鉴权)
		userGroup := v1.Group("/user")
		userGroup.Use(middleware.AuthMiddleware())
		{
			userGroup.GET("/profile", userAPI.GetProfile)
			userGroup.PUT("/profile", userAPI.UpdateProfile)
			userGroup.POST("/avatar", userAPI.UploadAvatar)
			userGroup.GET("/stats", userAPI.GetUserStats)
		}

		// 博客相关路由 (需鉴权)
		blogGroup := v1.Group("/blogs")
		blogGroup.Use(middleware.AuthMiddleware())
		{
			blogGroup.GET("", blogAPI.GetUserBlogs)
			blogGroup.DELETE("", blogAPI.BatchDeleteBlogs)
			blogGroup.PUT("/:id", blogAPI.UpdateBlog)
			blogGroup.GET("/:id/export", blogAPI.ExportSeries)
			blogGroup.POST("/:id/continue", streamAPI.ContinueBlogStreamHandler)
		}

		// 项目分析相关路由 (需鉴权)
		projectGroup := v1.Group("/project")
		projectGroup.Use(middleware.AuthMiddleware())
		{
			projectGroup.POST("/analyze", projectAPI.Analyze)
			projectGroup.POST("/parse", projectAPI.Parse)
		}

		// 流式生成路由 (需鉴权)
		streamGroup := v1.Group("/stream")
		streamGroup.Use(middleware.AuthMiddleware())
		{
			streamGroup.POST("/analyze", streamAPI.AnalyzeStreamHandler)
			streamGroup.POST("/generate", streamAPI.GenerateBlogStreamHandler)
		}
	}

	// 启动服务，默认监听 8080 端口
	log.Println("Server is running on port 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

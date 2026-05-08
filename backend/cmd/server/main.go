package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	authdomain "inkwords-backend/internal/domain/auth"
	blogdomain "inkwords-backend/internal/domain/blog"
	projectdomain "inkwords-backend/internal/domain/project"
	streamdomain "inkwords-backend/internal/domain/stream"
	userdomain "inkwords-backend/internal/domain/user"
	"inkwords-backend/internal/infra/cache"
	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/infra/parser"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	transportv1 "inkwords-backend/internal/transport/http/v1"
	"inkwords-backend/internal/transport/http/v1/api"
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

	// 允许上传大文件，限制为 100MB
	r.MaxMultipartMemory = 100 << 20

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
	userService := service.NewUserService(db.DB)
	blogService := service.NewBlogService()
	generatorService := service.NewGeneratorService()
	decompositionService := service.NewDecompositionService()
	gitFetcher := parser.NewGitFetcher()
	docParser := parser.NewDocParser()

	authRepo := authdomain.NewGormRepository(db.DB)
	authDomainService := authdomain.NewService(authRepo)
	authDomainHandler := authdomain.NewHandler(authDomainService)

	blogRepo := blogdomain.NewGormRepository(db.DB)
	blogDomainService := blogdomain.NewService(blogRepo)
	blogDomainHandler := blogdomain.NewHandlerWithLegacy(blogDomainService, blogService)

	userRepo := userdomain.NewGormRepository(db.DB)
	userDomainService := userdomain.NewService(userRepo)
	userDomainHandler := userdomain.NewHandler(userDomainService)

	streamDomainService := streamdomain.NewService(generatorService, decompositionService, userService)
	streamDomainHandler := streamdomain.NewHandler(streamDomainService, streamdomain.NewGormBlogReadable())

	projectDomainService := projectdomain.NewService(decompositionService, gitFetcher, docParser, userService)
	projectDomainHandler := projectdomain.NewHandler(projectDomainService)

	authAPI := api.NewAuthAPIWithDeps(authDomainHandler)
	userAPI := api.NewUserAPIWithDeps(userService, userDomainHandler)
	streamAPI := api.NewStreamAPIWithDeps(generatorService, decompositionService, userService, streamDomainHandler)
	projectAPI := api.NewProjectAPIWithDeps(decompositionService, gitFetcher, docParser, userService, projectDomainHandler)
	blogAPI := api.NewBlogAPIWithDeps(blogService, blogDomainHandler)

	transportv1.Register(r, middleware.AuthMiddleware(), transportv1.Handlers{
		Auth: transportv1.AuthHandlers{
			Register:      authAPI.Register,
			Login:         authAPI.Login,
			BindGithub:    authAPI.BindGithub,
			GetCaptcha:    authAPI.GetCaptcha,
			OAuthRedirect: authAPI.OAuthRedirect,
			OAuthCallback: authAPI.OAuthCallback,
		},
		User: transportv1.UserHandlers{
			GetProfile:    userAPI.GetProfile,
			UpdateProfile: userAPI.UpdateProfile,
			UploadAvatar:  userAPI.UploadAvatar,
			GetUserStats:  userAPI.GetUserStats,
		},
		Blog: transportv1.BlogHandlers{
			GetUserBlogs:           blogAPI.GetUserBlogs,
			CreateDraftBlog:        blogAPI.CreateDraftBlog,
			BatchDeleteBlogs:       blogAPI.BatchDeleteBlogs,
			UpdateBlog:             blogAPI.UpdateBlog,
			ExportSeries:           blogAPI.ExportSeries,
			ExportSeriesPDF:        blogAPI.ExportSeriesPDF,
			ExportToObsidian:       blogAPI.ExportToObsidian,
			ExportSeriesToObsidian: blogAPI.ExportSeriesToObsidian,
			ContinueBlog:           streamAPI.ContinueBlogStreamHandler,
			PolishBlog:             streamAPI.PolishBlogStreamHandler,
		},
		Project: transportv1.ProjectHandlers{
			ScanGithubRepo: projectAPI.ScanGithubRepo,
			Analyze:        projectAPI.Analyze,
			Parse:          projectAPI.Parse,
		},
		Stream: transportv1.StreamHandlers{
			ScanStreamHandler:     streamAPI.ScanStreamHandler,
			AnalyzeStreamHandler:  streamAPI.AnalyzeStreamHandler,
			GenerateStreamHandler: streamAPI.GenerateBlogStreamHandler,
		},
	})

	// 启动服务，默认监听 8080 端口
	log.Println("Server is running on port 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

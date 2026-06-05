package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	authdomain "inkwords-backend/internal/domain/auth"
	blogdomain "inkwords-backend/internal/domain/blog"
	projectdomain "inkwords-backend/internal/domain/project"
	reviewdomain "inkwords-backend/internal/domain/review"
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

type shutdownableServer interface {
	Shutdown(context.Context) error
}

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

	// 允许上传大文件，限制为 888MB
	r.MaxMultipartMemory = 888 << 20

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
	promptReqService := service.NewPromptRequirementsService(db.DB)
	generatorService := service.NewGeneratorServiceWithPersistence(
		promptReqService,
		blogdomain.NewGeneratedBlogPersistence(db.DB),
	)
	decompositionService := service.NewDecompositionServiceWithPersistences(
		promptReqService,
		blogdomain.NewSeriesPersistence(db.DB),
		blogdomain.NewContinuePersistence(db.DB),
	)
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

	reviewRepo := reviewdomain.NewGormRepository(db.DB)
	reviewNoteSource := buildReviewNoteSource()
	reviewDomainService := reviewdomain.NewService(reviewRepo, reviewNoteSource)
	reviewDomainHandler := reviewdomain.NewHandler(reviewDomainService)

	authAPI := api.NewAuthAPIWithDeps(authDomainHandler)
	userAPI := api.NewUserAPIWithDeps(userService, userDomainHandler)
	streamAPI := api.NewStreamAPIWithDeps(generatorService, decompositionService, userService, streamDomainHandler)
	projectAPI := api.NewProjectAPIWithDeps(decompositionService, gitFetcher, docParser, userService, projectDomainHandler)
	blogAPI := api.NewBlogAPIWithDeps(blogService, blogDomainHandler)

	authMiddleware := middleware.AuthMiddleware()

	transportv1.RegisterCore(r, authMiddleware, transportv1.CoreHandlers{
		Auth: transportv1.AuthHandlers{
			Register:      authAPI.Register,
			Login:         authAPI.Login,
			BindGithub:    authAPI.BindGithub,
			GetCaptcha:    authAPI.GetCaptcha,
			OAuthRedirect: authAPI.OAuthRedirect,
			OAuthCallback: authAPI.OAuthCallback,
		},
		User: transportv1.UserHandlers{
			GetProfile:           userAPI.GetProfile,
			UpdateProfile:        userAPI.UpdateProfile,
			UploadAvatar:         userAPI.UploadAvatar,
			GetUserStats:         userAPI.GetUserStats,
			GetPromptSettings:    userAPI.GetPromptSettings,
			UpdatePromptSettings: userAPI.UpdatePromptSettings,
		},
		Blog: transportv1.CoreBlogHandlers{
			GetUserBlogs:           blogAPI.GetUserBlogs,
			CreateDraftBlog:        blogAPI.CreateDraftBlog,
			BatchDeleteBlogs:       blogAPI.BatchDeleteBlogs,
			UpdateBlog:             blogAPI.UpdateBlog,
		},
		Project: transportv1.CoreProjectHandlers{
			ScanGithubRepo: projectAPI.ScanGithubRepo,
			Analyze:        projectAPI.Analyze,
		},
	})

	transportv1.RegisterExport(r, authMiddleware, transportv1.ExportHandlers{
		ExportSeries:           blogAPI.ExportSeries,
		ExportSeriesPDF:        blogAPI.ExportSeriesPDF,
		ExportToObsidian:       blogAPI.ExportToObsidian,
		ExportSeriesToObsidian: blogAPI.ExportSeriesToObsidian,
	})

	transportv1.RegisterParser(r, authMiddleware, transportv1.ParserHandlers{
		Parse: projectAPI.Parse,
	})

	transportv1.RegisterReview(r, authMiddleware, transportv1.ReviewOnlyHandlers{
		Review: transportv1.ReviewHandlers{
			GetTodayCard:  reviewDomainHandler.GetTodayCard,
			GetHistory:    reviewDomainHandler.GetHistory,
			PickRandom:    reviewDomainHandler.PickRandom,
			ListNotes:     reviewDomainHandler.ListNotes,
			CreateSession: reviewDomainHandler.CreateSession,
			GetSession:    reviewDomainHandler.GetSession,
			Respond:       reviewDomainHandler.Respond,
			RequestHint:   reviewDomainHandler.RequestHint,
			Finish:        reviewDomainHandler.Finish,
		},
	})

	transportv1.RegisterStream(r, authMiddleware, transportv1.StreamOnlyHandlers{
		Blog: transportv1.StreamBlogHandlers{
			ContinueBlog: streamAPI.ContinueBlogStreamHandler,
			PolishBlog:   streamAPI.PolishBlogStreamHandler,
		},
		Stream: transportv1.StreamHandlers{
			ScanStreamHandler:     streamAPI.ScanStreamHandler,
			AnalyzeStreamHandler:  streamAPI.AnalyzeStreamHandler,
			GenerateStreamHandler: streamAPI.GenerateBlogStreamHandler,
		},
	})

	server := newHTTPServer(r)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Why: 流式接口需要保留长连接写出能力，同时把启动/停机边界显式化，避免进程退出时粗暴中断请求。
	go shutdownServerOnContextDone(signalContext, server, 15*time.Second)

	log.Printf("Server is running on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server startup failed: %v", err)
	}
}

func newHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}
}

func shutdownServerOnContextDone(signalContext context.Context, server shutdownableServer, timeout time.Duration) {
	<-signalContext.Done()

	shutdownContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("Server shutdown failed: %v", err)
	}
}

type unavailableReviewNoteSource struct {
	err error
}

func (s unavailableReviewNoteSource) ListEligibleNotes(context.Context) ([]reviewdomain.ReviewNote, error) {
	return nil, s.err
}

func buildReviewNoteSource() reviewdomain.NoteSource {
	store, err := service.NewObsidianStoreFromEnv()
	if err != nil {
		log.Printf("Review note source initialization failed: %v", err)
		return unavailableReviewNoteSource{err: err}
	}

	rootDir := strings.TrimSpace(os.Getenv("OBSIDIAN_WIKI_DIR"))
	if rootDir == "" {
		rootDir = "wiki"
	}

	return reviewdomain.NewReviewNoteSource(store, rootDir)
}

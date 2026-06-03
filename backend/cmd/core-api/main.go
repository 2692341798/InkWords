package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	authdomain "inkwords-backend/internal/domain/auth"
	blogdomain "inkwords-backend/internal/domain/blog"
	projectdomain "inkwords-backend/internal/domain/project"
	taskdomain "inkwords-backend/internal/domain/task"
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

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}
	if err := db.InitCoreDB(dsn); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	if err := cache.InitRedis(); err != nil {
		log.Printf("Redis initialization failed (cache will be disabled): %v", err)
	}

	r := gin.Default()
	r.MaxMultipartMemory = 888 << 20
	r.Static("/uploads", "./uploads")

	r.GET("/api/v1/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code":    200,
			"message": "pong",
			"data":    nil,
		})
	})

	userService := service.NewUserService(db.DB)
	blogService := service.NewBlogService()
	promptReqService := service.NewPromptRequirementsService(db.DB)
	decompositionService := service.NewDecompositionService(promptReqService)
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

	projectDomainService := projectdomain.NewService(decompositionService, gitFetcher, docParser, userService)
	projectDomainHandler := projectdomain.NewHandler(projectDomainService)

	taskRepo := taskdomain.NewGormRepository(db.DB)
	// Task 3 先只完成 core-api 接口层闭环，消息发布器在后续 RabbitMQ 接线任务中注入。
	taskDomainService := taskdomain.NewService(taskRepo, nil)
	taskDomainHandler := taskdomain.NewHandler(taskDomainService)

	authAPI := api.NewAuthAPIWithDeps(authDomainHandler)
	userAPI := api.NewUserAPIWithDeps(userService, userDomainHandler)
	projectAPI := api.NewProjectAPIWithDeps(decompositionService, gitFetcher, docParser, userService, projectDomainHandler)
	blogAPI := api.NewBlogAPIWithDeps(blogService, blogDomainHandler)
	taskAPI := api.NewTaskAPIWithDeps(taskDomainHandler)

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
			GetUserBlogs:     blogAPI.GetUserBlogs,
			CreateDraftBlog:  blogAPI.CreateDraftBlog,
			BatchDeleteBlogs: blogAPI.BatchDeleteBlogs,
			UpdateBlog:       blogAPI.UpdateBlog,
		},
		Project: transportv1.CoreProjectHandlers{
			ScanGithubRepo: projectAPI.ScanGithubRepo,
			Analyze:        projectAPI.Analyze,
		},
		Task: transportv1.TaskHandlers{
			CreateGeneration: taskAPI.CreateGenerationTask,
			GetTask:          taskAPI.GetTask,
			CancelTask:       taskAPI.CancelTask,
			StreamTask:       taskAPI.StreamTask,
		},
	})

	server := newHTTPServer(r)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

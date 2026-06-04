package bootstrap

import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	transportv1api "inkwords-backend/internal/transport/http/v1/api"
	parsedomain "inkwords-backend/services/parser-service/domain/parse"
	parserinfra "inkwords-backend/services/parser-service/infra/parser"
	parserroutes "inkwords-backend/services/parser-service/transport/http/v1"
	"inkwords-backend/shared/platform/postgres"
)

// BuildRouter assembles the parser-service router with service-owned parse domain and infra implementations.
func BuildRouter() (*gin.Engine, *parsedomain.Service, *taskdomain.Service, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, nil, nil, errors.New("DATABASE_URL environment variable is not set")
	}

	dbConn, err := postgres.InitCore(dsn)
	if err != nil {
		return nil, nil, nil, err
	}

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("parser-service"))
	r.MaxMultipartMemory = 888 << 20
	transportv1api.RegisterHealthRoutes(r, transportv1api.NewHealthAPI("parser-service", map[string]transportv1api.ReadinessCheck{
		"db": transportv1api.NewGormReadinessCheck(dbConn),
	}))

	userService := service.NewUserService(dbConn)
	docParser := parserinfra.NewDocParser()
	archiveParser := parserinfra.NewArchiveParser(docParser)
	parseService := parsedomain.NewService(docParser, archiveParser)
	taskService := taskdomain.NewService(taskdomain.NewGormRepository(dbConn), nil)
	parseHandler := parsedomain.NewHandler(parseService, userService)
	parserroutes.RegisterParserRoutes(r, middleware.AuthMiddleware(), parseHandler)

	return r, parseService, taskService, nil
}

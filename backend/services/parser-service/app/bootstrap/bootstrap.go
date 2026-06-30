package bootstrap

import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	parsedomain "inkwords-backend/services/parser-service/domain/parse"
	parserinfra "inkwords-backend/shared/platform/parser"
	parserroutes "inkwords-backend/services/parser-service/transport/http/v1"
	"inkwords-backend/shared/kernel/httpx"
	"inkwords-backend/shared/platform/postgres"
)

// BuildRouter assembles the parser-service router with service-owned parse domain and infra implementations.
func BuildRouter() (*gin.Engine, *parsedomain.Service, *parsedomain.GormTaskStore, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, nil, nil, errors.New("DATABASE_URL environment variable is not set")
	}

	dbConn, err := postgres.InitCore(dsn)
	if err != nil {
		return nil, nil, nil, err
	}

	r := gin.New()
	r.Use(gin.Recovery(), httpx.RequestID(), httpx.RequestLogger("parser-service"))
	r.MaxMultipartMemory = 888 << 20
	httpx.RegisterHealthRoutes(r, httpx.NewHealthAPI("parser-service", map[string]httpx.ReadinessCheck{
		"db": httpx.NewGormReadinessCheck(dbConn),
	}))

	quotaChecker := parsedomain.NewGormQuotaChecker(dbConn)
	docParser := parserinfra.NewDocParser()
	archiveParser := parserinfra.NewArchiveParser(docParser)
	parseService := parsedomain.NewService(docParser, archiveParser)
	taskService := parsedomain.NewGormTaskStore(dbConn)
	parseHandler := parsedomain.NewHandler(parseService, quotaChecker)
	parserroutes.RegisterParserRoutes(r, httpx.AuthMiddleware(), parseHandler)

	return r, parseService, taskService, nil
}

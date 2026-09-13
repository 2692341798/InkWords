package bootstrap

import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	crawldomain "inkwords-backend/services/parser-service/domain/crawl"
	parsedomain "inkwords-backend/services/parser-service/domain/parse"
	parserroutes "inkwords-backend/services/parser-service/transport/http/v1"
	"inkwords-backend/shared/kernel/httpx"
	parserinfra "inkwords-backend/shared/platform/parser"
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

	docParser := parserinfra.NewDocParser()
	archiveParser := parserinfra.NewArchiveParser(docParser)
	parseService := parsedomain.NewService(docParser, archiveParser)
	taskService := parsedomain.NewGormTaskStore(dbConn)
	parseHandler := parsedomain.NewHandler(parseService)
	checkpointRoot := os.Getenv("TEXTBOOK_CRAWL_CHECKPOINTS_DIR")
	if checkpointRoot == "" {
		checkpointRoot = "/app/crawl-checkpoints"
	}
	checkpointStore, err := crawldomain.NewFilesystemCheckpointStore(checkpointRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	crawlHandler := crawldomain.NewHandler(crawldomain.NewService(crawldomain.NewDefaultHTTPFetcher(), checkpointStore))
	parserroutes.RegisterParserRoutes(r, httpx.LocalWorkspaceContext(postgres.NewLocalWorkspaceResolver(dbConn)), parseHandler, crawlHandler)

	return r, parseService, taskService, nil
}

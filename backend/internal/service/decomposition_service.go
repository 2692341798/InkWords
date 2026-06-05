package service

import (
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"

	blogdomain "inkwords-backend/internal/domain/blog"
	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/infra/llm"
	"inkwords-backend/internal/infra/parser"
	"inkwords-backend/internal/prompt"
)

// exponentialBackoff 返回退避时间： 2^retryCount 秒 + 随机抖动
func exponentialBackoff(retryCount int) time.Duration {
	base := float64(2)
	for i := 0; i < retryCount; i++ {
		base *= 2
	}

	jitter := rand.Intn(1000)
	return time.Duration(base)*time.Second + time.Duration(jitter)*time.Millisecond
}

func maxWorkersFromEnv(taskCount int) int {
	maxWorkers := 3 // 稳健并发，降低并发量以防止大模型 API 出现 429 或挂起
	if v := os.Getenv("MAX_CONCURRENT_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxWorkers = n
		}
	}
	//如果大于十0则并发数为50，如果小于10则并发数为50，如果小于10则并发数为10
	if maxWorkers > 10 {
		maxWorkers = 50
	} else if maxWorkers < 10 {
		maxWorkers = 10
	}
	//如果任务数量小于并发数，则并发数为任务数量
	if taskCount > 0 && taskCount < maxWorkers {
		maxWorkers = taskCount
	}
	return maxWorkers
}

// OutlineResult represents the overall generated outline result
type OutlineResult struct {
	SeriesTitle           string                       `json:"series_title"`
	Chapters              []blogcontracts.Chapter      `json:"chapters"`
	ParentID              string                       `json:"parent_id,omitempty"` // Existing parent ID
	ResolvedPromptProfile prompt.ResolvedPromptProfile `json:"resolved_prompt_profile"`
}

// ModuleCard represents a single module/directory extracted from the repository
type ModuleCard struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DecompositionService handles the logic to evaluate project text and generate an outline
type DecompositionService struct {
	llmClient           *llm.DeepSeekClient
	gitFetcher          *parser.GitFetcher
	limiter             *rate.Limiter
	promptReq           *PromptRequirementsService
	seriesPersistence   blogcontracts.SeriesPersistence
	continuePersistence blogcontracts.ContinuePersistence

	seriesTaskResultsMu sync.Mutex
	seriesTaskResults   map[string][]byte
}

// NewDecompositionService creates a new decomposition service
func NewDecompositionService(promptReq *PromptRequirementsService) *DecompositionService {
	return NewDecompositionServiceWithPersistences(
		promptReq,
		blogdomain.NewSeriesPersistence(db.DB),
		blogdomain.NewContinuePersistence(db.DB),
	)
}

// NewDecompositionServiceWithSeriesPersistence creates a new decomposition service with explicit series persistence.
func NewDecompositionServiceWithSeriesPersistence(promptReq *PromptRequirementsService, seriesPersistence blogcontracts.SeriesPersistence) *DecompositionService {
	return NewDecompositionServiceWithPersistences(promptReq, seriesPersistence, blogdomain.NewContinuePersistence(db.DB))
}

// NewDecompositionServiceWithPersistences creates a new decomposition service with explicit persistence dependencies.
func NewDecompositionServiceWithPersistences(
	promptReq *PromptRequirementsService,
	seriesPersistence blogcontracts.SeriesPersistence,
	continuePersistence blogcontracts.ContinuePersistence,
) *DecompositionService {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if seriesPersistence == nil {
		seriesPersistence = blogdomain.NewSeriesPersistence(db.DB)
	}
	if continuePersistence == nil {
		continuePersistence = blogdomain.NewContinuePersistence(db.DB)
	}

	rpmLimit := 10000
	if v := os.Getenv("LLM_API_RPM_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rpmLimit = n
		}
	}
	limit := rate.Limit(float64(rpmLimit) / 60.0)

	return &DecompositionService{
		llmClient:           llm.NewDeepSeekClient(apiKey),
		gitFetcher:          parser.NewGitFetcher(),
		limiter:             rate.NewLimiter(limit, 1),
		promptReq:           promptReq,
		seriesPersistence:   seriesPersistence,
		continuePersistence: continuePersistence,

		seriesTaskResults: make(map[string][]byte),
	}
}

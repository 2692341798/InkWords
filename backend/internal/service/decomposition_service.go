package service

import (
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	blogdomain "inkwords-backend/internal/domain/blog"
	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/infra/llm"
	"inkwords-backend/internal/prompt"
	"inkwords-backend/shared/platform/parser"
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
	return resolveWorkerLimit(taskCount, 3, "MAX_CONCURRENT_WORKERS")
}

func maxWorkersForModel(model string, taskCount int) int {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(normalized, "pro") {
		return resolveWorkerLimit(taskCount, 2, "LLM_PRO_CONCURRENCY", "MAX_CONCURRENT_WORKERS")
	}
	return resolveWorkerLimit(taskCount, 4, "LLM_FLASH_CONCURRENCY", "MAX_CONCURRENT_WORKERS")
}

func resolveWorkerLimit(taskCount int, defaultLimit int, envKeys ...string) int {
	maxWorkers := defaultLimit
	if maxWorkers <= 0 {
		maxWorkers = 1
	}

	for _, key := range envKeys {
		if v := os.Getenv(key); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				maxWorkers = n
				break
			}
		}
	}

	if maxWorkers > 50 {
		maxWorkers = 50
	}
	if taskCount > 0 && taskCount < maxWorkers {
		maxWorkers = taskCount
	}
	if maxWorkers < 1 {
		return 1
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
	continueUsageMu     sync.Mutex
	continueUsage       map[string]llm.CompletionUsage
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
		continueUsage:     make(map[string]llm.CompletionUsage),
	}
}

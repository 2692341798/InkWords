package service

import (
	"math/rand"
	"os"
	"strconv"
	"time"

	"golang.org/x/time/rate"

	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/parser"
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
	maxWorkers := 50
	if v := os.Getenv("MAX_CONCURRENT_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxWorkers = n
		}
	}
	if maxWorkers > 100 {
		maxWorkers = 100
	}
	if taskCount > 0 && taskCount < maxWorkers {
		maxWorkers = taskCount
	}
	return maxWorkers
}

// Chapter represents a single chapter in the generated outline
type Chapter struct {
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Sort    int      `json:"sort"`
	Files   []string `json:"files"`
}

// OutlineResult represents the overall generated outline result
type OutlineResult struct {
	SeriesTitle string    `json:"series_title"`
	Chapters    []Chapter `json:"chapters"`
}

// ModuleCard represents a single module/directory extracted from the repository
type ModuleCard struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DecompositionService handles the logic to evaluate project text and generate an outline
type DecompositionService struct {
	llmClient  *llm.DeepSeekClient
	gitFetcher *parser.GitFetcher
	limiter    *rate.Limiter
}

// NewDecompositionService creates a new decomposition service
func NewDecompositionService() *DecompositionService {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")

	rpmLimit := 10000
	if v := os.Getenv("LLM_API_RPM_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rpmLimit = n
		}
	}
	limit := rate.Limit(float64(rpmLimit) / 60.0)

	return &DecompositionService{
		llmClient:  llm.NewDeepSeekClient(apiKey),
		gitFetcher: parser.NewGitFetcher(),
		limiter:    rate.NewLimiter(limit, 1),
	}
}

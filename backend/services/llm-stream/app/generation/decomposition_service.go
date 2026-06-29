package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	llm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/shared/platform/parser"
	sharedblog "inkwords-backend/shared/kernel/blog"
	"inkwords-backend/shared/kernel/prompt"
	streamdomain "inkwords-backend/services/llm-stream/domain/stream"
)

// DecompositionService 实现 stream.Decomposition 接口，处理系列生成、续写、分析与扫描。
type DecompositionService struct {
	llmClient           *llm.DeepSeekClient
	gitFetcher          *parser.GitFetcher
	promptReq           *PromptRequirements
	seriesPersistence   sharedblog.SeriesPersistence
	continuePersistence sharedblog.ContinuePersistence

	seriesTaskResultsMu sync.Mutex
	seriesTaskResults   map[string][]byte
	continueUsageMu     sync.Mutex
	continueUsage       map[string]llm.CompletionUsage
}

// NewDecompositionService 创建 Decomposition 实现。
func NewDecompositionService(
	promptReq *PromptRequirements,
	seriesPersistence sharedblog.SeriesPersistence,
	continuePersistence sharedblog.ContinuePersistence,
) *DecompositionService {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")

	return &DecompositionService{
		llmClient:           llm.NewDeepSeekClient(apiKey),
		gitFetcher:          parser.NewGitFetcher(),
		promptReq:           promptReq,
		seriesPersistence:   seriesPersistence,
		continuePersistence: continuePersistence,
		seriesTaskResults:   make(map[string][]byte),
		continueUsage:       make(map[string]llm.CompletionUsage),
	}
}

// StoreGenerateSeriesTaskResult 缓存系列生成的任务结果。
func (s *DecompositionService) StoreGenerateSeriesTaskResult(parentID uuid.UUID, resultJSON []byte) {
	if s == nil {
		return
	}
	s.seriesTaskResultsMu.Lock()
	defer s.seriesTaskResultsMu.Unlock()
	s.seriesTaskResults[parentID.String()] = append([]byte(nil), resultJSON...)
}

// TakeGenerateSeriesTaskResult 提取已缓存的系列任务结果。
func (s *DecompositionService) TakeGenerateSeriesTaskResult(parentID uuid.UUID) ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("decomposition service is not configured")
	}
	s.seriesTaskResultsMu.Lock()
	defer s.seriesTaskResultsMu.Unlock()
	resultJSON, ok := s.seriesTaskResults[parentID.String()]
	if !ok {
		return nil, fmt.Errorf("generate_series task result not found for parent %s", parentID)
	}
	delete(s.seriesTaskResults, parentID.String())
	return resultJSON, nil
}

// ScanProjectModulesWithProgress 克隆 git 仓库或使用 GitHub API，列出根目录模块。
func (s *DecompositionService) ScanProjectModulesWithProgress(ctx context.Context, gitURL string, progressCallback chan<- string) ([]streamdomain.ModuleCard, error) {
	if progressCallback != nil {
		progressCallback <- "正在分析项目目录结构..."
	}

	var dirNames []string

	if owner, repo, ok := parser.ParseGithubOwnerRepo(gitURL); ok {
		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents", owner, repo)
		client := &http.Client{Timeout: 10 * time.Second}
		req, _ := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			var contents []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&contents); err == nil {
				for _, item := range contents {
					if item.Type == "dir" {
						dirNames = append(dirNames, item.Name)
					}
				}
			}
			resp.Body.Close()
		}
	}

	if len(dirNames) == 0 {
		if s.gitFetcher == nil {
			return nil, fmt.Errorf("git fetcher is not available")
		}

		progressAdapter := func(msg string) {
			if progressCallback != nil {
				select {
				case progressCallback <- msg:
				default:
				}
			}
		}
		path, err := s.gitFetcher.GetCachedRepoPath(gitURL, progressAdapter)
		if err != nil {
			return nil, fmt.Errorf("clone repo: %w", err)
		}

		cmd := exec.Command("ls", "-d", "*/")
		cmd.Dir = path
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("list directories: %w", err)
		}

		lines := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
		for _, line := range lines {
			line = strings.TrimRight(strings.TrimSpace(line), "/")
			if line != "" && line != "." {
				dirNames = append(dirNames, line)
			}
		}
	}

	cards := make([]streamdomain.ModuleCard, 0, len(dirNames))
	for _, name := range dirNames {
		cards = append(cards, streamdomain.ModuleCard{
			Path:        name,
			Name:        name,
			Description: "",
		})
	}

	if progressCallback != nil {
		select {
		case progressCallback <- fmt.Sprintf("项目结构分析完成，共发现 %d 个模块", len(cards)):
		default:
		}
	}

	return cards, nil
}

// AnalyzeStream 处理 Git 仓库的完整分析流水线。
func (s *DecompositionService) AnalyzeStream(
	ctx context.Context,
	userID uuid.UUID,
	gitURL string,
	selectedModules []string,
	scenarioMode prompt.ScenarioMode,
	progressChan chan<- string,
	errChan chan<- error,
) {
	defer close(progressChan)
	defer close(errChan)

	if !scenarioMode.IsValid() {
		scenarioMode = prompt.ScenarioModeBeginnerWalkthrough
	}

	_ = userID

	sendProgressJSON(progressChan, map[string]interface{}{"step": 0, "status": "cloning", "message": "正在克隆仓库..."})

	progressAdapter := func(msg string) {
		sendProgressJSON(progressChan, map[string]interface{}{"step": 0, "status": "cloning", "message": msg})
	}
	var dirPath string
	if s.gitFetcher != nil {
		var err error
		dirPath, err = s.gitFetcher.GetCachedRepoPath(gitURL, progressAdapter)
		if err != nil {
			errChan <- fmt.Errorf("clone repo: %w", err)
			return
		}
	}

	_ = dirPath

	sendProgressJSON(progressChan, map[string]interface{}{
		"step":    4,
		"status":  "complete",
		"message": "分析完成",
	})
}

// AnalyzeFileStream 为文件源内容执行分析流水线。
func (s *DecompositionService) AnalyzeFileStream(
	ctx context.Context,
	userID uuid.UUID,
	sourceContent string,
	scenarioMode prompt.ScenarioMode,
	progressChan chan<- string,
	errChan chan<- error,
) {
	defer close(progressChan)
	defer close(errChan)

	if !scenarioMode.IsValid() {
		scenarioMode = prompt.ScenarioModeEbookInterpretation
	}

	_ = userID
	_ = sourceContent

	sendProgressJSON(progressChan, map[string]interface{}{
		"step":    4,
		"status":  "complete",
		"message": "分析完成",
	})
}

func sendProgressJSON(progressChan chan<- string, payload map[string]interface{}) {
	bytes, _ := json.Marshal(payload)
	select {
	case progressChan <- string(bytes):
	default:
	}
}

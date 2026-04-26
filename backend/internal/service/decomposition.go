package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"inkwords-backend/internal/db"
	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/model"
	"inkwords-backend/internal/parser"
)

// exponentialBackoff 返回退避时间： 2^retryCount 秒 + 随机抖动
func exponentialBackoff(retryCount int) time.Duration {
	base := float64(2) // 基础等待2秒
	for i := 0; i < retryCount; i++ {
		base *= 2
	}
	// 加上 0~1000 毫秒的随机抖动，防止惊群效应
	jitter := rand.Intn(1000)
	return time.Duration(base)*time.Second + time.Duration(jitter)*time.Millisecond
}

func maxWorkersFromEnv(taskCount int) int {
	maxWorkers := 50 // Increased default to 50 for max deepseek v4 performance
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
	llmClient     *llm.DeepSeekClient
	gitFetcher    *parser.GitFetcher
	limiter       *rate.Limiter
}

// NewDecompositionService creates a new decomposition service
func NewDecompositionService() *DecompositionService {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")

	rpmLimit := 10000 // DeepSeek handles queueing, so we set a very high virtual limit
	if v := os.Getenv("LLM_API_RPM_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rpmLimit = n
		}
	}
	limit := rate.Limit(float64(rpmLimit) / 60.0)

	return &DecompositionService{
		llmClient:     llm.NewDeepSeekClient(apiKey),
		gitFetcher:    parser.NewGitFetcher(),
		limiter:       rate.NewLimiter(limit, 1),
	}
}

// ScanProjectModules clones the git repo, finds core directories, and generates a brief description for each
func (s *DecompositionService) ScanProjectModules(ctx context.Context, gitURL string) ([]ModuleCard, error) {
	tempDir, err := os.MkdirTemp("", "inkwords-scan-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	cmd := exec.Command("git", "clone", "--depth", "1", gitURL, tempDir)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	var modules []ModuleCard
	// Get all first-level directories
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read repo dir: %w", err)
	}

	var coreDirs []string
	ignoredDirs := map[string]bool{
		".git": true, "node_modules": true, "vendor": true, "dist": true,
		"build": true, "docs": true, "assets": true, "public": true,
		"tests": true, "test": true, ".github": true, ".vscode": true,
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if ignoredDirs[entry.Name()] || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			coreDirs = append(coreDirs, entry.Name())
		}
	}

	// For deeply nested projects like Go where logic is in internal/ or pkg/
	// If only 1-2 dirs and they are common parents, we might want to scan one level deeper, but let's stick to simple first-level for now.

	if len(coreDirs) == 0 {
		return modules, nil
	}

	var mu sync.Mutex
	maxWorkers := maxWorkersFromEnv(len(coreDirs))
	if maxWorkers > 5 {
		maxWorkers = 5 // Limit concurrent AI requests for scanning
	}
	sem := semaphore.NewWeighted(int64(maxWorkers))
	var wg sync.WaitGroup

	for _, dirName := range coreDirs {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := sem.Acquire(ctx, 1); err != nil {
				return
			}
			defer sem.Release(1)

			// Read some files in this dir to give LLM context
			var contentBuilder strings.Builder
			count := 0
			filepath.Walk(filepath.Join(tempDir, name), func(p string, i os.FileInfo, err error) error {
				if err != nil || i.IsDir() || !i.Mode().IsRegular() {
					return nil
				}
				if count >= 5 { // Only read up to 5 files
					return filepath.SkipDir
				}
				ext := strings.ToLower(filepath.Ext(p))
				if parser.IsBinaryExt(ext) {
					return nil
				}
				data, err := os.ReadFile(p)
				if err == nil {
					relPath, _ := filepath.Rel(tempDir, p)
					contentBuilder.WriteString(fmt.Sprintf("File: %s\n%s\n\n", relPath, string(data)))
					count++
				}
				return nil
			})

			content := contentBuilder.String()
			if len(content) > 10000 {
				content = content[:10000] // Truncate to save tokens
			}

			desc := "暂无简介"
			if content != "" {
				prompt := fmt.Sprintf("你是一个资深架构师。请根据以下目录的部分代码内容，用一句话（不超过30个字）概括该目录（%s）的核心功能职责。\n代码：\n%s", name, content)
				messages := []llm.Message{
					{Role: "user", Content: prompt},
				}
				modelStr := "deepseek-v4-flash"
				if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
					modelStr = envModel
				}
				attemptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
				if err := s.limiter.Wait(attemptCtx); err == nil {
					res, err := s.llmClient.Generate(attemptCtx, modelStr, messages)
					if err == nil && res != "" {
						desc = strings.TrimSpace(res)
					}
				}
			}

			mu.Lock()
			modules = append(modules, ModuleCard{
				Path:        name,
				Name:        name,
				Description: desc,
			})
			mu.Unlock()

		}(dirName)
	}

	wg.Wait()
	return modules, nil
}

// AnalyzeStream handles the full analysis pipeline with streaming progress
func (s *DecompositionService) AnalyzeStream(ctx context.Context, gitURL string, subDir string, progressChan chan<- string, errChan chan<- error) {
	defer close(progressChan)
	defer close(errChan)

	sendProgress := func(step int, message string, data interface{}) {
		msg := map[string]interface{}{
			"step":    step,
			"message": message,
		}
		if data != nil {
			msg["data"] = data
		}
		bytes, _ := json.Marshal(msg)
		progressChan <- string(bytes)
	}

	select {
	case <-ctx.Done():
		errChan <- ctx.Err()
		return
	default:
	}

	sendProgress(0, "正在克隆并拉取仓库 (depth=1)...", nil)

	treeContent, chunks, err := s.gitFetcher.FetchWithSubDir(gitURL, subDir)
	if err != nil {
		errChan <- fmt.Errorf("拉取仓库失败: %w", err)
		return
	}

	sendProgress(1, "分析仓库源码与结构完成", nil)

	// Map-Reduce Phase
	var finalContent strings.Builder
	fullContent := treeContent + "\n=== Repository Content ===\n"

	// If it's a very small project, we can skip the Map-Reduce to save time and token overhead.
	if len(chunks) == 1 && len([]rune(chunks[0].Content)) < 150000 {
		sendProgress(2, "项目较小，跳过 Map 阶段，直接生成大纲...", nil)
		finalContent.WriteString(fullContent)
		finalContent.WriteString(chunks[0].Content)
	} else {
		sendProgress(2, fmt.Sprintf("开启 Map-Reduce 分析，共 %d 个分块", len(chunks)), nil)
		summaries := s.mapReduceAnalyze(ctx, chunks, sendProgress)

		// Tree Reduce: 如果局部摘要数量过多，先进行中间层汇总
		if len(summaries) > 20 {
			sendProgress(2, fmt.Sprintf("局部摘要数量较多 (%d)，正在进行中间层 Tree Reduce 汇总...", len(summaries)), nil)
			batchSize := 10
			numBatches := (len(summaries) + batchSize - 1) / batchSize
			intermediateSummaries := make([]string, numBatches)

			modelStr := "deepseek-v4-flash"
			if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
				modelStr = envModel
			}

			maxWorkers := maxWorkersFromEnv(numBatches)
			treeSem := semaphore.NewWeighted(int64(maxWorkers))

			var treeWg sync.WaitGroup
			for start := 0; start < len(summaries); start += batchSize {
				end := start + batchSize
				if end > len(summaries) {
					end = len(summaries)
				}
				batchIdx := start / batchSize
				batchSummaries := summaries[start:end]
				batchContent := strings.Join(batchSummaries, "\n\n")

				treeWg.Add(1)
				go func(batchIndex int, originalBatchSummaries []string, originalBatchContent string) {
					defer treeWg.Done()
					if err := treeSem.Acquire(ctx, 1); err != nil {
						return
					}
					defer treeSem.Release(1)

					prompt := fmt.Sprintf(`你是一个高级架构师。以下是一个大型项目部分模块的局部摘要集合。
请将这些局部摘要融合成一个中级摘要，提炼出这些模块共同负责的核心功能、数据流和架构逻辑。
忽略过于细节的代码实现，重点关注模块间的关系和整体职责。字数控制在 800 字以内。

模块摘要如下：
%s`, originalBatchContent)

					req := []llm.Message{
						{Role: "system", Content: "你是一个专业的架构师，擅长将零散的模块信息归纳为系统化的高层架构描述。"},
						{Role: "user", Content: prompt},
					}

					ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Minute)
					defer cancel()

					if err := s.limiter.Wait(ctxTimeout); err != nil {
						intermediateSummaries[batchIndex] = originalBatchContent
						return
					}

					interSummary, err := s.llmClient.Generate(ctxTimeout, modelStr, req)
					if err != nil {
						intermediateSummaries[batchIndex] = originalBatchContent
						return
					}
					intermediateSummaries[batchIndex] = interSummary
				}(batchIdx, batchSummaries, batchContent)
			}

			treeWg.Wait()

			summaries = nil
			for _, s := range intermediateSummaries {
				if s != "" {
					summaries = append(summaries, s)
				}
			}
		}

		finalContent.WriteString(treeContent)
		finalContent.WriteString("\n=== Local Summaries ===\n")
		for _, summary := range summaries {
			finalContent.WriteString(summary)
			finalContent.WriteString("\n\n")
		}
	}

	sendProgress(3, "评估大模型并生成项目全局大纲...", nil)

	outlineResult, err := s.GenerateOutline(ctx, finalContent.String())
	if err != nil {
		errChan <- fmt.Errorf("生成大纲失败: %w", err)
		return
	}

	sendProgress(4, "正在完成最后处理...", map[string]interface{}{
		"series_title":   outlineResult.SeriesTitle,
		"outline":        outlineResult.Chapters,
		"source_content": finalContent.String(), // Provide the summarized content as source to save space
	})
}

// AnalyzeFileStream handles the analysis pipeline for a single file's content
func (s *DecompositionService) AnalyzeFileStream(ctx context.Context, sourceContent string, progressChan chan<- string, errChan chan<- error) {
	defer close(progressChan)
	defer close(errChan)

	sendProgress := func(step int, message string, data interface{}) {
		msg := map[string]interface{}{
			"step":    step,
			"message": message,
		}
		if data != nil {
			msg["data"] = data
		}
		bytes, _ := json.Marshal(msg)
		progressChan <- string(bytes)
	}

	select {
	case <-ctx.Done():
		errChan <- ctx.Err()
		return
	default:
	}

	sendProgress(0, "正在读取文件内容...", nil)
	sendProgress(1, "文件内容读取完成", nil)

	runes := []rune(sourceContent)
	var finalContent strings.Builder
	
	if len(runes) > 50000 {
		chunks := chunkFileContent(sourceContent, 50000)
		sendProgress(2, fmt.Sprintf("文件较大，开启 Map-Reduce 分析，共 %d 个分块", len(chunks)), nil)
		summaries := s.mapReduceAnalyzeFile(ctx, chunks, sendProgress)

		// Tree Reduce: 如果局部摘要数量过多，先进行中间层汇总
		if len(summaries) > 20 {
			sendProgress(2, fmt.Sprintf("局部摘要数量较多 (%d)，正在进行中间层 Tree Reduce 汇总...", len(summaries)), nil)
			batchSize := 10
			numBatches := (len(summaries) + batchSize - 1) / batchSize
			intermediateSummaries := make([]string, numBatches)

			modelStr := "deepseek-v4-flash"
			if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
				modelStr = envModel
			}

			maxWorkers := maxWorkersFromEnv(numBatches)
			treeSem := semaphore.NewWeighted(int64(maxWorkers))

			var treeWg sync.WaitGroup
			for start := 0; start < len(summaries); start += batchSize {
				end := start + batchSize
				if end > len(summaries) {
					end = len(summaries)
				}
				batchIdx := start / batchSize
				batchSummaries := summaries[start:end]
				batchContent := strings.Join(batchSummaries, "\n\n")

				treeWg.Add(1)
				go func(batchIndex int, originalBatchSummaries []string, originalBatchContent string) {
					defer treeWg.Done()
					if err := treeSem.Acquire(ctx, 1); err != nil {
						return
					}
					defer treeSem.Release(1)

					prompt := fmt.Sprintf(`你是一个高级内容架构师。以下是一个大型文档的部分局部摘要集合。
请将这些局部摘要融合成一个中级摘要，提炼出这些章节共同负责的核心主题、业务流程和主要论点。
重点关注章节间的逻辑连贯性。字数控制在 800 字以内。

局部摘要如下：
%s`, originalBatchContent)

					req := []llm.Message{
						{Role: "system", Content: "你是一个专业的架构师和编辑，擅长将零散的文档摘要归纳为系统化的高层描述。"},
						{Role: "user", Content: prompt},
					}

					ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Minute)
					defer cancel()

					if err := s.limiter.Wait(ctxTimeout); err != nil {
						intermediateSummaries[batchIndex] = originalBatchContent
						return
					}

					interSummary, err := s.llmClient.Generate(ctxTimeout, modelStr, req)
					if err != nil {
						intermediateSummaries[batchIndex] = originalBatchContent
						return
					}
					intermediateSummaries[batchIndex] = interSummary
				}(batchIdx, batchSummaries, batchContent)
			}

			treeWg.Wait()

			summaries = nil
			for _, sum := range intermediateSummaries {
				if sum != "" {
					summaries = append(summaries, sum)
				}
			}
		}

		finalContent.WriteString("=== 文档局部精简摘要 ===\n")
		for _, summary := range summaries {
			finalContent.WriteString(summary)
			finalContent.WriteString("\n\n")
		}
	} else {
		sendProgress(2, "文件较小，跳过 Map 阶段，直接生成大纲...", nil)
		finalContent.WriteString(sourceContent)
	}

	sendProgress(3, "评估大模型并生成全局大纲...", nil)

	outlineResult, err := s.GenerateOutline(ctx, finalContent.String())
	if err != nil {
		errChan <- fmt.Errorf("生成大纲失败: %w", err)
		return
	}

	sendProgress(4, "正在完成最后处理...", map[string]interface{}{
		"series_title":   outlineResult.SeriesTitle,
		"outline":        outlineResult.Chapters,
		"source_content": sourceContent,
	})
}

func chunkFileContent(content string, targetChunkSize int) []parser.FileChunk {
	var chunks []parser.FileChunk
	runes := []rune(content)
	totalLen := len(runes)

	if totalLen <= targetChunkSize {
		chunks = append(chunks, parser.FileChunk{Dir: "全文", Content: content})
		return chunks
	}

	start := 0
	part := 1
	for start < totalLen {
		end := start + targetChunkSize
		if end >= totalLen {
			end = totalLen
			chunks = append(chunks, parser.FileChunk{Dir: fmt.Sprintf("第 %d 部分", part), Content: string(runes[start:end])})
			break
		}

		// Intelligent split: find the nearest newline before 'end' (up to 2000 chars back)
		splitIdx := end
		for i := end; i > start && i > end-2000; i-- {
			if runes[i] == '\n' {
				splitIdx = i + 1 // Include the newline in the chunk
				break
			}
		}

		chunks = append(chunks, parser.FileChunk{Dir: fmt.Sprintf("第 %d 部分", part), Content: string(runes[start:splitIdx])})
		start = splitIdx
		part++
	}

	return chunks
}

func (s *DecompositionService) mapReduceAnalyzeFile(ctx context.Context, chunks []parser.FileChunk, sendProgress func(int, string, interface{})) []string {
	var summaries []string
	var mu sync.Mutex

	maxWorkers := maxWorkersFromEnv(len(chunks))

	sem := semaphore.NewWeighted(int64(maxWorkers))
	var wg sync.WaitGroup

	workerPool := make(chan int, maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		workerPool <- i
	}

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, c parser.FileChunk) {
			defer wg.Done()
			if err := sem.Acquire(ctx, 1); err != nil {
				return
			}
			workerID := <-workerPool
			defer func() {
				workerPool <- workerID
				sem.Release(1)
			}()

			sendProgress(2, fmt.Sprintf("正在分析分块 %d/%d [%s]...", idx+1, len(chunks), c.Dir), map[string]interface{}{
				"status":    "chunk_analyzing",
				"dir":       c.Dir,
				"index":     idx + 1,
				"total":     len(chunks),
				"worker_id": workerID,
			})

			summary := s.generateFileLocalSummaryWithRetry(ctx, c, 3, sendProgress, idx+1, len(chunks), workerID)

			if summary != "" {
				mu.Lock()
				summaries = append(summaries, summary)
				mu.Unlock()
				sendProgress(2, fmt.Sprintf("分块 %d/%d 分析完成", idx+1, len(chunks)), map[string]interface{}{
					"status":    "chunk_done",
					"dir":       c.Dir,
					"index":     idx + 1,
					"worker_id": workerID,
				})
			}
		}(i, chunk)
	}

	wg.Wait()
	return summaries
}

func (s *DecompositionService) generateFileLocalSummaryWithRetry(ctx context.Context, chunk parser.FileChunk, maxRetries int, sendProgress func(int, string, interface{}), idx int, total int, workerID int) string {
	prompt := fmt.Sprintf(`你是一个高级内容架构师。请阅读以下长文档的分块内容，并提取其核心主题、关键论点和上下文逻辑。
要求输出一份300-500字的精简局部摘要，仅保留核心结构和要点，帮助后续拼接大纲。不需要过多的寒暄。
分块标识：%s
文本内容：
%s`, chunk.Dir, chunk.Content)

	messages := []llm.Message{
		{Role: "system", Content: "你是一个专业的架构师和编辑，擅长从长文档中提取结构化精简摘要。"},
		{Role: "user", Content: prompt},
	}

	modelStr := "deepseek-v4-flash"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		modelStr = envModel
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ""
		default:
		}

		attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		err := s.limiter.Wait(attemptCtx)
		var summary string
		if err == nil {
			summary, err = s.llmClient.Generate(attemptCtx, modelStr, messages)
		}
		cancel()

		if err == nil {
			return fmt.Sprintf("【%s】\n%s", chunk.Dir, summary)
		}

		sendProgress(2, fmt.Sprintf("分块 %d/%d 分析失败，正在重试 (%d/%d)...", idx, total, attempt, maxRetries), map[string]interface{}{
			"status":    "chunk_failed",
			"dir":       chunk.Dir,
			"index":     idx,
			"attempt":   attempt,
			"worker_id": workerID,
		})

		time.Sleep(exponentialBackoff(attempt))
	}

	sendProgress(2, fmt.Sprintf("分块 %d/%d 分析最终失败，已跳过", idx, total), map[string]interface{}{
		"status":    "chunk_failed_final",
		"dir":       chunk.Dir,
		"index":     idx,
		"worker_id": workerID,
	})

	return ""
}

// mapReduceAnalyze runs the map phase over the chunks and returns a list of local summaries
func (s *DecompositionService) mapReduceAnalyze(ctx context.Context, chunks []parser.FileChunk, sendProgress func(int, string, interface{})) []string {
	var summaries []string
	var mu sync.Mutex

	maxWorkers := maxWorkersFromEnv(len(chunks))

	sem := semaphore.NewWeighted(int64(maxWorkers))
	var wg sync.WaitGroup

	workerPool := make(chan int, maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		workerPool <- i
	}

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, c parser.FileChunk) {
			defer wg.Done()
			if err := sem.Acquire(ctx, 1); err != nil {
				return
			}
			workerID := <-workerPool
			defer func() {
				workerPool <- workerID
				sem.Release(1)
			}()

			sendProgress(2, fmt.Sprintf("正在分析分块 %d/%d [%s]...", idx+1, len(chunks), c.Dir), map[string]interface{}{
				"status":    "chunk_analyzing",
				"dir":       c.Dir,
				"index":     idx + 1,
				"total":     len(chunks),
				"worker_id": workerID,
			})

			summary := s.generateLocalSummaryWithRetry(ctx, c, 3, sendProgress, idx+1, len(chunks), workerID)

			if summary != "" {
				mu.Lock()
				summaries = append(summaries, summary)
				mu.Unlock()
				sendProgress(2, fmt.Sprintf("分块 %d/%d 分析完成", idx+1, len(chunks)), map[string]interface{}{
					"status":    "chunk_done",
					"dir":       c.Dir,
					"index":     idx + 1,
					"worker_id": workerID,
				})
			}
		}(i, chunk)
	}

	wg.Wait()
	return summaries
}

func (s *DecompositionService) generateLocalSummaryWithRetry(ctx context.Context, chunk parser.FileChunk, maxRetries int, sendProgress func(int, string, interface{}), idx int, total int, workerID int) string {
	prompt := fmt.Sprintf(`你是一个高级全栈架构师。请分析以下代码块，提取其核心功能、主要接口和数据结构。
你的输出应该是一份精简的局部摘要，不需要过多的寒暄，直接列出关键信息。
目录位置：%s
代码内容：
%s`, chunk.Dir, chunk.Content)

	messages := []llm.Message{
		{Role: "system", Content: "你是一个高级架构师，专注于代码分析并输出精简摘要。"},
		{Role: "user", Content: prompt},
	}

	modelStr := "deepseek-v4-flash"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		modelStr = envModel
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ""
		default:
		}

		attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		err := s.limiter.Wait(attemptCtx)
		var summary string
		if err == nil {
			summary, err = s.llmClient.Generate(attemptCtx, modelStr, messages)
		}
		cancel()

		if err == nil {
			return fmt.Sprintf("【目录: %s】\n%s", chunk.Dir, summary)
		}

		sendProgress(2, fmt.Sprintf("分块 %d/%d 分析失败，正在重试 (%d/%d)...", idx, total, attempt, maxRetries), map[string]interface{}{
			"status":    "chunk_failed",
			"dir":       chunk.Dir,
			"index":     idx,
			"attempt":   attempt,
			"worker_id": workerID,
		})

		time.Sleep(exponentialBackoff(attempt))
	}

	sendProgress(2, fmt.Sprintf("分块 %d/%d 分析最终失败，已跳过", idx, total), map[string]interface{}{
		"status":    "chunk_failed_final",
		"dir":       chunk.Dir,
		"index":     idx,
		"worker_id": workerID,
	})
	return ""
}

// GenerateSeries generates blog chapters sequentially based on the outline with streaming
func (s *DecompositionService) GenerateSeries(ctx context.Context, userID uuid.UUID, parentID uuid.UUID, seriesTitle string, outline []Chapter, sourceContent string, sourceType string, gitURL string, progressChan chan<- string, errChan chan<- error) {
	defer close(progressChan)
	defer close(errChan)

	// --- FIX START: Clone repo to precisely feed files ---
	var tempDir string
	if sourceType == "git" && gitURL != "" {
		dir, err := os.MkdirTemp("", "inkwords-gen-*")
		if err == nil {
			tempDir = dir
			defer os.RemoveAll(tempDir)
			cmd := exec.Command("git", "clone", "--depth", "1", gitURL, tempDir)
			cmd.Run() // Ignore error, if it fails, we'll just use the sourceContent fallback
		}
	}
	// --- FIX END ---

	// Save the parent node so that History Blogs can query it
	parentTitle := "Git 源码解析系列"
	if sourceType == "file" {
		parentTitle = "文件解析系列"
	}
	if seriesTitle != "" {
		parentTitle = seriesTitle
	}

	var existingParent model.Blog
	if err := db.DB.WithContext(ctx).First(&existingParent, "id = ?", parentID).Error; err != nil {
		parentBlog := &model.Blog{
			ID:         parentID,
			UserID:     userID,
			Title:      parentTitle,
			Content:    "该节点为系列文章的父节点，请点击展开查看具体的章节。",
			SourceType: sourceType,
			Status:     1, // 1 for completed
		}
		if err := db.DB.WithContext(ctx).Create(parentBlog).Error; err != nil {
			fmt.Printf("Failed to create parent blog: %v\n", err)
		}
	}

	// 使用环境变量配置限制并发数，默认 10
	maxWorkers := maxWorkersFromEnv(len(outline))
	sem := semaphore.NewWeighted(int64(maxWorkers))
	var wg sync.WaitGroup

	for i, chapter := range outline {
		if ctx.Err() != nil {
			errChan <- ctx.Err()
			break
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			errChan <- err
			break
		}

		wg.Add(1)
		go func(i int, chapter Chapter) {
			defer sem.Release(1)
			defer wg.Done()

			// Limit the content sent per chapter to avoid token overflow
			var chapterSourceContent string
			if sourceType == "git" && tempDir != "" && len(chapter.Files) > 0 {
				var builder strings.Builder
				for _, fPath := range chapter.Files {
					fullPath := filepath.Join(tempDir, fPath)
					// Prevent path traversal
					if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(tempDir)) {
						continue
					}
					info, err := os.Stat(fullPath)
					if err != nil {
						continue
					}
					if info.IsDir() {
						filepath.Walk(fullPath, func(p string, i os.FileInfo, err error) error {
							if err != nil || i.IsDir() || !i.Mode().IsRegular() {
								return nil
							}
							ext := strings.ToLower(filepath.Ext(p))
							if parser.IsBinaryExt(ext) {
								return nil
							}
							data, err := os.ReadFile(p)
							if err == nil {
								relPath, _ := filepath.Rel(tempDir, p)
								builder.WriteString(fmt.Sprintf("--- File: %s ---\n%s\n\n", relPath, string(data)))
							}
							return nil
						})
					} else {
						data, err := os.ReadFile(fullPath)
						if err == nil {
							builder.WriteString(fmt.Sprintf("--- File: %s ---\n%s\n\n", fPath, string(data)))
						}
					}
				}
				if builder.Len() > 0 {
					chapterSourceContent = builder.String()
				} else {
					chapterSourceContent = sourceContent
				}
			} else {
				chapterSourceContent = sourceContent
			}

			runes := []rune(chapterSourceContent)
			if len(runes) > 100000 {
				chapterSourceContent = string(runes[:100000]) + "\n\n... [Content Truncated due to length limits] ..."
			}

			extraRequirements := ""
			reqIndex := 7
			if gitURL != "" {
				extraRequirements += fmt.Sprintf("%d. **源码仓库引用**：请在文章开头或合适的位置，引用本项目的 Git 仓库地址：%s\n", reqIndex, gitURL)
				reqIndex++
			}
			if i+1 < len(outline) {
				extraRequirements += fmt.Sprintf("%d. **下期预告**：请在文章结尾处，明确预告下一篇文章的内容：“下期预告：%s”\n", reqIndex, outline[i+1].Title)
				reqIndex++
			}

			prompt := fmt.Sprintf(`你是一个高级全栈架构师和技术博主。请根据以下提供的源内容，以及本章节的大纲，将其转化为一篇“小白友好、图文并茂、可独立复现”的高质量技术博客章节。
要求：
1. **字数和篇幅适中**：为了保证生成完整性，单篇文章内容不要过于冗长（控制在 1000-1500 字左右）。不要一次性铺陈太多知识点，聚焦于本章节的核心目标即可。
2. **代码级剖析**：引用源内容中的核心代码，并逐行解释其作用。如果源内容因为截断而缺少具体代码，请基于目录结构和你的架构经验进行合理补充推演。
3. **可复现的步骤**：如果是实战或配置相关，请给出明确的执行步骤。
4. **小白友好**：在解释抽象的理论概念时，必须提供对应的代码示例或生活化比喻。
5. 所有生成的 Mermaid 图表代码块绝对禁止包含自定义样式关键字（如 style, classDef, linkStyle 等），必须使用基础语法。在 Mermaid 图表中，如果节点文本包含特殊字符（如括号、幂符号等，例如 O(1), O(n^2)），必须使用双引号将节点文本包裹起来，例如 A["O(1)"] 而不是 A[O(1)]。
6. **完整性约束**：请务必完整输出，不要遗漏关键知识点。如果内容较长，请合理分配篇幅，确保文章结构完整，包含结尾总结。
%s
源内容：
%s

本章节大纲：
- 标题: %s
- 摘要: %s
- 排序: %d
`, extraRequirements, chapterSourceContent, chapter.Title, chapter.Summary, chapter.Sort)

			messages := []llm.Message{
				{Role: "system", Content: "你是一个高级技术博客作者。"},
				{Role: "user", Content: prompt},
			}

			llmModel := "deepseek-v4-flash"
			if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
				llmModel = envModel
			}

			var streamErr error
			maxRetries := 3
			var content string

			for attempt := 1; attempt <= maxRetries; attempt++ {
					select {
					case <-ctx.Done():
						errChan <- ctx.Err()
						return
					default:
					}

					if attempt > 1 {
						retryMsg := map[string]interface{}{
							"status":       "retrying",
							"chapter_sort": chapter.Sort,
							"attempt":      attempt,
							"parent_id":    parentID.String(),
						}
						retryBytes, _ := json.Marshal(retryMsg)
						progressChan <- string(retryBytes)
					} else {
						startMsg := map[string]interface{}{
							"status":       "generating",
							"chapter_sort": chapter.Sort,
							"title":        chapter.Title,
							"parent_id":    parentID.String(),
						}
						startBytes, _ := json.Marshal(startMsg)
						progressChan <- string(startBytes)
					}

					streamCtx, streamCancel := context.WithCancel(ctx)
					chapterChunkChan := make(chan string, 100)
					chapterErrChan := make(chan error, 1)

					var fullContentBuilder strings.Builder

					// Generator loop (handles auto-continuation)
					go func() {
						defer close(chapterChunkChan)
						defer close(chapterErrChan)

						currentMessages := make([]llm.Message, len(messages))
						copy(currentMessages, messages)

						for {
							tempChunkChan := make(chan string)
							var assistantContent string
							var wg sync.WaitGroup
							wg.Add(1)

							go func() {
								defer wg.Done()
								for chunk := range tempChunkChan {
									assistantContent += chunk
									chapterChunkChan <- chunk
								}
							}()

							finishReason, err := s.llmClient.GenerateStream(streamCtx, llmModel, currentMessages, tempChunkChan)
							wg.Wait()

							if err != nil {
								chapterErrChan <- err
								return
							}

							currentMessages = append(currentMessages, llm.Message{
								Role:    "assistant",
								Content: assistantContent,
							})

							if finishReason != "length" {
								return
							}

							// Auto-continue
							continueMsg := llm.Message{
								Role:    "user",
								Content: "刚才你的回答被截断了，请严格从上文最后一个字符开始无缝续写。绝对不要输出“好的，我们继续”等任何过渡性废话，直接输出后续的Markdown或代码内容。",
							}
							currentMessages = append(currentMessages, continueMsg)
						}
					}()

					idleTimeout := 60 * time.Second
					timer := time.NewTimer(idleTimeout)

					streamErr = nil
					done := false

					for !done {
						select {
						case <-ctx.Done():
							streamCancel()
							timer.Stop()
							errChan <- ctx.Err()
							return
						case <-timer.C:
							streamCancel()
							streamErr = fmt.Errorf("AI generation idle timeout (no data for %v)", idleTimeout)
							done = true
						case err, ok := <-chapterErrChan:
							if ok && err != nil {
								streamErr = err
								done = true
							} else if !ok {
								chapterErrChan = nil
							}
						case chunk, ok := <-chapterChunkChan:
							if !ok {
								done = true
							} else {
								if !timer.Stop() {
									select {
									case <-timer.C:
									default:
									}
								}
								timer.Reset(idleTimeout)

								fullContentBuilder.WriteString(chunk)
								streamMsg := map[string]interface{}{
									"status":       "streaming",
									"chapter_sort": chapter.Sort,
									"content":      chunk,
								}
								streamBytes, _ := json.Marshal(streamMsg)
								progressChan <- string(streamBytes)
							}
						}
					}

					timer.Stop()
					streamCancel()

					if streamErr == nil {
						content = fullContentBuilder.String()
						break
					}

					time.Sleep(exponentialBackoff(attempt))
				}

				if streamErr != nil {
					errMsg := map[string]interface{}{
						"status":       "error",
						"chapter_sort": chapter.Sort,
						"message":      fmt.Sprintf("chapter %d generation failed after %d attempts: %v", chapter.Sort, maxRetries, streamErr),
					}
					errBytes, _ := json.Marshal(errMsg)
					progressChan <- string(errBytes)
					return
				}

			// Calculate word count
			wordCount := len([]rune(content))

			// Extract Tech Stacks using LLM
			var techStacks datatypes.JSON
			extractPrompt := "请从以下文章内容中提取出涉及的核心技术栈名称（如 React, Go, Docker 等），以 JSON 数组格式返回，不要有任何其他多余字符：\n\n" + content
			extractMessages := []llm.Message{
				{Role: "user", Content: extractPrompt},
			}

			extractedJSON, err := s.llmClient.Generate(ctx, llmModel, extractMessages)
			if err == nil && len(extractedJSON) > 0 {
				// basic validation that it is a json array
				var parsed []string
				if json.Unmarshal([]byte(extractedJSON), &parsed) == nil {
					techStacks = datatypes.JSON(extractedJSON)
				}
			}

			// Save to database
			blog := &model.Blog{
				UserID:      userID,
				ParentID:    &parentID,
				ChapterSort: chapter.Sort,
				Title:       chapter.Title,
				Content:     content,
				SourceType:  sourceType,
				Status:      1, // 1 for completed
				WordCount:   wordCount,
				TechStacks:  techStacks,
			}

			if err := db.DB.WithContext(ctx).Create(blog).Error; err != nil {
				errMsg := map[string]interface{}{
					"status":       "error",
					"chapter_sort": chapter.Sort,
					"message":      fmt.Sprintf("failed to save chapter %d to db: %v", chapter.Sort, err),
				}
				errBytes, _ := json.Marshal(errMsg)
				progressChan <- string(errBytes)
				return
			} else {
				// Update user tokens used
				estimatedTokens := len([]rune(content)) * 2
				db.DB.Model(&model.User{}).Where("id = ?", userID).UpdateColumn("tokens_used", gorm.Expr("tokens_used + ?", estimatedTokens))
			}

			// Send progress: completed
			endMsg := map[string]interface{}{
				"status":       "completed",
				"chapter_sort": chapter.Sort,
				"title":        chapter.Title,
			}
			endBytes, _ := json.Marshal(endMsg)
			progressChan <- string(endBytes)
		}(i, chapter)
	}

	wg.Wait()
}

// GenerateOutline evaluates project text and generates a JSON outline
func (s *DecompositionService) GenerateOutline(ctx context.Context, sourceContent string) (*OutlineResult, error) {
	// DeepSeek V4 max context is 1M tokens.
	// Limit source content to ~2,000,000 characters to utilize the long context while keeping a safe margin.
	// 2,000,000 characters is roughly 500k - 700k tokens, leaving plenty of room for system prompts and the completion.
	runes := []rune(sourceContent)
	if len(runes) > 2000000 {
		sourceContent = string(runes[:2000000]) + "\n\n... [Content Truncated due to length limits] ..."
	}

	instruction := `你是一个高级架构师。请评估前面提供的项目文本，并生成一个系列博客的大纲。
对于大型项目、源码仓库或复杂内容，**强制拆分为细粒度系列博客**。
要求一个技术点分为一个博客，博客篇数上不封顶，只要有需要，技术点可以拆的更加详细。
输出必须是纯JSON格式，包含 series_title 和 chapters 两个字段，不包含任何Markdown标记或其他文字。
JSON 格式如下：
{
  "series_title": "系列博客的标题",
  "chapters": [
    {
      "title": "章节标题",
      "summary": "该章节的详细摘要或内容要点（指导后续生成的具体方向）",
      "sort": 1,
      "files": ["强相关的具体文件路径或目录（必须是相对路径）"]
    }
  ]
}`

	messages := []llm.Message{
		{Role: "system", Content: "项目文本内容如下：\n" + sourceContent},
		{Role: "user", Content: instruction},
	}

	model := "deepseek-v4-flash"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		model = envModel
	}

	content, err := s.llmClient.Generate(ctx, model, messages)
	if err != nil {
		return nil, fmt.Errorf("llm generation failed: %w", err)
	}

	// Clean up content just in case the LLM returned markdown blocks
	content = strings.TrimPrefix(strings.TrimSpace(content), "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var outline OutlineResult
	if err := json.Unmarshal([]byte(content), &outline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal llm output: %w, output: %s", err, content)
	}

	return &outline, nil
}

// ContinueGeneration handles the SSE stream to continue generating content for an existing blog
func (s *DecompositionService) ContinueGeneration(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error) {
	defer close(chunkChan)
	defer close(errChan)

	// Fetch existing blog
	var blog model.Blog
	if err := db.DB.WithContext(ctx).First(&blog, "id = ? AND user_id = ?", blogID, userID).Error; err != nil {
		errChan <- fmt.Errorf("blog not found: %w", err)
		return
	}

	prompt := "刚才你的回答被截断了，请严格从上文最后一个字符开始无缝续写。绝对不要输出“好的，我们继续”等任何过渡性废话，直接输出后续的Markdown或代码内容。"
	messages := []llm.Message{
		{Role: "system", Content: "你是一个高级技术博客作者。"},
		{Role: "assistant", Content: blog.Content},
		{Role: "user", Content: prompt},
	}

	llmModel := "deepseek-v4-flash"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		llmModel = envModel
	}

	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	internalChunkChan := make(chan string, 100)
	internalErrChan := make(chan error, 1)

	go func() {
		defer close(internalChunkChan)
		defer close(internalErrChan)

		currentMessages := make([]llm.Message, len(messages))
		copy(currentMessages, messages)

		for {
			tempChunkChan := make(chan string)
			var assistantContent string
			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				for chunk := range tempChunkChan {
					assistantContent += chunk
					internalChunkChan <- chunk
				}
			}()

			finishReason, err := s.llmClient.GenerateStream(streamCtx, llmModel, currentMessages, tempChunkChan)
			wg.Wait()

			if err != nil {
				internalErrChan <- err
				return
			}

			currentMessages = append(currentMessages, llm.Message{
				Role:    "assistant",
				Content: assistantContent,
			})

			if finishReason != "length" {
				return
			}

			// Auto-continue
			continueMsg := llm.Message{
				Role:    "user",
				Content: "刚才你的回答被截断了，请严格从上文最后一个字符开始无缝续写。绝对不要输出“好的，我们继续”等任何过渡性废话，直接输出后续的Markdown或代码内容。",
			}
			currentMessages = append(currentMessages, continueMsg)
		}
	}()

	var newContentBuilder strings.Builder
	idleTimeout := 60 * time.Second
	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		case <-timer.C:
			streamCancel()
			errChan <- fmt.Errorf("AI generation idle timeout (no data for %v)", idleTimeout)
			return
		case err, ok := <-internalErrChan:
			if ok && err != nil {
				errChan <- err
				return
			}
			if !ok {
				internalErrChan = nil
			}
		case chunk, ok := <-internalChunkChan:
			if !ok {
				// Stream finished
				finalNewContent := newContentBuilder.String()
				if finalNewContent != "" {
					updatedContent := blog.Content + finalNewContent
					if err := db.DB.WithContext(ctx).Model(&blog).Update("content", updatedContent).Error; err != nil {
						fmt.Printf("Failed to update blog content: %v\n", err)
					}
				}
				return
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(idleTimeout)

			newContentBuilder.WriteString(chunk)
			chunkChan <- chunk
		}
	}
}

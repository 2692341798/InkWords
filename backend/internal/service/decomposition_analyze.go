package service

import (
	"context"
	"encoding/json"
	"fmt"
	"inkwords-backend/internal/db"
	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/model"
	"inkwords-backend/internal/parser"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"
)

func getStatusFromStep(step int) string {
	switch step {
	case 0:
		return "cloning"
	case 1:
		return "scanning"
	case 2:
		return "analyzing"
	case 3:
		return "outline"
	case 4:
		return "complete"
	default:
		return "unknown"
	}
}

// AnalyzeStream handles the full analysis pipeline with streaming progress
func (s *DecompositionService) AnalyzeStream(ctx context.Context, userID uuid.UUID, gitURL string, selectedModules []string, progressChan chan<- string, errChan chan<- error) {
	defer close(progressChan)
	defer close(errChan)

	sendProgress := func(step int, message string, data interface{}) {
		msg := map[string]interface{}{
			"step":    step,
			"status":  getStatusFromStep(step),
			"message": message,
		}
		if data != nil {
			msg["data"] = data
			if step == 4 {
				msg["content"] = data // For frontend compatibility expecting 'content' in step 4
			}
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

	sendProgress(0, "正在获取仓库目录与源码内容...", nil)

	var treeContent string
	var chunks []parser.FileChunk
	var err error

	if len(selectedModules) > 0 {
		var allChunks []parser.FileChunk
		var treeBuilder strings.Builder
		for _, mod := range selectedModules {
			tree, modChunks, fetchErr := s.gitFetcher.FetchWithSubDir(gitURL, mod, func(msg string) {
				sendProgress(0, fmt.Sprintf("[%s] %s", mod, msg), nil)
			})
			if fetchErr != nil {
				errChan <- fmt.Errorf("拉取仓库模块 %s 失败: %w", mod, fetchErr)
				return
			}
			treeBuilder.WriteString(fmt.Sprintf("--- 目录结构 (%s) ---\n%s\n", mod, tree))
			allChunks = append(allChunks, modChunks...)
		}
		treeContent = treeBuilder.String()
		chunks = allChunks
	} else {
		treeContent, chunks, err = s.gitFetcher.FetchWithSubDir(gitURL, "", func(msg string) {
			sendProgress(0, msg, nil)
		})
		if err != nil {
			errChan <- fmt.Errorf("拉取仓库失败: %w", err)
			return
		}
	}

	sendProgress(1, "分析仓库源码与结构完成", nil)

	// Map-Reduce Phase
	var finalContent strings.Builder
	fullContent := treeContent + "\n=== Repository Content ===\n"

	if len(chunks) == 1 && len([]rune(chunks[0].Content)) < 1500000 {
		sendProgress(2, "项目较小，跳过 Map 阶段，直接生成大纲...", nil)
		finalContent.WriteString(fullContent)
		finalContent.WriteString(chunks[0].Content)
	} else {
		sendProgress(2, fmt.Sprintf("开启 Map-Reduce 分析，共 %d 个分块", len(chunks)), nil)
		summaries := s.mapReduceAnalyze(ctx, chunks, sendProgress)

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

	var existingParent *model.Blog
	var existingChildren []model.Blog
	if gitURL != "" {
		var p model.Blog
		if err := db.DB.WithContext(ctx).Where("user_id = ? AND source_type = 'git' AND source_url = ? AND parent_id IS NULL", userID, gitURL).First(&p).Error; err == nil {
			existingParent = &p
			db.DB.WithContext(ctx).Where("parent_id = ?", p.ID).Order("chapter_sort asc").Find(&existingChildren)
			sendProgress(3, "检测到已有该项目的博客系列，正在生成增量更新大纲...", nil)
		}
	}

	outlineResult, err := s.GenerateOutline(ctx, finalContent.String(), existingParent, existingChildren)
	if err != nil {
		errChan <- fmt.Errorf("生成大纲失败: %w", err)
		return
	}

	if existingParent != nil {
		outlineResult.ParentID = existingParent.ID.String()
	}

	sendProgress(4, "正在完成最后处理...", map[string]interface{}{
		"series_title":   outlineResult.SeriesTitle,
		"outline":        outlineResult.Chapters,
		"source_content": finalContent.String(),
		"parent_id":      outlineResult.ParentID,
	})
}

// AnalyzeFileStream handles the analysis pipeline for a single file's content
func (s *DecompositionService) AnalyzeFileStream(ctx context.Context, userID uuid.UUID, sourceContent string, progressChan chan<- string, errChan chan<- error) {
	defer close(progressChan)
	defer close(errChan)

	sendProgress := func(step int, message string, data interface{}) {
		msg := map[string]interface{}{
			"step":    step,
			"status":  getStatusFromStep(step),
			"message": message,
		}
		if data != nil {
			msg["data"] = data
			if step == 4 {
				msg["content"] = data
			}
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

	if len(runes) > 1000000 {
		chunks := chunkFileContent(sourceContent, 1000000)
		sendProgress(2, fmt.Sprintf("文件较大，开启 Map-Reduce 分析，共 %d 个分块", len(chunks)), nil)
		summaries := s.mapReduceAnalyzeFile(ctx, chunks, sendProgress)

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

	outlineResult, err := s.GenerateOutline(ctx, finalContent.String(), nil, nil)
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

		splitIdx := end
		for i := end; i > start && i > end-2000; i-- {
			if runes[i] == '\n' {
				splitIdx = i + 1
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

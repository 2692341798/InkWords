package service

import (
	"context"
	"fmt"
	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/parser"
	"os"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

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
你的输出应该是一份具有独立价值的局部摘要（不需要过多的寒暄，直接列出关键信息）。
特别注意：**请详细记录下核心的类名、函数名以及重要架构设计，尽量不要丢弃有价值的技术细节**，以便后续能为它们单独生成博客解析。
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

package service

import (
	"context"
	"encoding/json"
	"fmt"
	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"inkwords-backend/internal/infra/llm"
)

func sendSeriesProgressPayload(progressChan chan<- string, payload map[string]interface{}) {
	bytes, _ := json.Marshal(payload)
	progressChan <- string(bytes)
}

func sendSeriesSystemProgress(progressChan chan<- string, message string) {
	sendSeriesProgressPayload(progressChan, map[string]interface{}{
		"status":  "progress",
		"message": message,
	})
}

// finalizeSeriesChapterDraft 只在终稿阶段向前端透出增量内容，避免草稿和审稿中间态污染用户正在阅读的正文。
func (s *DecompositionService) finalizeSeriesChapterDraft(
	ctx context.Context,
	llmModel string,
	input seriesQualityPipelineInput,
	seriesPrefix string,
	understanding SeriesChapterUnderstanding,
	draft SeriesChapterDraft,
	review SeriesChapterReview,
) (SeriesChapterFinal, error) {
	chunkChan := make(chan string, 100)
	errChan := make(chan error, 1)
	usageChan := make(chan llm.CompletionUsage, 1)
	var finalBuilder strings.Builder

	go func() {
		options := llm.DefaultChatOptions()
		options.UserID = input.UserID
		_, usage, err := s.llmClient.GenerateStreamWithOptions(ctx, llmModel, []llm.Message{
			{Role: "system", Content: seriesPrefix + "\n当前阶段：定向补强与轻统稿"},
			{Role: "user", Content: buildSeriesFinalizePrompt(input, understanding, draft, review)},
		}, chunkChan, options)
		usageChan <- usage
		errChan <- err
	}()

	for chunk := range chunkChan {
		finalBuilder.WriteString(chunk)
		sendSeriesProgressPayload(input.ProgressChan, map[string]interface{}{
			"status":       "streaming",
			"chapter_sort": input.Chapter.Sort,
			"title":        input.Chapter.Title,
			"content":      chunk,
		})
	}

	if err := <-errChan; err != nil {
		return SeriesChapterFinal{}, err
	}
	usage := <-usageChan
	sendSeriesProgressPayload(input.ProgressChan, map[string]interface{}{
		"status":                   "usage",
		"chapter_sort":             input.Chapter.Sort,
		"title":                    input.Chapter.Title,
		"prompt_tokens":            usage.PromptTokens,
		"completion_tokens":        usage.CompletionTokens,
		"prompt_cache_hit_tokens":  usage.PromptCacheHitTokens,
		"prompt_cache_miss_tokens": usage.PromptCacheMissTokens,
	})

	return SeriesChapterFinal{
		FinalMarkdown:    finalBuilder.String(),
		ResolvedIssues:   append([]string(nil), review.RevisionActions...),
		ResidualRisks:    nil,
		Usage:            usageFromCompletionUsage(usage),
		QualityScorecard: review.Scorecard,
		RevisionActions:  append([]string(nil), review.RevisionActions...),
	}, nil
}

func (s *DecompositionService) streamSeriesChapterContent(
	ctx context.Context,
	parentID uuid.UUID,
	chapter blogcontracts.Chapter,
	messages []llm.Message,
	progressChan chan<- string,
) (string, error) {
	var streamErr error
	var content string
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		if attempt > 1 {
			sendSeriesProgressPayload(progressChan, map[string]interface{}{
				"status":       "retrying",
				"chapter_sort": chapter.Sort,
				"attempt":      attempt,
				"parent_id":    parentID.String(),
			})
		} else {
			sendSeriesProgressPayload(progressChan, map[string]interface{}{
				"status":       "generating",
				"chapter_sort": chapter.Sort,
				"title":        chapter.Title,
				"parent_id":    parentID.String(),
			})
		}

		streamCtx, streamCancel := context.WithCancel(ctx)
		chapterChunkChan := make(chan string, 100)
		chapterErrChan := make(chan error, 1)
		var fullContentBuilder strings.Builder

		llmModel := "deepseek-v4-flash"
		if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
			llmModel = envModel
		}

		go func() {
			defer close(chapterChunkChan)
			defer close(chapterErrChan)

			currentMessages := make([]llm.Message, len(messages))
			copy(currentMessages, messages)

			for {
				tempChunkChan := make(chan string)
				var assistantContent string
				var waitGroup sync.WaitGroup
				waitGroup.Add(1)

				go func() {
					defer waitGroup.Done()
					for chunk := range tempChunkChan {
						assistantContent += chunk
						chapterChunkChan <- chunk
					}
				}()

				finishReason, _, err := s.llmClient.GenerateStreamWithUsage(streamCtx, llmModel, currentMessages, tempChunkChan)
				waitGroup.Wait()
				if err != nil {
					chapterErrChan <- err
					return
				}

				currentMessages = append(currentMessages, llm.Message{Role: "assistant", Content: assistantContent})
				if finishReason != "length" {
					return
				}

				currentMessages = append(currentMessages, llm.Message{
					Role:    "user",
					Content: "刚才你的回答被截断了，请严格从上文最后一个字符开始无缝续写。绝对不要输出“好的，我们继续”等任何过渡性废话，直接输出后续的Markdown或代码内容。",
				})
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
				return "", ctx.Err()
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
					continue
				}

				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(idleTimeout)

				fullContentBuilder.WriteString(chunk)
				sendSeriesProgressPayload(progressChan, map[string]interface{}{
					"status":       "streaming",
					"chapter_sort": chapter.Sort,
					"content":      chunk,
				})
			}
		}

		timer.Stop()
		streamCancel()
		if streamErr == nil {
			content = fullContentBuilder.String()
			break
		}

		if !llm.IsRetryableError(streamErr) {
			break
		}
		time.Sleep(exponentialBackoff(attempt))
	}

	if streamErr != nil {
		return "", streamErr
	}

	return content, nil
}

func (s *DecompositionService) extractSeriesChapterTechStacks(ctx context.Context, llmModel, content string) datatypes.JSON {
	var techStacks datatypes.JSON
	extractPrompt := "请从以下文章内容中提取出涉及的核心技术栈名称（如 React, Go, Docker 等），以 JSON 数组格式返回，不要有任何其他多余字符。\n\n例如：[\"React\", \"Go\"]\n\n文章内容：\n\n" + content
	extractMessages := []llm.Message{{Role: "user", Content: extractPrompt}}
	extractedJSON, _, err := s.llmClient.GenerateJSONWithOptions(ctx, llmModel, extractMessages, llm.LightweightChatOptions("", 512))
	if err != nil || len(extractedJSON) == 0 {
		return techStacks
	}

	var parsed []string
	if json.Unmarshal([]byte(extractedJSON), &parsed) != nil {
		return techStacks
	}

	return datatypes.JSON(extractedJSON)
}

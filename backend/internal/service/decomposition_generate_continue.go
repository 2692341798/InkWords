package service

import (
	"context"
	"fmt"
	llm "inkwords-backend/shared/platform/llm"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ContinueTaskResultSnapshot captures the business facts needed to build the
// continue task_only result without directly touching core-api owned writes.
type ContinueTaskResultSnapshot struct {
	BlogID          string
	AppendedContent string
	FinalContent    string
	EstimatedTokens int
	Usage           SeriesChapterUsage
}

func (s *DecompositionService) ContinueGeneration(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error) {
	defer close(chunkChan)
	defer close(errChan)

	blog, err := s.continuePersistence.LoadContinueBlog(ctx, userID, blogID)
	if err != nil {
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
	internalUsageChan := make(chan llm.CompletionUsage, 1)

	go func() {
		defer close(internalChunkChan)
		defer close(internalErrChan)
		var totalUsage llm.CompletionUsage
		defer func() {
			internalUsageChan <- totalUsage
			close(internalUsageChan)
		}()

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

			options := llm.DefaultChatOptions()
			options.UserID = fmt.Sprintf("continue-%s", blogID.String())
			finishReason, usage, err := s.llmClient.GenerateStreamWithOptions(streamCtx, llmModel, currentMessages, tempChunkChan, options)
			wg.Wait()

			if err != nil {
				internalErrChan <- err
				return
			}
			totalUsage = addCompletionUsage(totalUsage, usage)

			currentMessages = append(currentMessages, llm.Message{
				Role:    "assistant",
				Content: assistantContent,
			})

			if finishReason != "length" {
				return
			}

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
				finalNewContent := newContentBuilder.String()
				if finalNewContent != "" && !taskOnlyPersistenceMode() {
					updatedContent := blog.Content + finalNewContent
					if err := s.continuePersistence.SaveContinuedBlog(ctx, blog, updatedContent); err != nil {
						fmt.Printf("Failed to update blog content: %v\n", err)
					}
				} else if finalNewContent != "" {
					if usage, ok := <-internalUsageChan; ok {
						s.storeContinueUsage(blogID, finalNewContent, usage)
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

// BuildContinueTaskResult builds the task_only result for continue without
// directly mutating blogs, so core-api can own the final business persistence.
func (s *DecompositionService) BuildContinueTaskResult(
	ctx context.Context,
	userID uuid.UUID,
	blogID uuid.UUID,
	appendedContent string,
) (ContinueTaskResultSnapshot, error) {
	blog, err := s.continuePersistence.LoadContinueBlog(ctx, userID, blogID)
	if err != nil {
		return ContinueTaskResultSnapshot{}, fmt.Errorf("load continue blog for task result: %w", err)
	}

	finalContent := blog.Content + appendedContent
	return ContinueTaskResultSnapshot{
		BlogID:          blog.ID.String(),
		AppendedContent: appendedContent,
		FinalContent:    finalContent,
		EstimatedTokens: len([]rune(appendedContent)) * 2,
		Usage:           usageFromCompletionUsage(s.takeContinueUsage(blogID, appendedContent)),
	}, nil
}

func continueUsageKey(blogID uuid.UUID, appendedContent string) string {
	return generatedUsageKey("continue-"+blogID.String(), appendedContent)
}

func (s *DecompositionService) storeContinueUsage(blogID uuid.UUID, appendedContent string, usage llm.CompletionUsage) {
	if s == nil {
		return
	}
	s.continueUsageMu.Lock()
	defer s.continueUsageMu.Unlock()
	if s.continueUsage == nil {
		s.continueUsage = make(map[string]llm.CompletionUsage)
	}
	key := continueUsageKey(blogID, appendedContent)
	s.continueUsage[key] = addCompletionUsage(s.continueUsage[key], usage)
}

func (s *DecompositionService) takeContinueUsage(blogID uuid.UUID, appendedContent string) llm.CompletionUsage {
	if s == nil {
		return llm.CompletionUsage{}
	}
	s.continueUsageMu.Lock()
	defer s.continueUsageMu.Unlock()
	if s.continueUsage == nil {
		return llm.CompletionUsage{}
	}
	key := continueUsageKey(blogID, appendedContent)
	usage := s.continueUsage[key]
	delete(s.continueUsage, key)
	return usage
}

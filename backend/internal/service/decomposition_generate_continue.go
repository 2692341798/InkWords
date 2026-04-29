package service

import (
	"context"
	"fmt"
	"inkwords-backend/internal/db"
	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/model"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

func (s *DecompositionService) ContinueGeneration(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error) {
	defer close(chunkChan)
	defer close(errChan)

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

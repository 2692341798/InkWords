package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"inkwords-backend/internal/db"
	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/model"
)

// GeneratorService handles the blog generation process
type GeneratorService struct {
	llmClient *llm.DeepSeekClient
}

// NewGeneratorService creates a new generator service
func NewGeneratorService() *GeneratorService {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	return &GeneratorService{
		llmClient: llm.NewDeepSeekClient(apiKey),
	}
}

// GenerateBlogStream assembles the prompt, calls the LLM, and pushes chunks to the channel
func (s *GeneratorService) GenerateBlogStream(ctx context.Context, userID uuid.UUID, sourceContent string, sourceType string, chunkChan chan<- string, errChan chan<- error) {
	prompt := fmt.Sprintf(`你是一个高级全栈架构师和技术博主。请根据以下提供的源内容，将其转化为一篇“小白友好、图文并茂、可独立复现”的高质量技术博客。
要求：
1. **字数充足，内容详实**：不要只写干瘪的总结。必须深入分析实现原理。
2. **代码级剖析**：对于每个技术点都添加更多的代码样例和图片来解释的更加详细。如果源内容包含代码，请引用核心代码并逐行解释其作用。
3. **可复现的步骤**：如果是实战或教程相关，请给出明确的执行步骤。
4. **小白友好**：在解释抽象的理论概念时，必须提供对应的代码示例或生活化比喻。
5. 所有生成的 Mermaid 图表代码块绝对禁止包含自定义样式关键字（如 style, classDef, linkStyle 等），必须使用基础语法。

源内容：
%s`, sourceContent)

	messages := []llm.Message{
		{Role: "system", Content: "你是一个高级技术博客作者。"},
		{Role: "user", Content: prompt},
	}

	modelType := "deepseek-chat" // or deepseek-reasoner depending on env/config
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		modelType = envModel
	}

	// Create an intermediate channel to intercept chunks for saving
	internalChunkChan := make(chan string)
	internalErrChan := make(chan error)

	streamCtx, streamCancel := context.WithCancel(ctx)

	// Receiver goroutine
	go func() {
		defer streamCancel()
		defer close(chunkChan)
		defer close(errChan)

		var fullContent string
		idleTimeout := 60 * time.Second // Increased to 60s
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
					// Stream finished successfully, save to DB
					s.saveToDB(ctx, userID, sourceType, fullContent)
					return
				}
				if !timer.Stop() {
					select { case <-timer.C: default: }
				}
				timer.Reset(idleTimeout)

				fullContent += chunk
				chunkChan <- chunk
			}
		}
	}()

	// Generator loop (handles auto-continuation)
	go func() {
		defer close(internalChunkChan)
		defer close(internalErrChan)
		
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

			finishReason, err := s.llmClient.GenerateStream(streamCtx, modelType, messages, tempChunkChan)
			wg.Wait() // Ensure all chunks are collected
			
			if err != nil {
				internalErrChan <- err
				return
			}
			
			// Append what the assistant just generated
			messages = append(messages, llm.Message{
				Role:    "assistant",
				Content: assistantContent,
			})

			if finishReason != "length" {
				return
			}
			
			// Auto-continue if it stopped due to length limit
			// We append the prompt to strictly continue without conversational filler
			continueMsg := llm.Message{
				Role:    "user",
				Content: "刚才你的回答被截断了，请严格从上文最后一个字符开始无缝续写。绝对不要输出“好的，我们继续”等任何过渡性废话，直接输出后续的Markdown或代码内容。",
			}
			messages = append(messages, continueMsg)
		}
	}()
}

// saveToDB persists the generated markdown to PostgreSQL
func (s *GeneratorService) saveToDB(ctx context.Context, userID uuid.UUID, sourceType string, content string) {
	// Extract a simple title from the first line or default
	title := "文件解析生成的博客"
	if len(content) > 0 {
		// Attempt to grab the first H1 or just first line
		// (A simplistic approach: we can just leave it as default or let user edit it)
	}

	blog := &model.Blog{
		UserID:      userID,
		Title:       title,
		Content:     content,
		SourceType:  sourceType,
		Status:      1, // completed
		ChapterSort: 1,
	}

	if err := db.DB.WithContext(ctx).Create(blog).Error; err != nil {
		fmt.Printf("Failed to save generated blog to DB: %v\n", err)
	} else {
		fmt.Printf("Saved generated blog to DB (ID: %s, Length: %d)\n", blog.ID, len(content))
	}
}

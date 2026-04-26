package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"inkwords-backend/internal/db"
	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/model"
)

// GeneratorService handles the blog generation process
type GeneratorService struct {
	llmClient     *llm.DeepSeekClient
}

// NewGeneratorService creates a new generator service
func NewGeneratorService() *GeneratorService {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	return &GeneratorService{
		llmClient:     llm.NewDeepSeekClient(apiKey),
	}
}

// GenerateBlogStream assembles the prompt, calls the LLM, and pushes chunks to the channel
func (s *GeneratorService) GenerateBlogStream(ctx context.Context, userID uuid.UUID, sourceContent string, sourceType string, chunkChan chan<- string, errChan chan<- error) {
	instruction := `你是一个高级全栈架构师和技术博主。请根据前面提供的源内容，将其转化为一篇“小白友好、图文并茂、可独立复现”的高质量技术博客。
要求：
1. **单点聚焦与深度剖析**：严格保证本篇文章只介绍**一个核心技术点**。请利用充足的上下文，深入分析其底层原理、设计思想和演进逻辑，不要只写干瘪的总结。字数篇幅不设上限，请尽可能详尽。
2. **丰富的代码示例**：在解释原理和应用时，尽可能多地提供代码示例（不仅仅是源码，还可以是辅助理解的伪代码或最佳实践用例）。如果源内容包含代码，请引用核心代码并逐行解释其作用。
3. **可复现的步骤**：如果是实战或教程相关，请给出明确的执行步骤。
4. **小白友好**：在解释抽象的理论概念时，必须提供对应的代码示例或生活化比喻。
5. 所有生成的 Mermaid 图表代码块绝对禁止包含自定义样式关键字（如 style, classDef, linkStyle 等），必须使用基础语法。在 Mermaid 图表中，如果节点文本包含特殊字符（如括号、幂符号等，例如 O(1), O(n^2)），必须使用双引号将节点文本包裹起来，例如 A["O(1)"] 而不是 A[O(1)]。`

	messages := []llm.Message{
		{Role: "system", Content: "项目源内容如下：\n" + sourceContent},
		{Role: "user", Content: instruction},
	}

	modelType := "deepseek-v4-flash" // or deepseek-v4-pro depending on env/config
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		modelType = envModel
	}

	// Create an intermediate channel to intercept chunks for saving
	internalChunkChan := make(chan string, 100)
	internalErrChan := make(chan error, 1)

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
					select {
					case <-timer.C:
					default:
					}
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
	title := "文件解析生成的博客"

	// Calculate word count
	wordCount := len([]rune(content))

	// Extract Tech Stacks using LLM
	var techStacks datatypes.JSON
	extractPrompt := "请从以下文章内容中提取出涉及的核心技术栈名称（如 React, Go, Docker 等），以 JSON 数组格式返回，不要有任何其他多余字符。\n\n例如：[\"React\", \"Go\"]\n\n文章内容：\n\n" + content
	messages := []llm.Message{
		{Role: "user", Content: extractPrompt},
	}
	modelType := "deepseek-v4-flash"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		modelType = envModel
	}

	extractedJSON, err := s.llmClient.GenerateJSON(ctx, modelType, messages)
	if err == nil && len(extractedJSON) > 0 {
		// basic validation that it is a json array
		var parsed []string
		if json.Unmarshal([]byte(extractedJSON), &parsed) == nil {
			techStacks = datatypes.JSON(extractedJSON)
		}
	}

	blog := &model.Blog{
		UserID:      userID,
		Title:       title,
		Content:     content,
		SourceType:  sourceType,
		Status:      1, // completed
		ChapterSort: 1,
		WordCount:   wordCount,
		TechStacks:  techStacks,
	}

	if err := db.DB.WithContext(ctx).Create(blog).Error; err != nil {
		fmt.Printf("Failed to save generated blog to DB: %v\n", err)
	} else {
		fmt.Printf("Saved generated blog to DB (ID: %s, Length: %d, TechStacks: %s)\n", blog.ID, len(content), string(techStacks))

		// Update user tokens used (rough estimation: 1 token ≈ 1.5 chars, let's just use rune count for simplicity)
		estimatedTokens := len([]rune(content)) * 2
		db.DB.Model(&model.User{}).Where("id = ?", userID).UpdateColumn("tokens_used", gorm.Expr("tokens_used + ?", estimatedTokens))
	}
}

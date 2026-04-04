package service

import (
	"context"
	"fmt"
	"os"

	"inkwords-backend/internal/llm"
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
func (s *GeneratorService) GenerateBlogStream(ctx context.Context, sourceContent string, chunkChan chan<- string, errChan chan<- error) {
	prompt := fmt.Sprintf(`你是一个高级全栈架构师和技术博主。请根据以下提供的源内容，将其转化为一篇“小白友好、图文并茂、可独立复现”的高质量技术博客。
如果是大型开源项目或长篇教程，请考虑将其拆分为系列博客。
在解释抽象的理论概念时，必须提供对应的代码示例。
所有生成的 Mermaid 图表代码块绝对禁止包含自定义样式关键字（如 style, classDef, linkStyle 等），必须使用基础语法。

源内容：
%s`, sourceContent)

	messages := []llm.Message{
		{Role: "system", Content: "你是一个高级技术博客作者。"},
		{Role: "user", Content: prompt},
	}

	model := "deepseek-chat" // or deepseek-reasoner depending on env/config
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		model = envModel
	}

	// Create an intermediate channel to intercept chunks for saving
	internalChunkChan := make(chan string)
	internalErrChan := make(chan error)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		var fullContent string
		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
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
					s.saveToDB(fullContent)
					return
				}
				fullContent += chunk
				chunkChan <- chunk
			}
		}
	}()

	s.llmClient.GenerateStream(ctx, model, messages, internalChunkChan, internalErrChan)
}

// saveToDB mocks the persistence of the generated markdown
func (s *GeneratorService) saveToDB(content string) {
	// TODO: Integrate with BlogRepo to save the content to PostgreSQL
	fmt.Printf("Saving generated blog to DB (Length: %d)\n", len(content))
}

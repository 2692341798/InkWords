package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/infra/llm"
	"inkwords-backend/internal/model"
	"inkwords-backend/internal/prompt"
)

// GeneratorService handles the blog generation process
type GeneratorService struct {
	llmClient *llm.DeepSeekClient
	promptReq *PromptRequirementsService
}

// NewGeneratorService creates a new generator service
func NewGeneratorService(promptReq *PromptRequirementsService) *GeneratorService {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	return &GeneratorService{
		llmClient: llm.NewDeepSeekClient(apiKey),
		promptReq: promptReq,
	}
}

// GenerateBlogStream assembles the prompt, calls the LLM, and pushes chunks to the channel.
func (s *GeneratorService) GenerateBlogStream(
	ctx context.Context,
	userID uuid.UUID,
	sourceContent string,
	sourceType string,
	scenarioMode prompt.ScenarioMode,
	style string,
	chunkChan chan<- string,
	errChan chan<- error,
) {
	s.GenerateBlogStreamWithProfile(
		ctx,
		userID,
		sourceContent,
		sourceType,
		scenarioMode,
		style,
		prompt.PromptProfile{},
		chunkChan,
		errChan,
	)
}

func (s *GeneratorService) GenerateBlogStreamWithProfile(
	ctx context.Context,
	userID uuid.UUID,
	sourceContent string,
	sourceType string,
	scenarioMode prompt.ScenarioMode,
	style string,
	profile prompt.PromptProfile,
	chunkChan chan<- string,
	errChan chan<- error,
) {
	if !scenarioMode.IsValid() {
		scenarioMode = prompt.DefaultScenarioModeForSource(sourceType)
	}
	profile = normalizePromptProfile(profile, scenarioMode)

	requirements := strings.TrimSpace(strings.Join([]string{
		prompt.DefaultScenarioRequirements(scenarioMode),
		prompt.DefaultStyleRequirements(scenarioMode, prompt.ArticleStyleGeneral),
	}, "\n\n"))
	requirements = strings.TrimSpace(strings.Join([]string{
		profile.GenerateRequirements,
		requirements,
	}, "\n\n"))
	if s.promptReq != nil {
		if resolved, err := s.promptReq.ResolveWithProfile(ctx, userID, scenarioMode, prompt.ArticleStyle(style), profile); err == nil && resolved != "" {
			requirements = resolved
		}
	}
	messages := buildSingleGenerateMessages(sourceContent, requirements, profile)

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
					// Why: task_only 模式下，最终业务事实应由 core-api 接管，这里不能再直写 blogs/users。
					// 默认仍保留旧行为，避免在切流前破坏现有生成链路。
					if !taskOnlyPersistenceMode() {
						if err := s.saveToDB(ctx, userID, sourceType, fullContent); err != nil {
							errChan <- err
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

func buildSingleGenerateMessages(sourceContent string, requirements string, profile prompt.PromptProfile) []llm.Message {
	instruction := fmt.Sprintf(`请根据前面提供的源内容，生成一篇中文正文。

写作要求：
%s

硬性约束：
1. 禁止输出“好的，收到你的需求”“作为高级架构师”等对话式前言。
2. 所有生成的 Mermaid 图表代码块绝对禁止包含自定义样式关键字（如 style, classDef, linkStyle 等），必须使用基础语法。
3. 在 Mermaid 图表中，如果节点文本包含特殊字符（如括号、幂符号等，例如 O(1), O(n^2)），必须使用双引号将节点文本包裹起来，例如 A["O(1)"] 而不是 A[O(1)]。`, requirements)

	return []llm.Message{
		{Role: "system", Content: profile.SystemRole + "\n\n项目源内容如下：\n" + sourceContent},
		{Role: "user", Content: instruction},
	}
}

// GeneratePolishDraftStream generates a polished blog draft via LLM streaming, without persisting it.
func (s *GeneratorService) GeneratePolishDraftStream(ctx context.Context, title string, content string, chunkChan chan<- string, errChan chan<- error) {
	const maxInputRunes = 15000000
	contentRunes := []rune(content)
	if len(contentRunes) > maxInputRunes {
		content = string(contentRunes[:maxInputRunes])
	}

	titleRunes := []rune(title)
	if len(titleRunes) > 2000 {
		title = string(titleRunes[:2000])
	}

	instruction := `你是一个高级技术博主和全栈架构师。现在请对用户提供的博客正文进行“全文润色”，输出一份可直接发布的高质量技术博客草稿。
要求：
1. 输出必须为中文。
2. 结构更清晰（H1-H4），逻辑更严密，适当补充示例代码、解释与可复现步骤（如适用）。
3. 严格禁止在 Mermaid 图表代码块中出现 style/classDef/linkStyle 等自定义样式关键字；如果节点文本包含括号或幂符号等特殊字符（如 O(1), O(n^2)），必须使用双引号包裹，例如 A["O(1)"]。
4. Markdown 格式必须严格正确：
   - 标题语法必须为 "# " / "## " / "### " / "#### "（# 后必须有空格），禁止出现 "##2." 这类没有空格的标题写法
   - 代码块必须使用三反引号围栏，且围栏必须单独成行：先输出“围栏行（含语言标识，如 c）”，换行后再写代码；结束围栏也必须单独成行
   - 列表、表格、引用块均使用标准 Markdown 语法；不要输出 HTML 标签
5. 输出必须严格按以下顺序组织，便于前端预览与应用：
   - 先输出一个“## 标题建议”小节，列出 3 个备选标题（编号列表）
   - 然后输出一行分隔线：---
   - 最后输出完整 Markdown 正文（可包含代码块、列表等）`

	messages := []llm.Message{
		{
			Role:    "system",
			Content: fmt.Sprintf("当前博客标题：%s\n\n当前博客正文（Markdown）如下：\n%s", title, content),
		},
		{Role: "user", Content: instruction},
	}

	modelType := "deepseek-v4-flash"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		modelType = envModel
	}

	internalChunkChan := make(chan string, 100)
	internalErrChan := make(chan error, 1)

	streamCtx, streamCancel := context.WithCancel(ctx)

	go func() {
		defer streamCancel()
		defer close(chunkChan)
		defer close(errChan)

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
					return
				}
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(idleTimeout)

				chunkChan <- chunk
			}
		}
	}()

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
			wg.Wait()

			if err != nil {
				internalErrChan <- err
				return
			}

			messages = append(messages, llm.Message{
				Role:    "assistant",
				Content: assistantContent,
			})

			if finishReason != "length" {
				return
			}

			messages = append(messages, llm.Message{
				Role:    "user",
				Content: "刚才你的回答被截断了，请严格从上文最后一个字符开始无缝续写。绝对不要输出“好的，我们继续”等任何过渡性废话，直接输出后续的Markdown或代码内容。",
			})
		}
	}()
}

func taskOnlyPersistenceMode() bool {
	return strings.EqualFold(os.Getenv("INKWORDS_TASK_PERSISTENCE_MODE"), "task_only")
}

// saveToDB persists the generated blog content to the database
func (s *GeneratorService) saveToDB(ctx context.Context, userID uuid.UUID, sourceType string, content string) error {
	if db.DB == nil {
		return fmt.Errorf("persist generated blog: database not configured")
	}

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

	if s.llmClient != nil {
		extractedJSON, err := s.llmClient.GenerateJSON(ctx, modelType, messages)
		if err == nil && len(extractedJSON) > 0 {
			// basic validation that it is a json array
			var parsed []string
			if json.Unmarshal([]byte(extractedJSON), &parsed) == nil {
				techStacks = datatypes.JSON(extractedJSON)
			}
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

	// Why: persisting the generated blog and its token accounting must stay consistent.
	// A transaction avoids silently storing a blog row while skipping the quota update.
	estimatedTokens := len([]rune(content)) * 2
	if err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(blog).Error; err != nil {
			return fmt.Errorf("create blog record: %w", err)
		}

		tokenUpdateResult := tx.Model(&model.User{}).
			Where("id = ?", userID).
			UpdateColumn("tokens_used", gorm.Expr("tokens_used + ?", estimatedTokens))
		if tokenUpdateResult.Error != nil {
			return fmt.Errorf("update user tokens: %w", tokenUpdateResult.Error)
		}
		if tokenUpdateResult.RowsAffected == 0 {
			return fmt.Errorf("update user tokens: user not found")
		}

		return nil
	}); err != nil {
		return fmt.Errorf("persist generated blog: %w", err)
	}

	return nil
}

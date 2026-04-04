package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"

	"inkwords-backend/internal/db"
	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/model"
)

// Chapter represents a single chapter in the generated outline
type Chapter struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Sort    int    `json:"sort"`
}

// DecompositionService handles the logic to evaluate project text and generate an outline
type DecompositionService struct {
	llmClient *llm.DeepSeekClient
}

// NewDecompositionService creates a new decomposition service
func NewDecompositionService() *DecompositionService {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	return &DecompositionService{
		llmClient: llm.NewDeepSeekClient(apiKey),
	}
}

// GenerateSeries generates blog chapters concurrently based on the outline
func (s *DecompositionService) GenerateSeries(ctx context.Context, userID uuid.UUID, parentID uuid.UUID, outline []Chapter, sourceContent string, sourceType string, progressChan chan<- string, errChan chan<- error) {
	defer close(progressChan)
	defer close(errChan)

	var wg sync.WaitGroup
	// Limit concurrency to avoid hitting rate limits or using too much memory
	semaphore := make(chan struct{}, 3)
	
	// Create channels for collecting results
	errs := make(chan error, len(outline))
	
	for _, chapter := range outline {
		wg.Add(1)
		go func(c Chapter) {
			defer wg.Done()
			
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			default:
			}

			// Send progress: starting
			progressChan <- fmt.Sprintf(`{"status":"generating","chapter_sort":%d,"title":"%s"}`, c.Sort, c.Title)
			
			prompt := fmt.Sprintf(`你是一个高级全栈架构师和技术博主。请根据以下提供的源内容，以及本章节的大纲，将其转化为一篇“小白友好、图文并茂、可独立复现”的高质量技术博客章节。
在解释抽象的理论概念时，必须提供对应的代码示例。
所有生成的 Mermaid 图表代码块绝对禁止包含自定义样式关键字（如 style, classDef, linkStyle 等），必须使用基础语法。

源内容：
%s

本章节大纲：
- 标题: %s
- 摘要: %s
- 排序: %d
`, sourceContent, c.Title, c.Summary, c.Sort)

			messages := []llm.Message{
				{Role: "system", Content: "你是一个高级技术博客作者。"},
				{Role: "user", Content: prompt},
			}

			llmModel := "deepseek-chat"
			if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
				llmModel = envModel
			}

			content, err := s.llmClient.Generate(ctx, llmModel, messages)
			if err != nil {
				errs <- fmt.Errorf("chapter %d generation failed: %w", c.Sort, err)
				return
			}
			
			// Save to database
			blog := &model.Blog{
				UserID:      userID,
				ParentID:    &parentID,
				ChapterSort: c.Sort,
				Title:       c.Title,
				Content:     content,
				SourceType:  sourceType,
				Status:      1, // 1 for completed
			}
			
			if err := db.DB.WithContext(ctx).Create(blog).Error; err != nil {
				errs <- fmt.Errorf("failed to save chapter %d to db: %w", c.Sort, err)
				return
			}
			
			// Send progress: completed
			progressChan <- fmt.Sprintf(`{"status":"completed","chapter_sort":%d,"title":"%s"}`, c.Sort, c.Title)
			
		}(chapter)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			errChan <- err
			return
		}
	}
}
// GenerateOutline evaluates project text and generates a JSON outline
func (s *DecompositionService) GenerateOutline(ctx context.Context, sourceContent string) ([]Chapter, error) {
	prompt := fmt.Sprintf(`你是一个高级架构师。请评估以下项目文本，并生成一个系列博客的大纲。
输出必须是纯JSON数组格式，不包含任何Markdown标记或其他文字。
每个元素包含以下字段：
- title: 章节标题
- summary: 该章节的摘要或内容要点
- sort: 排序（整数，从1开始）

项目文本：
%s`, sourceContent)

	messages := []llm.Message{
		{Role: "system", Content: "你是一个高级架构师，只输出符合要求的纯JSON数组。"},
		{Role: "user", Content: prompt},
	}

	model := "deepseek-chat"
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

	var outline []Chapter
	if err := json.Unmarshal([]byte(content), &outline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal llm output: %w, output: %s", err, content)
	}

	return outline, nil
}
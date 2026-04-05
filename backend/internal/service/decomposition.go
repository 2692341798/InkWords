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
	"golang.org/x/sync/semaphore"

	"inkwords-backend/internal/db"
	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/model"
	"inkwords-backend/internal/parser"
)

// Chapter represents a single chapter in the generated outline
type Chapter struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Sort    int    `json:"sort"`
}

// DecompositionService handles the logic to evaluate project text and generate an outline
type DecompositionService struct {
	llmClient  *llm.DeepSeekClient
	gitFetcher *parser.GitFetcher
}

// NewDecompositionService creates a new decomposition service
func NewDecompositionService() *DecompositionService {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	return &DecompositionService{
		llmClient:  llm.NewDeepSeekClient(apiKey),
		gitFetcher: parser.NewGitFetcher(),
	}
}

// AnalyzeStream handles the full analysis pipeline with streaming progress
func (s *DecompositionService) AnalyzeStream(ctx context.Context, gitURL string, progressChan chan<- string, errChan chan<- error) {
	defer close(progressChan)
	defer close(errChan)

	sendProgress := func(step int, message string, data interface{}) {
		msg := map[string]interface{}{
			"step":    step,
			"message": message,
		}
		if data != nil {
			msg["data"] = data
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

	sendProgress(0, "正在克隆并拉取仓库 (depth=1)...", nil)
	
	treeContent, chunks, err := s.gitFetcher.Fetch(gitURL)
	if err != nil {
		errChan <- fmt.Errorf("拉取仓库失败: %w", err)
		return
	}

	sendProgress(1, "分析仓库源码与结构完成", nil)

	// Map-Reduce Phase
	var finalContent strings.Builder
	fullContent := treeContent + "\n=== Repository Content ===\n"

	// If it's a very small project, we can skip the Map-Reduce to save time and token overhead.
	if len(chunks) == 1 && len([]rune(chunks[0].Content)) < 150000 {
		sendProgress(2, "项目较小，跳过 Map 阶段，直接生成大纲...", nil)
		finalContent.WriteString(fullContent)
		finalContent.WriteString(chunks[0].Content)
	} else {
		sendProgress(2, fmt.Sprintf("开启 Map-Reduce 分析，共 %d 个分块", len(chunks)), nil)
		summaries := s.mapReduceAnalyze(ctx, chunks, sendProgress)
		finalContent.WriteString(treeContent)
		finalContent.WriteString("\n=== Local Summaries ===\n")
		for _, summary := range summaries {
			finalContent.WriteString(summary)
			finalContent.WriteString("\n\n")
		}
	}

	sendProgress(3, "评估大模型并生成项目全局大纲...", nil)

	outline, err := s.GenerateOutline(ctx, finalContent.String())
	if err != nil {
		errChan <- fmt.Errorf("生成大纲失败: %w", err)
		return
	}

	sendProgress(4, "正在完成最后处理...", map[string]interface{}{
		"outline":        outline,
		"source_content": finalContent.String(), // Provide the summarized content as source to save space
	})
}

// mapReduceAnalyze runs the map phase over the chunks and returns a list of local summaries
func (s *DecompositionService) mapReduceAnalyze(ctx context.Context, chunks []parser.FileChunk, sendProgress func(int, string, interface{})) []string {
	var summaries []string
	var mu sync.Mutex

	sem := semaphore.NewWeighted(5) // Max 5 concurrent goroutines
	var wg sync.WaitGroup

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, c parser.FileChunk) {
			defer wg.Done()
			if err := sem.Acquire(ctx, 1); err != nil {
				return
			}
			defer sem.Release(1)

			sendProgress(2, fmt.Sprintf("正在分析分块 %d/%d [%s]...", idx+1, len(chunks), c.Dir), map[string]interface{}{
				"status": "chunk_analyzing",
				"dir":    c.Dir,
				"index":  idx + 1,
				"total":  len(chunks),
			})

			summary := s.generateLocalSummaryWithRetry(ctx, c, 3, sendProgress, idx+1, len(chunks))

			if summary != "" {
				mu.Lock()
				summaries = append(summaries, summary)
				mu.Unlock()
				sendProgress(2, fmt.Sprintf("分块 %d/%d 分析完成", idx+1, len(chunks)), map[string]interface{}{
					"status": "chunk_done",
					"dir":    c.Dir,
					"index":  idx + 1,
				})
			}
		}(i, chunk)
	}

	wg.Wait()
	return summaries
}

func (s *DecompositionService) generateLocalSummaryWithRetry(ctx context.Context, chunk parser.FileChunk, maxRetries int, sendProgress func(int, string, interface{}), idx int, total int) string {
	prompt := fmt.Sprintf(`你是一个高级全栈架构师。请分析以下代码块，提取其核心功能、主要接口和数据结构。
你的输出应该是一份精简的局部摘要，不需要过多的寒暄，直接列出关键信息。
目录位置：%s
代码内容：
%s`, chunk.Dir, chunk.Content)

	messages := []llm.Message{
		{Role: "system", Content: "你是一个高级架构师，专注于代码分析并输出精简摘要。"},
		{Role: "user", Content: prompt},
	}

	modelStr := "deepseek-chat"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		modelStr = envModel
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ""
		default:
		}

		attemptCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		summary, err := s.llmClient.Generate(attemptCtx, modelStr, messages)
		cancel()

		if err == nil {
			return fmt.Sprintf("【目录: %s】\n%s", chunk.Dir, summary)
		}

		sendProgress(2, fmt.Sprintf("分块 %d/%d 分析失败，正在重试 (%d/%d)...", idx, total, attempt, maxRetries), map[string]interface{}{
			"status": "chunk_failed",
			"dir":    chunk.Dir,
			"index":  idx,
			"attempt": attempt,
		})

		time.Sleep(time.Second * time.Duration(attempt*2)) // backoff
	}

	sendProgress(2, fmt.Sprintf("分块 %d/%d 分析最终失败，已跳过", idx, total), map[string]interface{}{
		"status": "chunk_failed_final",
		"dir":    chunk.Dir,
		"index":  idx,
	})
	return ""
}

// GenerateSeries generates blog chapters sequentially based on the outline with streaming
func (s *DecompositionService) GenerateSeries(ctx context.Context, userID uuid.UUID, parentID uuid.UUID, outline []Chapter, sourceContent string, sourceType string, progressChan chan<- string, errChan chan<- error) {
	defer close(progressChan)
	defer close(errChan)

	// --- FIX START: Save the parent node so that History Blogs can query it ---
	parentTitle := "Git 源码解析系列"
	if sourceType == "file" {
		parentTitle = "文件解析系列"
	}
	parentBlog := &model.Blog{
		ID:         parentID,
		UserID:     userID,
		Title:      parentTitle,
		Content:    "该节点为系列文章的父节点，请点击展开查看具体的章节。",
		SourceType: sourceType,
		Status:     1, // 1 for completed
	}
	if err := db.DB.WithContext(ctx).Create(parentBlog).Error; err != nil {
		fmt.Printf("Failed to create parent blog: %v\n", err)
	}
	// --- FIX END ---

	for _, chapter := range outline {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
		}

		// Limit the content sent per chapter to avoid token overflow
		chapterSourceContent := sourceContent
		runes := []rune(chapterSourceContent)
		if len(runes) > 300000 {
			chapterSourceContent = string(runes[:300000]) + "\n\n... [Content Truncated due to length limits] ..."
		}

		prompt := fmt.Sprintf(`你是一个高级全栈架构师和技术博主。请根据以下提供的源内容，以及本章节的大纲，将其转化为一篇“小白友好、图文并茂、可独立复现”的高质量技术博客章节。
要求：
1. **字数必须极度充足，内容极其详实（不少于 2000 字的长文）**：严禁只写干瘪的总结。必须深入分析代码实现原理，拆解每一个核心机制。
2. **代码级剖析**：引用源内容中的核心代码，并逐行解释其作用。如果源内容因为截断而缺少具体代码，请基于目录结构和你的架构经验进行合理补充推演。
3. **可复现的步骤**：如果是实战或配置相关，请给出明确的执行步骤。
4. **小白友好**：在解释抽象的理论概念时，必须提供对应的代码示例或生活化比喻。
5. 所有生成的 Mermaid 图表代码块绝对禁止包含自定义样式关键字（如 style, classDef, linkStyle 等），必须使用基础语法。

源内容：
%s

本章节大纲：
- 标题: %s
- 摘要: %s
- 排序: %d
`, chapterSourceContent, chapter.Title, chapter.Summary, chapter.Sort)

		messages := []llm.Message{
			{Role: "system", Content: "你是一个高级技术博客作者。"},
			{Role: "user", Content: prompt},
		}

		llmModel := "deepseek-chat"
		if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
			llmModel = envModel
		}

		var streamErr error
		var content string
		maxRetries := 3

		for attempt := 1; attempt <= maxRetries; attempt++ {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			if attempt > 1 {
				retryMsg := map[string]interface{}{
					"status":       "retrying",
					"chapter_sort": chapter.Sort,
					"attempt":      attempt,
				}
				retryBytes, _ := json.Marshal(retryMsg)
				progressChan <- string(retryBytes)
			} else {
				startMsg := map[string]interface{}{
					"status":       "generating",
					"chapter_sort": chapter.Sort,
					"title":        chapter.Title,
				}
				startBytes, _ := json.Marshal(startMsg)
				progressChan <- string(startBytes)
			}

			streamCtx, streamCancel := context.WithCancel(ctx)
			chapterChunkChan := make(chan string)
			chapterErrChan := make(chan error)
			
			var fullContentBuilder strings.Builder
			
			go s.llmClient.GenerateStream(streamCtx, llmModel, messages, chapterChunkChan, chapterErrChan)

			idleTimeout := 30 * time.Second
			timer := time.NewTimer(idleTimeout)

			streamErr = nil
			done := false

			for !done {
				select {
				case <-ctx.Done():
					streamCancel()
					timer.Stop()
					errChan <- ctx.Err()
					return
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
					} else {
						if !timer.Stop() {
							select { case <-timer.C: default: }
						}
						timer.Reset(idleTimeout)

						fullContentBuilder.WriteString(chunk)
						streamMsg := map[string]interface{}{
							"status":       "streaming",
							"chapter_sort": chapter.Sort,
							"content":      chunk,
						}
						streamBytes, _ := json.Marshal(streamMsg)
						progressChan <- string(streamBytes)
					}
				}
			}

			timer.Stop()
			streamCancel()

			if streamErr == nil {
				content = fullContentBuilder.String()
				break
			}

			time.Sleep(time.Second * time.Duration(attempt*2))
		}

		if streamErr != nil {
			errChan <- fmt.Errorf("chapter %d generation failed after %d attempts: %w", chapter.Sort, maxRetries, streamErr)
			return
		}

		// Save to database
		blog := &model.Blog{
			UserID:      userID,
			ParentID:    &parentID,
			ChapterSort: chapter.Sort,
			Title:       chapter.Title,
			Content:     content,
			SourceType:  sourceType,
			Status:      1, // 1 for completed
		}
		
		if err := db.DB.WithContext(ctx).Create(blog).Error; err != nil {
			errChan <- fmt.Errorf("failed to save chapter %d to db: %w", chapter.Sort, err)
			return
		}
		
		// Send progress: completed
		endMsg := map[string]interface{}{
			"status":       "completed",
			"chapter_sort": chapter.Sort,
			"title":        chapter.Title,
		}
		endBytes, _ := json.Marshal(endMsg)
		progressChan <- string(endBytes)
	}
}
// GenerateOutline evaluates project text and generates a JSON outline
func (s *DecompositionService) GenerateOutline(ctx context.Context, sourceContent string) ([]Chapter, error) {
	// DeepSeek max context is ~128k tokens. 
	// Limit source content to ~300,000 characters to avoid API 400 errors (invalid_request_error).
	// 300,000 characters is roughly 75k - 100k tokens, leaving plenty of room for system prompts and the completion.
	runes := []rune(sourceContent)
	if len(runes) > 300000 {
		sourceContent = string(runes[:300000]) + "\n\n... [Content Truncated due to length limits] ..."
	}

	prompt := fmt.Sprintf(`你是一个高级架构师。请评估以下项目文本，并生成一个系列博客的大纲。
对于大型项目、源码仓库或复杂内容，**强制拆分为系列博客**（如：基础概念篇、核心架构篇、实战复现篇等），必须至少包含 3 个章节。
输出必须是纯JSON数组格式，不包含任何Markdown标记或其他文字。
每个元素包含以下字段：
- title: 章节标题
- summary: 该章节的详细摘要或内容要点（指导后续生成的具体方向）
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
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"inkwords-backend/internal/db"
	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/model"
	"inkwords-backend/internal/parser"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// GenerateSeries generates blog chapters sequentially based on the outline with streaming
func (s *DecompositionService) GenerateSeries(ctx context.Context, userID uuid.UUID, parentID uuid.UUID, seriesTitle string, outline []Chapter, sourceContent string, sourceType string, gitURL string, progressChan chan<- string, errChan chan<- error) {
	defer close(progressChan)
	defer close(errChan)

	sendSystemProgress := func(msg string) {
		progressMsg := map[string]interface{}{
			"status":  "progress",
			"message": msg,
		}
		bytes, _ := json.Marshal(progressMsg)
		progressChan <- string(bytes)
	}

	sendSystemProgress("正在准备环境...")

	// --- FIX START: Clone repo to precisely feed files ---
	var cachePath string
	if sourceType == "git" && gitURL != "" {
		sendSystemProgress("正在准备环境与代码...")
		dir, err := s.gitFetcher.GetCachedRepoPath(gitURL, sendSystemProgress)
		if err == nil {
			cachePath = dir
		}
	}

	sendSystemProgress("正在初始化数据库与生成队列...")

	parentTitle := "Git 源码解析系列"
	if sourceType == "file" {
		parentTitle = "文件解析系列"
	}
	if seriesTitle != "" {
		parentTitle = seriesTitle
	}

	var existingParent model.Blog
	if err := db.DB.WithContext(ctx).First(&existingParent, "id = ?", parentID).Error; err != nil {
		parentBlog := &model.Blog{
			ID:         parentID,
			UserID:     userID,
			Title:      parentTitle,
			Content:    "正在生成系列导读...",
			SourceType: sourceType,
			SourceURL:  gitURL,
			IsSeries:   true,
			Status:     0,
		}
		if err := db.DB.WithContext(ctx).Create(parentBlog).Error; err != nil {
			fmt.Printf("Failed to create parent blog: %v\n", err)
		}
	} else {
		// Update SourceURL if empty
		if existingParent.SourceURL == "" && gitURL != "" {
			db.DB.WithContext(ctx).Model(&existingParent).Update("source_url", gitURL)
		}
	}

	maxWorkers := maxWorkersFromEnv(len(outline))
	sem := semaphore.NewWeighted(int64(maxWorkers))
	var wg sync.WaitGroup

	// To keep track of valid children IDs for deletion later
	var validChildrenIDs []uuid.UUID
	for _, chapter := range outline {
		if chapter.ID != "" {
			if id, err := uuid.Parse(chapter.ID); err == nil {
				validChildrenIDs = append(validChildrenIDs, id)
			}
		}
	}

	// Delete obsolete children before generating new ones
	if len(validChildrenIDs) > 0 {
		db.DB.WithContext(ctx).Where("parent_id = ? AND user_id = ? AND id NOT IN ?", parentID, userID, validChildrenIDs).Delete(&model.Blog{})
	} else {
		db.DB.WithContext(ctx).Where("parent_id = ? AND user_id = ?", parentID, userID).Delete(&model.Blog{})
	}

	sendSystemProgress("开始生成系列博客内容...")
	for i, chapter := range outline {
		if ctx.Err() != nil {
			errChan <- ctx.Err()
			break
		}

		if chapter.Action == "skip" && chapter.ID != "" {
			if blogID, err := uuid.Parse(chapter.ID); err == nil {
				// Update sort and title for existing skipped chapter
				db.DB.WithContext(ctx).Model(&model.Blog{}).Where("id = ? AND user_id = ?", blogID, userID).Updates(map[string]interface{}{
					"chapter_sort": chapter.Sort,
					"title":        chapter.Title,
				})
			}
			endMsg := map[string]interface{}{
				"status":       "completed",
				"chapter_sort": chapter.Sort,
				"title":        chapter.Title,
			}
			endBytes, _ := json.Marshal(endMsg)
			progressChan <- string(endBytes)
			continue
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			errChan <- err
			break
		}

		wg.Add(1)
		go func(i int, chapter Chapter) {
			defer sem.Release(1)
			defer wg.Done()

			// Limit the content sent per chapter to avoid token overflow
			var chapterSourceContent string
			if sourceType == "git" && cachePath != "" && len(chapter.Files) > 0 {
				var builder strings.Builder
				for _, fPath := range chapter.Files {
					fPath = strings.TrimSpace(fPath)
					if fPath == "" {
						continue
					}
					// check if it's a directory or a file
					cmdCheck := exec.Command("git", "cat-file", "-t", "HEAD:"+fPath)
					cmdCheck.Dir = cachePath
					objTypeBytes, err := cmdCheck.Output()
					if err != nil {
						continue
					}
					objType := strings.TrimSpace(string(objTypeBytes))

					if objType == "tree" {
						cmdLs := exec.Command("git", "ls-tree", "-r", "--name-only", "HEAD", fPath)
						cmdLs.Dir = cachePath
						outBytes, err := cmdLs.Output()
						if err == nil {
							files := strings.Split(strings.ReplaceAll(string(outBytes), "\r\n", "\n"), "\n")
							for _, p := range files {
								p = strings.TrimSpace(p)
								if p != "" && !parser.IsBinaryExt(strings.ToLower(filepath.Ext(p))) {
									cmdShow := exec.Command("git", "show", "HEAD:"+p)
									cmdShow.Dir = cachePath
									data, err := cmdShow.Output()
									if err == nil {
										builder.WriteString(fmt.Sprintf("--- File: %s ---\n%s\n\n", p, string(data)))
									}
								}
							}
						}
					} else {
						if !parser.IsBinaryExt(strings.ToLower(filepath.Ext(fPath))) {
							cmdShow := exec.Command("git", "show", "HEAD:"+fPath)
							cmdShow.Dir = cachePath
							data, err := cmdShow.Output()
							if err == nil {
								builder.WriteString(fmt.Sprintf("--- File: %s ---\n%s\n\n", fPath, string(data)))
							}
						}
					}
				}
				if builder.Len() > 0 {
					chapterSourceContent = builder.String()
				} else {
					chapterSourceContent = sourceContent
				}
			} else {
				chapterSourceContent = sourceContent
			}

			runes := []rune(chapterSourceContent)
			if len(runes) > 1000000 {
				chapterSourceContent = string(runes[:1000000]) + "\n\n... [Content Truncated due to length limits] ..."
			}

			extraRequirements := ""
			reqIndex := 7
			if gitURL != "" {
				extraRequirements += fmt.Sprintf("%d. **源码仓库引用**：请在文章开头或合适的位置，引用本项目的 Git 仓库地址：%s\n", reqIndex, gitURL)
				reqIndex++
			}
			if i+1 < len(outline) {
				extraRequirements += fmt.Sprintf("%d. **下期预告**：请在文章结尾处，明确预告下一篇文章的内容：“下期预告：%s”\n", reqIndex, outline[i+1].Title)
				reqIndex++
			}

			var oldContent string
			if chapter.Action == "regenerate" && chapter.ID != "" {
				if blogID, err := uuid.Parse(chapter.ID); err == nil {
					var oldBlog model.Blog
					if err := db.DB.WithContext(ctx).Select("content").First(&oldBlog, "id = ?", blogID).Error; err == nil {
						oldContent = oldBlog.Content
						oldRunes := []rune(oldContent)
						if len(oldRunes) > 500000 {
							oldContent = string(oldRunes[:500000]) + "\n\n... [Content Truncated due to length limits] ..."
						}
					}
				}
			}

			prompt := fmt.Sprintf(`请根据上述源内容，以及本章节的大纲，将其转化为一篇“小白友好、图文并茂、可独立复现”的高质量技术博客章节。
要求：
1. **单点聚焦与深度剖析**：严格保证本篇文章只介绍**一个核心技术点**。请利用充足的上下文，深入剖析其底层原理、设计思想和演进逻辑，不要停留在表面的 API 调用，字数篇幅不设上限，请尽可能详尽。
2. **丰富的代码示例**：在解释原理和应用时，尽可能多地提供代码示例（不仅仅是源码，还可以是辅助理解的伪代码或最佳实践用例），引用源内容中的核心代码并逐行解释其作用。如果源内容因为截断而缺少具体代码，请基于目录结构和你的架构经验进行合理补充推演。
3. **可复现的步骤**：如果是实战或配置相关，请给出明确的执行步骤。
4. **小白友好**：在解释抽象的理论概念时，必须提供对应的代码示例或生活化比喻。
5. 所有生成的 Mermaid 图表代码块绝对禁止包含自定义样式关键字（如 style, classDef, linkStyle 等），必须使用基础语法。在 Mermaid 图表中，如果节点文本包含特殊字符（如括号、幂符号等，例如 O(1), O(n^2)），必须使用双引号将节点文本包裹起来，例如 A["O(1)"] 而不是 A[O(1)]。
6. **完整性约束**：请务必完整输出，不要遗漏关键知识点。如果内容较长，请合理分配篇幅，确保文章结构完整，包含结尾总结。
%s

本章节大纲：
- 标题: %s
- 摘要: %s
- 排序: %d
`, extraRequirements, chapter.Title, chapter.Summary, chapter.Sort)

			if oldContent != "" {
				prompt += fmt.Sprintf("\n【注意：本章节为旧版博客的更新重写】\n以下是该章节在旧版本项目中的博客内容，供你作为松散参考。\n你可以参考旧内容中解释抽象概念的比喻、业务知识点或行文风格，但必须以本次提供的最新源码为准进行重写或调整，如果最新代码逻辑发生了改变，请以最新代码为准。\n旧版本内容：\n---\n%s\n---\n", oldContent)
			}

			messages := []llm.Message{
				{Role: "system", Content: "你是一个高级全栈架构师和技术博主。\n\n项目源内容如下：\n" + chapterSourceContent},
				{Role: "user", Content: prompt},
			}

			llmModel := "deepseek-v4-flash"
			if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
				llmModel = envModel
			}

			var streamErr error
			maxRetries := 3
			var content string

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
						"parent_id":    parentID.String(),
					}
					retryBytes, _ := json.Marshal(retryMsg)
					progressChan <- string(retryBytes)
				} else {
					startMsg := map[string]interface{}{
						"status":       "generating",
						"chapter_sort": chapter.Sort,
						"title":        chapter.Title,
						"parent_id":    parentID.String(),
					}
					startBytes, _ := json.Marshal(startMsg)
					progressChan <- string(startBytes)
				}

				streamCtx, streamCancel := context.WithCancel(ctx)
				chapterChunkChan := make(chan string, 100)
				chapterErrChan := make(chan error, 1)

				var fullContentBuilder strings.Builder

				go func() {
					defer close(chapterChunkChan)
					defer close(chapterErrChan)

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
								chapterChunkChan <- chunk
							}
						}()

						finishReason, err := s.llmClient.GenerateStream(streamCtx, llmModel, currentMessages, tempChunkChan)
						wg.Wait()

						if err != nil {
							chapterErrChan <- err
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

				idleTimeout := 60 * time.Second
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
								select {
								case <-timer.C:
								default:
								}
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

				time.Sleep(exponentialBackoff(attempt))
			}

			if streamErr != nil {
				errMsg := map[string]interface{}{
					"status":       "error",
					"chapter_sort": chapter.Sort,
					"message":      fmt.Sprintf("chapter %d generation failed after %d attempts: %v", chapter.Sort, maxRetries, streamErr),
				}
				errBytes, _ := json.Marshal(errMsg)
				progressChan <- string(errBytes)
				return
			}

			wordCount := len([]rune(content))

			// Extract Tech Stacks using LLM
			var techStacks datatypes.JSON
			extractPrompt := "请从以下文章内容中提取出涉及的核心技术栈名称（如 React, Go, Docker 等），以 JSON 数组格式返回，不要有任何其他多余字符。\n\n例如：[\"React\", \"Go\"]\n\n文章内容：\n\n" + content
			extractMessages := []llm.Message{
				{Role: "user", Content: extractPrompt},
			}

			extractedJSON, err := s.llmClient.GenerateJSON(ctx, llmModel, extractMessages)
			if err == nil && len(extractedJSON) > 0 {
				// basic validation that it is a json array
				var parsed []string
				if json.Unmarshal([]byte(extractedJSON), &parsed) == nil {
					techStacks = datatypes.JSON(extractedJSON)
				}
			}

			var updated bool
			if chapter.ID != "" {
				if blogID, err := uuid.Parse(chapter.ID); err == nil {
					if err := db.DB.WithContext(ctx).Model(&model.Blog{}).Where("id = ? AND user_id = ?", blogID, userID).Updates(map[string]interface{}{
						"chapter_sort": chapter.Sort,
						"title":        chapter.Title,
						"content":      content,
						"word_count":   wordCount,
						"tech_stacks":  techStacks,
					}).Error; err != nil {
						errMsg := map[string]interface{}{
							"status":       "error",
							"chapter_sort": chapter.Sort,
							"message":      fmt.Sprintf("failed to update chapter %d in db: %v", chapter.Sort, err),
						}
						errBytes, _ := json.Marshal(errMsg)
						progressChan <- string(errBytes)
						return
					}
					updated = true
				}
			}

			if !updated {
				blog := &model.Blog{
					UserID:      userID,
					ParentID:    &parentID,
					ChapterSort: chapter.Sort,
					Title:       chapter.Title,
					Content:     content,
					SourceType:  sourceType,
					Status:      1,
					WordCount:   wordCount,
					TechStacks:  techStacks,
				}

				if err := db.DB.WithContext(ctx).Create(blog).Error; err != nil {
					errMsg := map[string]interface{}{
						"status":       "error",
						"chapter_sort": chapter.Sort,
						"message":      fmt.Sprintf("failed to save chapter %d to db: %v", chapter.Sort, err),
					}
					errBytes, _ := json.Marshal(errMsg)
					progressChan <- string(errBytes)
					return
				}
			}

			estimatedTokens := len([]rune(content)) * 2
			db.DB.Model(&model.User{}).Where("id = ?", userID).UpdateColumn("tokens_used", gorm.Expr("tokens_used + ?", estimatedTokens))

			endMsg := map[string]interface{}{
				"status":       "completed",
				"chapter_sort": chapter.Sort,
				"title":        chapter.Title,
			}
			endBytes, _ := json.Marshal(endMsg)
			progressChan <- string(endBytes)
		}(i, chapter)
	}

	wg.Wait()

	if ctx.Err() == nil {

		s.generateSeriesIntro(ctx, userID, parentID, seriesTitle, outline, progressChan, errChan)
	}
}

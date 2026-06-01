package service

import (
	"context"
	"encoding/json"
	"fmt"
	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/model"
	"inkwords-backend/internal/prompt"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"
)

// GenerateSeries generates blog chapters sequentially based on the outline with streaming.
func (s *DecompositionService) GenerateSeries(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	seriesTitle string,
	outline []Chapter,
	sourceContent string,
	sourceType string,
	gitURL string,
	scenarioMode prompt.ScenarioMode,
	style string,
	progressChan chan<- string,
	errChan chan<- error,
) {
	s.GenerateSeriesWithProfile(
		ctx,
		userID,
		parentID,
		seriesTitle,
		outline,
		sourceContent,
		sourceType,
		gitURL,
		scenarioMode,
		style,
		prompt.PromptProfile{},
		progressChan,
		errChan,
	)
}

func (s *DecompositionService) GenerateSeriesWithProfile(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	seriesTitle string,
	outline []Chapter,
	sourceContent string,
	sourceType string,
	gitURL string,
	scenarioMode prompt.ScenarioMode,
	style string,
	profile prompt.PromptProfile,
	progressChan chan<- string,
	errChan chan<- error,
) {
	defer close(progressChan)
	defer close(errChan)

	if !scenarioMode.IsValid() {
		scenarioMode = prompt.DefaultScenarioModeForSource(sourceType)
	}
	profile = normalizePromptProfile(profile, scenarioMode)

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

	// Why: 系列章节的实际生成是异步并带重试的，若请求中途失败或某章最终失败，
	// 先落一条子博客草稿，历史树仍能展示完整系列结构，而不是只剩父级导读。
	for i := range outline {
		chapter := outline[i]
		if chapter.ID != "" || chapter.Action == "skip" {
			continue
		}

		draftBlog := &model.Blog{
			UserID:      userID,
			ParentID:    &parentID,
			ChapterSort: chapter.Sort,
			Title:       chapter.Title,
			Content:     "正在生成章节内容...",
			SourceType:  sourceType,
			Status:      0,
		}

		if err := db.DB.WithContext(ctx).Create(draftBlog).Error; err != nil {
			errChan <- fmt.Errorf("failed to create chapter draft %d: %w", chapter.Sort, err)
			return
		}

		outline[i].ID = draftBlog.ID.String()
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

			chapterSourceContent := resolveSeriesChapterSourceContent(sourceType, cachePath, sourceContent, chapter)
			oldContent := resolveSeriesOldContent(ctx, chapter)
			qualityResult, streamErr := s.runSeriesChapterQualityPipeline(ctx, seriesQualityPipelineInput{
				SeriesTitle:          parentTitle,
				ReaderProfile:        buildSeriesReaderProfile(scenarioMode),
				Outline:              outline,
				ChapterIndex:         i,
				Chapter:              chapter,
				ChapterSourceContent: chapterSourceContent,
				GitURL:               gitURL,
				OldContent:           oldContent,
				ProgressChan:         progressChan,
			})
			if streamErr != nil {
				if chapter.ID != "" {
					if blogID, err := uuid.Parse(chapter.ID); err == nil {
						db.DB.WithContext(ctx).Model(&model.Blog{}).Where("id = ? AND user_id = ?", blogID, userID).Updates(map[string]interface{}{
							"status":  2,
							"content": "章节生成失败，请重试。",
						})
					}
				}

				errMsg := map[string]interface{}{
					"status":       "error",
					"chapter_sort": chapter.Sort,
					"message":      fmt.Sprintf("chapter %d generation failed: %v", chapter.Sort, streamErr),
				}
				errBytes, _ := json.Marshal(errMsg)
				progressChan <- string(errBytes)
				return
			}

			content := qualityResult.FinalMarkdown
			wordCount := len([]rune(content))

			llmModel := "deepseek-v4-flash"
			techStacks := s.extractSeriesChapterTechStacks(ctx, llmModel, content)

			var updated bool
			if chapter.ID != "" {
				if blogID, err := uuid.Parse(chapter.ID); err == nil {
					if err := db.DB.WithContext(ctx).Model(&model.Blog{}).Where("id = ? AND user_id = ?", blogID, userID).Updates(map[string]interface{}{
						"chapter_sort": chapter.Sort,
						"title":        chapter.Title,
						"content":      content,
						"word_count":   wordCount,
						"tech_stacks":  techStacks,
						"source_type":  sourceType,
						"status":       1,
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
		s.generateSeriesIntro(ctx, userID, parentID, seriesTitle, outline, scenarioMode, prompt.ArticleStyle(style), profile, progressChan, errChan)
	}
}

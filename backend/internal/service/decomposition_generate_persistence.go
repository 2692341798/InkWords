package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/model"
)

func ensureSeriesParentAndDrafts(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	parentTitle string,
	sourceType string,
	gitURL string,
	outline []Chapter,
) ([]Chapter, error) {
	updatedOutline := append([]Chapter(nil), outline...)

	if err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingParent model.Blog
		err := tx.First(&existingParent, "id = ?", parentID).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("query parent blog: %w", err)
			}

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
			if err := tx.Create(parentBlog).Error; err != nil {
				return fmt.Errorf("create parent blog: %w", err)
			}
		} else if existingParent.SourceURL == "" && gitURL != "" {
			if err := tx.Model(&existingParent).Update("source_url", gitURL).Error; err != nil {
				return fmt.Errorf("update parent source url: %w", err)
			}
		}

		validChildrenIDs := collectValidSeriesChildIDs(updatedOutline)
		deleteQuery := tx.Where("parent_id = ? AND user_id = ?", parentID, userID)
		if len(validChildrenIDs) > 0 {
			deleteQuery = deleteQuery.Where("id NOT IN ?", validChildrenIDs)
		}
		if err := deleteQuery.Delete(&model.Blog{}).Error; err != nil {
			return fmt.Errorf("delete obsolete child blogs: %w", err)
		}

		// Why: 草稿创建与旧子节点清理必须是一个原子阶段，否则一旦新草稿创建失败，
		// 用户会看到系列树被删空，无法判断哪些章节原本已经存在。
		for i := range updatedOutline {
			chapter := updatedOutline[i]
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
			if err := tx.Create(draftBlog).Error; err != nil {
				return fmt.Errorf("create chapter draft %d: %w", chapter.Sort, err)
			}

			updatedOutline[i].ID = draftBlog.ID.String()
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return updatedOutline, nil
}

func collectValidSeriesChildIDs(outline []Chapter) []uuid.UUID {
	validChildrenIDs := make([]uuid.UUID, 0, len(outline))
	for _, chapter := range outline {
		if chapter.ID == "" {
			continue
		}
		if id, err := uuid.Parse(chapter.ID); err == nil {
			validChildrenIDs = append(validChildrenIDs, id)
		}
	}

	return validChildrenIDs
}

func persistSeriesChapterCompletion(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	sourceType string,
	chapter Chapter,
	content string,
	wordCount int,
	techStacks datatypes.JSON,
) error {
	if taskOnlyPersistenceMode() {
		return nil
	}

	estimatedTokens := len([]rune(content)) * 2

	return db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if chapter.ID != "" {
			blogID, err := uuid.Parse(chapter.ID)
			if err != nil {
				return fmt.Errorf("parse chapter blog id: %w", err)
			}

			updateResult := tx.Model(&model.Blog{}).
				Where("id = ? AND user_id = ?", blogID, userID).
				Updates(map[string]interface{}{
					"chapter_sort": chapter.Sort,
					"title":        chapter.Title,
					"content":      content,
					"word_count":   wordCount,
					"tech_stacks":  techStacks,
					"source_type":  sourceType,
					"status":       1,
				})
			if updateResult.Error != nil {
				return fmt.Errorf("update chapter blog: %w", updateResult.Error)
			}
			if updateResult.RowsAffected == 0 {
				return fmt.Errorf("update chapter blog: blog not found")
			}
		} else {
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
			if err := tx.Create(blog).Error; err != nil {
				return fmt.Errorf("create chapter blog: %w", err)
			}
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
	})
}

func handleSeriesChapterCompletion(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	sourceType string,
	chapter Chapter,
	content string,
	wordCount int,
	techStacks []string,
	collector *seriesTaskResultCollector,
) error {
	// Why: task_only 模式下系列章节的最终业务事实必须先收口进 result_json，
	// 否则 core-api 无法在任务成功路径里一次性接管父子博客持久化。
	if taskOnlyPersistenceMode() {
		collector.AddChapterSuccess(chapter, content, wordCount, techStacks)
		return nil
	}

	techStacksJSON, err := json.Marshal(techStacks)
	if err != nil {
		return fmt.Errorf("marshal series chapter tech stacks: %w", err)
	}

	return persistSeriesChapterCompletion(
		ctx,
		userID,
		parentID,
		sourceType,
		chapter,
		content,
		wordCount,
		datatypes.JSON(techStacksJSON),
	)
}

func handleSeriesChapterFailure(
	ctx context.Context,
	userID uuid.UUID,
	chapter Chapter,
	streamErr error,
	collector *seriesTaskResultCollector,
) {
	if taskOnlyPersistenceMode() {
		collector.AddChapterFailure(chapter, streamErr.Error())
		return
	}

	if chapter.ID == "" {
		return
	}

	blogID, err := uuid.Parse(chapter.ID)
	if err != nil {
		return
	}

	db.DB.WithContext(ctx).Model(&model.Blog{}).Where("id = ? AND user_id = ?", blogID, userID).Updates(map[string]interface{}{
		"status":  2,
		"content": "章节生成失败，请重试。",
	})
}

func decodeTechStacksJSON(raw datatypes.JSON) []string {
	var techStacks []string
	if len(raw) == 0 {
		return []string{}
	}
	if err := json.Unmarshal(raw, &techStacks); err != nil {
		return []string{}
	}
	return techStacks
}

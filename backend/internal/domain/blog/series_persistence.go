package blog

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"inkwords-backend/internal/model"
)

type seriesPersistence struct {
	db *gorm.DB
}

// NewSeriesPersistence creates the blog-domain adapter for series generation
// business facts that still belong to core-api owned blog tables.
func NewSeriesPersistence(database *gorm.DB) blogcontracts.SeriesPersistence {
	return &seriesPersistence{db: database}
}

func (p *seriesPersistence) EnsureSeriesParentAndDrafts(ctx context.Context, input blogcontracts.SeriesDraftPreflightInput) ([]blogcontracts.Chapter, error) {
	if p.db == nil {
		return nil, fmt.Errorf("series persistence database is not initialized")
	}

	updatedOutline := append([]blogcontracts.Chapter(nil), input.Outline...)
	if err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingParent model.Blog
		err := tx.Select("id", "user_id", "source_url").First(&existingParent, "id = ?", input.ParentID).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("query parent blog: %w", err)
			}

			parentBlog := &model.Blog{
				ID:         input.ParentID,
				UserID:     input.UserID,
				Title:      input.ParentTitle,
				Content:    "正在生成系列导读...",
				SourceType: input.SourceType,
				SourceURL:  input.GitURL,
				IsSeries:   true,
				Status:     0,
			}
			if err := tx.Create(parentBlog).Error; err != nil {
				return fmt.Errorf("create parent blog: %w", err)
			}
		} else {
			if existingParent.UserID != input.UserID {
				return fmt.Errorf("parent blog does not belong to user")
			}
		}
		if err == nil && existingParent.SourceURL == "" && input.GitURL != "" {
			if err := tx.Model(&existingParent).Update("source_url", input.GitURL).Error; err != nil {
				return fmt.Errorf("update parent source url: %w", err)
			}
		}

		validChildrenIDs := collectValidSeriesChildIDs(updatedOutline)
		deleteQuery := tx.Where("parent_id = ? AND user_id = ?", input.ParentID, input.UserID)
		if len(validChildrenIDs) > 0 {
			deleteQuery = deleteQuery.Where("id NOT IN ?", validChildrenIDs)
		}
		if err := deleteQuery.Delete(&model.Blog{}).Error; err != nil {
			return fmt.Errorf("delete obsolete child blogs: %w", err)
		}

		// Why: 草稿创建与旧子节点清理必须在同一事务中完成，避免出现树已删空但新草稿没建好的半成品状态。
		for i := range updatedOutline {
			chapter := updatedOutline[i]
			if chapter.ID != "" || chapter.Action == "skip" {
				continue
			}

			draftBlog := &model.Blog{
				UserID:      input.UserID,
				ParentID:    &input.ParentID,
				ChapterSort: chapter.Sort,
				Title:       chapter.Title,
				Content:     "正在生成章节内容...",
				SourceType:  input.SourceType,
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

func (p *seriesPersistence) SaveSeriesChapter(ctx context.Context, input blogcontracts.SeriesChapterPersistenceInput) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	estimatedTokens := len([]rune(input.Content)) * 2
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if input.BlogID != uuid.Nil {
			updateResult := tx.Model(&model.Blog{}).
				Where("id = ? AND user_id = ?", input.BlogID, input.UserID).
				Updates(map[string]any{
					"chapter_sort": input.Chapter.Sort,
					"title":        input.Chapter.Title,
					"content":      input.Content,
					"word_count":   input.WordCount,
					"tech_stacks":  input.TechStacks,
					"source_type":  input.SourceType,
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
				UserID:      input.UserID,
				ParentID:    &input.ParentID,
				ChapterSort: input.Chapter.Sort,
				Title:       input.Chapter.Title,
				Content:     input.Content,
				SourceType:  input.SourceType,
				Status:      1,
				WordCount:   input.WordCount,
				TechStacks:  input.TechStacks,
			}
			if err := tx.Create(blog).Error; err != nil {
				return fmt.Errorf("create chapter blog: %w", err)
			}
		}

		tokenUpdateResult := tx.Model(&model.User{}).
			Where("id = ?", input.UserID).
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

func (p *seriesPersistence) MarkSeriesChapterFailed(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	return p.db.WithContext(ctx).Model(&model.Blog{}).Where("id = ? AND user_id = ?", blogID, userID).Updates(map[string]any{
		"status":  2,
		"content": "章节生成失败，请重试。",
	}).Error
}

func (p *seriesPersistence) SaveSeriesIntro(ctx context.Context, parentID uuid.UUID, content string) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	return p.db.WithContext(ctx).Model(&model.Blog{}).Where("id = ?", parentID).Updates(map[string]any{
		"content": content,
		"status":  1,
	}).Error
}

func (p *seriesPersistence) MarkSeriesIntroFailed(ctx context.Context, parentID uuid.UUID) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	return p.db.WithContext(ctx).Model(&model.Blog{}).Where("id = ?", parentID).Updates(map[string]any{
		"status": 2,
	}).Error
}

func (p *seriesPersistence) LoadSeriesOldContent(ctx context.Context, blogID uuid.UUID) (string, error) {
	if p.db == nil {
		return "", fmt.Errorf("series persistence database is not initialized")
	}

	var blog model.Blog
	if err := p.db.WithContext(ctx).Select("content").First(&blog, "id = ?", blogID).Error; err != nil {
		return "", err
	}
	return blog.Content, nil
}

func (p *seriesPersistence) UpdateSkippedSeriesChapterMeta(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chapter blogcontracts.Chapter) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	return p.db.WithContext(ctx).Model(&model.Blog{}).Where("id = ? AND user_id = ?", blogID, userID).Updates(map[string]any{
		"chapter_sort": chapter.Sort,
		"title":        chapter.Title,
	}).Error
}

func collectValidSeriesChildIDs(outline []blogcontracts.Chapter) []uuid.UUID {
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

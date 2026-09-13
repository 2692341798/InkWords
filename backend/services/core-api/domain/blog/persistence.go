package blog

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	sharedblog "inkwords-backend/shared/kernel/blog"
)

type continuePersistence struct {
	db *gorm.DB
}

// NewContinuePersistence creates the core-api adapter for continue reads and
// final content updates owned by the blogs table.
func NewContinuePersistence(database *gorm.DB) sharedblog.ContinuePersistence {
	return &continuePersistence{db: database}
}

func (p *continuePersistence) LoadContinueBlog(ctx context.Context, workspaceID uuid.UUID, blogID uuid.UUID) (sharedblog.ContinueBlog, error) {
	if p.db == nil {
		return sharedblog.ContinueBlog{}, fmt.Errorf("continue persistence database is not initialized")
	}

	var blog Blog
	if err := p.db.WithContext(ctx).First(&blog, "id = ? AND workspace_id = ?", blogID, workspaceID).Error; err != nil {
		return sharedblog.ContinueBlog{}, err
	}
	return sharedblog.ContinueBlog{ID: blog.ID, WorkspaceID: blog.WorkspaceID, Content: blog.Content}, nil
}

func (p *continuePersistence) SaveContinuedBlog(ctx context.Context, blog sharedblog.ContinueBlog, updatedContent string) error {
	if p.db == nil {
		return fmt.Errorf("continue persistence database is not initialized")
	}

	return p.db.WithContext(ctx).
		Model(&Blog{}).
		Where("id = ?", blog.ID).
		Update("content", updatedContent).Error
}

type seriesPersistence struct {
	db *gorm.DB
}

// NewSeriesPersistence creates the core-api adapter for series generation
// business facts that still belong to blogs and user token accounting.
func NewSeriesPersistence(database *gorm.DB) sharedblog.SeriesPersistence {
	return &seriesPersistence{db: database}
}

func (p *seriesPersistence) EnsureSeriesParentAndDrafts(ctx context.Context, input sharedblog.SeriesDraftPreflightInput) ([]sharedblog.Chapter, error) {
	if p.db == nil {
		return nil, fmt.Errorf("series persistence database is not initialized")
	}

	updatedOutline := append([]sharedblog.Chapter(nil), input.Outline...)
	if err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingParent Blog
		err := tx.Select("id", "workspace_id", "source_url").First(&existingParent, "id = ?", input.ParentID).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("query parent blog: %w", err)
			}

			parentBlog := &Blog{
				ID:         input.ParentID,
				WorkspaceID: input.WorkspaceID,
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
		} else if existingParent.WorkspaceID != input.WorkspaceID {
			return fmt.Errorf("parent blog does not belong to workspace")
		}
		if err == nil && existingParent.SourceURL == "" && input.GitURL != "" {
			if err := tx.Model(&existingParent).Update("source_url", input.GitURL).Error; err != nil {
				return fmt.Errorf("update parent source url: %w", err)
			}
		}

		validChildrenIDs := collectValidSeriesChildIDs(updatedOutline)
		deleteQuery := tx.Where("parent_id = ? AND workspace_id = ?", input.ParentID, input.WorkspaceID)
		if len(validChildrenIDs) > 0 {
			deleteQuery = deleteQuery.Where("id NOT IN ?", validChildrenIDs)
		}
		if err := deleteQuery.Delete(&Blog{}).Error; err != nil {
			return fmt.Errorf("delete obsolete child blogs: %w", err)
		}

		// Why: draft creation and obsolete-child cleanup are one business fact.
		for i := range updatedOutline {
			chapter := updatedOutline[i]
			if chapter.ID != "" || chapter.Action == "skip" {
				continue
			}

			draftBlog := &Blog{
				WorkspaceID: input.WorkspaceID,
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

func (p *seriesPersistence) SaveSeriesChapter(ctx context.Context, input sharedblog.SeriesChapterPersistenceInput) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if input.BlogID != uuid.Nil {
			updateResult := tx.Model(&Blog{}).
				Where("id = ? AND workspace_id = ?", input.BlogID, input.WorkspaceID).
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
			blog := &Blog{
				WorkspaceID: input.WorkspaceID,
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

		return nil
	})
}

func (p *seriesPersistence) MarkSeriesChapterFailed(ctx context.Context, workspaceID uuid.UUID, blogID uuid.UUID) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	return p.db.WithContext(ctx).Model(&Blog{}).Where("id = ? AND workspace_id = ?", blogID, workspaceID).Updates(map[string]any{
		"status":  2,
		"content": "章节生成失败，请重试。",
	}).Error
}

func (p *seriesPersistence) SaveSeriesIntro(ctx context.Context, workspaceID uuid.UUID, parentID uuid.UUID, content string) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	updateResult := p.db.WithContext(ctx).Model(&Blog{}).Where("id = ? AND workspace_id = ?", parentID, workspaceID).Updates(map[string]any{
		"content": content,
		"status":  1,
	})
	if updateResult.Error != nil {
		return updateResult.Error
	}
	if updateResult.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (p *seriesPersistence) MarkSeriesIntroFailed(ctx context.Context, workspaceID uuid.UUID, parentID uuid.UUID) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	updateResult := p.db.WithContext(ctx).Model(&Blog{}).Where("id = ? AND workspace_id = ?", parentID, workspaceID).Updates(map[string]any{
		"status": 2,
	})
	if updateResult.Error != nil {
		return updateResult.Error
	}
	if updateResult.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (p *seriesPersistence) LoadSeriesOldContent(ctx context.Context, workspaceID uuid.UUID, blogID uuid.UUID) (string, error) {
	if p.db == nil {
		return "", fmt.Errorf("series persistence database is not initialized")
	}

	var blog Blog
	if err := p.db.WithContext(ctx).Select("content").First(&blog, "id = ? AND workspace_id = ?", blogID, workspaceID).Error; err != nil {
		return "", err
	}
	return blog.Content, nil
}

func (p *seriesPersistence) UpdateSkippedSeriesChapterMeta(ctx context.Context, workspaceID uuid.UUID, blogID uuid.UUID, chapter sharedblog.Chapter) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	return p.db.WithContext(ctx).Model(&Blog{}).Where("id = ? AND workspace_id = ?", blogID, workspaceID).Updates(map[string]any{
		"chapter_sort": chapter.Sort,
		"title":        chapter.Title,
	}).Error
}

func collectValidSeriesChildIDs(outline []sharedblog.Chapter) []uuid.UUID {
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

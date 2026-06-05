package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

// SeriesChapterPersistenceInput 描述系列章节完成后需要持久化的业务事实。
type SeriesChapterPersistenceInput struct {
	UserID     uuid.UUID
	ParentID   uuid.UUID
	BlogID     uuid.UUID
	Chapter    Chapter
	SourceType string
	Content    string
	WordCount  int
	TechStacks datatypes.JSON
}

// SeriesDraftPreflightInput 描述系列生成前置草稿准备阶段需要的上下文。
type SeriesDraftPreflightInput struct {
	UserID      uuid.UUID
	ParentID    uuid.UUID
	ParentTitle string
	SourceType  string
	GitURL      string
	Outline     []Chapter
}

// SeriesPersistence 收口系列生成阶段仍归 core-api 持有的业务表写入。
type SeriesPersistence interface {
	EnsureSeriesParentAndDrafts(ctx context.Context, input SeriesDraftPreflightInput) ([]Chapter, error)
	SaveSeriesChapter(ctx context.Context, input SeriesChapterPersistenceInput) error
	MarkSeriesChapterFailed(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) error
	SaveSeriesIntro(ctx context.Context, parentID uuid.UUID, content string) error
	MarkSeriesIntroFailed(ctx context.Context, parentID uuid.UUID) error
	LoadSeriesOldContent(ctx context.Context, blogID uuid.UUID) (string, error)
	UpdateSkippedSeriesChapterMeta(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chapter Chapter) error
}

type gormSeriesPersistence struct {
	db *gorm.DB
}

// NewGormSeriesPersistence creates the default GORM-backed series persistence adapter.
func NewGormSeriesPersistence(database *gorm.DB) SeriesPersistence {
	return &gormSeriesPersistence{db: database}
}

func (p *gormSeriesPersistence) EnsureSeriesParentAndDrafts(ctx context.Context, input SeriesDraftPreflightInput) ([]Chapter, error) {
	if p.db == nil {
		return nil, fmt.Errorf("series persistence database is not initialized")
	}

	updatedOutline := append([]Chapter(nil), input.Outline...)
	if err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingParent model.Blog
		err := tx.First(&existingParent, "id = ?", input.ParentID).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
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
		} else if existingParent.SourceURL == "" && input.GitURL != "" {
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

func (p *gormSeriesPersistence) SaveSeriesChapter(ctx context.Context, input SeriesChapterPersistenceInput) error {
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

func (p *gormSeriesPersistence) MarkSeriesChapterFailed(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	return p.db.WithContext(ctx).Model(&model.Blog{}).Where("id = ? AND user_id = ?", blogID, userID).Updates(map[string]any{
		"status":  2,
		"content": "章节生成失败，请重试。",
	}).Error
}

func (p *gormSeriesPersistence) SaveSeriesIntro(ctx context.Context, parentID uuid.UUID, content string) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	return p.db.WithContext(ctx).Model(&model.Blog{}).Where("id = ?", parentID).Updates(map[string]any{
		"content": content,
		"status":  1,
	}).Error
}

func (p *gormSeriesPersistence) MarkSeriesIntroFailed(ctx context.Context, parentID uuid.UUID) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	return p.db.WithContext(ctx).Model(&model.Blog{}).Where("id = ?", parentID).Updates(map[string]any{
		"status": 2,
	}).Error
}

func (p *gormSeriesPersistence) LoadSeriesOldContent(ctx context.Context, blogID uuid.UUID) (string, error) {
	if p.db == nil {
		return "", fmt.Errorf("series persistence database is not initialized")
	}

	var blog model.Blog
	if err := p.db.WithContext(ctx).Select("content").First(&blog, "id = ?", blogID).Error; err != nil {
		return "", err
	}
	return blog.Content, nil
}

func (p *gormSeriesPersistence) UpdateSkippedSeriesChapterMeta(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chapter Chapter) error {
	if p.db == nil {
		return fmt.Errorf("series persistence database is not initialized")
	}

	return p.db.WithContext(ctx).Model(&model.Blog{}).Where("id = ? AND user_id = ?", blogID, userID).Updates(map[string]any{
		"chapter_sort": chapter.Sort,
		"title":        chapter.Title,
	}).Error
}

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

// SeriesPersistence 收口系列生成阶段仍归 core-api 持有的业务表写入。
type SeriesPersistence interface {
	SaveSeriesChapter(ctx context.Context, input SeriesChapterPersistenceInput) error
	MarkSeriesChapterFailed(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) error
	SaveSeriesIntro(ctx context.Context, parentID uuid.UUID, content string) error
	MarkSeriesIntroFailed(ctx context.Context, parentID uuid.UUID) error
}

type gormSeriesPersistence struct {
	db *gorm.DB
}

// NewGormSeriesPersistence creates the default GORM-backed series persistence adapter.
func NewGormSeriesPersistence(database *gorm.DB) SeriesPersistence {
	return &gormSeriesPersistence{db: database}
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

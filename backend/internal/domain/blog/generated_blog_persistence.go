package blog

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"inkwords-backend/internal/model"
	"inkwords-backend/internal/service"
)

// generatedBlogPersistence keeps the default generated blog write path close to
// the blog domain so later repository consolidation can happen in one place.
type generatedBlogPersistence struct {
	db *gorm.DB
}

// NewGeneratedBlogPersistence creates the default blog-domain adapter for
// GeneratorService final blog writes.
func NewGeneratedBlogPersistence(database *gorm.DB) service.GeneratedBlogPersistence {
	return &generatedBlogPersistence{db: database}
}

func (p *generatedBlogPersistence) SaveGeneratedBlog(ctx context.Context, input service.GeneratedBlogPersistenceInput) error {
	if p.db == nil {
		return fmt.Errorf("database not configured")
	}

	blog := &model.Blog{
		UserID:      input.UserID,
		Title:       input.Title,
		Content:     input.Content,
		SourceType:  input.SourceType,
		Status:      1,
		ChapterSort: 1,
		WordCount:   input.WordCount,
		TechStacks:  input.TechStacks,
	}

	estimatedTokens := len([]rune(input.Content)) * 2

	// Why: 生成正文和 token 记账属于同一业务事实，必须事务一致，避免生成成功但配额不同步。
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(blog).Error; err != nil {
			return fmt.Errorf("create blog record: %w", err)
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


package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/model"
)

// GeneratedBlogPersistence defines the explicit persistence boundary for
// GeneratorService when it needs to store the final generated blog outcome.
type GeneratedBlogPersistence interface {
	SaveGeneratedBlog(ctx context.Context, input GeneratedBlogPersistenceInput) error
}

// GeneratedBlogPersistenceInput contains the persisted business facts produced
// by GeneratorService after LLM generation has completed.
type GeneratedBlogPersistenceInput struct {
	UserID     uuid.UUID
	Title      string
	Content    string
	SourceType string
	WordCount  int
	TechStacks datatypes.JSON
}

type databaseGeneratedBlogPersistence struct {
	database *gorm.DB
}

func newDatabaseGeneratedBlogPersistence(database *gorm.DB) GeneratedBlogPersistence {
	return &databaseGeneratedBlogPersistence{database: database}
}

func (p *databaseGeneratedBlogPersistence) SaveGeneratedBlog(ctx context.Context, input GeneratedBlogPersistenceInput) error {
	databaseHandle := p.database
	if databaseHandle == nil {
		databaseHandle = db.DB
	}
	if databaseHandle == nil {
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

	// Why: blog creation and token accounting must commit together, otherwise a
	// partial write would leave the generated artifact and user quota out of sync.
	return databaseHandle.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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

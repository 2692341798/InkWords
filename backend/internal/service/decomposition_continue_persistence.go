package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

// ContinuePersistence 收口 continue 链路对博客正文的读取与最终更新。
type ContinuePersistence interface {
	LoadContinueBlog(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) (model.Blog, error)
	SaveContinuedBlog(ctx context.Context, blog model.Blog, updatedContent string) error
}

type gormContinuePersistence struct {
	db *gorm.DB
}

// NewGormContinuePersistence creates the default GORM-backed continue persistence adapter.
func NewGormContinuePersistence(database *gorm.DB) ContinuePersistence {
	return &gormContinuePersistence{db: database}
}

func (p *gormContinuePersistence) LoadContinueBlog(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) (model.Blog, error) {
	if p.db == nil {
		return model.Blog{}, fmt.Errorf("continue persistence database is not initialized")
	}

	var blog model.Blog
	if err := p.db.WithContext(ctx).First(&blog, "id = ? AND user_id = ?", blogID, userID).Error; err != nil {
		return model.Blog{}, err
	}
	return blog, nil
}

func (p *gormContinuePersistence) SaveContinuedBlog(ctx context.Context, blog model.Blog, updatedContent string) error {
	if p.db == nil {
		return fmt.Errorf("continue persistence database is not initialized")
	}

	return p.db.WithContext(ctx).Model(&blog).Update("content", updatedContent).Error
}

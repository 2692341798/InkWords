package blog

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
	"inkwords-backend/internal/service"
)

type continuePersistence struct {
	db *gorm.DB
}

// NewContinuePersistence creates the blog-domain adapter for continue reads
// and final content updates.
func NewContinuePersistence(database *gorm.DB) service.ContinuePersistence {
	return &continuePersistence{db: database}
}

func (p *continuePersistence) LoadContinueBlog(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) (model.Blog, error) {
	if p.db == nil {
		return model.Blog{}, fmt.Errorf("continue persistence database is not initialized")
	}

	var blog model.Blog
	if err := p.db.WithContext(ctx).First(&blog, "id = ? AND user_id = ?", blogID, userID).Error; err != nil {
		return model.Blog{}, err
	}
	return blog, nil
}

func (p *continuePersistence) SaveContinuedBlog(ctx context.Context, blog model.Blog, updatedContent string) error {
	if p.db == nil {
		return fmt.Errorf("continue persistence database is not initialized")
	}

	return p.db.WithContext(ctx).Model(&blog).Update("content", updatedContent).Error
}


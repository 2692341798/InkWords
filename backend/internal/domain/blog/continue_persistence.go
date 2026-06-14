package blog

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"inkwords-backend/internal/model"
)

type continuePersistence struct {
	db *gorm.DB
}

// NewContinuePersistence creates the blog-domain adapter for continue reads
// and final content updates.
func NewContinuePersistence(database *gorm.DB) blogcontracts.ContinuePersistence {
	return &continuePersistence{db: database}
}

func (p *continuePersistence) LoadContinueBlog(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) (blogcontracts.ContinueBlog, error) {
	if p.db == nil {
		return blogcontracts.ContinueBlog{}, fmt.Errorf("continue persistence database is not initialized")
	}

	var blog model.Blog
	if err := p.db.WithContext(ctx).First(&blog, "id = ? AND user_id = ?", blogID, userID).Error; err != nil {
		return blogcontracts.ContinueBlog{}, err
	}
	return blogcontracts.ContinueBlog{ID: blog.ID, UserID: blog.UserID, Content: blog.Content}, nil
}

func (p *continuePersistence) SaveContinuedBlog(ctx context.Context, blog blogcontracts.ContinueBlog, updatedContent string) error {
	if p.db == nil {
		return fmt.Errorf("continue persistence database is not initialized")
	}

	return p.db.WithContext(ctx).
		Model(&model.Blog{}).
		Where("id = ?", blog.ID).
		Update("content", updatedContent).Error
}

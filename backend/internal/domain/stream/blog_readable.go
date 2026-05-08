package stream

import (
	"context"

	"github.com/google/uuid"

	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/model"
)

type BlogReadable interface {
	Exists(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) error
}

type GormBlogReadable struct{}

func NewGormBlogReadable() *GormBlogReadable {
	return &GormBlogReadable{}
}

func (r *GormBlogReadable) Exists(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) error {
	var blog model.Blog
	if err := db.DB.WithContext(ctx).Select("id").First(&blog, "id = ? AND user_id = ?", blogID, userID).Error; err != nil {
		return err
	}
	return nil
}

var _ BlogReadable = (*GormBlogReadable)(nil)

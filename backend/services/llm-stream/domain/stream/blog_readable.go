package stream

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BlogReadable interface {
	Exists(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) error
}

type readableBlog struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;index"`
}

func (readableBlog) TableName() string {
	return "blogs"
}

type GormBlogReadable struct {
	db *gorm.DB
}

func NewGormBlogReadable(db *gorm.DB) *GormBlogReadable {
	return &GormBlogReadable{db: db}
}

func (r *GormBlogReadable) Exists(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) error {
	var blog readableBlog
	if err := r.db.WithContext(ctx).Select("id").First(&blog, "id = ? AND user_id = ?", blogID, userID).Error; err != nil {
		return err
	}
	return nil
}

var _ BlogReadable = (*GormBlogReadable)(nil)

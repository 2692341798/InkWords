package export

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	GetByID(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) (Blog, error)
	GetSeriesBlogs(ctx context.Context, userID uuid.UUID, parentID uuid.UUID) ([]Blog, error)
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) GetByID(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) (Blog, error) {
	var blog Blog
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", blogID, userID).First(&blog).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Blog{}, ErrBlogNotFound
		}
		return Blog{}, err
	}
	return blog, nil
}

func (r *GormRepository) GetSeriesBlogs(ctx context.Context, userID uuid.UUID, parentID uuid.UUID) ([]Blog, error) {
	var blogs []Blog

	var parent Blog
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", parentID, userID).First(&parent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSeriesNotFound
		}
		return nil, err
	}
	blogs = append(blogs, parent)

	var children []Blog
	if err := r.db.WithContext(ctx).
		Where("parent_id = ? AND user_id = ?", parentID, userID).
		Order("chapter_sort ASC").
		Find(&children).Error; err != nil {
		return nil, err
	}

	blogs = append(blogs, children...)
	return blogs, nil
}

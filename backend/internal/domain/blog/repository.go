package blog

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

// Repository 定义 Blog 领域的数据访问接口。
type Repository interface {
	ListTopLevelBlogs(ctx context.Context, userID uuid.UUID, page int, size int) ([]model.Blog, error)
	ListChildrenByParentIDs(ctx context.Context, userID uuid.UUID, parentIDs []uuid.UUID) ([]model.Blog, error)
	GetByID(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) (*model.Blog, error)
	GetSeriesBlogs(ctx context.Context, userID uuid.UUID, parentID uuid.UUID) ([]model.Blog, error)
	Create(ctx context.Context, blog *model.Blog) error
	Update(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, updates map[string]any) (rowsAffected int64, err error)
	BatchDelete(ctx context.Context, userID uuid.UUID, blogIDs []uuid.UUID) error
}

// GormRepository 使用 GORM 实现 BlogRepository。
type GormRepository struct {
	db *gorm.DB
}

// NewGormRepository 创建 GormRepository。
func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) ListTopLevelBlogs(ctx context.Context, userID uuid.UUID, page int, size int) ([]model.Blog, error) {
	var parents []model.Blog
	offset := (page - 1) * size
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND parent_id IS NULL", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(size).
		Find(&parents).Error
	if err != nil {
		return nil, err
	}
	return parents, nil
}

func (r *GormRepository) ListChildrenByParentIDs(ctx context.Context, userID uuid.UUID, parentIDs []uuid.UUID) ([]model.Blog, error) {
	var children []model.Blog
	if len(parentIDs) == 0 {
		return []model.Blog{}, nil
	}
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND parent_id IN ?", userID, parentIDs).
		Order("chapter_sort ASC").
		Find(&children).Error
	if err != nil {
		return nil, err
	}
	return children, nil
}

func (r *GormRepository) GetByID(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) (*model.Blog, error) {
	var blog model.Blog
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", blogID, userID).First(&blog).Error; err != nil {
		return nil, err
	}
	return &blog, nil
}

func (r *GormRepository) GetSeriesBlogs(ctx context.Context, userID uuid.UUID, parentID uuid.UUID) ([]model.Blog, error) {
	var blogs []model.Blog

	var parent model.Blog
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", parentID, userID).First(&parent).Error
	if err != nil {
		return nil, err
	}
	blogs = append(blogs, parent)

	var children []model.Blog
	err = r.db.WithContext(ctx).Where("parent_id = ? AND user_id = ?", parentID, userID).Order("chapter_sort ASC").Find(&children).Error
	if err != nil {
		return nil, err
	}

	blogs = append(blogs, children...)
	return blogs, nil
}

func (r *GormRepository) Create(ctx context.Context, blog *model.Blog) error {
	return r.db.WithContext(ctx).Create(blog).Error
}

func (r *GormRepository) Update(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, updates map[string]any) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.Blog{}).
		Where("id = ? AND user_id = ?", blogID, userID).
		Updates(updates)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

func (r *GormRepository) BatchDelete(ctx context.Context, userID uuid.UUID, blogIDs []uuid.UUID) error {
	if len(blogIDs) == 0 {
		return nil
	}
	res := r.db.WithContext(ctx).
		Where("user_id = ? AND (id IN ? OR parent_id IN ?)", userID, blogIDs, blogIDs).
		Delete(&model.Blog{})
	return res.Error
}


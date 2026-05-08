package auth

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

type Repository interface {
	CountByEmailOrUsername(ctx context.Context, email string, username string) (int64, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByGithubIDOrEmail(ctx context.Context, githubID string, email string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	Save(ctx context.Context, user *model.User) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) CountByEmailOrUsername(ctx context.Context, email string, username string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.User{}).Where("email = ? OR username = ?", email, username).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *GormRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormRepository) GetByGithubIDOrEmail(ctx context.Context, githubID string, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("github_id = ?", githubID).Or("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormRepository) Create(ctx context.Context, user *model.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *GormRepository) Save(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}


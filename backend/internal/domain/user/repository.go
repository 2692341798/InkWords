package user

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"inkwords-backend/internal/model"
)

type Repository interface {
	GetUserByID(ctx context.Context, uid uuid.UUID) (*model.User, error)
	UpdateUsername(ctx context.Context, uid uuid.UUID, username string) error
	UpdateAvatarURL(ctx context.Context, uid uuid.UUID, avatarURL string) error
	CountArticles(ctx context.Context, uid uuid.UUID) (int64, error)
	SumWords(ctx context.Context, uid uuid.UUID) (int64, error)
	ListBlogsWithTechStacks(ctx context.Context, uid uuid.UUID) ([]model.Blog, error)
	GetPromptSettings(ctx context.Context, uid uuid.UUID) (*model.UserPromptSettings, error)
	UpsertPromptSettings(ctx context.Context, uid uuid.UUID, overrides datatypes.JSON) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) GetUserByID(ctx context.Context, uid uuid.UUID) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", uid).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormRepository) UpdateUsername(ctx context.Context, uid uuid.UUID, username string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", uid).Update("username", username).Error
}

func (r *GormRepository) UpdateAvatarURL(ctx context.Context, uid uuid.UUID, avatarURL string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", uid).Update("avatar_url", avatarURL).Error
}

func (r *GormRepository) CountArticles(ctx context.Context, uid uuid.UUID) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.Blog{}).
		Where("user_id = ? AND status = 1 AND parent_id IS NOT NULL", uid).
		Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *GormRepository) SumWords(ctx context.Context, uid uuid.UUID) (int64, error) {
	type result struct {
		TotalWords int64
	}
	var res result
	if err := r.db.WithContext(ctx).
		Model(&model.Blog{}).
		Select("sum(word_count) as total_words").
		Where("user_id = ? AND status = 1 AND parent_id IS NOT NULL", uid).
		Scan(&res).Error; err != nil {
		return 0, err
	}
	return res.TotalWords, nil
}

func (r *GormRepository) ListBlogsWithTechStacks(ctx context.Context, uid uuid.UUID) ([]model.Blog, error) {
	var blogs []model.Blog
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = 1 AND parent_id IS NOT NULL AND tech_stacks IS NOT NULL", uid).
		Find(&blogs).Error; err != nil {
		return nil, err
	}
	return blogs, nil
}

func (r *GormRepository) GetPromptSettings(ctx context.Context, uid uuid.UUID) (*model.UserPromptSettings, error) {
	var row model.UserPromptSettings
	if err := r.db.WithContext(ctx).First(&row, "user_id = ?", uid).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GormRepository) UpsertPromptSettings(ctx context.Context, uid uuid.UUID, overrides datatypes.JSON) error {
	row := &model.UserPromptSettings{
		UserID:    uid,
		Overrides: overrides,
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"overrides", "updated_at"}),
		}).
		Create(row).Error
}

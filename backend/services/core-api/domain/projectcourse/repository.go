package projectcourse

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrNotFound           = errors.New("project course not found")
	ErrVersionConflict    = errors.New("project course blueprint version conflict")
	ErrBlueprintImmutable = errors.New("approved project course blueprint is immutable")
)

type Repository interface {
	Create(ctx context.Context, course *ProjectCourse) error
	GetByID(ctx context.Context, userID, courseID uuid.UUID) (*ProjectCourse, error)
	UpdateBlueprintCAS(ctx context.Context, userID, courseID uuid.UUID, update BlueprintUpdate) error
	Approve(ctx context.Context, userID, courseID uuid.UUID, expectedVersion int) error
}

type GormRepository struct{ db *gorm.DB }

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

func (r *GormRepository) Create(ctx context.Context, course *ProjectCourse) error {
	return r.db.WithContext(ctx).Create(course).Error
}

func (r *GormRepository) GetByID(ctx context.Context, userID, courseID uuid.UUID) (*ProjectCourse, error) {
	var course ProjectCourse
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", courseID, userID).First(&course).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &course, nil
}

func (r *GormRepository) UpdateBlueprintCAS(ctx context.Context, userID, courseID uuid.UUID, update BlueprintUpdate) error {
	current, err := r.GetByID(ctx, userID, courseID)
	if err != nil {
		return err
	}
	if current.Status == StatusApproved || current.Status == StatusGenerating || current.Status == StatusCompleted {
		return ErrBlueprintImmutable
	}
	result := r.db.WithContext(ctx).Model(&ProjectCourse{}).
		Where("id = ? AND user_id = ? AND blueprint_version = ? AND status NOT IN ?", courseID, userID, update.ExpectedVersion, []string{StatusApproved, StatusGenerating, StatusCompleted}).
		Updates(map[string]any{
			"blueprint_json":    datatypes.JSON(update.BlueprintJSON),
			"coverage_json":     datatypes.JSON(update.CoverageJSON),
			"blueprint_version": update.ExpectedVersion + 1,
			"updated_at":        gorm.Expr("CURRENT_TIMESTAMP"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrVersionConflict
	}
	return nil
}

func (r *GormRepository) Approve(ctx context.Context, userID, courseID uuid.UUID, expectedVersion int) error {
	result := r.db.WithContext(ctx).Model(&ProjectCourse{}).
		Where("id = ? AND user_id = ? AND blueprint_version = ? AND status = ?", courseID, userID, expectedVersion, StatusAwaitingApproval).
		Updates(map[string]any{"status": StatusApproved, "updated_at": gorm.Expr("CURRENT_TIMESTAMP")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrVersionConflict
	}
	return nil
}

var _ Repository = (*GormRepository)(nil)

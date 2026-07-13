package projectcourse

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
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

// PersistProjectCourseResult is the core-api write boundary used by the task
// result reconciler. The worker owns computation; this repository owns facts.
func (r *GormRepository) PersistProjectCourseResult(ctx context.Context, result map[string]any) error {
	var payload struct {
		CourseID  string                      `json:"course_id"`
		Status    string                      `json:"status"`
		Snapshot  sharedkernel.SourceSnapshot `json:"snapshot"`
		Blueprint sharedkernel.Blueprint      `json:"blueprint"`
		Coverage  sharedkernel.CoverageMatrix `json:"coverage"`
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return err
	}
	courseID, err := uuid.Parse(payload.CourseID)
	if err != nil {
		return ErrNotFound
	}
	if err := payload.Snapshot.Validate(); err != nil {
		return err
	}
	if err := payload.Blueprint.Validate(); err != nil {
		return err
	}
	blueprintJSON, err := json.Marshal(payload.Blueprint)
	if err != nil {
		return err
	}
	coverageJSON, err := json.Marshal(payload.Coverage)
	if err != nil {
		return err
	}
	dbResult := r.db.WithContext(ctx).Model(&ProjectCourse{}).
		Where("id = ? AND status = ?", courseID, StatusAnalyzing).
		Updates(map[string]any{
			"resolved_commit_sha": payload.Snapshot.ResolvedCommitSHA,
			"blueprint_json":      datatypes.JSON(blueprintJSON),
			"coverage_json":       datatypes.JSON(coverageJSON),
			"blueprint_version":   payload.Blueprint.BlueprintVersion,
			"status":              StatusAwaitingApproval,
			"updated_at":          gorm.Expr("CURRENT_TIMESTAMP"),
		})
	if dbResult.Error != nil {
		return dbResult.Error
	}
	if dbResult.RowsAffected == 0 {
		return ErrVersionConflict
	}
	return nil
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

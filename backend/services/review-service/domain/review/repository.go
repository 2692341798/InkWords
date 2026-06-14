package review

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository 定义 review 领域所需的持久化接口。
type Repository interface {
	GetRecentSessions(ctx context.Context, userID uuid.UUID, limit int) ([]ReviewSession, error)
	CreateSession(ctx context.Context, session *ReviewSession) error
	GetSessionByID(ctx context.Context, sessionID uuid.UUID) (ReviewSession, error)
	ListTurns(ctx context.Context, sessionID uuid.UUID) ([]ReviewTurn, error)
	AppendTurn(ctx context.Context, turn *ReviewTurn) error
	UpdateSession(ctx context.Context, session *ReviewSession) error
}

// GormRepository 使用 GORM 实现 review 领域仓储。
type GormRepository struct {
	db *gorm.DB
}

// NewGormRepository 创建 review 领域的 GORM 仓储。
func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) GetRecentSessions(ctx context.Context, userID uuid.UUID, limit int) ([]ReviewSession, error) {
	if limit <= 0 {
		limit = 20
	}

	var sessions []ReviewSession
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&sessions).Error
	if err != nil {
		return nil, err
	}

	return sessions, nil
}

func (r *GormRepository) CreateSession(ctx context.Context, session *ReviewSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *GormRepository) GetSessionByID(ctx context.Context, sessionID uuid.UUID) (ReviewSession, error) {
	var session ReviewSession
	err := r.db.WithContext(ctx).
		Where("id = ?", sessionID).
		First(&session).Error
	if err != nil {
		return ReviewSession{}, err
	}
	return session, nil
}

func (r *GormRepository) ListTurns(ctx context.Context, sessionID uuid.UUID) ([]ReviewTurn, error) {
	var turns []ReviewTurn
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("turn_index ASC").
		Find(&turns).Error
	if err != nil {
		return nil, err
	}
	return turns, nil
}

func (r *GormRepository) AppendTurn(ctx context.Context, turn *ReviewTurn) error {
	return r.db.WithContext(ctx).Create(turn).Error
}

func (r *GormRepository) UpdateSession(ctx context.Context, session *ReviewSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

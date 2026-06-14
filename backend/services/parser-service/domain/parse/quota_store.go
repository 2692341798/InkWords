package parse

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type quotaUser struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	TokensUsed int       `gorm:"type:integer;default:0"`
	TokenLimit int       `gorm:"type:integer;default:1000000000"`
}

func (quotaUser) TableName() string {
	return "users"
}

type GormQuotaChecker struct {
	db *gorm.DB
}

func NewGormQuotaChecker(db *gorm.DB) *GormQuotaChecker {
	return &GormQuotaChecker{db: db}
}

func (q *GormQuotaChecker) CheckQuota(userID uuid.UUID) error {
	var user quotaUser
	if err := q.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	limit := user.TokenLimit
	if limit == 0 {
		limit = 1000000000
	}
	if user.TokensUsed >= limit {
		return errors.New("您的 Token 额度已耗尽，请升级订阅或联系管理员")
	}
	return nil
}

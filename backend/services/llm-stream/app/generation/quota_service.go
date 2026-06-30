package generation

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// userTokenProjection 是 QuotaChecker 查询 users 表所需的最小投影。
type userTokenProjection struct {
	TokensUsed int `gorm:"column:tokens_used"`
	TokenLimit int `gorm:"column:token_limit"`
}

// QuotaService 实现 stream.QuotaChecker，基于 users 表检查 Token 额度。
type QuotaService struct {
	db *gorm.DB
}

// NewQuotaService 创建 QuotaChecker 实现。
func NewQuotaService(db *gorm.DB) *QuotaService {
	return &QuotaService{db: db}
}

// CheckQuota 检查用户的 Token 是否超额。
func (s *QuotaService) CheckQuota(uid uuid.UUID) error {
	var row userTokenProjection
	if err := s.db.Table("users").
		Select("tokens_used, token_limit").
		Where("id = ?", uid).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	limit := row.TokenLimit
	if limit == 0 {
		limit = 1000000000
	}

	if row.TokensUsed >= limit {
		return errors.New("您的 Token 额度已耗尽，请升级订阅或联系管理员")
	}

	return nil
}

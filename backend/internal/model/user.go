package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User 代表系统的核心访问实体
type User struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Username         string         `gorm:"type:varchar(255);not null" json:"username"`
	Email            string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash     string         `gorm:"type:varchar(255)" json:"-"`
	GithubID         *string        `gorm:"type:varchar(255)" json:"github_id"`
	WechatOpenID     *string        `gorm:"type:varchar(255)" json:"wechat_openid"`
	AvatarURL           string         `gorm:"type:varchar(1024)" json:"avatar_url"`
	SubscriptionTier    int16          `gorm:"type:smallint;default:0" json:"subscription_tier"`
	TokensUsed          int            `gorm:"type:integer;default:0" json:"tokens_used"`
	FailedLoginAttempts int            `gorm:"default:0" json:"-"`
	LockedUntil         *time.Time     `json:"-"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate 在插入数据库前自动生成 UUID
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

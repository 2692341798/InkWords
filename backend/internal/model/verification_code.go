package model

import (
	"time"

	"gorm.io/gorm"
)

// VerificationCode 验证码模型
type VerificationCode struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Email     string         `gorm:"index;not null;type:varchar(255)" json:"email"`
	Code      string         `gorm:"not null;type:varchar(20)" json:"code"`
	Type      string         `gorm:"not null;type:varchar(50)" json:"type"` // "register" or "reset_password"
	ExpiresAt time.Time      `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

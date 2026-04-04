package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OAuthToken 第三方授权表，用于管理用户在掘金、CSDN等平台的一键发文授权
type OAuthToken struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	PlatformType string         `gorm:"type:varchar(50);not null" json:"platform_type"`
	AccessToken  string         `gorm:"type:text;not null" json:"access_token"`
	RefreshToken string         `gorm:"type:text" json:"refresh_token"`
	ExpiresIn    int            `gorm:"type:integer" json:"expires_in"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate 在插入数据库前自动生成 UUID
func (o *OAuthToken) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

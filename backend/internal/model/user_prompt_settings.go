package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type UserPromptSettings struct {
	UserID    uuid.UUID      `gorm:"type:uuid;primaryKey" json:"user_id"`
	Overrides datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"overrides"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}


package blog

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Blog is core-api blog management's projection of the shared blogs table.
type Blog struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;index:idx_user_parent_chapter;not null" json:"user_id"`
	ParentID    *uuid.UUID     `gorm:"type:uuid;index:idx_user_parent_chapter" json:"parent_id"`
	ChapterSort int            `gorm:"type:integer;index:idx_user_parent_chapter" json:"chapter_sort"`
	Title       string         `gorm:"type:varchar(255);not null" json:"title"`
	Content     string         `gorm:"type:text;not null" json:"content"`
	SourceType  string         `gorm:"type:varchar(50);not null" json:"source_type"`
	SourceURL   string         `gorm:"type:varchar(512)" json:"source_url"`
	IsSeries    bool           `gorm:"type:boolean;default:false" json:"is_series"`
	Status      int16          `gorm:"type:smallint;default:0" json:"status"`
	WordCount   int            `gorm:"type:integer;default:0" json:"word_count"`
	TechStacks  datatypes.JSON `gorm:"type:jsonb" json:"tech_stacks"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Blog) TableName() string {
	return "blogs"
}

// BeforeCreate keeps UUID generation in the service model instead of relying
// on database-specific defaults.
func (b *Blog) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

type userTokenBalance struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	TokensUsed int       `gorm:"type:integer;default:0"`
}

func (userTokenBalance) TableName() string {
	return "users"
}

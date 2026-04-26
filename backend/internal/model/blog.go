package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Blog 核心业务表，存储生成的 Markdown 内容及大项目拆解结构
type Blog struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;index:idx_user_parent_chapter;not null" json:"user_id"`
	ParentID    *uuid.UUID     `gorm:"type:uuid;index:idx_user_parent_chapter" json:"parent_id"`
	ChapterSort int            `gorm:"type:integer;index:idx_user_parent_chapter" json:"chapter_sort"`
	Title       string         `gorm:"type:varchar(255);not null" json:"title"`
	Content     string         `gorm:"type:text;not null" json:"content"`
	SourceType  string         `gorm:"type:varchar(50);not null" json:"source_type"`
	IsSeries    bool           `gorm:"type:boolean;default:false" json:"is_series"`
	Status      int16          `gorm:"type:smallint;default:0" json:"status"`
	WordCount   int            `gorm:"type:integer;default:0" json:"word_count"`
	TechStacks  datatypes.JSON `gorm:"type:jsonb" json:"tech_stacks"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate 在插入数据库前自动生成 UUID
func (b *Blog) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

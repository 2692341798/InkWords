package blog

import (
	"time"

	"github.com/google/uuid"
)

// BlogNode 表示博客历史记录树节点（与对外 JSON 结构保持一致）。
type BlogNode struct {
	ID          uuid.UUID   `json:"id"`
	Title       string      `json:"title"`
	Content     string      `json:"content"`
	SourceType  string      `json:"source_type"`
	Status      int16       `json:"status"`
	ChapterSort int         `json:"chapter_sort"`
	ParentID    *uuid.UUID  `json:"parent_id"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Children    []*BlogNode `json:"children"`
}

// UpdateRequest 表示更新博客内容的请求体。
type UpdateRequest struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}

// BatchDeleteRequest 表示批量删除博客的请求体。
type BatchDeleteRequest struct {
	BlogIDs []uuid.UUID `json:"blog_ids" binding:"required"`
}


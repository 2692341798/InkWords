package api

import "github.com/google/uuid"

type BatchDeleteBlogsRequest struct {
	BlogIDs []uuid.UUID `json:"blog_ids" binding:"required"`
}


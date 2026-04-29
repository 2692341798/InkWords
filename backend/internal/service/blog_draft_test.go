package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

func TestBlogService_CreateDraftBlog_createsTopLevelDraft(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&model.Blog{}))

	s := NewBlogServiceWithDB(db)
	userID := uuid.New()

	blog, err := s.CreateDraftBlog(context.Background(), userID)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, blog.ID)
	require.Equal(t, userID, blog.UserID)
	require.Nil(t, blog.ParentID)
	require.Equal(t, 0, blog.ChapterSort)
	require.Equal(t, "未命名博客", blog.Title)
	require.Equal(t, "", blog.Content)
	require.Equal(t, "manual", blog.SourceType)
	require.False(t, blog.IsSeries)

	var persisted model.Blog
	require.NoError(t, db.First(&persisted, "id = ?", blog.ID).Error)
	require.Equal(t, blog.ID, persisted.ID)
	require.Equal(t, userID, persisted.UserID)
	require.Equal(t, "未命名博客", persisted.Title)
	require.Equal(t, "", persisted.Content)
	require.Equal(t, "manual", persisted.SourceType)
}


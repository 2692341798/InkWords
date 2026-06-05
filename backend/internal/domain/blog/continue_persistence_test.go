package blog

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

func TestContinuePersistence_LoadAndSaveContinueBlog(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))

	userID := uuid.New()
	blogID := uuid.New()
	require.NoError(t, testDB.Create(&model.User{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:         blogID,
		UserID:     userID,
		Title:      "旧标题",
		Content:    "旧内容",
		SourceType: "file",
		Status:     1,
	}).Error)

	persistence := NewContinuePersistence(testDB)
	blog, err := persistence.LoadContinueBlog(context.Background(), userID, blogID)
	require.NoError(t, err)
	require.Equal(t, "旧内容", blog.Content)

	require.NoError(t, persistence.SaveContinuedBlog(context.Background(), blog, "旧内容追加内容"))

	var updated model.Blog
	require.NoError(t, testDB.First(&updated, "id = ?", blogID).Error)
	require.Equal(t, "旧内容追加内容", updated.Content)
}


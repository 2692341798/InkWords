package blog

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"inkwords-backend/internal/model"
)

func TestGeneratedBlogPersistence_SaveGeneratedBlog_PersistsBlogAndUpdatesTokens(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))

	userID := uuid.New()
	require.NoError(t, testDB.Create(&model.User{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)

	persistence := NewGeneratedBlogPersistence(testDB)
	err = persistence.SaveGeneratedBlog(context.Background(), blogcontracts.GeneratedBlogPersistenceInput{
		UserID:     userID,
		Title:      "文件解析生成的博客",
		Content:    "hello",
		SourceType: "file",
		WordCount:  5,
		TechStacks: datatypes.JSON([]byte(`["Go"]`)),
	})
	require.NoError(t, err)

	var blog model.Blog
	require.NoError(t, testDB.First(&blog).Error)
	require.Equal(t, userID, blog.UserID)
	require.Equal(t, "文件解析生成的博客", blog.Title)
	require.Equal(t, "hello", blog.Content)
	require.Equal(t, "file", blog.SourceType)

	var user model.User
	require.NoError(t, testDB.First(&user, "id = ?", userID).Error)
	require.Equal(t, 10, user.TokensUsed)
}

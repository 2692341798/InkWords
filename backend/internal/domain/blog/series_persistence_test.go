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

func openSeriesPersistenceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))
	return testDB
}

func TestSeriesPersistence_EnsureSeriesParentAndDrafts_PreparesTree(t *testing.T) {
	testDB := openSeriesPersistenceTestDB(t)

	userID := uuid.New()
	parentID := uuid.New()
	obsoleteChildID := uuid.New()
	require.NoError(t, testDB.Create(&model.User{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:       parentID,
		UserID:   userID,
		Title:    "旧系列",
		Content:  "旧导读",
		IsSeries: true,
		Status:   0,
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:          obsoleteChildID,
		UserID:      userID,
		ParentID:    &parentID,
		ChapterSort: 99,
		Title:       "旧章节",
		Content:     "旧内容",
		Status:      1,
	}).Error)

	persistence := NewSeriesPersistence(testDB)
	outline, err := persistence.EnsureSeriesParentAndDrafts(context.Background(), blogcontracts.SeriesDraftPreflightInput{
		UserID:      userID,
		ParentID:    parentID,
		ParentTitle: "新系列",
		SourceType:  "file",
		GitURL:      "https://example.com/repo",
		Outline: []blogcontracts.Chapter{
			{Title: "第一章", Summary: "摘要", Sort: 1},
			{Title: "第二章", Summary: "摘要", Sort: 2},
		},
	})
	require.NoError(t, err)
	require.Len(t, outline, 2)
	require.NotEmpty(t, outline[0].ID)
	require.NotEmpty(t, outline[1].ID)

	var parent model.Blog
	require.NoError(t, testDB.First(&parent, "id = ?", parentID).Error)
	require.Equal(t, "https://example.com/repo", parent.SourceURL)

	var children []model.Blog
	require.NoError(t, testDB.Where("parent_id = ?", parentID).Order("chapter_sort ASC").Find(&children).Error)
	require.Len(t, children, 2)
	require.Equal(t, "第一章", children[0].Title)
	require.Equal(t, "第二章", children[1].Title)
}

func TestSeriesPersistence_SaveSeriesChapter_UpdatesBlogAndTokens(t *testing.T) {
	testDB := openSeriesPersistenceTestDB(t)

	userID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()
	require.NoError(t, testDB.Create(&model.User{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:       parentID,
		UserID:   userID,
		Title:    "系列父稿",
		Content:  "导读",
		IsSeries: true,
		Status:   1,
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:          childID,
		UserID:      userID,
		ParentID:    &parentID,
		ChapterSort: 1,
		Title:       "旧标题",
		Content:     "旧内容",
		SourceType:  "file",
		Status:      0,
	}).Error)

	persistence := NewSeriesPersistence(testDB)
	err := persistence.SaveSeriesChapter(context.Background(), blogcontracts.SeriesChapterPersistenceInput{
		UserID:     userID,
		ParentID:   parentID,
		BlogID:     childID,
		Chapter:    blogcontracts.Chapter{Title: "新标题", Sort: 1},
		SourceType: "file",
		Content:    "章节终稿",
		WordCount:  4,
		TechStacks: datatypes.JSON([]byte(`["Go"]`)),
	})
	require.NoError(t, err)

	var child model.Blog
	require.NoError(t, testDB.First(&child, "id = ?", childID).Error)
	require.Equal(t, "新标题", child.Title)
	require.Equal(t, "章节终稿", child.Content)
	require.EqualValues(t, 1, child.Status)

	var user model.User
	require.NoError(t, testDB.First(&user, "id = ?", userID).Error)
	require.Equal(t, len([]rune("章节终稿"))*2, user.TokensUsed)
}

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

func TestSeriesPersistence_EnsureSeriesParentAndDrafts_RejectsForeignParent(t *testing.T) {
	testDB := openSeriesPersistenceTestDB(t)

	ownerID := uuid.New()
	attackerID := uuid.New()
	parentID := uuid.New()
	require.NoError(t, testDB.Create(&model.User{
		ID:       ownerID,
		Username: "owner",
		Email:    "owner@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.User{
		ID:       attackerID,
		Username: "attacker",
		Email:    "attacker@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:       parentID,
		UserID:   ownerID,
		Title:    "他人的系列",
		Content:  "导读",
		IsSeries: true,
		Status:   1,
	}).Error)

	persistence := NewSeriesPersistence(testDB)
	outline, err := persistence.EnsureSeriesParentAndDrafts(context.Background(), blogcontracts.SeriesDraftPreflightInput{
		UserID:      attackerID,
		ParentID:    parentID,
		ParentTitle: "尝试劫持的系列",
		SourceType:  "file",
		Outline: []blogcontracts.Chapter{
			{Title: "第 1 章", Summary: "不应创建", Sort: 1},
		},
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "parent blog does not belong to user")
	require.Nil(t, outline)

	var childCount int64
	require.NoError(t, testDB.Model(&model.Blog{}).
		Where("parent_id = ? AND user_id = ?", parentID, attackerID).
		Count(&childCount).Error)
	require.Zero(t, childCount)
}

func TestSeriesPersistence_LoadSeriesOldContent_RejectsForeignBlog(t *testing.T) {
	testDB := openSeriesPersistenceTestDB(t)

	ownerID := uuid.New()
	attackerID := uuid.New()
	blogID := uuid.New()
	require.NoError(t, testDB.Create(&model.User{
		ID:       ownerID,
		Username: "owner",
		Email:    "owner@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.User{
		ID:       attackerID,
		Username: "attacker",
		Email:    "attacker@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:      blogID,
		UserID:  ownerID,
		Title:   "他人的旧章节",
		Content: "敏感旧正文",
		Status:  1,
	}).Error)

	persistence := NewSeriesPersistence(testDB)
	content, err := persistence.LoadSeriesOldContent(context.Background(), attackerID, blogID)
	require.Error(t, err)
	require.Empty(t, content)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestSeriesPersistence_SaveSeriesIntro_RejectsForeignParent(t *testing.T) {
	testDB := openSeriesPersistenceTestDB(t)

	ownerID := uuid.New()
	attackerID := uuid.New()
	parentID := uuid.New()
	require.NoError(t, testDB.Create(&model.User{
		ID:       ownerID,
		Username: "owner",
		Email:    "owner@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.User{
		ID:       attackerID,
		Username: "attacker",
		Email:    "attacker@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:       parentID,
		UserID:   ownerID,
		Title:    "他人的系列导读",
		Content:  "原始导读",
		IsSeries: true,
		Status:   0,
	}).Error)

	persistence := NewSeriesPersistence(testDB)
	err := persistence.SaveSeriesIntro(context.Background(), attackerID, parentID, "越权改写")
	require.Error(t, err)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var parent model.Blog
	require.NoError(t, testDB.First(&parent, "id = ?", parentID).Error)
	require.Equal(t, "原始导读", parent.Content)
	require.EqualValues(t, 0, parent.Status)
}

func TestSeriesPersistence_MarkSeriesIntroFailed_RejectsForeignParent(t *testing.T) {
	testDB := openSeriesPersistenceTestDB(t)

	ownerID := uuid.New()
	attackerID := uuid.New()
	parentID := uuid.New()
	require.NoError(t, testDB.Create(&model.User{
		ID:       ownerID,
		Username: "owner",
		Email:    "owner@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.User{
		ID:       attackerID,
		Username: "attacker",
		Email:    "attacker@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:       parentID,
		UserID:   ownerID,
		Title:    "他人的系列导读",
		Content:  "原始导读",
		IsSeries: true,
		Status:   1,
	}).Error)

	persistence := NewSeriesPersistence(testDB)
	err := persistence.MarkSeriesIntroFailed(context.Background(), attackerID, parentID)
	require.Error(t, err)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var parent model.Blog
	require.NoError(t, testDB.First(&parent, "id = ?", parentID).Error)
	require.EqualValues(t, 1, parent.Status)
}

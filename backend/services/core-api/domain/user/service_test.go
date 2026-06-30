package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"inkwords-backend/services/core-api/domain/user"
)

// =============================================================================
// FakeRepository – in-memory fake implementing user.Repository
// 所有方法均可通过注入错误钩子模拟 Repo 层失败场景
// =============================================================================

type FakeRepository struct {
	users          map[uuid.UUID]*user.User
	promptSettings map[uuid.UUID]*user.UserPromptSettings

	GetUserByIDErr            error
	UpdateUsernameErr         error
	UpdateAvatarURLErr        error
	CountArticlesErr          error
	SumWordsErr               error
	ListBlogsWithTechStacksErr error
	GetPromptSettingsErr      error
	UpsertPromptSettingsErr   error
}

func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		users:          make(map[uuid.UUID]*user.User),
		promptSettings: make(map[uuid.UUID]*user.UserPromptSettings),
	}
}

func (f *FakeRepository) GetUserByID(_ context.Context, uid uuid.UUID) (*user.User, error) {
	if f.GetUserByIDErr != nil {
		return nil, f.GetUserByIDErr
	}
	u, ok := f.users[uid]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return u, nil
}

func (f *FakeRepository) UpdateUsername(_ context.Context, uid uuid.UUID, username string) error {
	if f.UpdateUsernameErr != nil {
		return f.UpdateUsernameErr
	}
	if u, ok := f.users[uid]; ok {
		u.Username = username
	}
	return nil
}

func (f *FakeRepository) UpdateAvatarURL(_ context.Context, uid uuid.UUID, avatarURL string) error {
	if f.UpdateAvatarURLErr != nil {
		return f.UpdateAvatarURLErr
	}
	if u, ok := f.users[uid]; ok {
		u.AvatarURL = avatarURL
	}
	return nil
}

func (f *FakeRepository) CountArticles(_ context.Context, _ uuid.UUID) (int64, error) {
	if f.CountArticlesErr != nil {
		return 0, f.CountArticlesErr
	}
	return 5, nil
}

func (f *FakeRepository) SumWords(_ context.Context, _ uuid.UUID) (int64, error) {
	if f.SumWordsErr != nil {
		return 0, f.SumWordsErr
	}
	return 12000, nil
}

func (f *FakeRepository) ListBlogsWithTechStacks(_ context.Context, _ uuid.UUID) ([]user.Blog, error) {
	if f.ListBlogsWithTechStacksErr != nil {
		return nil, f.ListBlogsWithTechStacksErr
	}
	return []user.Blog{}, nil
}

func (f *FakeRepository) GetPromptSettings(_ context.Context, uid uuid.UUID) (*user.UserPromptSettings, error) {
	if f.GetPromptSettingsErr != nil {
		return nil, f.GetPromptSettingsErr
	}
	row, ok := f.promptSettings[uid]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return row, nil
}

func (f *FakeRepository) UpsertPromptSettings(_ context.Context, uid uuid.UUID, overrides datatypes.JSON) error {
	if f.UpsertPromptSettingsErr != nil {
		return f.UpsertPromptSettingsErr
	}
	f.promptSettings[uid] = &user.UserPromptSettings{
		UserID:    uid,
		Overrides: overrides,
	}
	return nil
}

// =============================================================================
// 辅助函数
// =============================================================================

func ptr[T any](v T) *T { return &v }

func makeUser(id uuid.UUID) *user.User {
	return &user.User{
		ID:               id,
		Username:         "testuser",
		Email:            "test@example.com",
		AvatarURL:        "/uploads/avatars/default.png",
		SubscriptionTier: 1,
		TokensUsed:       5000,
		TokenLimit:       10000,
		GithubID:         ptr("gh_123"),
	}
}

// =============================================================================
// 场景 1：Profile 读取 —— 验证读取用户配置（含不存在用户处理）
// =============================================================================

func TestGetProfile_ExistingUser(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()
	fake := NewFakeRepository()
	fake.users[uid] = makeUser(uid)

	svc := user.NewService(fake)
	profile, err := svc.GetProfile(ctx, uid)

	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "testuser", profile.Username)
	assert.Equal(t, "test@example.com", profile.Email)
	assert.Equal(t, "/uploads/avatars/default.png", profile.AvatarURL)
	assert.Equal(t, int16(1), profile.SubscriptionTier)
	assert.Equal(t, 5000, profile.TokensUsed)
	assert.Equal(t, 10000, profile.TokenLimit)
	// 有 GithubID 时 connectedPlatforms 应包含 "github"
	assert.Contains(t, profile.ConnectedPlatforms, "github")
}

func TestGetProfile_NonExistentUser(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()
	fake := NewFakeRepository()

	svc := user.NewService(fake)
	profile, err := svc.GetProfile(ctx, uid)

	assert.Error(t, err)
	assert.Nil(t, profile)
	assert.Contains(t, err.Error(), "user not found")
}

func TestGetProfile_ZeroTokenLimitDefaults(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()
	u := makeUser(uid)
	u.TokenLimit = 0
	fake := NewFakeRepository()
	fake.users[uid] = u

	svc := user.NewService(fake)
	profile, err := svc.GetProfile(ctx, uid)

	require.NoError(t, err)
	// TokenLimit 为 0 时应兜底为 1_000_000_000
	assert.Equal(t, 1000000000, profile.TokenLimit)
}

func TestGetProfile_EmptyConnectedPlatforms(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()
	u := makeUser(uid)
	u.GithubID = nil
	u.WechatOpenID = nil
	fake := NewFakeRepository()
	fake.users[uid] = u

	svc := user.NewService(fake)
	profile, err := svc.GetProfile(ctx, uid)

	require.NoError(t, err)
	assert.Empty(t, profile.ConnectedPlatforms)
}

// =============================================================================
// 场景 2：Profile 更新 —— 验证更新用户头像/用户名等字段
// =============================================================================

func TestUpdateUsername_Valid(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()
	fake := NewFakeRepository()
	fake.users[uid] = makeUser(uid)

	svc := user.NewService(fake)
	err := svc.UpdateUsername(ctx, uid, "newUsername")

	require.NoError(t, err)
	assert.Equal(t, "newUsername", fake.users[uid].Username)
}

func TestUpdateUsername_TooShort(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()

	svc := user.NewService(NewFakeRepository())
	err := svc.UpdateUsername(ctx, uid, "a")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "用户名长度必须在 2 到 20 个字符之间")
}

func TestUpdateUsername_TooLong(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()

	svc := user.NewService(NewFakeRepository())
	err := svc.UpdateUsername(ctx, uid, "thisUsernameIsWayTooLong123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "用户名长度必须在 2 到 20 个字符之间")
}

func TestUpdateAvatarURL_Success(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()
	fake := NewFakeRepository()
	fake.users[uid] = makeUser(uid)

	svc := user.NewService(fake)
	err := svc.UpdateAvatarURL(ctx, uid, "/uploads/avatars/new_avatar.png")

	require.NoError(t, err)
	assert.Equal(t, "/uploads/avatars/new_avatar.png", fake.users[uid].AvatarURL)
}

// =============================================================================
// 场景 4：Avatar 验证 —— 验证非法 avatar URL 的被处理情况
// 当前 Service 层未做 URL 校验，所有字符串均直接透传至 Repository
// =============================================================================

func TestUpdateAvatarURL_EmptyStringAccepted(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()
	fake := NewFakeRepository()
	fake.users[uid] = makeUser(uid)

	svc := user.NewService(fake)
	err := svc.UpdateAvatarURL(ctx, uid, "")

	// 当前行为：空字符串被原样传递给 Repository，无报错
	require.NoError(t, err)
	assert.Equal(t, "", fake.users[uid].AvatarURL)
}

func TestUpdateAvatarURL_RepoErrorPropagated(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()
	fake := NewFakeRepository()
	fake.users[uid] = makeUser(uid)
	fake.UpdateAvatarURLErr = errors.New("db write failure")

	svc := user.NewService(fake)
	err := svc.UpdateAvatarURL(ctx, uid, "/avatars/x.png")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db write failure")
}

// =============================================================================
// 场景 3：Prompt 设置合并失败 —— 当存储 prompt 配置失败时应返回错误
// =============================================================================

func TestUpdatePromptSettings_MergeAndUpsert(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()
	fake := NewFakeRepository()
	// 预先存入一条已有配置
	existingOverrides := datatypes.JSON(`{"temperature":"0.7"}`)
	fake.promptSettings[uid] = &user.UserPromptSettings{
		UserID:    uid,
		Overrides: existingOverrides,
	}

	svc := user.NewService(fake)
	patch := map[string]string{"max_tokens": "4096"}
	err := svc.UpdatePromptSettings(ctx, uid, patch)

	require.NoError(t, err)
	// 验证合并结果：既有旧值保留又有新值写入
	saved, ok := fake.promptSettings[uid]
	require.True(t, ok)
	assert.Contains(t, string(saved.Overrides), "temperature")
	assert.Contains(t, string(saved.Overrides), "max_tokens")
}

func TestUpdatePromptSettings_UpsertFails(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()
	fake := NewFakeRepository()
	// 先存一条已有配置，让 merge 阶段正常完成
	fake.promptSettings[uid] = &user.UserPromptSettings{
		UserID:    uid,
		Overrides: datatypes.JSON(`{"model":"deepseek"}`),
	}
	// 注入 Upsert 阶段的失败
	fake.UpsertPromptSettingsErr = errors.New("upsert db error")

	svc := user.NewService(fake)
	err := svc.UpdatePromptSettings(ctx, uid, map[string]string{"version": "v2"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "upsert db error")
}

func TestUpdatePromptSettings_GetPromptSettingsFails(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()
	fake := NewFakeRepository()
	// GetPromptSettings 返回非 RecordNotFound 的错误
	fake.GetPromptSettingsErr = errors.New("connection refused")

	svc := user.NewService(fake)
	err := svc.UpdatePromptSettings(ctx, uid, map[string]string{"key": "val"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestGetPromptSettings_NoExistingSettings(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()
	fake := NewFakeRepository()

	svc := user.NewService(fake)
	resp, err := svc.GetPromptSettings(ctx, uid)

	// 无已有配置时不应报错，Overrides 应为空 map
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Styles)
	assert.NotEmpty(t, resp.Defaults)
	assert.Empty(t, resp.Overrides)
}

func TestGetPromptSettings_RepoErrorNotRecordNotFound(t *testing.T) {
	ctx := context.Background()
	uid := uuid.New()
	fake := NewFakeRepository()
	fake.GetPromptSettingsErr = errors.New("timeout")

	svc := user.NewService(fake)
	resp, err := svc.GetPromptSettings(ctx, uid)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "timeout")
}

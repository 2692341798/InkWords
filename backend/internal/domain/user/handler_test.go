package user_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"inkwords-backend/internal/domain/user"
	"inkwords-backend/internal/model"
)

type stubRepo struct {
	user *model.User
}

func (s *stubRepo) GetUserByID(ctx context.Context, uid uuid.UUID) (*model.User, error) {
	if s.user == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return s.user, nil
}

func (s *stubRepo) UpdateUsername(ctx context.Context, uid uuid.UUID, username string) error {
	return nil
}

func (s *stubRepo) UpdateAvatarURL(ctx context.Context, uid uuid.UUID, avatarURL string) error {
	return nil
}

func (s *stubRepo) CountArticles(ctx context.Context, uid uuid.UUID) (int64, error) {
	return 0, nil
}

func (s *stubRepo) SumWords(ctx context.Context, uid uuid.UUID) (int64, error) {
	return 0, nil
}

func (s *stubRepo) ListBlogsWithTechStacks(ctx context.Context, uid uuid.UUID) ([]model.Blog, error) {
	return []model.Blog{}, nil
}

func TestHandler_GetProfile_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	svc := user.NewService(&stubRepo{})
	h := user.NewHandler(svc)

	r.GET("/profile", func(c *gin.Context) {
		h.GetProfile(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetProfile_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	uid := uuid.New()
	repo := &stubRepo{
		user: &model.User{
			ID:               uid,
			Username:         "u",
			Email:            "e@example.com",
			AvatarURL:        "/uploads/avatars/a.png",
			SubscriptionTier: 0,
			TokensUsed:       123,
			TokenLimit:       0,
		},
	}

	svc := user.NewService(repo)
	h := user.NewHandler(svc)

	r.GET("/profile", func(c *gin.Context) {
		c.Set("user_id", uid)
		h.GetProfile(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code    int              `json:"code"`
		Message string           `json:"message"`
		Data    user.ProfileData `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "success", resp.Message)
	require.Equal(t, "u", resp.Data.Username)
	require.Equal(t, 1000000000, resp.Data.TokenLimit)
}


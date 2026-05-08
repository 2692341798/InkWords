package blog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"inkwords-backend/internal/domain/blog"
	"inkwords-backend/internal/model"
)

type stubRepo struct {
	parents  []model.Blog
	children []model.Blog
}

func (s *stubRepo) ListTopLevelBlogs(ctx context.Context, userID uuid.UUID, page int, size int) ([]model.Blog, error) {
	return s.parents, nil
}

func (s *stubRepo) ListChildrenByParentIDs(ctx context.Context, userID uuid.UUID, parentIDs []uuid.UUID) ([]model.Blog, error) {
	return s.children, nil
}

func (s *stubRepo) GetByID(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) (*model.Blog, error) {
	return nil, nil
}

func (s *stubRepo) GetSeriesBlogs(ctx context.Context, userID uuid.UUID, parentID uuid.UUID) ([]model.Blog, error) {
	return nil, nil
}

func (s *stubRepo) Create(ctx context.Context, blog *model.Blog) error {
	if blog.ID == uuid.Nil {
		blog.ID = uuid.New()
	}
	return nil
}

func (s *stubRepo) Update(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, updates map[string]any) (int64, error) {
	return 1, nil
}

func (s *stubRepo) BatchDelete(ctx context.Context, userID uuid.UUID, blogIDs []uuid.UUID) error {
	return nil
}

func TestHandler_GetUserBlogs_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	svc := blog.NewService(&stubRepo{})
	h := blog.NewHandler(svc)

	r.GET("/blogs", func(c *gin.Context) {
		h.GetUserBlogs(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/blogs?page=1&size=20", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetUserBlogs_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	userID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	now := time.Now().UTC()
	repo := &stubRepo{
		parents: []model.Blog{
			{ID: parentID, UserID: userID, Title: "P", Content: "PC", SourceType: "manual", Status: 0, ChapterSort: 0, ParentID: nil, CreatedAt: now, UpdatedAt: now},
		},
		children: []model.Blog{
			{ID: childID, UserID: userID, Title: "C", Content: "CC", SourceType: "manual", Status: 0, ChapterSort: 1, ParentID: &parentID, CreatedAt: now, UpdatedAt: now},
		},
	}

	svc := blog.NewService(repo)
	h := blog.NewHandler(svc)

	r.GET("/blogs", func(c *gin.Context) {
		c.Set("user_id", userID)
		h.GetUserBlogs(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/blogs?page=1&size=20", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code    int            `json:"code"`
		Message string         `json:"message"`
		Data    []*blog.BlogNode `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "success", resp.Message)
	require.Len(t, resp.Data, 1)
	require.Equal(t, parentID, resp.Data[0].ID)
	require.Len(t, resp.Data[0].Children, 1)
	require.Equal(t, childID, resp.Data[0].Children[0].ID)
}

func TestHandler_CreateDraftBlog_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	svc := blog.NewService(&stubRepo{})
	h := blog.NewHandler(svc)

	r.POST("/blogs/draft", func(c *gin.Context) {
		h.CreateDraftBlog(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/blogs/draft", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CreateDraftBlog_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	userID := uuid.New()
	repo := &stubRepo{}
	svc := blog.NewService(repo)
	h := blog.NewHandler(svc)

	r.POST("/blogs/draft", func(c *gin.Context) {
		c.Set("user_id", userID)
		h.CreateDraftBlog(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/blogs/draft", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    blog.BlogNode `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "success", resp.Message)
	require.NotEqual(t, uuid.Nil, resp.Data.ID)
	require.Equal(t, "未命名博客", resp.Data.Title)
	require.Len(t, resp.Data.Children, 0)
}

func TestHandler_UpdateBlog_BadID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	userID := uuid.New()
	svc := blog.NewService(&stubRepo{})
	h := blog.NewHandler(svc)

	r.PUT("/blogs/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		h.UpdateBlog(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/blogs/not-a-uuid", bytes.NewBufferString(`{"title":"t"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateBlog_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	userID := uuid.New()
	blogID := uuid.New()
	svc := blog.NewService(&stubRepo{})
	h := blog.NewHandler(svc)

	r.PUT("/blogs/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		h.UpdateBlog(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/blogs/"+blogID.String(), bytes.NewBufferString(`{"title":"t"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "success", resp.Message)
}

func TestHandler_BatchDeleteBlogs_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	userID := uuid.New()
	svc := blog.NewService(&stubRepo{})
	h := blog.NewHandler(svc)

	r.DELETE("/blogs", func(c *gin.Context) {
		c.Set("user_id", userID)
		h.BatchDeleteBlogs(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/blogs", bytes.NewBufferString(`{"blog_ids":["`+uuid.New().String()+`"]}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

package blog

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRepository 是 Repository 接口的内存实现，用于特征化测试。
// 所有操作都在内存中同步完成，确保测试确定性且无外部依赖。
type fakeRepository struct {
	mu    sync.RWMutex
	blogs map[uuid.UUID]Blog
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		blogs: make(map[uuid.UUID]Blog),
	}
}

func (r *fakeRepository) seed(blogs ...Blog) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range blogs {
		r.blogs[b.ID] = b
	}
}

func (r *fakeRepository) ListTopLevelBlogs(ctx context.Context, userID uuid.UUID, page int, size int) ([]Blog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var parents []Blog
	for _, b := range r.blogs {
		if b.UserID == userID && b.ParentID == nil {
			parents = append(parents, b)
		}
	}

	// Why: 简单排序以保持确定性输出 — ID 字典序模拟 ORDER BY created_at DESC。
	parents = sortBlogsByIDSuffixDesc(parents)

	offset := (page - 1) * size
	if offset >= len(parents) {
		return []Blog{}, nil
	}
	end := offset + size
	if end > len(parents) {
		end = len(parents)
	}
	return parents[offset:end], nil
}

func (r *fakeRepository) ListChildrenByParentIDs(ctx context.Context, userID uuid.UUID, parentIDs []uuid.UUID) ([]Blog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(parentIDs) == 0 {
		return []Blog{}, nil
	}

	parentSet := make(map[uuid.UUID]bool, len(parentIDs))
	for _, pid := range parentIDs {
		parentSet[pid] = true
	}

	var children []Blog
	for _, b := range r.blogs {
		if b.UserID == userID && b.ParentID != nil && parentSet[*b.ParentID] {
			children = append(children, b)
		}
	}
	return sortBlogsByChapterSort(children), nil
}

func (r *fakeRepository) GetByID(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) (*Blog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	b, ok := r.blogs[blogID]
	if !ok || b.UserID != userID {
		return nil, ErrBlogNotFound
	}
	return &b, nil
}

func (r *fakeRepository) GetSeriesBlogs(ctx context.Context, userID uuid.UUID, parentID uuid.UUID) ([]Blog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var parent *Blog
	for _, b := range r.blogs {
		if b.ID == parentID {
			bCopy := b
			parent = &bCopy
			break
		}
	}
	if parent == nil || parent.UserID != userID {
		return nil, ErrBlogNotFound
	}

	result := []Blog{*parent}
	for _, b := range r.blogs {
		if b.ParentID != nil && *b.ParentID == parentID && b.UserID == userID {
			result = append(result, b)
		}
	}
	return sortBlogsByChapterSort(result), nil
}

func (r *fakeRepository) Create(ctx context.Context, blog *Blog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if blog.ID == uuid.Nil {
		blog.ID = uuid.New()
	}
	now := time.Now().UTC()
	if blog.CreatedAt.IsZero() {
		blog.CreatedAt = now
	}
	if blog.UpdatedAt.IsZero() {
		blog.UpdatedAt = now
	}

	clone := *blog
	r.blogs[clone.ID] = clone
	return nil
}

func (r *fakeRepository) Update(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, updates map[string]any) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.blogs[blogID]
	if !ok || b.UserID != userID {
		return 0, ErrBlogNotFound
	}

	applied := false
	for k, v := range updates {
		switch k {
		case "title":
			if title, ok := v.(string); ok && b.Title != title {
				b.Title = title
				applied = true
			}
		case "content":
			if content, ok := v.(string); ok && b.Content != content {
				b.Content = content
				applied = true
			}
		}
	}
	if applied {
		b.UpdatedAt = time.Now().UTC()
		r.blogs[blogID] = b
		return 1, nil
	}
	return 0, nil
}

func (r *fakeRepository) BatchDelete(ctx context.Context, userID uuid.UUID, blogIDs []uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(blogIDs) == 0 {
		return nil
	}

	idSet := make(map[uuid.UUID]bool, len(blogIDs))
	for _, id := range blogIDs {
		idSet[id] = true
	}

	for id, b := range r.blogs {
		if b.UserID != userID {
			continue
		}
		if idSet[id] || (b.ParentID != nil && idSet[*b.ParentID]) {
			delete(r.blogs, id)
		}
	}
	return nil
}

// ———— 辅助排序函数，保证测试确定性 ————

func sortBlogsByIDSuffixDesc(blogs []Blog) []Blog {
	for i := 0; i < len(blogs); i++ {
		for j := i + 1; j < len(blogs); j++ {
			if blogs[i].ID.String() < blogs[j].ID.String() {
				blogs[i], blogs[j] = blogs[j], blogs[i]
			}
		}
	}
	return blogs
}

func sortBlogsByChapterSort(blogs []Blog) []Blog {
	for i := 0; i < len(blogs); i++ {
		for j := i + 1; j < len(blogs); j++ {
			if blogs[i].ChapterSort > blogs[j].ChapterSort {
				blogs[i], blogs[j] = blogs[j], blogs[i]
			}
		}
	}
	return blogs
}

// ———— 测试用例 ————

func TestGetUserBlogs_OwnershipFiltering(t *testing.T) {
	userA := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userB := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	blogA1 := Blog{
		ID:          uuid.MustParse("a0000001-0000-0000-0000-000000000001"),
		UserID:      userA,
		Title:       "User A Blog 1",
		Content:     "Content A1",
		SourceType:  "manual",
		Status:      1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	blogA2 := Blog{
		ID:          uuid.MustParse("a0000002-0000-0000-0000-000000000002"),
		UserID:      userA,
		Title:       "User A Blog 2",
		Content:     "Content A2",
		SourceType:  "manual",
		Status:      1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	blogB1 := Blog{
		ID:          uuid.MustParse("b0000001-0000-0000-0000-000000000001"),
		UserID:      userB,
		Title:       "User B Blog 1",
		Content:     "Content B1",
		SourceType:  "manual",
		Status:      1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	repo := newFakeRepository()
	repo.seed(blogA1, blogA2, blogB1)
	svc := NewService(repo)

	ctx := context.Background()

	t.Run("只返回用户A的博客", func(t *testing.T) {
		results, err := svc.GetUserBlogs(ctx, userA, 1, 20)
		require.NoError(t, err)
		require.Len(t, results, 2)
		for _, node := range results {
			assert.Equal(t, userA, findUserID(repo, node.ID))
		}
	})

	t.Run("只返回用户B的博客", func(t *testing.T) {
		results, err := svc.GetUserBlogs(ctx, userB, 1, 20)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "User B Blog 1", results[0].Title)
	})

	t.Run("用户C无博客返回空列表", func(t *testing.T) {
		userC := uuid.New()
		results, err := svc.GetUserBlogs(ctx, userC, 1, 20)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestGetUserBlogs_Pagination(t *testing.T) {
	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	repo := newFakeRepository()

	for i := 1; i <= 5; i++ {
		idBytes := [16]byte{}
		idBytes[15] = byte(i)
		blog := Blog{
			ID:         uuid.UUID(idBytes),
			UserID:     userID,
			Title:      "Blog " + string(rune('0'+i)),
			Content:    "Content",
			SourceType: "manual",
			Status:     1,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
		repo.seed(blog)
	}

	svc := NewService(repo)
	ctx := context.Background()

	t.Run("第一页返回2条", func(t *testing.T) {
		results, err := svc.GetUserBlogs(ctx, userID, 1, 2)
		require.NoError(t, err)
		require.Len(t, results, 2)
	})

	t.Run("第二页返回2条", func(t *testing.T) {
		results, err := svc.GetUserBlogs(ctx, userID, 2, 2)
		require.NoError(t, err)
		require.Len(t, results, 2)
	})

	t.Run("第三页返回1条", func(t *testing.T) {
		results, err := svc.GetUserBlogs(ctx, userID, 3, 2)
		require.NoError(t, err)
		require.Len(t, results, 1)
	})

	t.Run("第四页返回空", func(t *testing.T) {
		results, err := svc.GetUserBlogs(ctx, userID, 4, 2)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestGetUserBlogs_TreeStructure(t *testing.T) {
	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	parent := Blog{
		ID:          uuid.MustParse("10000000-0000-0000-0000-000000000001"),
		UserID:      userID,
		Title:       "Series Parent",
		Content:     "Parent Content",
		SourceType:  "git",
		Status:      1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	parentID := parent.ID
	child1 := Blog{
		ID:          uuid.MustParse("20000000-0000-0000-0000-000000000001"),
		UserID:      userID,
		ParentID:    &parentID,
		Title:       "Chapter 1",
		Content:     "Chapter 1 Content",
		SourceType:  "git",
		Status:      1,
		ChapterSort: 1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	child2 := Blog{
		ID:          uuid.MustParse("20000000-0000-0000-0000-000000000002"),
		UserID:      userID,
		ParentID:    &parentID,
		Title:       "Chapter 2",
		Content:     "Chapter 2 Content",
		SourceType:  "git",
		Status:      1,
		ChapterSort: 2,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	repo := newFakeRepository()
	repo.seed(parent, child1, child2)
	svc := NewService(repo)

	ctx := context.Background()
	results, err := svc.GetUserBlogs(ctx, userID, 1, 20)
	require.NoError(t, err)
	require.Len(t, results, 1)

	root := results[0]
	assert.Equal(t, "Series Parent", root.Title)
	require.Len(t, root.Children, 2)
	assert.Equal(t, "Chapter 1", root.Children[0].Title)
	assert.Equal(t, "Chapter 2", root.Children[1].Title)
}

func TestCreateDraftBlog_Success(t *testing.T) {
	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	repo := newFakeRepository()
	svc := NewService(repo)

	ctx := context.Background()
	draft, err := svc.CreateDraftBlog(ctx, userID)
	require.NoError(t, err)

	t.Run("草稿具有默认属性", func(t *testing.T) {
		assert.Equal(t, userID, draft.UserID)
		assert.Nil(t, draft.ParentID)
		assert.Equal(t, "未命名博客", draft.Title)
		assert.Empty(t, draft.Content)
		assert.Equal(t, "manual", draft.SourceType)
		assert.False(t, draft.IsSeries)
		assert.EqualValues(t, 0, draft.Status)
		assert.Equal(t, 0, draft.WordCount)
		assert.Equal(t, 0, draft.ChapterSort)
		assert.NotEqual(t, uuid.Nil, draft.ID)
	})

	t.Run("草稿已在仓库中持久化", func(t *testing.T) {
		results, err := svc.GetUserBlogs(ctx, userID, 1, 20)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, draft.ID, results[0].ID)
		assert.Equal(t, "未命名博客", results[0].Title)
	})
}

func TestCreateDraftBlog_MultipleUsersIndependent(t *testing.T) {
	userA := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userB := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	repo := newFakeRepository()
	svc := NewService(repo)

	ctx := context.Background()

	draftA, err := svc.CreateDraftBlog(ctx, userA)
	require.NoError(t, err)

	draftB, err := svc.CreateDraftBlog(ctx, userB)
	require.NoError(t, err)

	assert.NotEqual(t, draftA.ID, draftB.ID)
	assert.Equal(t, userA, draftA.UserID)
	assert.Equal(t, userB, draftB.UserID)

	resultsA, err := svc.GetUserBlogs(ctx, userA, 1, 20)
	require.NoError(t, err)
	assert.Len(t, resultsA, 1)

	resultsB, err := svc.GetUserBlogs(ctx, userB, 1, 20)
	require.NoError(t, err)
	assert.Len(t, resultsB, 1)
}

func TestUpdateBlog_Success(t *testing.T) {
	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	blogID := uuid.MustParse("e0000000-0000-0000-0000-000000000001")

	blog := Blog{
		ID:         blogID,
		UserID:     userID,
		Title:      "原始标题",
		Content:    "原始内容",
		SourceType: "manual",
		Status:     1,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	repo := newFakeRepository()
	repo.seed(blog)
	svc := NewService(repo)

	ctx := context.Background()

	newTitle := "新标题"
	newContent := "新内容"
	req := UpdateRequest{
		Title:   &newTitle,
		Content: &newContent,
	}

	err := svc.UpdateBlog(ctx, blogID, userID, req)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, userID, blogID)
	require.NoError(t, err)
	assert.Equal(t, "新标题", retrieved.Title)
	assert.Equal(t, "新内容", retrieved.Content)
}

func TestUpdateBlog_NotFound(t *testing.T) {
	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	nonExistentID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

	repo := newFakeRepository()
	svc := NewService(repo)

	ctx := context.Background()

	newTitle := "新标题"
	req := UpdateRequest{Title: &newTitle}

	err := svc.UpdateBlog(ctx, nonExistentID, userID, req)
	assert.True(t, errors.Is(err, ErrBlogNotFound), "更新不存在的博客应返回 ErrBlogNotFound")
}

func TestUpdateBlog_WrongUser(t *testing.T) {
	userA := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userB := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	blogID := uuid.MustParse("e0000000-0000-0000-0000-000000000001")

	blog := Blog{
		ID:         blogID,
		UserID:     userA,
		Title:      "User A 的博客",
		Content:    "Content",
		SourceType: "manual",
		Status:     1,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	repo := newFakeRepository()
	repo.seed(blog)
	svc := NewService(repo)

	ctx := context.Background()

	newTitle := "尝试篡改"
	req := UpdateRequest{Title: &newTitle}

	err := svc.UpdateBlog(ctx, blogID, userB, req)
	assert.True(t, errors.Is(err, ErrBlogNotFound), "其他用户尝试更新应返回 ErrBlogNotFound")

	retrieved, err := repo.GetByID(ctx, userA, blogID)
	require.NoError(t, err)
	assert.Equal(t, "User A 的博客", retrieved.Title, "原博客内容不应被修改")
}

func TestUpdateBlog_EmptyUpdates(t *testing.T) {
	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	blogID := uuid.MustParse("e0000000-0000-0000-0000-000000000001")

	blog := Blog{
		ID:         blogID,
		UserID:     userID,
		Title:      "原始标题",
		Content:    "原始内容",
		SourceType: "manual",
		Status:     1,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	repo := newFakeRepository()
	repo.seed(blog)
	svc := NewService(repo)

	ctx := context.Background()

	req := UpdateRequest{}
	err := svc.UpdateBlog(ctx, blogID, userID, req)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, userID, blogID)
	require.NoError(t, err)
	assert.Equal(t, "原始标题", retrieved.Title)
}

func TestBatchDeleteBlogs_Success(t *testing.T) {
	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	blog1 := Blog{
		ID:          uuid.MustParse("d0000000-0000-0000-0000-000000000001"),
		UserID:      userID,
		Title:       "待删除博客1",
		Content:     "Content 1",
		SourceType:  "manual",
		Status:      1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	blog2 := Blog{
		ID:          uuid.MustParse("d0000000-0000-0000-0000-000000000002"),
		UserID:      userID,
		Title:       "待删除博客2",
		Content:     "Content 2",
		SourceType:  "manual",
		Status:      1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	blog3 := Blog{
		ID:          uuid.MustParse("d0000000-0000-0000-0000-000000000003"),
		UserID:      userID,
		Title:       "保留博客",
		Content:     "Content 3",
		SourceType:  "manual",
		Status:      1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	repo := newFakeRepository()
	repo.seed(blog1, blog2, blog3)
	svc := NewService(repo)

	ctx := context.Background()

	err := svc.BatchDeleteBlogs(ctx, userID, []uuid.UUID{blog1.ID, blog2.ID})
	require.NoError(t, err)

	t.Run("已删除的博客不可见", func(t *testing.T) {
		results, err := svc.GetUserBlogs(ctx, userID, 1, 20)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, blog3.ID, results[0].ID)
	})

	t.Run("其他用户的博客不受影响", func(t *testing.T) {
		_, err := repo.GetByID(ctx, userID, blog1.ID)
		assert.True(t, errors.Is(err, ErrBlogNotFound))
	})
}

func TestBatchDeleteBlogs_OwnershipEnforced(t *testing.T) {
	userA := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userB := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	blogB := Blog{
		ID:          uuid.MustParse("b0000000-0000-0000-0000-000000000001"),
		UserID:      userB,
		Title:       "User B 的博客",
		Content:     "Content B",
		SourceType:  "manual",
		Status:      1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	blogA := Blog{
		ID:          uuid.MustParse("a0000000-0000-0000-0000-000000000001"),
		UserID:      userA,
		Title:       "User A 的博客",
		Content:     "Content A",
		SourceType:  "manual",
		Status:      1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	repo := newFakeRepository()
	repo.seed(blogA, blogB)
	svc := NewService(repo)

	ctx := context.Background()

	err := svc.BatchDeleteBlogs(ctx, userA, []uuid.UUID{blogB.ID})
	require.NoError(t, err)

	t.Run("用户A删除用户B的博客时，B的博客未被删除", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, userB, blogB.ID)
		require.NoError(t, err)
		assert.Equal(t, "User B 的博客", retrieved.Title)
	})

	t.Run("用户A自己的博客仍然存在", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, userA, blogA.ID)
		require.NoError(t, err)
		assert.Equal(t, "User A 的博客", retrieved.Title)
	})
}

func TestBatchDeleteBlogs_CascadingChildren(t *testing.T) {
	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	parentID := uuid.MustParse("10000000-0000-0000-0000-000000000001")

	parent := Blog{
		ID:          parentID,
		UserID:      userID,
		Title:       "系列父博客",
		Content:     "Parent",
		SourceType:  "git",
		Status:      1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	child1 := Blog{
		ID:          uuid.MustParse("20000000-0000-0000-0000-000000000001"),
		UserID:      userID,
		ParentID:    &parentID,
		Title:       "子章节1",
		Content:     "Child 1",
		SourceType:  "git",
		Status:      1,
		ChapterSort: 1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	child2 := Blog{
		ID:          uuid.MustParse("20000000-0000-0000-0000-000000000002"),
		UserID:      userID,
		ParentID:    &parentID,
		Title:       "子章节2",
		Content:     "Child 2",
		SourceType:  "git",
		Status:      1,
		ChapterSort: 2,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	repo := newFakeRepository()
	repo.seed(parent, child1, child2)
	svc := NewService(repo)

	ctx := context.Background()

	err := svc.BatchDeleteBlogs(ctx, userID, []uuid.UUID{parent.ID})
	require.NoError(t, err)

	t.Run("父博客已删除", func(t *testing.T) {
		_, err := repo.GetByID(ctx, userID, parent.ID)
		assert.True(t, errors.Is(err, ErrBlogNotFound))
	})

	t.Run("子博客级联删除", func(t *testing.T) {
		_, err := repo.GetByID(ctx, userID, child1.ID)
		assert.True(t, errors.Is(err, ErrBlogNotFound))
		_, err = repo.GetByID(ctx, userID, child2.ID)
		assert.True(t, errors.Is(err, ErrBlogNotFound))
	})

	t.Run("列表为空", func(t *testing.T) {
		results, err := svc.GetUserBlogs(ctx, userID, 1, 20)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestBatchDeleteBlogs_EmptyIDs(t *testing.T) {
	userID := uuid.New()
	repo := newFakeRepository()
	svc := NewService(repo)

	ctx := context.Background()
	err := svc.BatchDeleteBlogs(ctx, userID, []uuid.UUID{})
	require.NoError(t, err)
}

func TestGetSeriesBlogs_Success(t *testing.T) {
	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	parentID := uuid.MustParse("10000000-0000-0000-0000-000000000001")

	parent := Blog{
		ID:          parentID,
		UserID:      userID,
		Title:       "系列导读",
		Content:     "Intro",
		SourceType:  "git",
		Status:      1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	child := Blog{
		ID:          uuid.MustParse("20000000-0000-0000-0000-000000000001"),
		UserID:      userID,
		ParentID:    &parentID,
		Title:       "第一章",
		Content:     "Chapter 1",
		SourceType:  "git",
		Status:      1,
		ChapterSort: 1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	repo := newFakeRepository()
	repo.seed(parent, child)
	svc := NewService(repo)

	ctx := context.Background()
	series, err := svc.GetSeriesBlogs(ctx, parentID, userID)
	require.NoError(t, err)
	require.Len(t, series, 2)
	assert.Equal(t, "系列导读", series[0].Title)
	assert.Equal(t, "第一章", series[1].Title)
}

func TestGetSeriesBlogs_NotFound(t *testing.T) {
	userID := uuid.New()
	nonExistentID := uuid.New()

	repo := newFakeRepository()
	svc := NewService(repo)

	ctx := context.Background()
	_, err := svc.GetSeriesBlogs(ctx, nonExistentID, userID)
	assert.True(t, errors.Is(err, ErrBlogNotFound))
}

// findUserID 通过 blog ID 从仓库查找对应的 UserID，仅用于测试断言。
func findUserID(repo *fakeRepository, blogID uuid.UUID) uuid.UUID {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	if b, ok := repo.blogs[blogID]; ok {
		return b.UserID
	}
	return uuid.Nil
}

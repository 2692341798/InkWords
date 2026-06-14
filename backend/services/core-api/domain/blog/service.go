package blog

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ErrBlogNotFound indicates that the requested blog does not exist for the user.
var ErrBlogNotFound = errors.New("blog not found")

// Service 提供 Blog 领域的业务能力。
type Service struct {
	repo Repository
}

// NewService 创建 BlogService。
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetUserBlogs 获取用户博客列表，并组织成树状结构（与历史接口保持一致）。
func (s *Service) GetUserBlogs(ctx context.Context, userID uuid.UUID, page int, size int) ([]*BlogNode, error) {
	parents, err := s.repo.ListTopLevelBlogs(ctx, userID, page, size)
	if err != nil {
		return nil, err
	}

	if len(parents) == 0 {
		return []*BlogNode{}, nil
	}

	parentIDs := make([]uuid.UUID, 0, len(parents))
	for _, p := range parents {
		parentIDs = append(parentIDs, p.ID)
	}

	children, err := s.repo.ListChildrenByParentIDs(ctx, userID, parentIDs)
	if err != nil {
		return nil, err
	}

	childrenMap := make(map[uuid.UUID][]*BlogNode)
	for _, c := range children {
		cNode := &BlogNode{
			ID:          c.ID,
			Title:       c.Title,
			Content:     c.Content,
			SourceType:  c.SourceType,
			Status:      c.Status,
			ChapterSort: c.ChapterSort,
			ParentID:    c.ParentID,
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
			Children:    []*BlogNode{},
		}
		if c.ParentID != nil {
			childrenMap[*c.ParentID] = append(childrenMap[*c.ParentID], cNode)
		}
	}

	result := make([]*BlogNode, 0, len(parents))
	for _, p := range parents {
		pNode := &BlogNode{
			ID:          p.ID,
			Title:       p.Title,
			Content:     p.Content,
			SourceType:  p.SourceType,
			Status:      p.Status,
			ChapterSort: p.ChapterSort,
			ParentID:    p.ParentID,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
			Children:    childrenMap[p.ID],
		}
		if pNode.Children == nil {
			pNode.Children = []*BlogNode{}
		}
		result = append(result, pNode)
	}

	return result, nil
}

// GetSeriesBlogs 获取系列博客（父节点及所有子节点）。
func (s *Service) GetSeriesBlogs(ctx context.Context, parentID uuid.UUID, userID uuid.UUID) ([]Blog, error) {
	return s.repo.GetSeriesBlogs(ctx, userID, parentID)
}

// BatchDeleteBlogs 批量删除博客及其子节点。
func (s *Service) BatchDeleteBlogs(ctx context.Context, userID uuid.UUID, blogIDs []uuid.UUID) error {
	return s.repo.BatchDelete(ctx, userID, blogIDs)
}

// UpdateBlog 更新博客内容。
func (s *Service) UpdateBlog(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) error {
	updates := map[string]any{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if len(updates) == 0 {
		return nil
	}

	rowsAffected, err := s.repo.Update(ctx, userID, id, updates)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrBlogNotFound
	}
	return nil
}

// CreateDraftBlog 创建手写草稿博客。
func (s *Service) CreateDraftBlog(ctx context.Context, userID uuid.UUID) (Blog, error) {
	blog := Blog{
		UserID:      userID,
		ParentID:    nil,
		ChapterSort: 0,
		Title:       "未命名博客",
		Content:     "",
		SourceType:  "manual",
		SourceURL:   "",
		IsSeries:    false,
		Status:      0,
		WordCount:   0,
		TechStacks:  datatypes.JSON([]byte("[]")),
	}

	if err := s.repo.Create(ctx, &blog); err != nil {
		return Blog{}, err
	}
	return blog, nil
}

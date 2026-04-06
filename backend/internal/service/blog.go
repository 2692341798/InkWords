package service

import (
"context"
"errors"
"time"

"github.com/google/uuid"
"gorm.io/gorm"

"inkwords-backend/internal/db"
"inkwords-backend/internal/model"
)

// BlogNode 博客历史记录树节点
type BlogNode struct {
ID          uuid.UUID   `json:"id"`
Title       string      `json:"title"`
Content     string      `json:"content"`
SourceType  string      `json:"source_type"`
Status      int16       `json:"status"`
ChapterSort int         `json:"chapter_sort"`
ParentID    *uuid.UUID  `json:"parent_id"`
CreatedAt   time.Time   `json:"created_at"`
UpdatedAt   time.Time   `json:"updated_at"`
Children    []*BlogNode `json:"children"`
}

// BlogService 博客业务逻辑处理
type BlogService struct {
db *gorm.DB
}

// NewBlogService 创建博客服务实例
func NewBlogService() *BlogService {
return &BlogService{
db: db.DB,
}
}

// GetUserBlogs 获取用户的博客列表，并组织成树状结构
func (s *BlogService) GetUserBlogs(ctx context.Context, userID uuid.UUID, page, size int) ([]*BlogNode, error) {
var parents []model.Blog
offset := (page - 1) * size

// 查询顶级博客 (parent_id is null)
err := s.db.WithContext(ctx).
Where("user_id = ? AND parent_id IS NULL", userID).
Order("created_at DESC").
Offset(offset).
Limit(size).
Find(&parents).Error
if err != nil {
return nil, err
}

if len(parents) == 0 {
return []*BlogNode{}, nil
}

// 收集所有的 parent ID
parentIDs := make([]uuid.UUID, 0, len(parents))
for _, p := range parents {
parentIDs = append(parentIDs, p.ID)
}

// 查出这些父节点下的所有子节点
var children []model.Blog
err = s.db.WithContext(ctx).
Where("user_id = ? AND parent_id IN ?", userID, parentIDs).
Order("chapter_sort ASC").
Find(&children).Error
if err != nil {
return nil, err
}

// 组织成树状结构
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

var result []*BlogNode
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

// GetSeriesBlogs 获取系列博客（父节点及所有子节点）
func (s *BlogService) GetSeriesBlogs(ctx context.Context, parentID uuid.UUID, userID uuid.UUID) ([]model.Blog, error) {
	var blogs []model.Blog
	
	var parent model.Blog
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", parentID, userID).First(&parent).Error
	if err != nil {
		return nil, err
	}
	
	blogs = append(blogs, parent)
	
	var children []model.Blog
	err = s.db.WithContext(ctx).Where("parent_id = ? AND user_id = ?", parentID, userID).Order("chapter_sort ASC").Find(&children).Error
	if err != nil {
		return nil, err
	}
	
	blogs = append(blogs, children...)
	return blogs, nil
}

// BatchDeleteBlogs 批量删除博客及其子节点
func (s *BlogService) BatchDeleteBlogs(ctx context.Context, userID uuid.UUID, blogIDs []uuid.UUID) error {
	if len(blogIDs) == 0 {
		return nil
	}

	// 删除选中的博客，或者其父节点在选中列表中的博客
	res := s.db.WithContext(ctx).
		Where("user_id = ? AND (id IN ? OR parent_id IN ?)", userID, blogIDs, blogIDs).
		Delete(&model.Blog{})

	return res.Error
}

// UpdateBlogRequest 更新博客内容的请求体
type UpdateBlogRequest struct {
Title   *string `json:"title"`
Content *string `json:"content"`
}

// UpdateBlog 更新博客内容
func (s *BlogService) UpdateBlog(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateBlogRequest) error {
	updates := map[string]interface{}{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}

// 如果没有更新内容则直接返回
if len(updates) == 0 {
return nil
}

// 执行更新
res := s.db.WithContext(ctx).Model(&model.Blog{}).
Where("id = ? AND user_id = ?", id, userID).
Updates(updates)

if res.Error != nil {
return res.Error
}
if res.RowsAffected == 0 {
return errors.New("blog not found or no permission")
}

return nil
}

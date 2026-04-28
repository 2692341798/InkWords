package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// ExportSeriesToObsidian 批量导出整个系列博客到 Obsidian 挂载目录
func (s *BlogService) ExportSeriesToObsidian(ctx context.Context, parentID uuid.UUID, userID uuid.UUID) error {
	blogs, err := s.GetSeriesBlogs(ctx, parentID, userID)
	if err != nil {
		return err
	}
	if len(blogs) == 0 {
		return errors.New("系列博客为空")
	}

	parentTitle := blogs[0].Title
	if parentTitle == "" {
		parentTitle = "未命名系列"
	}

	obsidianPath := os.Getenv("OBSIDIAN_VAULT_PATH_INTERNAL")
	if obsidianPath == "" {
		obsidianPath = "/app/obsidian"
	}

	// 为该系列创建一个专属文件夹（概念子目录）
	seriesDir := filepath.Join(obsidianPath, parentTitle)
	if err := os.MkdirAll(seriesDir, 0755); err != nil {
		return fmt.Errorf("无法创建系列目录: %v", err)
	}

	now := time.Now().Format("2006-01-02")
	nowTimeStr := time.Now().Format("2006-01-02 15:04:05")

	var childTitles []string

	// 写入所有子博客
	for i := 1; i < len(blogs); i++ {
		child := blogs[i]
		childTitle := child.Title
		if childTitle == "" {
			childTitle = fmt.Sprintf("未命名子博客-%d", i)
		}
		childTitles = append(childTitles, childTitle)

		frontmatter := fmt.Sprintf(`---
type: concept
title: "%s"
created: %s
updated: %s
tags:
  - "#domain/tech"
status: seed
related:
  - "[[%s]]"
---
`, childTitle, now, now, parentTitle)

		content := fmt.Sprintf("%s\n# %s\n\n%s", frontmatter, childTitle, child.Content)
		filePath := filepath.Join(seriesDir, fmt.Sprintf("%s.md", childTitle))
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("写入子博客失败: %v", err)
		}
	}

	// 写入父博客 (Series Overview)
	relatedStr := ""
	if len(childTitles) > 0 {
		relatedStr = "related:\n"
		for _, title := range childTitles {
			relatedStr += fmt.Sprintf("  - \"[[%s]]\"\n", title)
		}
	}

	parentFrontmatter := fmt.Sprintf(`---
type: concept
title: "%s"
created: %s
updated: %s
tags:
  - "#domain/tech"
status: seed
%s---
`, parentTitle, now, now, strings.TrimSpace(relatedStr)+"\n")

	parentContent := fmt.Sprintf("%s\n# %s\n\n%s", parentFrontmatter, parentTitle, blogs[0].Content)
	parentFilePath := filepath.Join(seriesDir, fmt.Sprintf("%s.md", parentTitle))
	if err := os.WriteFile(parentFilePath, []byte(parentContent), 0644); err != nil {
		return fmt.Errorf("写入父博客失败: %v", err)
	}

	// ----------------- 更新 Obsidian 全局状态文件 -----------------
	// 1. 更新 index.md
	indexPath := filepath.Join(obsidianPath, "index.md")
	if _, err := os.Stat(indexPath); err == nil {
		// 如果存在，追加到某个地方（这里简化为追加到末尾或如果有特定结构则追加）
		indexFile, err := os.OpenFile(indexPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			indexFile.WriteString(fmt.Sprintf("\n- [[%s]]", parentTitle))
			indexFile.Close()
		}
	}

	// 2. 更新 log.md
	logPath := filepath.Join(obsidianPath, "log.md")
	if logContent, err := os.ReadFile(logPath); err == nil {
		// 在 "---" 分隔符或特定标题下追加
		lines := strings.Split(string(logContent), "\n")
		var newLines []string
		inserted := false
		for _, line := range lines {
			newLines = append(newLines, line)
			if !inserted && strings.HasPrefix(line, "---") && len(newLines) > 10 { // 找到正文中的第一个 ---
				newLines = append(newLines, fmt.Sprintf("- **%s**: Ingest 系列博客: [[%s]]，共包含 %d 篇子博客。", nowTimeStr, parentTitle, len(childTitles)))
				inserted = true
			}
		}
		if !inserted {
			newLines = append(newLines, fmt.Sprintf("- **%s**: Ingest 系列博客: [[%s]]，共包含 %d 篇子博客。", nowTimeStr, parentTitle, len(childTitles)))
		}
		os.WriteFile(logPath, []byte(strings.Join(newLines, "\n")), 0644)
	}

	// 3. 覆盖更新 hot.md
	hotPath := filepath.Join(obsidianPath, "hot.md")
	hotContent := fmt.Sprintf(`---
type: meta
title: "🔥 热点上下文 (Hot)"
created: %s
updated: %s
tags:
  - "#meta/hot"
status: mature
---

# 🔥 当前热点上下文

最近摄入的知识：
- [[%s]]
`, now, now, parentTitle)
	for _, title := range childTitles {
		hotContent += fmt.Sprintf("- [[%s]]\n", title)
	}
	os.WriteFile(hotPath, []byte(hotContent), 0644)

	return nil
}
func (s *BlogService) ExportToObsidian(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) error {
	var blog model.Blog
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", blogID, userID).First(&blog).Error
	if err != nil {
		return err
	}

	title := blog.Title
	if title == "" {
		title = "未命名博客"
	}

	// 构建 Frontmatter
	now := time.Now().Format("2006-01-02")
	frontmatter := fmt.Sprintf(`---
type: concept
title: "%s"
created: %s
updated: %s
tags:
  - "#domain/tech"
status: seed
---
`, title, now, now)

	content := fmt.Sprintf("%s\n# %s\n\n%s", frontmatter, title, blog.Content)

	// 写入文件
	obsidianPath := os.Getenv("OBSIDIAN_VAULT_PATH_INTERNAL")
	if obsidianPath == "" {
		obsidianPath = "/app/obsidian"
	}

	// 确保目录存在
	if err := os.MkdirAll(obsidianPath, 0755); err != nil {
		return fmt.Errorf("无法创建 Obsidian 目录: %v", err)
	}

	fileName := fmt.Sprintf("%s.md", title)
	filePath := filepath.Join(obsidianPath, fileName)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入 Obsidian 失败: %v", err)
	}

	return nil
}

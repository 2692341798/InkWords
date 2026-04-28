# Export to Obsidian Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Export generated blogs to a local Obsidian Vault by directly writing Markdown files with YAML Frontmatter via a mounted Docker volume.

**Architecture:** We will add a new `POST /api/v1/blogs/:id/export/obsidian` endpoint in the Go backend. It fetches the blog from the database, prepends standard Obsidian LLM Wiki YAML Frontmatter, and writes the file to the path specified by the `OBSIDIAN_VAULT_PATH` environment variable (defaulting to `/app/obsidian`). The frontend will get a new "Export to Obsidian" button in the blog UI.

**Tech Stack:** Go, Gin, GORM, React, Tailwind, Shadcn UI.

---

### Task 1: Configure Environment and Docker

**Files:**
- Modify: `/Users/huangqijun/Documents/墨言博客助手/InkWords/.env.example`
- Modify: `/Users/huangqijun/Documents/墨言博客助手/InkWords/docker-compose.yml`

- [x] **Step 1: Add environment variable to `.env.example`**

```env
# 导出到 Obsidian 的本地挂载路径（宿主机路径）
OBSIDIAN_VAULT_PATH=./obsidian_vault
```

- [x] **Step 2: Update `docker-compose.yml` to mount the volume**

Find the `backend` service and add the volume mount for Obsidian.

```yaml
    volumes:
      - ./.env:/app/.env
      - ${OBSIDIAN_VAULT_PATH:-./obsidian_vault}:/app/obsidian
```

- [x] **Step 3: Commit changes**

```bash
git add .env.example docker-compose.yml
git commit -m "feat: add obsidian vault volume mount to docker-compose"
```

### Task 2: Implement Backend Service Logic

**Files:**
- Modify: `/Users/huangqijun/Documents/墨言博客助手/InkWords/backend/internal/service/blog.go`

- [x] **Step 1: Add `ExportToObsidian` method to `BlogService`**

```go
// ExportToObsidian 导出博客到 Obsidian 挂载目录
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
```
*Note: Make sure to add `os` and `path/filepath`, `fmt` to the imports if not present.*

- [x] **Step 2: Commit changes**

```bash
git add backend/internal/service/blog.go
git commit -m "feat: add ExportToObsidian service logic"
```

### Task 3: Implement Backend API Endpoint

**Files:**
- Modify: `/Users/huangqijun/Documents/墨言博客助手/InkWords/backend/internal/api/blog.go`
- Modify: `/Users/huangqijun/Documents/墨言博客助手/InkWords/backend/cmd/server/main.go`

- [x] **Step 1: Add `ExportToObsidian` handler to `BlogAPI`**

```go
// ExportToObsidian 导出博客到 Obsidian
func (a *BlogAPI) ExportToObsidian(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    http.StatusUnauthorized,
			"message": "未授权的访问",
			"data":    nil,
		})
		return
	}

	uid, ok := userIDStr.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "用户 ID 类型错误",
			"data":    nil,
		})
		return
	}

	blogIDStr := c.Param("id")
	blogID, err := uuid.Parse(blogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "无效的博客 ID",
			"data":    nil,
		})
		return
	}

	if err := a.blogService.ExportToObsidian(c.Request.Context(), blogID, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data":    nil,
	})
}
```

- [x] **Step 2: Register the route in `main.go`**

In `backend/cmd/server/main.go`, find the blog routes block and add the new POST route:

```go
			blogs := apiV1.Group("/blogs")
			{
				blogs.GET("", blogAPI.GetUserBlogs)
				blogs.DELETE("", blogAPI.BatchDeleteBlogs)
				blogs.PUT("/:id", blogAPI.UpdateBlog)
				blogs.GET("/:id/export", blogAPI.ExportSeries)
				blogs.POST("/:id/export/obsidian", blogAPI.ExportToObsidian)
			}
```

- [x] **Step 3: Commit changes**

```bash
git add backend/internal/api/blog.go backend/cmd/server/main.go
git commit -m "feat: add ExportToObsidian API route"
```

### Task 4: Frontend UI Integration

**Files:**
- Modify: `/Users/huangqijun/Documents/墨言博客助手/InkWords/frontend/src/components/Editor.tsx`

- [ ] **Step 1: Add export API call function to `Editor.tsx`**

Inside the `Editor` component, add:

```typescript
  const handleExportToObsidian = async () => {
    if (!blogId) return;
    try {
      const response = await fetch(`/api/v1/blogs/${blogId}/export/obsidian`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      const data = await response.json();
      if (data.code === 200) {
        alert('成功导出到 Obsidian 仓库！');
      } else {
        alert(data.message || '导出失败');
      }
    } catch (error) {
      console.error('Export error:', error);
      alert('导出发生错误');
    }
  };
```

- [ ] **Step 2: Add "Export to Obsidian" button to the UI**

In `Editor.tsx`, locate the top action bar (where "保存", "返回" etc. might be) and add the new button.

```tsx
          <Button 
            variant="outline" 
            size="sm"
            onClick={handleExportToObsidian}
          >
            <Download className="h-4 w-4 mr-2" />
            导出到 Obsidian
          </Button>
```
*(Make sure to import the appropriate icon from `lucide-react` if used, like `Download` or `FileText`)*

- [ ] **Step 3: Commit changes**

```bash
git add frontend/src/components/Editor.tsx
git commit -m "feat: add Export to Obsidian button in Editor"
```

### Task 5: Verify and Document

**Files:**
- Modify: `/Users/huangqijun/Documents/墨言博客助手/InkWords/.trae/documents/InkWords_API.md`
- Modify: `/Users/huangqijun/Documents/墨言博客助手/InkWords/.trae/documents/InkWords_Development_Plan_and_Log.md`

- [ ] **Step 1: Document the new API**

Add the `POST /api/v1/blogs/:id/export/obsidian` documentation to `InkWords_API.md`.

- [ ] **Step 2: Update Development Log**

Record the completion of the "Export to Obsidian" feature in `InkWords_Development_Plan_and_Log.md`.

- [ ] **Step 3: Commit changes**

```bash
git add .trae/documents/
git commit -m "docs: update API and dev log for Obsidian export feature"
```

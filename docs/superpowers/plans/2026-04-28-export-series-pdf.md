# Export Series PDF Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在侧边栏批量模式中支持对系列父节点导出“合并 PDF”（封面 + 目录 + 正文），勾选多个系列时逐个触发多次下载。

**Architecture:** 后端将系列 Markdown 渲染为 HTML（封面/目录/章节分隔分页），使用容器内 Chromium headless 打印为 PDF 并以附件形式返回；前端在 Sidebar 批量操作栏新增「导出 PDF」按钮，按顺序调用导出接口并逐个触发下载，失败不中断。

**Tech Stack:** Go + Gin + GORM，goldmark（Markdown→HTML），Chromium headless（HTML→PDF），React + Tailwind + sonner toast。

---

## Files Overview

**Backend**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/api/blog.go`
- Create: `backend/internal/service/pdf_export.go`
- Create: `backend/internal/service/pdf_export_test.go`
- Modify: `backend/Dockerfile`

**Frontend**
- Modify: `frontend/src/components/Sidebar.tsx`

**Docs**
- Reference: `docs/superpowers/specs/2026-04-28-export-series-pdf-design.md`

---

### Task 1: Backend - Add goldmark dependency

**Files:**
- Modify: `backend/go.mod`

- [ ] **Step 1: Add dependency**

Add to `require` block:

```go
github.com/yuin/goldmark v1.7.13
```

- [ ] **Step 2: Run go mod tidy**

Run:

```bash
cd backend && go mod tidy
```

Expected: `go.sum` updated, no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/go.mod backend/go.sum
git commit -m "chore(backend): add goldmark for markdown to html rendering"
```

---

### Task 2: Backend - Implement PDF export service (TDD)

**Files:**
- Create: `backend/internal/service/pdf_export.go`
- Create: `backend/internal/service/pdf_export_test.go`

- [ ] **Step 1: Write failing unit test for HTML generation**

```go
package service

import (
	"strings"
	"testing"
	"time"

	"inkwords-backend/internal/model"
)

func TestBuildSeriesPDFHTML(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	blogs := []model.Blog{
		{Title: "系列导读", Content: "# 导读\n\nhello", ChapterSort: 0},
		{Title: "第一章", Content: "## A\n\n内容", ChapterSort: 1},
		{Title: "第二章", Content: "内容2", ChapterSort: 2},
	}

	html, filename, err := buildSeriesPDFHTML("我的系列", now, blogs)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if filename == "" || !strings.Contains(filename, ".pdf") {
		t.Fatalf("expected pdf filename, got %q", filename)
	}
	if !strings.Contains(html, "我的系列") {
		t.Fatalf("expected cover title in html")
	}
	if !strings.Contains(html, "目录") {
		t.Fatalf("expected toc in html")
	}
	if !strings.Contains(html, "第一章") || !strings.Contains(html, "第二章") {
		t.Fatalf("expected chapters in toc/content")
	}
	if !strings.Contains(html, "page-break-before") {
		t.Fatalf("expected page break CSS")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd backend && go test ./internal/service -run TestBuildSeriesPDFHTML -v
```

Expected: FAIL (`buildSeriesPDFHTML` undefined).

- [ ] **Step 3: Implement minimal HTML builder**

```go
package service

import (
	"bytes"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"

	"inkwords-backend/internal/model"
)

var seriesPDFMarkdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

func buildSeriesPDFHTML(seriesTitle string, now time.Time, blogs []model.Blog) (string, string, error) {
	safeTitle := strings.TrimSpace(seriesTitle)
	if safeTitle == "" {
		safeTitle = "未命名系列"
	}

	filename := fmt.Sprintf("%s.pdf", sanitizeObsidianFileName(safeTitle))

	type chapter struct {
		Title string
		HTML  string
		ID    string
	}

	chapters := make([]chapter, 0, len(blogs))
	for i, b := range blogs {
		title := strings.TrimSpace(b.Title)
		if title == "" {
			title = fmt.Sprintf("未命名_%d", i)
		}
		anchor := fmt.Sprintf("ch-%d", i)

		var buf bytes.Buffer
		if err := seriesPDFMarkdown.Convert([]byte(b.Content), &buf); err != nil {
			return "", "", err
		}
		chapters = append(chapters, chapter{Title: title, HTML: buf.String(), ID: anchor})
	}

	dateStr := now.Format("2006-01-02 15:04")

	var out strings.Builder
	out.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"/>")
	out.WriteString("<style>")
	out.WriteString("@page{size:A4;margin:18mm 16mm;}")
	out.WriteString("body{font-family:\"Noto Sans CJK SC\", \"Noto Sans\", \"PingFang SC\", \"Microsoft YaHei\", sans-serif;color:#111;}")
	out.WriteString("h1,h2,h3{margin:0 0 8px 0;} p{line-height:1.7;} pre{white-space:pre-wrap;word-break:break-word;}")
	out.WriteString(".page{page-break-after:always;}")
	out.WriteString(".page:last-child{page-break-after:auto;}")
	out.WriteString(".cover{display:flex;flex-direction:column;justify-content:center;min-height:80vh;}")
	out.WriteString(".meta{color:#666;font-size:12px;margin-top:12px;}")
	out.WriteString(".toc a{text-decoration:none;color:#111;}")
	out.WriteString(".toc li{margin:6px 0;}")
	out.WriteString(".chapter{page-break-before:always;}")
	out.WriteString("</style></head><body>")

	out.WriteString("<section class=\"page cover\">")
	out.WriteString("<h1>")
	out.WriteString(html.EscapeString(safeTitle))
	out.WriteString("</h1>")
	out.WriteString("<div class=\"meta\">导出时间：")
	out.WriteString(html.EscapeString(dateStr))
	out.WriteString("</div>")
	out.WriteString("</section>")

	out.WriteString("<section class=\"page toc\">")
	out.WriteString("<h2>目录</h2><ol>")
	for _, ch := range chapters {
		out.WriteString("<li><a href=\"#")
		out.WriteString(html.EscapeString(ch.ID))
		out.WriteString("\">")
		out.WriteString(html.EscapeString(ch.Title))
		out.WriteString("</a></li>")
	}
	out.WriteString("</ol></section>")

	for _, ch := range chapters {
		out.WriteString("<section class=\"chapter\" id=\"")
		out.WriteString(html.EscapeString(ch.ID))
		out.WriteString("\">")
		out.WriteString("<h2>")
		out.WriteString(html.EscapeString(ch.Title))
		out.WriteString("</h2>")
		out.WriteString(ch.HTML)
		out.WriteString("</section>")
	}

	out.WriteString("</body></html>")
	return out.String(), filename, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd backend && go test ./internal/service -run TestBuildSeriesPDFHTML -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/pdf_export.go backend/internal/service/pdf_export_test.go
git commit -m "feat(backend): build html template for series pdf export"
```

---

### Task 3: Backend - Execute Chromium to render PDF (TDD)

**Files:**
- Modify: `backend/internal/service/pdf_export.go`
- Modify: `backend/internal/service/pdf_export_test.go`

- [ ] **Step 1: Add failing test for renderer command builder**

```go
package service

import (
	"strings"
	"testing"
)

func TestBuildChromiumArgs(t *testing.T) {
	htmlPath := "/tmp/a.html"
	pdfPath := "/tmp/a.pdf"
	args := buildChromiumArgs(htmlPath, pdfPath)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--headless") {
		t.Fatalf("expected headless arg")
	}
	if !strings.Contains(joined, "--print-to-pdf=") {
		t.Fatalf("expected print-to-pdf arg")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd backend && go test ./internal/service -run TestBuildChromiumArgs -v
```

Expected: FAIL (`buildChromiumArgs` undefined).

- [ ] **Step 3: Implement command builder + renderer**

Append to `pdf_export.go`:

```go
package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"inkwords-backend/internal/model"
)

func buildChromiumArgs(htmlPath, pdfPath string) []string {
	return []string{
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--print-to-pdf=" + pdfPath,
		htmlPath,
	}
}

func (s *BlogService) ExportSeriesToPDF(ctx context.Context, parentID uuid.UUID, userID uuid.UUID) (string, string, error) {
	blogs, err := s.GetSeriesBlogs(ctx, parentID, userID)
	if err != nil {
		return "", "", err
	}
	if len(blogs) == 0 {
		return "", "", fmt.Errorf("找不到该系列博客")
	}

	seriesTitle := blogs[0].Title
	htmlContent, filename, err := buildSeriesPDFHTML(seriesTitle, time.Now(), blogs)
	if err != nil {
		return "", "", err
	}

	tmpDir := os.TempDir()
	htmlFile, err := os.CreateTemp(tmpDir, "inkwords-series-*.html")
	if err != nil {
		return "", "", err
	}
	defer func() {
		_ = os.Remove(htmlFile.Name())
	}()
	if _, err := htmlFile.WriteString(htmlContent); err != nil {
		_ = htmlFile.Close()
		return "", "", err
	}
	_ = htmlFile.Close()

	pdfPath := filepath.Join(tmpDir, fmt.Sprintf("inkwords-series-%s.pdf", uuid.NewString()))

	chromiumPath := os.Getenv("CHROMIUM_BIN")
	if chromiumPath == "" {
		chromiumPath = "chromium"
	}

	cmd := exec.CommandContext(ctx, chromiumPath, buildChromiumArgs(htmlFile.Name(), pdfPath)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(pdfPath)
		return "", "", fmt.Errorf("生成 PDF 失败")
	} else {
		_ = out
	}

	return pdfPath, filename, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd backend && go test ./internal/service -run TestBuildChromiumArgs -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/pdf_export.go backend/internal/service/pdf_export_test.go
git commit -m "feat(backend): add chromium renderer for series pdf export"
```

---

### Task 4: Backend - Add API handler and route for PDF export

**Files:**
- Modify: `backend/internal/api/blog.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add handler method**

Add to `BlogAPI` in `backend/internal/api/blog.go`:

```go
// ExportSeriesPDF 导出系列博客为合并 PDF
func (a *BlogAPI) ExportSeriesPDF(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "未授权的访问", "data": nil})
		return
	}
	uid, ok := userIDStr.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "用户 ID 类型错误", "data": nil})
		return
	}

	blogID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "无效的博客 ID", "data": nil})
		return
	}

	pdfPath, filename, err := a.blogService.ExportSeriesToPDF(c.Request.Context(), blogID, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": err.Error(), "data": nil})
		return
	}
	defer func() { _ = os.Remove(pdfPath) }()

	c.FileAttachment(pdfPath, filename)
}
```

- [ ] **Step 2: Wire route**

Add in `backend/cmd/server/main.go` under blogGroup:

```go
blogGroup.GET("/:id/export/pdf", blogAPI.ExportSeriesPDF)
```

- [ ] **Step 3: Build & run unit tests**

Run:

```bash
cd backend && go test ./... -v
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/api/blog.go backend/cmd/server/main.go
git commit -m "feat(backend): add series pdf export api"
```

---

### Task 5: Backend - Update Dockerfile runtime stage to include Chromium + fonts

**Files:**
- Modify: `backend/Dockerfile`

- [ ] **Step 1: Add packages**

Update runtime stage `apk add` to include:

```dockerfile
chromium
nss
freetype
harfbuzz
ttf-freefont
font-noto-cjk
```

- [ ] **Step 2: Rebuild via docker compose**

Run:

```bash
docker compose down && docker compose up -d --build
```

Expected: backend container starts successfully.

- [ ] **Step 3: Commit**

```bash
git add backend/Dockerfile
git commit -m "chore(backend): install chromium and cjk fonts for pdf export"
```

---

### Task 6: Frontend - Add batch export PDF button and logic (Sidebar)

**Files:**
- Modify: `frontend/src/components/Sidebar.tsx`

- [ ] **Step 1: Add state for pdf exporting**

Add state:

```ts
const [isExportingPDF, setIsExportingPDF] = useState(false)
```

- [ ] **Step 2: Implement handler**

Add function:

```ts
const handleBatchExportSeriesPDF = async () => {
  if (selectedSeriesRoots.length === 0) {
    toast.error('请先选择一个系列父节点')
    return
  }

  const token = localStorage.getItem('token')
  setIsExportingPDF(true)

  try {
    toast.loading(`正在导出 PDF：0/${selectedSeriesRoots.length}`, { id: 'export-series-pdf' })

    let done = 0
    for (const series of selectedSeriesRoots) {
      try {
        const res = await fetch(`/api/v1/blogs/${series.id}/export/pdf`, {
          headers: {
            ...(token ? { 'Authorization': `Bearer ${token}` } : {})
          }
        })

        if (!res.ok) {
          const data = await res.json().catch(() => null)
          throw new Error(data?.message || '导出失败')
        }

        const blob = await res.blob()
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `${series.title || 'series'}.pdf`
        document.body.appendChild(a)
        a.click()
        URL.revokeObjectURL(url)
        document.body.removeChild(a)
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : '导出失败'
        toast.error(`《${series.title || '未命名系列'}》导出失败：${message}`)
      } finally {
        done += 1
        toast.loading(`正在导出 PDF：${done}/${selectedSeriesRoots.length}`, { id: 'export-series-pdf' })
      }
    }

    toast.success(`已开始下载 ${selectedSeriesRoots.length} 份 PDF`, { id: 'export-series-pdf' })
  } finally {
    setIsExportingPDF(false)
  }
}
```

- [ ] **Step 3: Add button in batch action bar**

Add button near “导出 ZIP”:

```tsx
<Button
  variant="default"
  size="sm"
  className="h-7 text-xs bg-zinc-900 hover:bg-zinc-800"
  onClick={handleBatchExportSeriesPDF}
  disabled={selectedSeriesRoots.length === 0 || isExporting || isDeleting || isSyncingSeriesToObsidian || isExportingPDF}
>
  {isExportingPDF ? <Loader2 className="w-3 h-3 animate-spin mr-1" /> : null}
  导出 PDF
</Button>
```

- [ ] **Step 4: Build frontend**

Run:

```bash
cd frontend && npm run build
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/Sidebar.tsx
git commit -m "feat(frontend): add batch export series pdf button"
```

---

### Task 7: End-to-end verification

**Files:**
- None (runtime check)

- [ ] **Step 1: Restart via docker compose**

```bash
docker compose down && docker compose up -d --build
```

- [ ] **Step 2: Manual verification**

At `http://localhost`:
- 登录
- 选择/展开历史博客，进入批量模式（文件夹图标）
- 勾选 2 个系列父节点
- 点击「导出 PDF」
- 观察浏览器触发两次下载（每个系列 1 个 PDF）

- [ ] **Step 3: Record evidence**

Provide:
- 下载到的 PDF 文件名示例（至少 2 个）
- 后端容器日志中无崩溃（若有错误，贴出关键报错行）

---

## Self-Review Checklist (run mentally before execution)

- No directory page numbers (confirmed).
- Button in Sidebar batch bar (B plan).
- Existing APIs remain unchanged; new `/export/pdf` endpoint added.
- Docker runtime includes Chromium + CJK fonts.

---

Plan complete and saved to `docs/superpowers/plans/2026-04-28-export-series-pdf.md`.

Two execution options:
1. Subagent-Driven (recommended) — dispatch a fresh subagent per task, review between tasks
2. Inline Execution — execute tasks in this session with checkpoints

Which approach?


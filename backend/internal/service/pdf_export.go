package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

var seriesPDFMarkdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
)

var ErrSeriesNotFound = errors.New("series not found")

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

		chapters = append(chapters, chapter{
			Title: title,
			HTML:  buf.String(),
			ID:    anchor,
		})
	}

	dateStr := now.Format("2006-01-02 15:04")

	var out strings.Builder
	out.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"/>")
	out.WriteString("<meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src data:;\"/>")
	out.WriteString("<style>")
	out.WriteString("@page{size:A4;margin:18mm 16mm;}")
	out.WriteString("body{font-family:\"Noto Sans CJK SC\",\"Noto Sans\",\"PingFang SC\",\"Microsoft YaHei\",sans-serif;color:#111;}")
	out.WriteString("h1{font-size:28px;margin:0 0 8px 0;} h2{font-size:20px;margin:22px 0 10px 0;} h3{font-size:16px;margin:18px 0 8px 0;}")
	out.WriteString("p,li{line-height:1.75;font-size:14px;} pre{white-space:pre-wrap;word-break:break-word;font-size:12px;background:#f7f7f8;padding:12px;border-radius:8px;}")
	out.WriteString("code{font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,\"Liberation Mono\",\"Courier New\",monospace;}")
	out.WriteString(".page{page-break-after:always;} .page:last-child{page-break-after:auto;}")
	out.WriteString(".cover{display:flex;flex-direction:column;justify-content:center;min-height:80vh;}")
	out.WriteString(".meta{color:#666;font-size:12px;margin-top:12px;}")
	out.WriteString(".toc a{text-decoration:none;color:#111;} .toc li{margin:6px 0;}")
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", ErrSeriesNotFound
		}
		return "", "", err
	}
	if len(blogs) == 0 {
		return "", "", ErrSeriesNotFound
	}

	seriesTitle := blogs[0].Title
	htmlContent, filename, err := buildSeriesPDFHTML(seriesTitle, time.Now(), blogs)
	if err != nil {
		return "", "", err
	}

	htmlFile, err := os.CreateTemp("", "inkwords-series-*.html")
	if err != nil {
		return "", "", err
	}
	defer func() { _ = os.Remove(htmlFile.Name()) }()

	if _, err := htmlFile.WriteString(htmlContent); err != nil {
		_ = htmlFile.Close()
		return "", "", err
	}
	_ = htmlFile.Close()

	pdfPath := filepath.Join(os.TempDir(), fmt.Sprintf("inkwords-series-%s.pdf", uuid.NewString()))

	chromiumPath := os.Getenv("CHROMIUM_BIN")
	if chromiumPath == "" {
		chromiumPath = "chromium"
		if _, err := exec.LookPath(chromiumPath); err != nil {
			if _, err2 := exec.LookPath("chromium-browser"); err2 == nil {
				chromiumPath = "chromium-browser"
			}
		}
	}

	cmd := exec.CommandContext(ctx, chromiumPath, buildChromiumArgs(htmlFile.Name(), pdfPath)...)
	if _, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(pdfPath)
		return "", "", fmt.Errorf("生成 PDF 失败")
	}

	return pdfPath, filename, nil
}

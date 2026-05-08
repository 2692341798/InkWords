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
		t.Fatalf("expected page break css")
	}
}

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


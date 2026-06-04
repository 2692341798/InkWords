package review

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"inkwords-backend/internal/model"
	"inkwords-backend/internal/service"
)

// ReviewNote 表示可进入复习入口的一篇候选笔记。
type ReviewNote struct {
	NotePath      string
	Title         string
	SourceTitle   string
	Body          string
	PreferredMode string
}

// ReviewNoteSource 负责从 Obsidian concepts 目录中读取候选复习笔记。
type ReviewNoteSource struct {
	store   service.ObsidianStore
	rootDir string
}

// NewReviewNoteSource 创建基于 ObsidianStore 的复习笔记源。
func NewReviewNoteSource(store service.ObsidianStore, rootDir string) *ReviewNoteSource {
	return &ReviewNoteSource{
		store:   store,
		rootDir: strings.TrimSuffix(rootDir, "/"),
	}
}

// ListEligibleNotes 返回通过首版过滤规则的 concept 笔记列表。
func (s *ReviewNoteSource) ListEligibleNotes(ctx context.Context) ([]ReviewNote, error) {
	entries, err := s.store.List(ctx, path.Join(s.rootDir, "concepts"))
	if err != nil {
		return nil, fmt.Errorf("列出复习笔记失败: %w", err)
	}

	notes := make([]ReviewNote, 0, len(entries))
	for _, entry := range entries {
		notePath, ok := s.normalizeConceptNotePath(entry)
		if !ok {
			continue
		}

		content, err := s.store.Read(ctx, notePath)
		if err != nil {
			return nil, fmt.Errorf("读取复习笔记失败: %w", err)
		}

		meta, body := parseFrontmatter(string(content))
		if !IsEligibleReviewNote(notePath, meta, body) {
			continue
		}

		note := ReviewNote{
			NotePath:      notePath,
			Title:         firstNonEmpty(meta.Title, inferTitleFromPath(notePath)),
			SourceTitle:   extractSourceTitle(meta.Related),
			Body:          strings.TrimSpace(body),
			PreferredMode: firstNonEmpty(meta.Review.PreferredMode, model.ReviewModeLightRecall),
		}
		notes = append(notes, note)
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].NotePath < notes[j].NotePath
	})

	return notes, nil
}

func (s *ReviewNoteSource) normalizeConceptNotePath(entry string) (string, bool) {
	trimmed := strings.TrimSpace(entry)
	if trimmed == "" || strings.HasSuffix(trimmed, "/") || !strings.HasSuffix(trimmed, ".md") {
		return "", false
	}
	var notePath string
	if strings.HasPrefix(trimmed, s.rootDir+"/") {
		notePath = trimmed
	} else if strings.HasPrefix(trimmed, "concepts/") {
		notePath = path.Join(s.rootDir, trimmed)
	} else {
		notePath = path.Join(s.rootDir, "concepts", trimmed)
	}
	if isSystemNotePath(notePath) {
		return "", false
	}
	return notePath, true
}

func extractSourceTitle(related []string) string {
	for _, item := range related {
		trimmed := strings.Trim(item, "\"")
		if !strings.HasPrefix(trimmed, "[[sources/") {
			continue
		}
		inner := strings.TrimSuffix(strings.TrimPrefix(trimmed, "[["), "]]")
		parts := strings.SplitN(inner, "|", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			return strings.TrimSpace(parts[1])
		}
		return path.Base(strings.TrimSpace(parts[0]))
	}
	return ""
}

func inferTitleFromPath(notePath string) string {
	base := path.Base(notePath)
	return strings.TrimSuffix(base, path.Ext(base))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

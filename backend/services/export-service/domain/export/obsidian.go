package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/google/uuid"

	platformllm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/shared/platform/obsidian"
)

func (s *Service) ExportToObsidian(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return ErrExportNotConfigured
	}

	blog, err := s.repo.GetByID(ctx, userID, blogID)
	if err != nil {
		return err
	}

	store, err := s.getObsidianStore()
	if err != nil {
		return fmt.Errorf("obsidian REST API 未配置: %w", err)
	}

	nowTime := s.now()
	opts := wikiScaffoldOptions{DomainSlug: "tech", DomainTag: "#domain/tech"}
	if err := ensureWikiScaffold(ctx, store, s.rootDir, nowTime, opts); err != nil {
		return fmt.Errorf("初始化知识库目录失败: %w", err)
	}

	title := sanitizeExportFileName(blog.Title)
	now := nowTime.Format("2006-01-02")

	frontmatter := fmt.Sprintf(`---
type: concept
title: "%s"
created: %s
updated: %s
tags:
  - "%s"
status: seed
---
`, title, now, now, opts.DomainTag)

	content := fmt.Sprintf("%s\n# %s\n\n%s", frontmatter, title, blog.Content)
	filePath := path.Join(s.rootDir, "concepts", fmt.Sprintf("%s.md", title))

	if err := store.Put(ctx, filePath, "text/markdown", []byte(content)); err != nil {
		return fmt.Errorf("写入 Obsidian 失败: %w", err)
	}

	if err := ensureWikiScaffold(ctx, store, s.rootDir, nowTime, opts); err != nil {
		return fmt.Errorf("更新知识库索引失败: %w", err)
	}

	return nil
}

//nolint:gocyclo
func (s *Service) ExportSeriesToObsidian(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) error {
	blogs, err := s.GetSeriesBlogs(ctx, blogID, userID)
	if err != nil {
		return err
	}

	store, err := s.getObsidianStore()
	if err != nil {
		return fmt.Errorf("obsidian REST API 未配置: %w", err)
	}
	if s.jsonGenerator == nil {
		return errors.New("llm json generator is not configured")
	}

	nowTime := s.now()
	opts := wikiScaffoldOptions{DomainSlug: "tech", DomainTag: "#domain/tech"}
	if err := ensureWikiScaffold(ctx, store, s.rootDir, nowTime, opts); err != nil {
		return fmt.Errorf("初始化知识库目录失败: %w", err)
	}

	parentTitle := sanitizeExportFileName(blogs[0].Title)
	if parentTitle == "未命名" {
		parentTitle = "未命名系列"
	}

	now := nowTime.Format("2006-01-02")
	nowTimeStr := nowTime.Format("2006-01-02 15:04:05")

	type extractedData struct {
		Entities []string `json:"entities"`
		Concepts []string `json:"concepts"`
	}

	childTitles := make([]string, 0, len(blogs)-1)
	childContents := make([]string, 0, len(blogs)-1)
	extractedMap := make(map[string]extractedData)
	var mu sync.Mutex

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	for idx := 1; idx < len(blogs); idx++ {
		child := blogs[idx]
		childTitle := sanitizeExportFileName(child.Title)
		if childTitle == "未命名" {
			childTitle = fmt.Sprintf("未命名子博客-%d", idx)
		}

		childTitles = append(childTitles, childTitle)
		childContents = append(childContents, child.Content)

		wg.Add(1)
		go func(title, content string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			messages := []platformllm.Message{
				{Role: "system", Content: "你正在帮助构建个人知识库（Second Brain）。请从文本中抽取 3-5 个最重要的实体（人、组织、工具、项目等）与 3-5 个最重要的概念（理论、模式、框架等）。严格返回 JSON：{\"entities\": [\"...\"], \"concepts\": [\"...\"]}。不要输出任何多余文本。"},
				{Role: "user", Content: content},
			}

			jsonResp, err := s.jsonGenerator.GenerateJSON(ctx, s.model, messages)
			if err != nil {
				return
			}

			var data extractedData
			if json.Unmarshal([]byte(jsonResp), &data) != nil {
				return
			}

			for idx := range data.Entities {
				data.Entities[idx] = sanitizeExportFileName(data.Entities[idx])
			}
			for idx := range data.Concepts {
				data.Concepts[idx] = sanitizeExportFileName(data.Concepts[idx])
			}

			mu.Lock()
			extractedMap[title] = data
			mu.Unlock()
		}(childTitle, child.Content)
	}
	wg.Wait()

	for idx, childTitle := range childTitles {
		data := extractedMap[childTitle]
		relatedLinks := fmt.Sprintf("  - \"[[sources/%s|%s]]\"\n", parentTitle, parentTitle)
		for _, entity := range data.Entities {
			relatedLinks += fmt.Sprintf("  - \"[[entities/%s|%s]]\"\n", entity, entity)
		}
		for _, concept := range data.Concepts {
			relatedLinks += fmt.Sprintf("  - \"[[concepts/%s|%s]]\"\n", concept, concept)
		}

		frontmatter := fmt.Sprintf(`---
type: concept
title: "%s"
created: %s
updated: %s
tags:
  - "%s"
status: developing
related:
%s---
`, childTitle, now, now, opts.DomainTag, relatedLinks)

		content := fmt.Sprintf("%s\n# %s\n\n%s", frontmatter, childTitle, childContents[idx])
		filePath := path.Join(s.rootDir, "concepts", fmt.Sprintf("%s.md", childTitle))
		if err := store.Put(ctx, filePath, "text/markdown", []byte(content)); err != nil {
			return fmt.Errorf("写入子博客失败: %w", err)
		}

		for _, entity := range data.Entities {
			entityPath := path.Join(s.rootDir, "entities", fmt.Sprintf("%s.md", entity))
			_, err := store.Read(ctx, entityPath)
			if err == nil {
				continue
			}
			if !obsidian.IsNotFound(err) {
				return fmt.Errorf("读取实体失败: %w", err)
			}

			entityContent := fmt.Sprintf(`---
type: entity
title: "%s"
created: %s
updated: %s
tags:
  - "%s"
status: seed
related:
  - "[[concepts/%s|%s]]"
---

# %s

Context extracted from [[concepts/%s|%s]].
`, entity, now, now, opts.DomainTag, childTitle, childTitle, entity, childTitle, childTitle)
			if err := store.Put(ctx, entityPath, "text/markdown", []byte(entityContent)); err != nil {
				return fmt.Errorf("写入实体失败: %w", err)
			}
		}

		for _, concept := range data.Concepts {
			conceptPath := path.Join(s.rootDir, "concepts", fmt.Sprintf("%s.md", concept))
			_, err := store.Read(ctx, conceptPath)
			if err == nil {
				continue
			}
			if !obsidian.IsNotFound(err) {
				return fmt.Errorf("读取概念失败: %w", err)
			}

			conceptContent := fmt.Sprintf(`---
type: concept
title: "%s"
created: %s
updated: %s
tags:
  - "%s"
status: seed
related:
  - "[[concepts/%s|%s]]"
---

# %s

Context extracted from [[concepts/%s|%s]].
`, concept, now, now, opts.DomainTag, childTitle, childTitle, concept, childTitle, childTitle)
			if err := store.Put(ctx, conceptPath, "text/markdown", []byte(conceptContent)); err != nil {
				return fmt.Errorf("写入概念失败: %w", err)
			}
		}
	}

	parentRelatedStr := "related:\n"
	for _, title := range childTitles {
		parentRelatedStr += fmt.Sprintf("  - \"[[concepts/%s|%s]]\"\n", title, title)
	}

	parentFrontmatter := fmt.Sprintf(`---
type: source
title: "%s"
created: %s
updated: %s
tags:
  - "%s"
status: mature
%s---
`, parentTitle, now, now, opts.DomainTag, parentRelatedStr)

	parentContent := fmt.Sprintf("%s\n# %s\n\n%s", parentFrontmatter, parentTitle, blogs[0].Content)
	parentFilePath := path.Join(s.rootDir, "sources", fmt.Sprintf("%s.md", parentTitle))
	if err := store.Put(ctx, parentFilePath, "text/markdown", []byte(parentContent)); err != nil {
		return fmt.Errorf("写入父博客失败: %w", err)
	}

	indexPath := path.Join(s.rootDir, "index.md")
	indexContentBytes, err := store.Read(ctx, indexPath)
	if err != nil && !obsidian.IsNotFound(err) {
		return fmt.Errorf("读取 index 失败: %w", err)
	}
	indexContent := string(indexContentBytes)
	if indexContent != "" && !strings.HasSuffix(indexContent, "\n") {
		indexContent += "\n"
	}
	if indexContent != "" {
		indexContent += "\n"
	}
	indexContent += fmt.Sprintf("- [[sources/%s|%s]]", parentTitle, parentTitle)
	if err := store.Put(ctx, indexPath, "text/markdown", []byte(indexContent)); err != nil {
		return fmt.Errorf("写入 index 失败: %w", err)
	}

	logPath := path.Join(s.rootDir, "log.md")
	logContent, err := store.Read(ctx, logPath)
	if err == nil {
		lines := strings.Split(string(logContent), "\n")
		newLines := make([]string, 0, len(lines)+1)
		inserted := false
		for _, line := range lines {
			newLines = append(newLines, line)
			if !inserted && strings.HasPrefix(line, "---") && len(newLines) > 10 {
				newLines = append(newLines, fmt.Sprintf("- **%s**: Ingest 系列源: [[sources/%s|%s]]，生成 %d 篇概念卡，并抽取实体/概念。", nowTimeStr, parentTitle, parentTitle, len(childTitles)))
				inserted = true
			}
		}
		if !inserted {
			newLines = append(newLines, fmt.Sprintf("- **%s**: Ingest 系列源: [[sources/%s|%s]]，生成 %d 篇概念卡，并抽取实体/概念。", nowTimeStr, parentTitle, parentTitle, len(childTitles)))
		}
		if err := store.Put(ctx, logPath, "text/markdown", []byte(strings.Join(newLines, "\n"))); err != nil {
			return fmt.Errorf("写入 log 失败: %w", err)
		}
	} else if err != nil && !obsidian.IsNotFound(err) {
		return fmt.Errorf("读取 log 失败: %w", err)
	}

	hotPath := path.Join(s.rootDir, "hot.md")
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

最近摄入的源 (Source)：
- [[sources/%s|%s]]

关联概念 (Concepts)：
`, now, now, parentTitle, parentTitle)
	for _, title := range childTitles {
		hotContent += fmt.Sprintf("- [[concepts/%s|%s]]\n", title, title)
	}
	if err := store.Put(ctx, hotPath, "text/markdown", []byte(hotContent)); err != nil {
		return fmt.Errorf("写入 hot 失败: %w", err)
	}

	if err := ensureWikiScaffold(ctx, store, s.rootDir, nowTime, opts); err != nil {
		return fmt.Errorf("更新知识库索引失败: %w", err)
	}

	return nil
}

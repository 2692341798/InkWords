package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/model"
)

// ExportToObsidian 导出单篇博客到 Obsidian（写入 concepts/ 并更新索引页）
func (s *BlogService) ExportToObsidian(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) error {
	var blog model.Blog
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", blogID, userID).First(&blog).Error
	if err != nil {
		return err
	}

	store, err := newRestAPIStoreFromEnv()
	if err != nil {
		return fmt.Errorf("Obsidian REST API 未配置: %w", err)
	}

	rootDir := strings.TrimSpace(os.Getenv("OBSIDIAN_WIKI_DIR"))
	if rootDir == "" {
		rootDir = "wiki"
	}

	nowTime := time.Now()
	opts := wikiScaffoldOptions{DomainSlug: "tech", DomainTag: "#domain/tech"}
	if err := ensureWikiScaffold(ctx, store, rootDir, nowTime, opts); err != nil {
		return fmt.Errorf("初始化知识库目录失败: %w", err)
	}

	title := sanitizeObsidianFileName(blog.Title)
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
	filePath := path.Join(rootDir, "concepts", fmt.Sprintf("%s.md", title))

	if err := store.Put(ctx, filePath, "text/markdown", []byte(content)); err != nil {
		return fmt.Errorf("写入 Obsidian 失败: %w", err)
	}

	if err := ensureWikiScaffold(ctx, store, rootDir, nowTime, opts); err != nil {
		return fmt.Errorf("更新知识库索引失败: %w", err)
	}

	return nil
}

// ExportSeriesToObsidian 批量导出系列博客到 Obsidian，并按 Karpathy LLM Wiki Pattern 生成 sources/concepts/entities 与索引页
func (s *BlogService) ExportSeriesToObsidian(ctx context.Context, parentID uuid.UUID, userID uuid.UUID) error {
	blogs, err := s.GetSeriesBlogs(ctx, parentID, userID)
	if err != nil {
		return err
	}
	if len(blogs) == 0 {
		return errors.New("系列博客为空")
	}

	store, err := newRestAPIStoreFromEnv()
	if err != nil {
		return fmt.Errorf("Obsidian REST API 未配置: %w", err)
	}

	rootDir := strings.TrimSpace(os.Getenv("OBSIDIAN_WIKI_DIR"))
	if rootDir == "" {
		rootDir = "wiki"
	}

	nowTime := time.Now()
	opts := wikiScaffoldOptions{DomainSlug: "tech", DomainTag: "#domain/tech"}
	if err := ensureWikiScaffold(ctx, store, rootDir, nowTime, opts); err != nil {
		return fmt.Errorf("初始化知识库目录失败: %w", err)
	}

	parentTitle := sanitizeObsidianFileName(blogs[0].Title)
	if parentTitle == "未命名" {
		parentTitle = "未命名系列"
	}

	now := nowTime.Format("2006-01-02")
	nowTimeStr := nowTime.Format("2006-01-02 15:04:05")

	llmClient := llm.NewDeepSeekClient(os.Getenv("DEEPSEEK_API_KEY"))
	modelName := os.Getenv("DEEPSEEK_MODEL")
	if modelName == "" {
		modelName = "deepseek-v4-flash"
	}

	type extractedData struct {
		Entities []string `json:"entities"`
		Concepts []string `json:"concepts"`
	}

	var childTitles []string
	var childContents []string

	extractedMap := make(map[string]extractedData)
	var mu sync.Mutex

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for i := 1; i < len(blogs); i++ {
		child := blogs[i]
		childTitle := sanitizeObsidianFileName(child.Title)
		if childTitle == "未命名" {
			childTitle = fmt.Sprintf("未命名子博客-%d", i)
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

			messages := []llm.Message{
				{Role: "system", Content: "你正在帮助构建个人知识库（Second Brain）。请从文本中抽取 3-5 个最重要的实体（人、组织、工具、项目等）与 3-5 个最重要的概念（理论、模式、框架等）。严格返回 JSON：{\"entities\": [\"...\"], \"concepts\": [\"...\"]}。不要输出任何多余文本。"},
				{Role: "user", Content: content},
			}

			jsonResp, err := llmClient.GenerateJSON(ctx, modelName, messages)
			if err != nil {
				return
			}

			var data extractedData
			if json.Unmarshal([]byte(jsonResp), &data) != nil {
				return
			}

			for i := range data.Entities {
				data.Entities[i] = sanitizeObsidianFileName(data.Entities[i])
			}
			for i := range data.Concepts {
				data.Concepts[i] = sanitizeObsidianFileName(data.Concepts[i])
			}

			mu.Lock()
			extractedMap[title] = data
			mu.Unlock()
		}(childTitle, child.Content)
	}

	wg.Wait()

	for i, childTitle := range childTitles {
		data := extractedMap[childTitle]
		relatedLinks := fmt.Sprintf("  - \"[[sources/%s|%s]]\"\n", parentTitle, parentTitle)
		for _, e := range data.Entities {
			relatedLinks += fmt.Sprintf("  - \"[[entities/%s|%s]]\"\n", e, e)
		}
		for _, c := range data.Concepts {
			relatedLinks += fmt.Sprintf("  - \"[[concepts/%s|%s]]\"\n", c, c)
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

		content := fmt.Sprintf("%s\n# %s\n\n%s", frontmatter, childTitle, childContents[i])
		filePath := path.Join(rootDir, "concepts", fmt.Sprintf("%s.md", childTitle))
		if err := store.Put(ctx, filePath, "text/markdown", []byte(content)); err != nil {
			return fmt.Errorf("写入子博客失败: %w", err)
		}

		for _, e := range data.Entities {
			ePath := path.Join(rootDir, "entities", fmt.Sprintf("%s.md", e))
			_, err := store.Read(ctx, ePath)
			if err == nil {
				continue
			}
			if !isObsidianNotFound(err) {
				return fmt.Errorf("读取实体失败: %w", err)
			}

			eContent := fmt.Sprintf(`---
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
`, e, now, now, opts.DomainTag, childTitle, childTitle, e, childTitle, childTitle)
			if err := store.Put(ctx, ePath, "text/markdown", []byte(eContent)); err != nil {
				return fmt.Errorf("写入实体失败: %w", err)
			}
		}

		for _, c := range data.Concepts {
			cPath := path.Join(rootDir, "concepts", fmt.Sprintf("%s.md", c))
			_, err := store.Read(ctx, cPath)
			if err == nil {
				continue
			}
			if !isObsidianNotFound(err) {
				return fmt.Errorf("读取概念失败: %w", err)
			}

			cContent := fmt.Sprintf(`---
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
`, c, now, now, opts.DomainTag, childTitle, childTitle, c, childTitle, childTitle)
			if err := store.Put(ctx, cPath, "text/markdown", []byte(cContent)); err != nil {
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
	parentFilePath := path.Join(rootDir, "sources", fmt.Sprintf("%s.md", parentTitle))
	if err := store.Put(ctx, parentFilePath, "text/markdown", []byte(parentContent)); err != nil {
		return fmt.Errorf("写入父博客失败: %w", err)
	}

	indexPath := path.Join(rootDir, "index.md")
	indexContentBytes, err := store.Read(ctx, indexPath)
	if err != nil && !isObsidianNotFound(err) {
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

	logPath := path.Join(rootDir, "log.md")
	logContent, err := store.Read(ctx, logPath)
	if err == nil {
		lines := strings.Split(string(logContent), "\n")
		var newLines []string
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
	} else if err != nil && !isObsidianNotFound(err) {
		return fmt.Errorf("读取 log 失败: %w", err)
	}

	hotPath := path.Join(rootDir, "hot.md")
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

	if err := ensureWikiScaffold(ctx, store, rootDir, nowTime, opts); err != nil {
		return fmt.Errorf("更新知识库索引失败: %w", err)
	}

	return nil
}

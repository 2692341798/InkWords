package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type wikiScaffoldOptions struct {
	DomainSlug string
	DomainTag  string
}

func ensureWikiScaffold(basePath string, now time.Time, opts wikiScaffoldOptions) error {
	dirs := []string{".raw", "sources", "concepts", "entities", "domains", filepath.Join("domains", opts.DomainSlug)}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(basePath, dir), 0755); err != nil {
			return err
		}
	}

	nowDate := now.Format("2006-01-02")

	if err := writeFolderIndex(basePath, nowDate, "sources", "sources/_index", "Sources Index", []string{"#meta/index"}); err != nil {
		return err
	}
	if err := writeFolderIndex(basePath, nowDate, "concepts", "concepts/_index", "Concepts Index", []string{"#meta/index"}); err != nil {
		return err
	}
	if err := writeFolderIndex(basePath, nowDate, "entities", "entities/_index", "Entities Index", []string{"#meta/index"}); err != nil {
		return err
	}

	if err := writeDomainsIndex(basePath, nowDate, opts); err != nil {
		return err
	}

	return nil
}

func writeDomainsIndex(basePath string, nowDate string, opts wikiScaffoldOptions) error {
	domainsIndexPath := filepath.Join(basePath, "domains", "_index.md")
	content := fmt.Sprintf(`---
type: domain
title: "Domains Index"
created: %s
updated: %s
tags:
  - "#meta/index"
status: mature
---

# Domains Index

- [[domains/%s/_index|%s]]
`, nowDate, nowDate, opts.DomainSlug, strings.ToUpper(opts.DomainSlug[:1])+opts.DomainSlug[1:])

	if err := os.WriteFile(domainsIndexPath, []byte(content), 0644); err != nil {
		return err
	}

	domainIndexPath := filepath.Join(basePath, "domains", opts.DomainSlug, "_index.md")
	domainIndex := fmt.Sprintf(`---
type: domain
title: "%s"
created: %s
updated: %s
tags:
  - "%s"
status: mature
---

# %s

## Sources
- [[sources/_index|Sources Index]]

## Concepts
- [[concepts/_index|Concepts Index]]

## Entities
- [[entities/_index|Entities Index]]
`, opts.DomainSlug, nowDate, nowDate, opts.DomainTag, opts.DomainSlug)

	return os.WriteFile(domainIndexPath, []byte(domainIndex), 0644)
}

func writeFolderIndex(basePath string, nowDate string, folder string, notePath string, title string, tags []string) error {
	entries, err := os.ReadDir(filepath.Join(basePath, folder))
	if err != nil {
		return err
	}

	var noteNames []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		if name == "_index.md" {
			continue
		}
		noteNames = append(noteNames, strings.TrimSuffix(name, ".md"))
	}
	sort.Strings(noteNames)

	tagLines := ""
	if len(tags) > 0 {
		tagLines = "tags:\n"
		for _, t := range tags {
			tagLines += fmt.Sprintf("  - \"%s\"\n", t)
		}
	}

	listLines := ""
	for _, n := range noteNames {
		listLines += fmt.Sprintf("- [[%s/%s|%s]]\n", folder, n, n)
	}
	if listLines == "" {
		listLines = "- （暂无）\n"
	}

	content := fmt.Sprintf(`---
type: meta
title: "%s"
created: %s
updated: %s
%sstatus: mature
---

# %s

%s`, title, nowDate, nowDate, tagLines, title, listLines)

	return os.WriteFile(filepath.Join(basePath, notePath+".md"), []byte(content), 0644)
}

func sanitizeObsidianFileName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return "未命名"
	}
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = strings.ReplaceAll(s, ":", "：")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}


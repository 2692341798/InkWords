package service

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
)

type wikiScaffoldOptions struct {
	DomainSlug string
	DomainTag  string
}

func ensureWikiScaffold(ctx context.Context, store ObsidianStore, rootDir string, now time.Time, opts wikiScaffoldOptions) error {
	nowDate := now.Format("2006-01-02")

	if err := writeFolderIndex(ctx, store, rootDir, nowDate, "sources", "sources/_index", "Sources Index", []string{"#meta/index"}); err != nil {
		return err
	}
	if err := writeFolderIndex(ctx, store, rootDir, nowDate, "concepts", "concepts/_index", "Concepts Index", []string{"#meta/index"}); err != nil {
		return err
	}
	if err := writeFolderIndex(ctx, store, rootDir, nowDate, "entities", "entities/_index", "Entities Index", []string{"#meta/index"}); err != nil {
		return err
	}

	if err := writeDomainsIndex(ctx, store, rootDir, nowDate, opts); err != nil {
		return err
	}

	return nil
}

func writeDomainsIndex(ctx context.Context, store ObsidianStore, rootDir string, nowDate string, opts wikiScaffoldOptions) error {
	domainsIndexPath := path.Join(rootDir, "domains", "_index.md")
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

	if err := store.Put(ctx, domainsIndexPath, "text/markdown", []byte(content)); err != nil {
		return err
	}

	domainIndexPath := path.Join(rootDir, "domains", opts.DomainSlug, "_index.md")
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

	return store.Put(ctx, domainIndexPath, "text/markdown", []byte(domainIndex))
}

func writeFolderIndex(ctx context.Context, store ObsidianStore, rootDir string, nowDate string, folder string, notePath string, title string, tags []string) error {
	entries, err := store.List(ctx, path.Join(rootDir, folder))
	if err != nil && !isObsidianNotFound(err) {
		return err
	}

	var noteNames []string
	for _, entry := range entries {
		name := path.Base(entry)
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

	return store.Put(ctx, path.Join(rootDir, notePath+".md"), "text/markdown", []byte(content))
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

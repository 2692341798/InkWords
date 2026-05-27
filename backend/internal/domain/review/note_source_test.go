package review

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReviewNoteSource_ListEligibleNotes_FiltersAndBuildsMetadata(t *testing.T) {
	t.Parallel()

	store := &fakeObsidianStore{
		files: map[string][]byte{
			"wiki/concepts/并发控制与速率限制.md": []byte(`---
type: concept
title: "并发控制与速率限制"
related:
  - "[[sources/InkWords 内容生成平台架构解析系列|InkWords 内容生成平台架构解析系列]]"
review:
  enabled: true
  preferred_mode: detailed_qa
---

# 并发控制与速率限制

` + strings.Repeat("并发控制与速率限制的有效正文。", 20)),
			"wiki/concepts/种子页.md": []byte(`---
type: concept
title: "种子页"
---

Context extracted from [[concepts/并发控制与速率限制|并发控制与速率限制]].`),
		},
		dirs: map[string][]string{
			"wiki/concepts": {"并发控制与速率限制.md", "种子页.md", "_index.md", "子目录/"},
		},
	}

	source := NewReviewNoteSource(store, "wiki")

	notes, err := source.ListEligibleNotes(context.Background())
	require.NoError(t, err)
	require.Len(t, notes, 1)
	require.Equal(t, "wiki/concepts/并发控制与速率限制.md", notes[0].NotePath)
	require.Equal(t, "并发控制与速率限制", notes[0].Title)
	require.Equal(t, "InkWords 内容生成平台架构解析系列", notes[0].SourceTitle)
	require.Equal(t, "detailed_qa", notes[0].PreferredMode)
	require.Contains(t, notes[0].Body, "有效正文")
}

var errFakeObsidianNotFound = errors.New("fake obsidian not found")

type fakeObsidianStore struct {
	files map[string][]byte
	dirs  map[string][]string
}

func (f *fakeObsidianStore) Read(_ context.Context, path string) ([]byte, error) {
	body, ok := f.files[path]
	if !ok {
		return nil, errFakeObsidianNotFound
	}
	return body, nil
}

func (f *fakeObsidianStore) Put(_ context.Context, _ string, _ string, _ []byte) error {
	panic("unexpected Put call")
}

func (f *fakeObsidianStore) Post(_ context.Context, _ string, _ string, _ []byte) error {
	panic("unexpected Post call")
}

func (f *fakeObsidianStore) Patch(_ context.Context, _ string, _ map[string]string, _ string, _ []byte) error {
	panic("unexpected Patch call")
}

func (f *fakeObsidianStore) List(_ context.Context, dirPath string) ([]string, error) {
	items, ok := f.dirs[dirPath]
	if !ok {
		return nil, errFakeObsidianNotFound
	}
	return items, nil
}

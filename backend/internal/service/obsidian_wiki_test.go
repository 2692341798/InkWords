package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestEnsureWikiScaffoldCreatesIndexNotes(t *testing.T) {
	store := &fakeStore{
		files: map[string][]byte{},
		dirs: map[string][]string{
			"wiki/concepts": {"A.md"},
			"wiki/entities": {},
			"wiki/sources":  {},
			"wiki/domains":  {},
		},
	}

	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	if err := ensureWikiScaffold(context.Background(), store, "wiki", now, wikiScaffoldOptions{DomainSlug: "tech", DomainTag: "#domain/tech"}); err != nil {
		t.Fatal(err)
	}

	indexPath := "wiki/concepts/_index.md"
	indexBytes := store.files[indexPath]
	if len(indexBytes) == 0 || !strings.Contains(string(indexBytes), "[[concepts/A|A]]") {
		t.Fatalf("concepts index missing link, got: %s", string(indexBytes))
	}

	domainsIndexPath := "wiki/domains/_index.md"
	if len(store.files[domainsIndexPath]) == 0 {
		t.Fatalf("domains index not created")
	}
}

type fakeStore struct {
	files map[string][]byte
	dirs  map[string][]string
}

func (f *fakeStore) Read(_ context.Context, p string) ([]byte, error) {
	b, ok := f.files[p]
	if !ok {
		return nil, obsidianHTTPError{StatusCode: 404}
	}
	return b, nil
}

func (f *fakeStore) Put(_ context.Context, p string, _ string, body []byte) error {
	f.files[p] = body
	return nil
}

func (f *fakeStore) Post(_ context.Context, p string, _ string, body []byte) error {
	f.files[p] = append(f.files[p], body...)
	return nil
}

func (f *fakeStore) Patch(_ context.Context, p string, _ map[string]string, _ string, body []byte) error {
	f.files[p] = body
	return nil
}

func (f *fakeStore) List(_ context.Context, dirPath string) ([]string, error) {
	entries, ok := f.dirs[dirPath]
	if !ok {
		return nil, obsidianHTTPError{StatusCode: 404}
	}
	return entries, nil
}

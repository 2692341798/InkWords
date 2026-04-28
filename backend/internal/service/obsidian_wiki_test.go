package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnsureWikiScaffoldCreatesIndexNotes(t *testing.T) {
	baseDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(baseDir, "concepts"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "concepts", "A.md"), []byte("# A"), 0644); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	if err := ensureWikiScaffold(baseDir, now, wikiScaffoldOptions{DomainSlug: "tech", DomainTag: "#domain/tech"}); err != nil {
		t.Fatal(err)
	}

	indexPath := filepath.Join(baseDir, "concepts", "_index.md")
	b, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read concepts index: %v", err)
	}
	if string(b) == "" || !contains(string(b), "[[concepts/A|A]]") {
		t.Fatalf("concepts index missing link, got: %s", string(b))
	}

	domainsIndexPath := filepath.Join(baseDir, "domains", "_index.md")
	if _, err := os.Stat(domainsIndexPath); err != nil {
		t.Fatalf("domains index not created: %v", err)
	}
}

func contains(s string, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s string, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}


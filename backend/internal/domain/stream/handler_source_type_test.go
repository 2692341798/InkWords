package stream

import "testing"

func TestResolveAnalyzeSourceType(t *testing.T) {
	t.Run("treats source content without git url as file", func(t *testing.T) {
		req := GenerateRequest{
			SourceContent: "parsed file content",
		}

		if got := resolveAnalyzeSourceType(req); got != "file" {
			t.Fatalf("expected file source type, got %q", got)
		}
	})

	t.Run("keeps explicit git source type", func(t *testing.T) {
		req := GenerateRequest{
			SourceType: "git",
			GitURL:     "https://github.com/example/repo",
		}

		if got := resolveAnalyzeSourceType(req); got != "git" {
			t.Fatalf("expected git source type, got %q", got)
		}
	})
}

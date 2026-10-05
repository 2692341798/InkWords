package bookrender

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func testBook(t *testing.T) sharedtextbook.CanonicalBookAST {
	t.Helper()
	b, err := sharedtextbook.NewCanonicalBookAST("中文打印验证", time.Unix(1, 0), []sharedtextbook.CanonicalBookChapter{{ID: "c1", Order: 1, Title: "测试章节", Markdown: "# 测试章节\n\n## 可追溯的示例\n\n这是正文。\n\n```go\nfunc main() {}\n```", ContentHash: "sha256:" + strings.Repeat("a", 64)}})
	require.NoError(t, err)
	return b
}

func TestChromiumPrintsRealPageNumbersWhenExplicitlyConfigured(t *testing.T) {
	executable := os.Getenv("TEST_TEXTBOOK_CHROMIUM_BIN")
	if executable == "" {
		t.Skip("explicit real Chromium opt-in required")
	}
	book := testBook(t)
	if path := os.Getenv("TEST_TEXTBOOK_BOOK_AST"); path != "" {
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &book))
		require.NoError(t, book.Validate())
	}
	result, err := NewChromium(executable).RenderPDF(t.Context(), book)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(result.Content), "%PDF-"))
	require.Contains(t, result.RenderLog, "status=success")
	require.Equal(t, layoutVersion, result.ToolVersions["pdf_layout"])
	require.NotNil(t, result.FontEvidence)
	require.NoError(t, result.FontEvidence.ValidateBinding(book, result.Content))
	require.Equal(t, "partial", result.FontEvidence.BodyObservationStatus)
	require.NotEmpty(t, result.FontEvidence.Fonts)
	require.NotEmpty(t, result.FontEvidence.InstalledFaces)
	require.False(t, result.FontEvidence.PublicationReady())
	if os.Getenv("TEXTBOOK_PDF_FONT_PROFILE") == reviewedFontProfile {
		require.NotNil(t, result.FontEvidence.SourceSet)
		require.Len(t, result.FontEvidence.SourceSet.FileHashes, 4)
		require.True(t, result.FontEvidence.SourceSet.InventoryStable)
	}
	if path := os.Getenv("TEST_TEXTBOOK_PDF_OUTPUT"); path != "" {
		require.NoError(t, os.WriteFile(path, result.Content, 0o600))
		raw, err := json.MarshalIndent(result.FontEvidence, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path+".font-evidence.json", raw, 0o600))
	}
}

func TestChromiumNeverAddsSandboxBypassToTheActualCommand(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args")
	exe := filepath.Join(dir, "capture-browser")
	require.NoError(t, os.WriteFile(exe, []byte(fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > '%s'\nexit 1\n", argsPath)), 0o700))
	_, err := NewChromium(exe).RenderPDF(t.Context(), testBook(t))
	require.Error(t, err)
	args, err := os.ReadFile(argsPath)
	require.NoError(t, err)
	for _, forbidden := range []string{"--no-sandbox", "--disable-setuid-sandbox", "--disable-web-security", "--disable-ipc-flooding-protection"} {
		require.NotContains(t, string(args), forbidden)
	}
	require.Contains(t, string(args), "--headless")
}

func TestChromiumDeadlineIncludesWaitingForItsOnlySlot(t *testing.T) {
	r := NewChromiumWithTimeout("/not-launched", 20*time.Millisecond)
	r.slots <- struct{}{}
	start := time.Now()
	_, err := r.RenderPDF(context.Background(), testBook(t))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), time.Second)
}

func TestChromiumUnavailableAndFailedStartRemainExplicit(t *testing.T) {
	_, err := NewChromium("").RenderPDF(t.Context(), testBook(t))
	require.ErrorContains(t, err, "not configured")
	result, err := NewChromium("/nonexistent-browser").RenderPDF(t.Context(), testBook(t))
	require.Error(t, err)
	require.Contains(t, result.RenderLog, "status=failed")
	require.Empty(t, result.Content)
}

package export

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	shared "inkwords-backend/shared/kernel/textbook"
)

type testBookImages map[string][]byte

func (s testBookImages) ReadBookImage(hash string) ([]byte, error) {
	b, ok := s[hash]
	if !ok {
		return nil, fmt.Errorf("missing image")
	}
	return b, nil
}

func imageBook(t *testing.T) (shared.CanonicalBookAST, testBookImages) {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 20, 10))))
	hash := fmt.Sprintf("sha256:%x", sha256.Sum256(b.Bytes()))
	book := citedBook(t)
	book.Chapters[0].Assets = []shared.CanonicalBookAsset{{ID: "image-1", StableRef: "inkwords-asset:route", ContentHash: hash, AltText: "路由 [结构]"}}
	return book, testBookImages{hash: b.Bytes()}
}

func TestBookImagesEmbedFrozenBytesAndPreserveManuscript(t *testing.T) {
	book, source := imageBook(t)
	code := "`![示例](inkwords-asset:route)`\n\n```md\n![示例](inkwords-asset:route)\n```"
	book.Chapters[0].Markdown += "\n\n正文前\n\n![带 **强调** 的图][diagram]\n\n正文后\n\n[diagram]: inkwords-asset:route\n\n" + code
	original := book.Chapters[0].Markdown
	md, err := RenderBookMarkdown(book, source)
	require.NoError(t, err)
	require.Equal(t, original, book.Chapters[0].Markdown)
	require.Contains(t, string(md), code)
	require.Equal(t, 1, strings.Count(string(md), "data:image/png;base64,"))
	require.Less(t, strings.Index(string(md), "正文前"), strings.Index(string(md), "data:image/png"))
	require.Less(t, strings.Index(string(md), "data:image/png"), strings.Index(string(md), "正文后"))
	for _, b := range source {
		require.Contains(t, string(md), base64.StdEncoding.EncodeToString(b))
	}
	html, err := RenderBookHTML(book, source)
	require.NoError(t, err)
	require.Contains(t, string(html), `<img src="data:image/png;base64,`)
	require.Contains(t, string(html), `class="book-image-caption"`)
	require.NotContains(t, string(html), `\[`)
	require.False(t, hasOnlyTextFontSurfaces(html))
	e := NewPDFFontEvidence(book, html, []byte("%PDF-fixture"), time.Unix(1, 0))
	require.NoError(t, e.ValidateBinding(book, []byte("%PDF-fixture"), source))
	require.Error(t, e.ValidateBinding(book, []byte("%PDF-fixture")))
}

func TestBookImagesAppendUnplacedAndRejectInvalidBytes(t *testing.T) {
	book, source := imageBook(t)
	book.Chapters[0].Assets[0].AltText = "main_test.go 路由 [结构]"
	md, err := RenderBookMarkdown(book, source)
	require.NoError(t, err)
	require.Contains(t, string(md), "![main\\_test.go 路由 \\[结构\\]](data:image/png;base64,")
	html, err := RenderBookHTML(book, source)
	require.NoError(t, err)
	require.Contains(t, string(html), `class="book-image-caption">main_test.go 路由 [结构]</span>`)
	_, err = RenderBookMarkdown(book, testBookImages{})
	require.Error(t, err)
	for hash := range source {
		source[hash] = []byte("corrupted")
	}
	_, err = RenderBookMarkdown(book, source)
	require.Error(t, err)
	bad := []byte("not a PNG")
	hash := fmt.Sprintf("sha256:%x", sha256.Sum256(bad))
	book.Chapters[0].Assets[0].ContentHash = hash
	_, err = RenderBookMarkdown(book, testBookImages{hash: bad})
	require.Error(t, err)
}

func TestBookImagesResolveInlineEmptyAndReferenceImagesButRejectForeignTargets(t *testing.T) {
	for _, syntax := range []string{"![](inkwords-asset:route)", "![a \\[b\\]](<inkwords-asset:route> \"title\")", "![route][]\n\n[route]: inkwords-asset:route", "![route]\n\n[route]: inkwords-asset:route", "> ![route](inkwords-asset:route)"} {
		t.Run(syntax, func(t *testing.T) {
			book, source := imageBook(t)
			book.Chapters[0].Markdown += "\n\n" + syntax
			md, err := RenderBookMarkdown(book, source)
			require.NoError(t, err)
			require.Equal(t, 1, strings.Count(string(md), "data:image/png;base64,"))
		})
	}
	for _, target := range []string{"https://example.com/a.png", "file:///etc/passwd", "../../secret", "data:image/svg+xml,bad", "inkwords-asset:other"} {
		book, source := imageBook(t)
		book.Chapters[0].Markdown += "\n\n![图](" + target + ")"
		_, err := RenderBookMarkdown(book, source)
		require.Error(t, err, target)
	}
}

func TestPandocEmbedsFrozenImageBytes(t *testing.T) {
	book, source := imageBook(t)
	renderer := testPandocRenderer(t).WithBookImages(source)
	projection, err := renderer.RenderDOCX(context.Background(), book)
	require.NoError(t, err)
	assertDOCXImage(t, projection.Content, source)
}

func TestImageBookPackageBindsPDFEvidenceAndRejectsMissingAssets(t *testing.T) {
	book, source := imageBook(t)
	html, err := RenderBookHTML(book, source)
	require.NoError(t, err)
	content := []byte("%PDF-fixture")
	evidence := NewPDFFontEvidence(book, html, content, time.Unix(1, 0))
	input := BookPackageInput{Book: book, PDF: &BookPDFProjection{Content: content, FontEvidence: &evidence}, BuildManifest: json.RawMessage(`{"format":"inkwords.book-build.v1"}`)}
	var failed bytes.Buffer
	require.Error(t, NewBookPackageBuilder().Build(&failed, input))
	require.Empty(t, failed.Bytes(), "no partial successful ZIP when frozen media is missing")
	for hash, data := range source {
		input.Assets = append(input.Assets, BookPackageFile{Path: strings.TrimPrefix(hash, "sha256:") + ".png", Content: data})
	}
	var output bytes.Buffer
	require.NoError(t, NewBookPackageBuilder().Build(&output, input))
	z, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	require.NoError(t, err)
	for _, f := range z.File {
		if f.Name == "projections/book.md" {
			require.Contains(t, readZipFile(t, f), "data:image/png;base64,")
		}
	}
	input.Assets[0].Content = []byte("tampered")
	require.Error(t, NewBookPackageBuilder().Build(&bytes.Buffer{}, input))
}

func testPandocRenderer(t *testing.T) *PandocBookRenderer {
	t.Helper()
	p, err := exec.LookPath("pandoc")
	if err != nil {
		t.Skip("Pandoc is not installed")
	}
	ref, err := filepath.Abs(filepath.Join("..", "..", "assets", "publishing", "inkwords-reference.docx"))
	require.NoError(t, err)
	return NewPandocBookRenderer(p, ref)
}

func assertDOCXImage(t *testing.T, content []byte, images testBookImages) {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)
	count := 0
	for _, file := range z.File {
		if !strings.HasPrefix(file.Name, "word/media/") {
			continue
		}
		r, err := file.Open()
		require.NoError(t, err)
		b, err := io.ReadAll(r)
		require.NoError(t, err)
		require.NoError(t, r.Close())
		hash := fmt.Sprintf("sha256:%x", sha256.Sum256(b))
		require.Equal(t, images[hash], b)
		count++
	}
	require.Equal(t, len(images), count)
}

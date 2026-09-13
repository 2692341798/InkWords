package export

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestPandocBookRendererCreatesDOCXFromCanonicalAST(t *testing.T) {
	pandocPath, err := exec.LookPath("pandoc")
	if err != nil {
		t.Skip("Pandoc is not installed")
	}
	referencePath, err := filepath.Abs(filepath.Join("..", "..", "assets", "publishing", "inkwords-reference.docx"))
	require.NoError(t, err)

	book := citedBook(t)
	book.Chapters[0].Markdown = "# 从请求到处理函数\n\n## 对照原理\n\n" + book.Chapters[0].Markdown + "\n\n相邻来源[evidence:route][evidence:route]。\n\n## 对照原理\n\n第二次出现的同名标题。\n"
	book.Chapters[0].Markdown += "\n\n|方案|边界|\n|---|---|\n|教学树|只处理静态路径|\n"
	longCode := strings.Repeat("// 保留空行与原代码\n\n", 35) + "func main() {}"
	book.Chapters[0].Markdown += "\n\n```go\n" + longCode + "\n```\n"
	renderer := NewPandocBookRenderer(pandocPath, referencePath)
	result, err := renderer.RenderDOCX(context.Background(), book)

	require.NoError(t, err)
	require.Equal(t, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", result.MediaType)
	require.True(t, bytes.HasPrefix(result.Content, []byte("PK\x03\x04")))
	require.Contains(t, result.ToolVersions["pandoc"], "pandoc")
	referenceBytes, err := os.ReadFile(referencePath)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("sha256:%x", sha256.Sum256(referenceBytes)), result.ToolVersions["docx_reference_sha256"])
	archive, err := zip.NewReader(bytes.NewReader(result.Content), int64(len(result.Content)))
	require.NoError(t, err)
	parts := map[string]string{}
	for _, file := range archive.File {
		if strings.HasPrefix(file.Name, "word/") && (strings.HasSuffix(file.Name, ".xml") || strings.HasSuffix(file.Name, ".rels")) {
			reader, err := file.Open()
			require.NoError(t, err)
			content, err := io.ReadAll(reader)
			require.NoError(t, err)
			require.NoError(t, reader.Close())
			parts[file.Name] = string(content)
		}
	}
	require.Contains(t, parts["word/styles.xml"], `w:styleId="Table"`)
	require.Contains(t, parts["word/styles.xml"], `w:styleId="Compact"`)
	require.Contains(t, parts["word/document.xml"], "<w:tbl>")
	require.Contains(t, parts["word/document.xml"], `w:tblStyle w:val="Table"`)
	require.Contains(t, parts["word/document.xml"], "w:footnoteReference")
	require.NotContains(t, parts["word/document.xml"], "[evidence:route]")
	require.Contains(t, parts["word/footnotes.xml"], "node.getValue")
	require.Contains(t, parts["word/_rels/footnotes.xml.rels"], "/blob/"+strings.Repeat("b", 40)+"/tree.go#L10-L20")
	require.Contains(t, parts["word/document.xml"], `w:pStyle w:val="Title"`)
	require.Contains(t, parts["word/document.xml"], "目录")
	require.Contains(t, parts["word/document.xml"], `w:hyperlink w:anchor=`)
	require.Contains(t, parts["word/document.xml"], `w:type="page"`)
	require.Contains(t, parts["word/footer1.xml"], "PAGE")
	require.Contains(t, parts["word/document.xml"], "footerReference")
	require.Contains(t, parts["word/document.xml"], `w:vertAlign w:val="superscript"`)
	require.Contains(t, parts["word/document.xml"], `>,</w:t>`)
	require.Equal(t, "inkwords.docx-layout.v2", result.ToolVersions["docx_layout"])
	// Repeated Chinese headings must link to distinct, existing bookmarks;
	// merely checking for hyperlink tags would miss broken navigation.
	bookmarks := map[string]bool{}
	var anchors []string
	decoder := xml.NewDecoder(strings.NewReader(parts["word/document.xml"]))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		for _, attribute := range start.Attr {
			if start.Name.Local == "bookmarkStart" && attribute.Name.Local == "name" {
				bookmarks[attribute.Value] = true
			}
			if start.Name.Local == "hyperlink" && attribute.Name.Local == "anchor" {
				anchors = append(anchors, attribute.Value)
			}
		}
	}
	require.Len(t, anchors, 3)
	require.NotEqual(t, anchors[1], anchors[2])
	for _, anchor := range anchors {
		require.True(t, bookmarks[anchor], "missing TOC destination %q", anchor)
	}
	units := docxCodeParagraphs(t, parts["word/document.xml"])
	require.Greater(t, len(units), 2)
	require.Equal(t, "[evidence:fenced]\n"+longCode, strings.Join(units, "\n"))
	for _, unit := range units {
		require.LessOrEqual(t, len(strings.Split(unit, "\n")), 30)
	}
}

func docxCodeParagraphs(t *testing.T, document string) []string {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(document))
	var units []string
	var paragraph strings.Builder
	isCode := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return units
		}
		require.NoError(t, err)
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "p":
				paragraph.Reset()
				isCode = false
			case "pStyle":
				for _, attribute := range token.Attr {
					if attribute.Name.Local == "val" && attribute.Value == "SourceCode" {
						isCode = true
					}
				}
			case "t":
				var value string
				require.NoError(t, decoder.DecodeElement(&value, &token))
				paragraph.WriteString(value)
			case "br":
				paragraph.WriteByte('\n')
			}
		case xml.EndElement:
			if token.Name.Local == "p" && isCode {
				units = append(units, paragraph.String())
			}
		}
	}
}

func TestPandocBookRendererRejectsMissingReferenceDocument(t *testing.T) {
	book := canonicalBookForDOCXTest(t)
	renderer := NewPandocBookRenderer("pandoc", filepath.Join(t.TempDir(), "missing.docx"))

	_, err := renderer.RenderDOCX(context.Background(), book)

	require.ErrorContains(t, err, "reference document")
}

func canonicalBookForDOCXTest(t *testing.T) sharedtextbook.CanonicalBookAST {
	t.Helper()
	hash := "sha256:" + strings.Repeat("a", 64)
	book, err := sharedtextbook.NewCanonicalBookAST("InkWords 测试教材", time.Unix(1, 0), []sharedtextbook.CanonicalBookChapter{{
		ID:          "chapter-1",
		Order:       1,
		Title:       "从问题开始",
		Markdown:    "# 从问题开始\n\n用一个最小示例解释机制。",
		ContentHash: hash,
	}})
	require.NoError(t, err)
	return book
}

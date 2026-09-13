package parser

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStructuredParser_ParseMarkdownPreservesHeadingsCodeLinksAndOffsets(t *testing.T) {
	content := "# Gin 入门\n\n先读 [官方文档](https://gin-gonic.com/docs/)。\n\n## 路由\n\n```go\nr.GET(\"/ping\", handler)\n```\n\n解释请求如何进入处理函数。\n"
	result, err := NewStructuredParser().Parse(strings.NewReader(content), StructuredParseRequest{
		SourceID:         "source-gin",
		SnapshotID:       "snapshot-gin-1",
		CanonicalLocator: "https://github.com/gin-gonic/gin/blob/abc/readme.md",
		ArtifactPath:     "README.md",
		Filename:         "README.md",
	})
	require.NoError(t, err)
	require.NoError(t, result.Document.Validate())
	require.Len(t, result.Chunks, 3)
	require.Equal(t, []string{"Gin 入门"}, result.Chunks[0].HeadingPath)
	require.Equal(t, []string{"Gin 入门", "路由"}, result.Chunks[1].HeadingPath)
	require.Equal(t, "go", result.Chunks[1].CodeLanguage)
	require.Equal(t, 7, result.Chunks[1].Locator.StartLine)
	require.Equal(t, 8, result.Chunks[1].Locator.EndLine)
	require.Equal(t, content[result.Chunks[1].StartByte:result.Chunks[1].EndByte], "```go\nr.GET(\"/ping\", handler)")
	require.Len(t, result.Links, 1)
	require.Equal(t, "官方文档", result.Links[0].Text)
	require.Equal(t, "https://gin-gonic.com/docs/", result.Links[0].Target)
	for _, chunk := range result.Chunks {
		require.NoError(t, chunk.Validate())
	}
}

func TestStructuredParser_ParseTextUsesParagraphsAndRejectsUnknownEncoding(t *testing.T) {
	request := StructuredParseRequest{SourceID: "source-text", SnapshotID: "snapshot-text", CanonicalLocator: "file:///lesson.txt", ArtifactPath: "lesson.txt", Filename: "lesson.txt"}
	result, err := NewStructuredParser().Parse(strings.NewReader("第一段第一行\n第一段第二行\n\n第二段"), request)
	require.NoError(t, err)
	require.Len(t, result.Chunks, 2)
	require.Equal(t, 1, result.Chunks[0].Paragraph)
	require.Equal(t, 2, result.Chunks[1].Paragraph)

	_, err = NewStructuredParser().Parse(strings.NewReader("\xff\xfe"), request)
	require.ErrorContains(t, err, "UTF-8")
}

func TestStructuredParser_ParseCodePreservesGitSourceLineRanges(t *testing.T) {
	content := "package gin\n\n// GET registers a route.\nfunc (group *RouterGroup) GET(relativePath string) {\n\tgroup.handle(\"GET\", relativePath)\n}\n"
	result, err := NewStructuredParser().Parse(strings.NewReader(content), StructuredParseRequest{SourceID: "source-gin", SnapshotID: "snapshot-gin", CanonicalLocator: "https://github.com/gin-gonic/gin", ArtifactPath: "routergroup.go", Filename: "routergroup.go"})
	require.NoError(t, err)
	require.Equal(t, "text/x-go", result.Document.MediaType)
	require.Len(t, result.Chunks, 2)
	require.Equal(t, "go", result.Chunks[1].CodeLanguage)
	require.Equal(t, 3, result.Chunks[1].Locator.StartLine)
	require.Equal(t, 6, result.Chunks[1].Locator.EndLine)
}

func TestDocParser_ParseStructuredPDFPreservesPageProvenance(t *testing.T) {
	result, err := NewDocParser().ParseStructured(bytes.NewReader(minimalTextPDF("PDF route evidence")), StructuredParseRequest{SourceID: "source-pdf", SnapshotID: "snapshot-pdf", CanonicalLocator: "file:///lesson.pdf", ArtifactPath: "lesson.pdf", Filename: "lesson.pdf"})
	require.NoError(t, err)
	require.Equal(t, "application/pdf", result.Document.MediaType)
	require.Len(t, result.Chunks, 1)
	require.Equal(t, 1, result.Chunks[0].Locator.Page)
	require.Equal(t, "PDF route evidence", result.Chunks[0].SearchText)
	require.NoError(t, result.Chunks[0].Validate())
}

func minimalTextPDF(text string) []byte {
	stream := fmt.Sprintf("BT /F1 12 Tf 72 720 Td (%s) Tj ET", text)
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
	}
	var document strings.Builder
	document.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = document.Len()
		fmt.Fprintf(&document, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}
	xrefOffset := document.Len()
	fmt.Fprintf(&document, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&document, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&document, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset)
	return []byte(document.String())
}

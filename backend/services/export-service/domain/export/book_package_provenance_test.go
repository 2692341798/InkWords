package export

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	shared "inkwords-backend/shared/kernel/textbook"
)

func TestBookPackagePreservesActualRendererVersions(t *testing.T) {
	book, err := shared.NewCanonicalBookAST("模板追溯", time.Unix(1, 0), []shared.CanonicalBookChapter{{ID: "chapter", Order: 1, Title: "章节", Markdown: "正文", ContentHash: "sha256:" + strings.Repeat("a", 64)}})
	require.NoError(t, err)
	docxVersions := map[string]string{"pandoc": "pandoc 3.8", "docx_reference_sha256": "sha256:" + strings.Repeat("b", 64)}
	pdfVersions := map[string]string{"chromium": "fixed-renderer"}
	var output bytes.Buffer
	require.NoError(t, NewBookPackageBuilder().Build(&output, BookPackageInput{Book: book,
		DOCX: &BookDOCXProjection{Content: []byte("PK\x03\x04docx"), ToolVersions: docxVersions},
		PDF:  &BookPDFProjection{Content: []byte("%PDF-1.7"), ToolVersions: pdfVersions},
	}))
	archive, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	require.NoError(t, err)
	for _, file := range archive.File {
		if file.Name != "manifest.json" {
			continue
		}
		var manifest struct {
			Renderers map[string]map[string]string `json:"renderer_tool_versions"`
		}
		require.NoError(t, json.Unmarshal([]byte(readZipFile(t, file)), &manifest))
		require.Equal(t, docxVersions, manifest.Renderers["docx"])
		require.Equal(t, pdfVersions, manifest.Renderers["pdf"])
		return
	}
	t.Fatal("missing package manifest")
}

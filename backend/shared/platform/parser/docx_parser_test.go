package parser

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractDOCXTextPreservesHeadingListTableCodeAndOrder(t *testing.T) {
	content := []byte(`<?xml version="1.0"?><w:document xmlns:w="urn:test"><w:body>
<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>路由</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr/></w:pPr><w:r><w:t>注册 GET</w:t></w:r></w:p>
<w:tbl><w:tr><w:tc><w:p><w:r><w:t>方法</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>路径</w:t></w:r></w:p></w:tc></w:tr></w:tbl>
<w:p><w:pPr><w:pStyle w:val="Code"/></w:pPr><w:r><w:t>r.GET(&quot;/ping&quot;)</w:t></w:r></w:p>
</w:body></w:document>`)
	text, err := extractDOCXText(content)
	require.NoError(t, err)
	require.Equal(t, "# 路由\n\n- 注册 GET\n\n方法\n\n路径\n\n```text\nr.GET(\"/ping\")\n```", text)
}

func TestDocParser_ParseStructuredDOCXProducesCiteableChunks(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entry, err := writer.Create("word/document.xml")
	require.NoError(t, err)
	_, err = entry.Write([]byte(`<?xml version="1.0"?><w:document xmlns:w="urn:test"><w:body><w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>安装</w:t></w:r></w:p><w:p><w:r><w:t>执行 go run .</w:t></w:r></w:p></w:body></w:document>`))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	result, err := NewDocParser().ParseStructured(bytes.NewReader(archive.Bytes()), StructuredParseRequest{
		SourceID: "source-docx", SnapshotID: "snapshot-docx", CanonicalLocator: "file:///guide.docx", ArtifactPath: "guide.docx", Filename: "guide.docx",
	})
	require.NoError(t, err)
	require.Equal(t, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", result.Document.MediaType)
	require.Len(t, result.Chunks, 1)
	require.Equal(t, []string{"安装"}, result.Chunks[0].HeadingPath)
	require.Equal(t, "执行 go run .", result.Chunks[0].SearchText)
	require.NoError(t, result.Chunks[0].Validate())
}

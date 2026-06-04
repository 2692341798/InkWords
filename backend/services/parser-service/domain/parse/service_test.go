package parse

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	parserinfra "inkwords-backend/services/parser-service/infra/parser"
)

func TestService_Parse_ReturnsArchiveSummaryForZip(t *testing.T) {
	docParser := parserinfra.NewDocParser()
	archiveParser := parserinfra.NewArchiveParser(docParser)
	service := NewService(docParser, archiveParser)

	archive := buildZipArchive(t, map[string]string{
		"01-intro.md": "# 第一课\n课程目标",
		"src/main.go": "package main\nfunc main() {}\n",
	})

	result, err := service.Parse(bytes.NewReader(archive), "courseware.zip")
	require.NoError(t, err)
	assert.NotNil(t, result.ArchiveSummary)
	assert.Contains(t, result.SourceContent, "--- 文件: src/main.go ---")
}

func TestService_Parse_OmitsArchiveSummaryForNormalFile(t *testing.T) {
	docParser := parserinfra.NewDocParser()
	archiveParser := parserinfra.NewArchiveParser(docParser)
	service := NewService(docParser, archiveParser)

	result, err := service.Parse(bytes.NewReader([]byte("# title")), "lesson.md")
	require.NoError(t, err)
	assert.Nil(t, result.ArchiveSummary)
	assert.Contains(t, result.SourceContent, "# title")
}

func buildZipArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	for name, content := range files {
		entry, err := writer.Create(name)
		require.NoError(t, err)

		_, err = entry.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, writer.Close())
	return buf.Bytes()
}

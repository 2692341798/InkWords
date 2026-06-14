package parser

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchiveParser_ParseArchive_KeepsUsefulFilesAndDeduplicates(t *testing.T) {
	parser := NewArchiveParser(NewDocParser())
	archive := buildZipArchive(t, map[string]string{
		"01-intro.md":        "# 第一课\n课程目标",
		"02-demo.txt":        "示例代码说明",
		"copies/02-demo.txt": "示例代码说明",
		"src/main.go":        "package main\nfunc main() {}\n",
		"assets/logo.png":    "not-text",
	})

	result, err := parser.ParseArchive(bytes.NewReader(archive), "courseware.zip")
	require.NoError(t, err)
	assert.Contains(t, result.SourceContent, "--- 文件: 01-intro.md ---")
	assert.Contains(t, result.SourceContent, "--- 文件: 02-demo.txt ---")
	assert.Contains(t, result.SourceContent, "--- 文件: src/main.go ---")
	assert.Equal(t, 5, result.ArchiveSummary.TotalFiles)
	assert.Equal(t, 4, result.ArchiveSummary.SupportedFiles)
	assert.Equal(t, 3, result.ArchiveSummary.KeptFiles)
	assert.Equal(t, 1, result.ArchiveSummary.DuplicateFiles)
	assert.Equal(t, 1, result.ArchiveSummary.IgnoredFiles)
}

func TestArchiveParser_ParseArchive_RejectsPathTraversal(t *testing.T) {
	parser := NewArchiveParser(NewDocParser())
	archive := buildZipArchive(t, map[string]string{
		"../../escape.md": "# bad",
	})

	_, err := parser.ParseArchive(bytes.NewReader(archive), "danger.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "非法压缩包路径")
}

func TestArchiveParser_ParseArchive_ReturnsErrorWhenNoUsefulFiles(t *testing.T) {
	parser := NewArchiveParser(NewDocParser())
	archive := buildZipArchive(t, map[string]string{
		"notes/logo.png": "binary",
		"empty.txt":      "\n\n",
	})

	_, err := parser.ParseArchive(bytes.NewReader(archive), "empty.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "压缩包中没有可解析的文本文件")
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

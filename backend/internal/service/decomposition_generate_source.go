package service

import (
	"fmt"
	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"os/exec"
	"path/filepath"
	"strings"

	"inkwords-backend/shared/platform/parser"
)

const (
	seriesChapterSourceRuneLimit = 1000000
	seriesOldContentRuneLimit    = 500000
	seriesContentTruncatedSuffix = "\n\n... [Content Truncated due to length limits] ..."
)

func resolveSeriesChapterSourceContent(sourceType, cachePath, fallbackSourceContent string, chapter blogcontracts.Chapter) string {
	if sourceType != "git" || cachePath == "" || len(chapter.Files) == 0 {
		return truncateSeriesContent(fallbackSourceContent, seriesChapterSourceRuneLimit)
	}

	var builder strings.Builder
	for _, filePath := range chapter.Files {
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}

		cmdCheck := exec.Command("git", "cat-file", "-t", "HEAD:"+filePath)
		cmdCheck.Dir = cachePath
		objectTypeBytes, err := cmdCheck.Output()
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(objectTypeBytes)) == "tree" {
			appendSeriesDirectorySource(&builder, cachePath, filePath)
			continue
		}

		appendSeriesFileSource(&builder, cachePath, filePath)
	}

	if builder.Len() == 0 {
		return truncateSeriesContent(fallbackSourceContent, seriesChapterSourceRuneLimit)
	}

	return truncateSeriesContent(builder.String(), seriesChapterSourceRuneLimit)
}

func appendSeriesDirectorySource(builder *strings.Builder, cachePath, dirPath string) {
	cmdList := exec.Command("git", "ls-tree", "-r", "--name-only", "HEAD", dirPath)
	cmdList.Dir = cachePath
	output, err := cmdList.Output()
	if err != nil {
		return
	}

	files := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	for _, filePath := range files {
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}
		appendSeriesFileSource(builder, cachePath, filePath)
	}
}

func appendSeriesFileSource(builder *strings.Builder, cachePath, filePath string) {
	if parser.IsBinaryExt(strings.ToLower(filepath.Ext(filePath))) {
		return
	}

	cmdShow := exec.Command("git", "show", "HEAD:"+filePath)
	cmdShow.Dir = cachePath
	data, err := cmdShow.Output()
	if err != nil {
		return
	}

	builder.WriteString(fmt.Sprintf("--- File: %s ---\n%s\n\n", filePath, string(data)))
}

func truncateSeriesContent(content string, runeLimit int) string {
	runes := []rune(content)
	if len(runes) <= runeLimit {
		return content
	}

	return string(runes[:runeLimit]) + seriesContentTruncatedSuffix
}

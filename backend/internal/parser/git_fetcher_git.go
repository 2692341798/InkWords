package parser

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

func buildChunksFromDirContents(dirContents map[string]*strings.Builder) []FileChunk {
	chunks := make([]FileChunk, 0, len(dirContents))
	for dir, builder := range dirContents {
		content := builder.String()
		runes := []rune(content)

		if len(runes) > maxChunkChars {
			numChunks := (len(runes) + maxChunkChars - 1) / maxChunkChars
			for i := 0; i < numChunks; i++ {
				start := i * maxChunkChars
				end := (i + 1) * maxChunkChars
				if end > len(runes) {
					end = len(runes)
				}
				chunks = append(chunks, FileChunk{
					Dir:     fmt.Sprintf("%s (Part %d/%d)", dir, i+1, numChunks),
					Content: string(runes[start:end]),
				})
			}
			continue
		}

		chunks = append(chunks, FileChunk{
			Dir:     dir,
			Content: content,
		})
	}
	return chunks
}

func (f *GitFetcher) fetchWithGitCLI(repoURL, subDir string, progressCallback func(string)) (string, []FileChunk, error) {
	cachePath, err := f.GetCachedRepoPath(repoURL, progressCallback)
	if err != nil {
		return "", nil, err
	}

	if progressCallback != nil {
		progressCallback("读取文件列表...")
	}

	args := []string{"ls-tree", "-r", "--name-only", "HEAD"}
	if subDir != "" && subDir != "." {
		args = append(args, subDir)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = cachePath
	outBytes, err := cmd.Output()
	if err != nil {
		return "", nil, fmt.Errorf("failed to list files with ls-tree: %w", err)
	}

	files := strings.Split(strings.ReplaceAll(string(outBytes), "\r\n", "\n"), "\n")

	var treeBuilder strings.Builder
	treeBuilder.WriteString("=== Repository Structure ===\n")

	dirContents := make(map[string]*strings.Builder)

	for _, path := range files {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		if isIgnoredPath(path) {
			continue
		}

		treeBuilder.WriteString("- " + path + "\n")

		cmdShow := exec.Command("git", "show", "HEAD:"+path)
		cmdShow.Dir = cachePath
		data, err := cmdShow.Output()
		if err != nil {
			continue
		}

		if !utf8.Valid(data) || bytes.Contains(data, []byte{0}) {
			continue
		}

		dir := filepath.Dir(path)
		if dir == "." {
			dir = "/"
		}

		contentStr := string(data)
		runes := []rune(contentStr)
		if len(runes) > maxChunkChars {
			contentStr = string(runes[:maxChunkChars]) + "\n\n... [File Content Truncated due to length limits] ..."
		}

		fileContent := fmt.Sprintf("--- File: %s ---\n%s\n\n", path, contentStr)

		builder, exists := dirContents[dir]
		if !exists {
			builder = &strings.Builder{}
		}

		builder.WriteString(fileContent)
		dirContents[dir] = builder
	}

	chunks := buildChunksFromDirContents(dirContents)
	if len(chunks) > maxTotalChunks {
		chunks = chunks[:maxTotalChunks]
		treeBuilder.WriteString("\n\n" + largeRepoTruncationHint)
	}

	return treeBuilder.String(), chunks, nil
}

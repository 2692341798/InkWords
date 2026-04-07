package parser

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// FileChunk represents a chunk of code, aggregated by directory or truncated if too large
type FileChunk struct {
	Dir     string
	Content string
}

// GitFetcher is responsible for cloning a Git repository, extracting text from its files,
// and then deleting the cloned repository.
type GitFetcher struct{}

// NewGitFetcher creates a new instance of GitFetcher.
func NewGitFetcher() *GitFetcher {
	return &GitFetcher{}
}

const maxChunkChars = 300000

// Fetch clones the git repository to a temporary directory, extracts text from all non-ignored files,
// aggregates them by directory into chunks, and then deletes the temporary directory.
// Returns the repository structure tree, the list of file chunks, and an error if any.
func (f *GitFetcher) Fetch(repoURL string) (string, []FileChunk, error) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "inkwords-git-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Burn after reading: guarantee the temporary directory is deleted
	defer os.RemoveAll(tempDir)

	// Clone the repository with depth 1 to speed up fetching
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tempDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", nil, fmt.Errorf("failed to clone repository: %w, stderr: %s", err, stderr.String())
	}

	// Traverse and extract text
	var treeBuilder strings.Builder
	treeBuilder.WriteString("=== Repository Structure ===\n")

	dirContents := make(map[string]strings.Builder)

	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories that we want to ignore
		if info.IsDir() {
			dirName := info.Name()
			if dirName == ".git" || dirName == "node_modules" || dirName == "dist" || dirName == "build" || dirName == ".idea" || dirName == ".vscode" || dirName == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a regular file
		if !info.Mode().IsRegular() {
			return nil
		}

		fileName := info.Name()
		// Skip known large generated or dependency files
		if fileName == "package-lock.json" || fileName == "yarn.lock" || fileName == "pnpm-lock.yaml" || fileName == "go.sum" || fileName == "Cargo.lock" {
			return nil
		}

		// Skip binary files and some extensions
		ext := strings.ToLower(filepath.Ext(path))
		if IsBinaryExt(ext) {
			return nil
		}

		relPath, _ := filepath.Rel(tempDir, path)
		treeBuilder.WriteString("- " + relPath + "\n")

		// Read file content
		data, err := os.ReadFile(path)
		if err != nil {
			// Skip files that cannot be read
			return nil
		}

		// Check if the content is valid UTF-8 and not binary
		if !utf8.Valid(data) || bytes.Contains(data, []byte{0}) {
			return nil
		}

		dir := filepath.Dir(relPath)
		if dir == "." {
			dir = "/"
		}

		// Truncate file content if it exceeds the limit
		contentStr := string(data)
		runes := []rune(contentStr)
		if len(runes) > maxChunkChars {
			contentStr = string(runes[:maxChunkChars]) + "\n\n... [File Content Truncated due to length limits] ..."
		}

		fileContent := fmt.Sprintf("--- File: %s ---\n%s\n\n", relPath, contentStr)

		// Create a new builder if it doesn't exist
		builder, exists := dirContents[dir]
		if !exists {
			builder = strings.Builder{}
		}

		// Check if adding this file will exceed the limit for the directory chunk
		if len([]rune(builder.String()))+len([]rune(fileContent)) > maxChunkChars && builder.Len() > 0 {
			// Save the current chunk and start a new one with a suffix to indicate it's split
			// But for simplicity in Map phase, we'll just keep adding and later split them
		}
		builder.WriteString(fileContent)
		dirContents[dir] = builder

		return nil
	})

	if err != nil {
		return "", nil, fmt.Errorf("failed to traverse repository files: %w", err)
	}

	var chunks []FileChunk
	for dir, builder := range dirContents {
		content := builder.String()
		runes := []rune(content)

		// If the directory content is still too large, split it
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
		} else {
			chunks = append(chunks, FileChunk{
				Dir:     dir,
				Content: content,
			})
		}
	}

	return treeBuilder.String(), chunks, nil
}

func IsBinaryExt(ext string) bool {
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true, ".ico": true, ".webp": true,
		".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
		".mp4": true, ".mp3": true, ".wav": true, ".avi": true, ".mov": true,
		".ttf": true, ".woff": true, ".woff2": true, ".eot": true,
		".pyc": true, ".class": true, ".jar": true, ".war": true,
	}
	return binaryExts[ext]
}

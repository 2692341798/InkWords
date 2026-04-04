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

// GitFetcher is responsible for cloning a Git repository, extracting text from its files,
// and then deleting the cloned repository.
type GitFetcher struct{}

// NewGitFetcher creates a new instance of GitFetcher.
func NewGitFetcher() *GitFetcher {
	return &GitFetcher{}
}

// Fetch clones the git repository to a temporary directory, extracts text from all non-ignored files,
// concatenates them, and then deletes the temporary directory.
func (f *GitFetcher) Fetch(repoURL string) (string, error) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "inkwords-git-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Burn after reading: guarantee the temporary directory is deleted
	defer os.RemoveAll(tempDir)

	// Clone the repository with depth 1 to speed up fetching
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tempDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to clone repository: %w, stderr: %s", err, stderr.String())
	}

	// Traverse and extract text
	var contentBuilder strings.Builder
	var treeBuilder strings.Builder
	
	treeBuilder.WriteString("=== Repository Structure ===\n")

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
		if isBinaryExt(ext) {
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

		contentBuilder.WriteString(fmt.Sprintf("--- File: %s ---\n", relPath))
		contentBuilder.Write(data)
		contentBuilder.WriteString("\n\n")

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to traverse repository files: %w", err)
	}

	return treeBuilder.String() + "\n=== Repository Content ===\n" + contentBuilder.String(), nil
}

func isBinaryExt(ext string) bool {
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

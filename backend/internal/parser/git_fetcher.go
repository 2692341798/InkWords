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

func isIgnoredPath(path string, info os.FileInfo) bool {
	// 忽略常见依赖和构建产物目录
	ignoredDirs := []string{
		"node_modules", "vendor", "dist", "build", "out", "target", "bin",
		".git", ".svn", ".idea", ".vscode", "__pycache__", "testdata", "docs", "examples", "scripts", "assets",
	}

	for _, dir := range ignoredDirs {
		if strings.Contains(path, "/"+dir+"/") || strings.HasPrefix(path, dir+"/") {
			return true
		}
	}

	if !info.IsDir() {
		name := strings.ToLower(info.Name())
		// 忽略测试文件
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".test.js") || strings.HasSuffix(name, ".spec.js") || strings.HasSuffix(name, ".test.ts") || strings.HasSuffix(name, ".spec.ts") {
			return true
		}
		// 忽略非代码的静态资源与文档
		ignoredExts := []string{
			".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg", ".mp4", ".mp3", ".wav", ".zip", ".tar", ".gz", ".rar", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".ttf", ".woff", ".woff2", ".eot",
		}
		for _, ext := range ignoredExts {
			if strings.HasSuffix(name, ext) {
				return true
			}
		}
	}
	return false
}

// GitFetcher is responsible for cloning a Git repository, extracting text from its files,
// and then deleting the cloned repository.
type GitFetcher struct{}

// NewGitFetcher creates a new instance of GitFetcher.
func NewGitFetcher() *GitFetcher {
	return &GitFetcher{}
}

const maxChunkChars = 300000
const maxTotalChunks = 15

const largeRepoTruncationHint = "【系统提示】由于该项目体量极其庞大，系统已执行优雅降级，自动截断了后续文件（仅保留了前15个核心模块的分块）。请你在生成的博客引言或开头中，自然地向读者说明：由于项目过于庞大，本文仅抽取分析了其核心的若干模块代码，并未包含全量内容。"

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
	cmd := exec.Command("git", "-c", "http.postBuffer=524288000", "-c", "http.maxRequestBuffer=100M", "-c", "core.compression=0", "clone", "--depth", "1", repoURL, tempDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", nil, fmt.Errorf("failed to clone repository: %w, stderr: %s", err, stderr.String())
	}

	// Traverse and extract text
	var treeBuilder strings.Builder
	treeBuilder.WriteString("=== Repository Structure ===\n")

	dirContents := make(map[string]*strings.Builder)

	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(tempDir, path)
		if relPath == "." {
			return nil
		}

		if isIgnoredPath(relPath, info) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories that we want to ignore
		if info.IsDir() {
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
			builder = &strings.Builder{}
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

	if len(chunks) > maxTotalChunks {
		chunks = chunks[:maxTotalChunks]
		treeBuilder.WriteString("\n\n" + largeRepoTruncationHint)
	}

	return treeBuilder.String(), chunks, nil
}

func (f *GitFetcher) FetchWithSubDir(repoURL string, subDir string) (string, []FileChunk, error) {
	if strings.TrimSpace(subDir) == "" {
		return f.Fetch(repoURL)
	}

	tempDir, err := os.MkdirTemp("", "inkwords-git-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	subDir = filepath.ToSlash(filepath.Clean(strings.TrimSpace(subDir)))
	subDir = strings.TrimPrefix(subDir, "/")

	var stderr bytes.Buffer
	// 加入重试机制和增大 http 缓冲区，防止大仓库拉取时因网络波动或缓冲区过小导致的 RPC failed (如 curl 56 OpenSSL SSL_read error)
	cmd := exec.Command("git", "-c", "http.postBuffer=524288000", "-c", "http.maxRequestBuffer=100M", "-c", "core.compression=0", "clone", "--filter=blob:none", "--no-checkout", "--depth", "1", repoURL, tempDir)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		stderr.Reset()
		cmd = exec.Command("git", "-c", "http.postBuffer=524288000", "-c", "http.maxRequestBuffer=100M", "-c", "core.compression=0", "clone", "--no-checkout", "--depth", "1", repoURL, tempDir)
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("failed to clone repository: %w, stderr: %s", err, stderr.String())
		}
	}

	stderr.Reset()
	cmd = exec.Command("git", "sparse-checkout", "init", "--cone")
	cmd.Dir = tempDir
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", nil, fmt.Errorf("failed to init sparse-checkout: %w, stderr: %s", err, stderr.String())
	}

	stderr.Reset()
	cmd = exec.Command("git", "sparse-checkout", "set", subDir)
	cmd.Dir = tempDir
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", nil, fmt.Errorf("failed to set sparse-checkout: %w, stderr: %s", err, stderr.String())
	}

	stderr.Reset()
	// 重试机制：对于大型仓库，checkout 可能会因网络拉取 blob 失败而报错，因此增加重试和 http 配置
	var checkoutErr error
	var checkoutStderr string
	for i := 0; i < 3; i++ {
		cmd = exec.Command("git", "-c", "http.postBuffer=524288000", "-c", "http.maxRequestBuffer=100M", "-c", "core.compression=0", "checkout")
		cmd.Dir = tempDir
		cmd.Stderr = &stderr
		checkoutErr = cmd.Run()
		if checkoutErr == nil {
			break
		}
		checkoutStderr = stderr.String()
		stderr.Reset()
	}
	if checkoutErr != nil {
		return "", nil, fmt.Errorf("failed to checkout repository after retries: %w, stderr: %s", checkoutErr, checkoutStderr)
	}

	walkDir := filepath.Join(tempDir, filepath.FromSlash(filepath.Clean(subDir)))
	if _, err := os.Stat(walkDir); err != nil {
		if os.IsNotExist(err) {
			return "", nil, fmt.Errorf("指定的子目录不存在: %s", subDir)
		}
		return "", nil, fmt.Errorf("failed to stat sub directory: %w", err)
	}

	var treeBuilder strings.Builder
	treeBuilder.WriteString("=== Repository Structure ===\n")

	dirContents := make(map[string]*strings.Builder)

	err = filepath.Walk(walkDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(tempDir, path)
		if relPath == "." {
			return nil
		}

		if isIgnoredPath(relPath, info) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		fileName := info.Name()
		if fileName == "package-lock.json" || fileName == "yarn.lock" || fileName == "pnpm-lock.yaml" || fileName == "go.sum" || fileName == "Cargo.lock" {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if IsBinaryExt(ext) {
			return nil
		}

		treeBuilder.WriteString("- " + relPath + "\n")

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		if !utf8.Valid(data) || bytes.Contains(data, []byte{0}) {
			return nil
		}

		dir := filepath.Dir(relPath)
		if dir == "." {
			dir = "/"
		}

		contentStr := string(data)
		runes := []rune(contentStr)
		if len(runes) > maxChunkChars {
			contentStr = string(runes[:maxChunkChars]) + "\n\n... [File Content Truncated due to length limits] ..."
		}

		fileContent := fmt.Sprintf("--- File: %s ---\n%s\n\n", relPath, contentStr)

		builder, exists := dirContents[dir]
		if !exists {
			builder = &strings.Builder{}
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

	if len(chunks) > maxTotalChunks {
		chunks = chunks[:maxTotalChunks]
		treeBuilder.WriteString("\n\n" + largeRepoTruncationHint)
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

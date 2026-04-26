package parser

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/sync/semaphore"
)

// FileChunk represents a chunk of code, aggregated by directory or truncated if too large
type FileChunk struct {
	Dir     string
	Content string
}

func isIgnoredPath(path string) bool {
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

	name := strings.ToLower(filepath.Base(path))
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
	return false
}

var repoCacheMu sync.Mutex

func getRepoCacheDir(gitURL string) string {
	hash := md5.Sum([]byte(gitURL))
	return filepath.Join(os.TempDir(), "inkwords_repos", fmt.Sprintf("%x", hash))
}

// GetCachedRepoPath ensures a local partial clone of the repository exists and returns its path.
func (f *GitFetcher) GetCachedRepoPath(repoURL string, progressCallback func(string)) (string, error) {
	repoCacheMu.Lock()
	defer repoCacheMu.Unlock()

	cachePath := getRepoCacheDir(repoURL)

	if _, err := os.Stat(filepath.Join(cachePath, ".git")); err == nil {
		if progressCallback != nil {
			progressCallback("使用本地仓库缓存...")
		}
		return cachePath, nil
	}

	if progressCallback != nil {
		progressCallback("开始拉取仓库数据 (浅克隆)...")
	}

	os.RemoveAll(cachePath)
	os.MkdirAll(filepath.Dir(cachePath), 0755)

	var stderr bytes.Buffer
	var stderrWriter io.Writer = &stderr
	if progressCallback != nil {
		stderrWriter = io.MultiWriter(&stderr, &progressWriter{cb: progressCallback})
	}

	cmd := exec.Command("git", "-c", "http.postBuffer=1048576000", "-c", "http.maxRequestBuffer=100M", "-c", "core.compression=0", "-c", "http.lowSpeedLimit=1000", "-c", "http.lowSpeedTime=60", "clone", "--filter=blob:none", "--no-checkout", "--depth", "1", "--single-branch", repoURL, cachePath)
	cmd.Stderr = stderrWriter
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	if err := cmd.Run(); err != nil {
		stderr.Reset()
		if progressCallback != nil {
			progressCallback("部分克隆失败，尝试完整浅克隆...")
		}
		cmd = exec.Command("git", "-c", "http.postBuffer=1048576000", "-c", "http.maxRequestBuffer=100M", "-c", "core.compression=0", "-c", "http.lowSpeedLimit=1000", "-c", "http.lowSpeedTime=60", "clone", "--no-checkout", "--depth", "1", "--single-branch", repoURL, cachePath)
		cmd.Stderr = stderrWriter
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if err := cmd.Run(); err != nil {
			os.RemoveAll(cachePath)
			return "", fmt.Errorf("failed to clone repository: %w, stderr: %s", err, stderr.String())
		}
	}

	return cachePath, nil
}

// GitFetcher is responsible for cloning a Git repository, extracting text from its files,
// and then deleting the cloned repository.
type GitFetcher struct{}

// NewGitFetcher creates a new instance of GitFetcher.
func NewGitFetcher() *GitFetcher {
	return &GitFetcher{}
}

const maxChunkChars = 2000000 // 2M chars ~ 800k tokens
const maxTotalChunks = 15

const largeRepoTruncationHint = "【系统提示】由于该项目体量极其庞大，系统已执行优雅降级，自动截断了后续文件（仅保留了前15个核心模块的分块）。请你在生成的博客引言或开头中，自然地向读者说明：由于项目过于庞大，本文仅抽取分析了其核心的若干模块代码，并未包含全量内容。"

// ParseGithubOwnerRepo extracts the owner and repository name from a GitHub URL.
func ParseGithubOwnerRepo(urlStr string) (owner, repo string, ok bool) {
	urlStr = strings.TrimSpace(urlStr)
	urlStr = strings.TrimSuffix(urlStr, ".git")
	urlStr = strings.TrimSuffix(urlStr, "/")

	if strings.HasPrefix(urlStr, "https://github.com/") || strings.HasPrefix(urlStr, "http://github.com/") {
		parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(urlStr, "https://"), "http://"), "/")
		if len(parts) >= 3 && parts[0] == "github.com" {
			return parts[1], parts[2], true
		}
	} else if strings.HasPrefix(urlStr, "git@github.com:") {
		parts := strings.Split(strings.TrimPrefix(urlStr, "git@github.com:"), "/")
		if len(parts) == 2 {
			return parts[0], parts[1], true
		}
	}
	return "", "", false
}

type GitTreeResponse struct {
	Sha       string `json:"sha"`
	Url       string `json:"url"`
	Tree      []struct {
		Path string `json:"path"`
		Type string `json:"type"`
		Size int    `json:"size"`
	} `json:"tree"`
	Truncated bool `json:"truncated"`
}

// Fetch clones the git repository to a temporary directory, extracts text from all non-ignored files,
// aggregates them by directory into chunks, and then deletes the temporary directory.
// Returns the repository structure tree, the list of file chunks, and an error if any.
func (f *GitFetcher) Fetch(repoURL string, progressCallback func(string)) (string, []FileChunk, error) {
	return f.FetchWithSubDir(repoURL, "/", progressCallback)
}

type progressWriter struct {
	cb func(string)
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	if pw.cb != nil {
		// Split by either \n or \r to handle Git's progress output
		str := string(p)
		str = strings.ReplaceAll(str, "\r", "\n")
		lines := strings.Split(str, "\n")
		for _, line := range lines {
			if line = strings.TrimSpace(line); line != "" {
				pw.cb(line)
			}
		}
	}
	return len(p), nil
}

// FetchWithSubDir fetches the repository and filters by subDir if provided.
// Returns the tree structure, chunks, and an error if any.
func (f *GitFetcher) FetchWithSubDir(repoURL string, subDir string, progressCallback func(string)) (string, []FileChunk, error) {
	subDir = filepath.ToSlash(filepath.Clean(strings.TrimSpace(subDir)))
	subDir = strings.TrimPrefix(subDir, "/")

	// 1. Try GitHub API first if it's a GitHub URL
	if owner, repo, ok := ParseGithubOwnerRepo(repoURL); ok {
		treeContent, chunks, err := f.fetchWithGithubAPI(owner, repo, subDir, progressCallback)
		if err == nil {
			return treeContent, chunks, nil
		}
		// If API fails (e.g. rate limit, or not found), fallback to git sparse-checkout
		if progressCallback != nil {
			progressCallback(fmt.Sprintf("GitHub API failed for %s/%s. Falling back to git clone...", owner, repo))
		}
		fmt.Printf("GitHub API failed for %s/%s: %v. Falling back to git clone...\n", owner, repo, err)
	}

	// 2. Use Cached Partial Clone
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

func (f *GitFetcher) fetchWithGithubAPI(owner, repo, subDir string, progressCallback func(string)) (string, []FileChunk, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/HEAD?recursive=1", owner, repo)
	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch tree: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var treeResp GitTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&treeResp); err != nil {
		return "", nil, fmt.Errorf("failed to decode tree: %w", err)
	}

	var filesToFetch []string
	prefix := subDir
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	for _, item := range treeResp.Tree {
		if item.Type == "blob" {
			if prefix == "" || strings.HasPrefix(item.Path, prefix) {
				name := filepath.Base(item.Path)
				if !isIgnoredPath(item.Path) {
					ext := strings.ToLower(filepath.Ext(item.Path))
					if !IsBinaryExt(ext) && name != "package-lock.json" && name != "yarn.lock" && name != "pnpm-lock.yaml" && name != "go.sum" && name != "Cargo.lock" {
						filesToFetch = append(filesToFetch, item.Path)
					}
				}
			}
		}
	}

	if len(filesToFetch) == 0 {
		return "", nil, fmt.Errorf("no valid files found in directory %s", subDir)
	}

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("API: 发现 %d 个待拉取文件，正在并行获取...", len(filesToFetch)))
	}

	var treeBuilder strings.Builder
	treeBuilder.WriteString("=== Repository Structure ===\n")
	for _, fPath := range filesToFetch {
		treeBuilder.WriteString("- " + fPath + "\n")
	}

	dirContents := make(map[string]*strings.Builder)
	var mu sync.Mutex

	sem := semaphore.NewWeighted(20) // max 20 concurrent requests
	var wg sync.WaitGroup
	var fetchErrs []error

	for _, fPath := range filesToFetch {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			if err := sem.Acquire(ctx, 1); err != nil {
				return
			}
			defer sem.Release(1)

			rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/HEAD/%s", owner, repo, path)
			reqRaw, _ := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
			if token := os.Getenv("GITHUB_TOKEN"); token != "" {
				reqRaw.Header.Set("Authorization", "token "+token)
			}
			
			var data []byte
			var fetchErr error
			// Retry up to 3 times
			for attempt := 0; attempt < 3; attempt++ {
				if progressCallback != nil {
					progressCallback(fmt.Sprintf("Downloading %s (attempt %d/3)...", path, attempt+1))
				}
				respRaw, err := client.Do(reqRaw)
				if err == nil && respRaw.StatusCode == http.StatusOK {
					data, fetchErr = io.ReadAll(respRaw.Body)
					respRaw.Body.Close()
					if fetchErr == nil {
						break
					}
				} else if respRaw != nil {
					respRaw.Body.Close()
					fetchErr = fmt.Errorf("status %d", respRaw.StatusCode)
				} else {
					fetchErr = err
				}
				time.Sleep(500 * time.Millisecond)
			}

			if fetchErr != nil || data == nil {
				mu.Lock()
				fetchErrs = append(fetchErrs, fmt.Errorf("failed to fetch %s: %v", path, fetchErr))
				mu.Unlock()
				return
			}

			if !utf8.Valid(data) || bytes.Contains(data, []byte{0}) {
				return
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

			mu.Lock()
			builder, exists := dirContents[dir]
			if !exists {
				builder = &strings.Builder{}
				dirContents[dir] = builder
			}
			builder.WriteString(fileContent)
			mu.Unlock()
		}(fPath)
	}

	wg.Wait()

	if len(dirContents) == 0 && len(fetchErrs) > 0 {
		return "", nil, fmt.Errorf("failed to fetch any files, errors: %v", fetchErrs[0])
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

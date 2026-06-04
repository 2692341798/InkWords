package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/sync/semaphore"
)

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

	sem := semaphore.NewWeighted(20)
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

	chunks := buildChunksFromDirContents(dirContents)
	if len(chunks) > maxTotalChunks {
		chunks = chunks[:maxTotalChunks]
		treeBuilder.WriteString("\n\n" + largeRepoTruncationHint)
	}

	return treeBuilder.String(), chunks, nil
}

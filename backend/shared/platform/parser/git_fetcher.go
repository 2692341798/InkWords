package parser

import (
	"fmt"
	"path/filepath"
	"strings"
)

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

func (f *GitFetcher) Fetch(repoURL string, progressCallback func(string)) (string, []FileChunk, error) {
	return f.FetchWithSubDir(repoURL, "/", progressCallback)
}

func (f *GitFetcher) FetchWithSubDir(repoURL string, subDir string, progressCallback func(string)) (string, []FileChunk, error) {
	result, err := f.FetchSnapshot(repoURL, subDir, "HEAD", progressCallback)
	if err != nil {
		return "", nil, err
	}
	return result.TreeContent, result.Chunks, nil
}

// FetchSnapshot 在读取内容前锁定 requestedRef 对应的 commit SHA。
// 旧 Fetch/FetchWithSubDir 保持原有返回值，但内部也经过该路径，避免 HEAD 内容在一次任务内漂移。
func (f *GitFetcher) FetchSnapshot(repoURL string, subDir string, requestedRef string, progressCallback func(string)) (GitFetchResult, error) {
	subDir = filepath.ToSlash(filepath.Clean(strings.TrimSpace(subDir)))
	subDir = strings.TrimPrefix(subDir, "/")
	requestedRef = strings.TrimSpace(requestedRef)
	if requestedRef == "" {
		requestedRef = "HEAD"
	}

	if owner, repo, ok := ParseGithubOwnerRepo(repoURL); ok {
		resolvedSHA, treeContent, chunks, err := f.fetchWithGithubAPI(owner, repo, subDir, requestedRef, progressCallback)
		if err == nil {
			return GitFetchResult{RequestedRef: requestedRef, ResolvedCommitSHA: resolvedSHA, TreeContent: treeContent, Chunks: chunks}, nil
		}
		if progressCallback != nil {
			progressCallback(fmt.Sprintf("GitHub API failed for %s/%s. Falling back to git sparse-checkout clone...", owner, repo))
		}
		fmt.Printf("GitHub API failed for %s/%s: %v. Falling back to git sparse-checkout clone...\n", owner, repo, err)
	}

	resolvedSHA, treeContent, chunks, err := f.fetchWithGitCLI(repoURL, subDir, requestedRef, progressCallback)
	if err != nil {
		return GitFetchResult{}, err
	}
	return GitFetchResult{RequestedRef: requestedRef, ResolvedCommitSHA: resolvedSHA, TreeContent: treeContent, Chunks: chunks}, nil
}

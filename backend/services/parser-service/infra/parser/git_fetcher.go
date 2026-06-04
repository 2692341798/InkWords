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
	subDir = filepath.ToSlash(filepath.Clean(strings.TrimSpace(subDir)))
	subDir = strings.TrimPrefix(subDir, "/")

	if owner, repo, ok := ParseGithubOwnerRepo(repoURL); ok {
		treeContent, chunks, err := f.fetchWithGithubAPI(owner, repo, subDir, progressCallback)
		if err == nil {
			return treeContent, chunks, nil
		}
		if progressCallback != nil {
			progressCallback(fmt.Sprintf("GitHub API failed for %s/%s. Falling back to git sparse-checkout clone...", owner, repo))
		}
		fmt.Printf("GitHub API failed for %s/%s: %v. Falling back to git sparse-checkout clone...\n", owner, repo, err)
	}

	return f.fetchWithGitCLI(repoURL, subDir, progressCallback)
}

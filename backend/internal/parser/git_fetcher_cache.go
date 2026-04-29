package parser

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var repoCacheMu sync.Mutex

func getRepoCacheDir(gitURL string) string {
	hash := md5.Sum([]byte(gitURL))
	return filepath.Join(os.TempDir(), "inkwords_repos", fmt.Sprintf("%x", hash))
}

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

	_ = os.RemoveAll(cachePath)
	_ = os.MkdirAll(filepath.Dir(cachePath), 0755)

	var stderr bytes.Buffer
	var stderrWriter io.Writer = &stderr
	if progressCallback != nil {
		stderrWriter = io.MultiWriter(&stderr, &progressWriter{cb: progressCallback})
	}

	cmd := exec.Command(
		"git",
		"-c", "http.postBuffer=1048576000",
		"-c", "http.maxRequestBuffer=100M",
		"-c", "core.compression=0",
		"-c", "http.lowSpeedLimit=1000",
		"-c", "http.lowSpeedTime=60",
		"clone", "--filter=blob:none", "--no-checkout", "--depth", "1", "--single-branch",
		repoURL, cachePath,
	)
	cmd.Stderr = stderrWriter
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	if err := cmd.Run(); err != nil {
		stderr.Reset()
		if progressCallback != nil {
			progressCallback("部分克隆失败，尝试完整浅克隆...")
		}
		cmd = exec.Command(
			"git",
			"-c", "http.postBuffer=1048576000",
			"-c", "http.maxRequestBuffer=100M",
			"-c", "core.compression=0",
			"-c", "http.lowSpeedLimit=1000",
			"-c", "http.lowSpeedTime=60",
			"clone", "--no-checkout", "--depth", "1", "--single-branch",
			repoURL, cachePath,
		)
		cmd.Stderr = stderrWriter
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if err := cmd.Run(); err != nil {
			_ = os.RemoveAll(cachePath)
			return "", fmt.Errorf("failed to clone repository: %w, stderr: %s", err, stderr.String())
		}
	}

	return cachePath, nil
}

type progressWriter struct {
	cb func(string)
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	if pw.cb != nil {
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

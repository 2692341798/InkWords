package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"inkwords-backend/internal/parser"
)

// ScanProjectModules clones the git repo without blobs (partial clone) or uses GitHub API,
// lists the root directories, and returns them. This process is very fast
// and skips LLM generation.
func (s *DecompositionService) ScanProjectModules(ctx context.Context, gitURL string) ([]ModuleCard, error) {
	return s.ScanProjectModulesWithProgress(ctx, gitURL, nil)
}

// ScanProjectModulesWithProgress clones the git repo without blobs (partial clone) or uses GitHub API,
// lists the root directories, and returns them. This process is very fast
// and skips LLM generation, emitting progress.
func (s *DecompositionService) ScanProjectModulesWithProgress(ctx context.Context, gitURL string, progressCallback chan<- string) ([]ModuleCard, error) {
	if progressCallback != nil {
		progressCallback <- "正在分析项目目录结构..."
	}

	var dirNames []string

	// 1. Try GitHub REST API first (fastest, no clone required)
	if owner, repo, ok := parser.ParseGithubOwnerRepo(gitURL); ok {
		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents", owner, repo)
		client := &http.Client{Timeout: 10 * time.Second}
		req, _ := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		
		// If GITHUB_TOKEN is available, use it to avoid rate limiting
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			var contents []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&contents); err == nil {
				for _, item := range contents {
					if item.Type == "dir" {
						dirNames = append(dirNames, item.Name)
					}
				}
			}
			resp.Body.Close()
		} else if resp != nil {
			resp.Body.Close()
		}
	}

	// 2. Fallback to git partial clone if API failed or not a GitHub URL
	if len(dirNames) == 0 {
		tempDir, err := os.MkdirTemp("", "inkwords-scan-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(tempDir)

		var stderr bytes.Buffer
		// 使用 partial clone (无 blob) 且不 checkout 任何文件，极大提升拉取速度
		cmd := exec.Command("git", "-c", "http.postBuffer=1048576000", "-c", "http.maxRequestBuffer=100M", "-c", "core.compression=0", "-c", "http.lowSpeedLimit=1000", "-c", "http.lowSpeedTime=60", "clone", "--filter=blob:none", "--no-checkout", "--depth", "1", gitURL, tempDir)
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			stderr.Reset()
			// 如果部分克隆失败（例如旧版本 Git 或不支持的 Git 服务器），降级为常规不检出的浅克隆
			cmd = exec.Command("git", "-c", "http.postBuffer=1048576000", "-c", "http.maxRequestBuffer=100M", "-c", "core.compression=0", "-c", "http.lowSpeedLimit=1000", "-c", "http.lowSpeedTime=60", "clone", "--no-checkout", "--depth", "1", gitURL, tempDir)
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("failed to clone repository: %w, stderr: %s", err, stderr.String())
			}
		}

		// 仅列出顶级目录 (Tree 对象)，不获取任何文件内容
		cmdTree := exec.Command("git", "ls-tree", "-d", "--name-only", "HEAD")
		cmdTree.Dir = tempDir
		outBytes, err := cmdTree.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to run git ls-tree: %w", err)
		}
		
		dirNames = strings.Split(strings.TrimSpace(string(outBytes)), "\n")
	}

	var modules []ModuleCard
	ignoredDirs := map[string]bool{
		".git": true, "node_modules": true, "vendor": true, "dist": true,
		"build": true, "docs": true, "assets": true, "public": true,
		"tests": true, "test": true, ".github": true, ".vscode": true,
	}

	for _, dirName := range dirNames {
		dirName = strings.TrimSpace(dirName)
		if dirName == "" || ignoredDirs[dirName] || strings.HasPrefix(dirName, ".") {
			continue
		}
		
		modules = append(modules, ModuleCard{
			Path:        dirName,
			Name:        dirName,
			Description: "代码目录模块 (点击解析后查看大纲)", // 统一使用默认描述
		})
	}

	return modules, nil
}

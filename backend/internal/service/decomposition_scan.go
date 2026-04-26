package service

import (
	"bytes"
	"context"
	"fmt"
	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/parser"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// ScanProjectModules clones the git repo, finds core directories, and generates a brief description for each
func (s *DecompositionService) ScanProjectModules(ctx context.Context, gitURL string) ([]ModuleCard, error) {
	tempDir, err := os.MkdirTemp("", "inkwords-scan-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var stderr bytes.Buffer
	// 加入重试机制和增大 http 缓冲区，防止大仓库拉取时因网络波动或缓冲区过小导致的 RPC failed
	cmd := exec.Command("git", "-c", "http.postBuffer=1048576000", "-c", "http.maxRequestBuffer=100M", "-c", "core.compression=0", "-c", "http.lowSpeedLimit=1000", "-c", "http.lowSpeedTime=60", "clone", "--depth", "1", gitURL, tempDir)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w, stderr: %s", err, stderr.String())
	}

	var modules []ModuleCard

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read repo dir: %w", err)
	}

	var coreDirs []string
	ignoredDirs := map[string]bool{
		".git": true, "node_modules": true, "vendor": true, "dist": true,
		"build": true, "docs": true, "assets": true, "public": true,
		"tests": true, "test": true, ".github": true, ".vscode": true,
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if ignoredDirs[entry.Name()] || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			coreDirs = append(coreDirs, entry.Name())
		}
	}

	if len(coreDirs) == 0 {
		return modules, nil
	}

	var mu sync.Mutex
	maxWorkers := maxWorkersFromEnv(len(coreDirs))
	if maxWorkers > 5 {
		maxWorkers = 5
	}
	sem := semaphore.NewWeighted(int64(maxWorkers))
	var wg sync.WaitGroup

	for _, dirName := range coreDirs {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := sem.Acquire(ctx, 1); err != nil {
				return
			}
			defer sem.Release(1)

			// Read some files in this dir to give LLM context
			var contentBuilder strings.Builder
			count := 0
			filepath.Walk(filepath.Join(tempDir, name), func(p string, i os.FileInfo, err error) error {
				if err != nil || i.IsDir() || !i.Mode().IsRegular() {
					return nil
				}
				if count >= 5 {
					return filepath.SkipDir
				}
				ext := strings.ToLower(filepath.Ext(p))
				if parser.IsBinaryExt(ext) {
					return nil
				}
				data, err := os.ReadFile(p)
				if err == nil {
					relPath, _ := filepath.Rel(tempDir, p)
					contentBuilder.WriteString(fmt.Sprintf("File: %s\n%s\n\n", relPath, string(data)))
					count++
				}
				return nil
			})

			content := contentBuilder.String()
			if len(content) > 10000 {
				content = content[:10000]
			}

			desc := "暂无简介"
			if content != "" {
				prompt := fmt.Sprintf("你是一个资深架构师。请根据以下目录的部分代码内容，用一句话（不超过30个字）概括该目录（%s）的核心功能职责。\n代码：\n%s", name, content)
				messages := []llm.Message{
					{Role: "user", Content: prompt},
				}
				modelStr := "deepseek-v4-flash"
				if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
					modelStr = envModel
				}
				attemptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
				if err := s.limiter.Wait(attemptCtx); err == nil {
					res, err := s.llmClient.Generate(attemptCtx, modelStr, messages)
					if err == nil && res != "" {
						desc = strings.TrimSpace(res)
					}
				}
			}

			mu.Lock()
			modules = append(modules, ModuleCard{
				Path:        name,
				Name:        name,
				Description: desc,
			})
			mu.Unlock()

		}(dirName)
	}

	wg.Wait()
	return modules, nil
}

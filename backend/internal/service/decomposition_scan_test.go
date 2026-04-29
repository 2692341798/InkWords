package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func executeCommand(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	// Set environment variables to allow git commit without global config
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	return cmd.Run()
}

func TestDecompositionService_ScanProjectModules(t *testing.T) {
	// Create a local git repository to test
	tempDir, err := os.MkdirTemp("", "inkwords-test-scan-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// We can use a file:// URL to clone the local repo
	cmdName := "git"

	err = executeCommand(tempDir, cmdName, "init")
	require.NoError(t, err)

	// Create some core directories
	err = os.MkdirAll(filepath.Join(tempDir, "frontend"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "frontend", "main.js"), []byte("console.log('frontend');"), 0644)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(tempDir, "backend"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "backend", "main.go"), []byte("package main\nfunc main() {}"), 0644)
	require.NoError(t, err)

	err = executeCommand(tempDir, cmdName, "add", ".")
	require.NoError(t, err)

	err = executeCommand(tempDir, cmdName, "commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Without DEEPSEEK_API_KEY, the LLM client might return an error or empty string,
	// but the ScanProjectModules should gracefully handle it and return "暂无简介".
	os.Unsetenv("DEEPSEEK_API_KEY")

	svc := NewDecompositionService()
	repoURL := "file://" + filepath.ToSlash(tempDir)

	modules, err := svc.ScanProjectModules(context.Background(), repoURL)
	require.NoError(t, err)

	// Should contain "frontend" and "backend"
	assert.Len(t, modules, 2)

	var names []string
	for _, m := range modules {
		names = append(names, m.Name)
		assert.Equal(t, "代码目录模块 (点击解析后查看大纲)", m.Description, "Should default to standard description")
	}

	assert.Contains(t, names, "frontend")
	assert.Contains(t, names, "backend")
}

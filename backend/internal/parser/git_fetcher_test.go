package parser

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitFetcher_Fetch(t *testing.T) {
	// Create a local git repository to test
	tempDir, err := os.MkdirTemp("", "inkwords-test-repo-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	// Create some normal files
	err = os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("# Test Repo\n"), 0644)
	require.NoError(t, err)

	// Create ignored directory
	nodeModules := filepath.Join(tempDir, "node_modules")
	err = os.MkdirAll(nodeModules, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(nodeModules, "ignore.js"), []byte("console.log('ignore me');"), 0644)
	require.NoError(t, err)

	// Create a binary file
	err = os.WriteFile(filepath.Join(tempDir, "image.png"), []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, 0644)
	require.NoError(t, err)

	// Commit files
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	// Set environment variables to allow git commit without global config
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	err = cmd.Run()
	require.NoError(t, err)

	// Instantiate the fetcher
	fetcher := NewGitFetcher()

	// Using the local directory path as a file:// URL works with git clone
	repoURL := "file://" + filepath.ToSlash(tempDir)
	content, err := fetcher.Fetch(repoURL)
	require.NoError(t, err)

	// Validate content
	assert.Contains(t, content, "package main")
	assert.Contains(t, content, "--- File: main.go ---")
	assert.Contains(t, content, "# Test Repo")
	assert.Contains(t, content, "--- File: README.md ---")

	// Validate ignored contents
	assert.NotContains(t, content, "ignore me")
	assert.NotContains(t, content, "image.png")
	assert.NotContains(t, content, "node_modules")
}

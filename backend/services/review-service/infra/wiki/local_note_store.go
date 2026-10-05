package wiki

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
)

const localVaultRootEnv = "OBSIDIAN_LOCAL_VAULT_ROOT"

// localNoteStore exposes only the read operations needed by the review domain.
// os.Root keeps every resolved file beneath the configured vault, including
// when a note path contains traversal segments or follows a symbolic link.
type localNoteStore struct {
	root *os.Root
}

func newLocalNoteStore(vaultRoot string) (*localNoteStore, error) {
	trimmedRoot := strings.TrimSpace(vaultRoot)
	if trimmedRoot == "" {
		return nil, errors.New("本地 Obsidian 知识库根目录未配置")
	}
	root, err := os.OpenRoot(trimmedRoot)
	if err != nil {
		return nil, fmt.Errorf("打开本地 Obsidian 知识库失败: %w", err)
	}
	return &localNoteStore{root: root}, nil
}

func (store *localNoteStore) Read(ctx context.Context, notePath string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cleanPath, err := cleanVaultPath(notePath)
	if err != nil {
		return nil, err
	}
	content, err := store.root.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("读取本地知识库文件失败: %w", err)
	}
	return content, nil
}

func (store *localNoteStore) List(ctx context.Context, dirPath string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cleanPath, err := cleanVaultPath(dirPath)
	if err != nil {
		return nil, err
	}
	dir, err := store.root.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("打开本地知识库目录失败: %w", err)
	}
	defer func() { _ = dir.Close() }()

	entries, err := dir.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("列出本地知识库目录失败: %w", err)
	}
	items := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		items = append(items, name)
	}
	sort.Strings(items)
	return items, nil
}

func cleanVaultPath(candidate string) (string, error) {
	cleanPath := strings.TrimSpace(strings.ReplaceAll(candidate, "\\", "/"))
	if !fs.ValidPath(cleanPath) {
		return "", fmt.Errorf("知识库路径不安全: %q", candidate)
	}
	return cleanPath, nil
}

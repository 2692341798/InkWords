package teachingartifact

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreStagesAndResolvesOneImmutableGeneratedTree(t *testing.T) {
	store := NewStoreWithMaxBytes(t.TempDir(), 1024)
	files := []File{{Path: "go.mod", Content: []byte("module example.com/demo\n\ngo 1.25\n")}, {Path: "cmd/main.go", Content: []byte("package main\n")}}
	first, err := store.Stage(context.Background(), files)
	require.NoError(t, err)
	require.Equal(t, first.Token, first.ArtifactHash)
	require.Equal(t, 2, first.FileCount)

	second, err := store.Stage(context.Background(), []File{files[1], files[0]})
	require.NoError(t, err)
	require.Equal(t, first, second)

	root, err := store.Resolve(first.Token)
	require.NoError(t, err)
	require.Equal(t, first.ArtifactHash, mustTreeHash(t, root))
	require.FileExists(t, filepath.Join(root, "cmd", "main.go"))
	_, err = store.Resolve("../../etc/passwd")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestStoreRejectsUnsafeOrOversizedGeneratedFiles(t *testing.T) {
	store := NewStoreWithMaxBytes(t.TempDir(), 3)
	_, err := store.Stage(context.Background(), []File{{Path: "../main.go", Content: []byte("ok")}})
	require.ErrorIs(t, err, ErrUnsafeFile)
	_, err = store.Stage(context.Background(), []File{{Path: "main.go", Content: []byte("1234")}})
	require.ErrorIs(t, err, ErrTooLarge)
	_, err = store.Stage(context.Background(), []File{{Path: "main.go", Content: []byte("x")}, {Path: "main.go", Content: []byte("y")}})
	require.ErrorIs(t, err, ErrUnsafeFile)
}

func TestTreeHashRejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o600))
	require.NoError(t, os.Symlink("main.go", filepath.Join(root, "link.go")))
	_, err := TreeHash(root)
	require.True(t, errors.Is(err, ErrUnsafeFile))
}

func mustTreeHash(t *testing.T, root string) string {
	t.Helper()
	hash, err := TreeHash(root)
	require.NoError(t, err)
	return hash
}

package teachingartifact

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestSharedPublisherRepairsPrivateTreeWithoutChangingItsHash(t *testing.T) {
	if os.Getgid() == 0 {
		t.Skip("test requires an unprivileged caller group")
	}
	root := t.TempDir()
	files := []File{{Path: "src/main.go", Content: []byte("package main\n")}}
	store := NewStore(root)
	original, err := store.Stage(t.Context(), files)
	require.NoError(t, err)
	artifactRoot, err := store.Resolve(original.Token)
	require.NoError(t, err)
	private, err := os.Stat(filepath.Join(artifactRoot, "src/main.go"))
	require.NoError(t, err)
	require.EqualValues(t, 0o600, private.Mode().Perm())
	shared, err := store.WithReadGroup(os.Getgid()).Stage(t.Context(), files)
	require.NoError(t, err)
	require.Equal(t, original, shared)
	require.Equal(t, original.ArtifactHash, mustTreeHash(t, artifactRoot))
	for _, path := range []string{root, artifactRoot, filepath.Join(artifactRoot, "src"), filepath.Join(artifactRoot, "src/main.go")} {
		info, err := os.Stat(path)
		require.NoError(t, err)
		if info.IsDir() {
			require.EqualValues(t, 0o750, info.Mode().Perm())
		} else {
			require.EqualValues(t, 0o640, info.Mode().Perm())
		}
	}
}

func TestReadGroupRejectsSymlinksAndInvalidGroup(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	require.NoError(t, os.WriteFile(target, []byte("private"), 0o600))
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(target, link))
	require.ErrorIs(t, setReaderPermission(link, false, 20001), ErrUnsafeFile)
	require.Error(t, setReaderPermission(target, false, 0))
	info, err := os.Stat(target)
	require.NoError(t, err)
	require.EqualValues(t, 0o600, info.Mode().Perm())
}

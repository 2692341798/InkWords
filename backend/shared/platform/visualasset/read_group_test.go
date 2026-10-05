package visualasset

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadGroupRepairsExistingPrivateVisualWithoutChangingBytes(t *testing.T) {
	if os.Getgid() == 0 {
		t.Skip("requires a nonzero caller group")
	}
	root := t.TempDir()
	store := NewStore(root)
	original, err := store.Stage(t.Context(), "image/png", bytes.NewBufferString("same visual bytes"))
	require.NoError(t, err)
	path, _, err := store.Resolve(original.Token)
	require.NoError(t, err)
	before, err := os.Stat(path)
	require.NoError(t, err)
	require.EqualValues(t, 0o600, before.Mode().Perm())
	again, err := store.WithReadGroup(os.Getgid()).Stage(t.Context(), "image/png", bytes.NewBufferString("same visual bytes"))
	require.NoError(t, err)
	require.Equal(t, original, again)
	for name, mode := range map[string]os.FileMode{root: 0o750, path: 0o640} {
		info, err := os.Stat(name)
		require.NoError(t, err)
		require.Equal(t, mode, info.Mode().Perm())
	}
	_, _, err = store.Resolve(original.Token)
	require.NoError(t, err)
}

func TestReadGroupRefusesSymlinksAndInvalidGroups(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "private")
	require.NoError(t, os.WriteFile(target, []byte("private"), 0o600))
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(target, link))
	require.Error(t, setVisualReadPermission(link, false, os.Getgid()))
	require.Error(t, setVisualReadPermission(target, false, 0))
	info, err := os.Stat(target)
	require.NoError(t, err)
	require.EqualValues(t, 0o600, info.Mode().Perm())
}

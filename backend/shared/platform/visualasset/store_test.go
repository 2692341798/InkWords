package visualasset

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStageUsesContentAddressedTypeRestrictedFiles(t *testing.T) {
	store := NewStore(t.TempDir())
	first, err := store.Stage(context.Background(), "image/png", bytes.NewBufferString("png-bytes"))
	require.NoError(t, err)
	second, err := store.Stage(context.Background(), "image/png", bytes.NewBufferString("png-bytes"))
	require.NoError(t, err)
	require.Equal(t, first.ContentHash, second.ContentHash)
	_, err = store.Stage(context.Background(), "application/pdf", bytes.NewBufferString("not an image"))
	require.ErrorIs(t, err, ErrUnsupported)
	_, err = (&Store{root: t.TempDir(), maxBytes: 2}).Stage(context.Background(), "image/png", bytes.NewBufferString("too-big"))
	require.ErrorIs(t, err, ErrTooLarge)
}

func TestResolveReadsOnlyMatchingContentAddressedVisual(t *testing.T) {
	store := NewStore(t.TempDir())
	staged, err := store.Stage(context.Background(), "image/png", bytes.NewBufferString("visual-bytes"))
	require.NoError(t, err)

	path, mediaType, err := store.Resolve(staged.ContentHash)
	require.NoError(t, err)
	require.Equal(t, "image/png", mediaType)
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "visual-bytes", string(contents))

	_, _, err = store.Resolve("sha256:" + strings.Repeat("0", 64))
	require.ErrorIs(t, err, ErrInvalidToken)
}

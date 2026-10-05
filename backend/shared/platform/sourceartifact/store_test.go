package sourceartifact

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreStagesDeduplicatedContentWithoutExposingPaths(t *testing.T) {
	store := NewStoreWithMaxBytes(t.TempDir(), 64)
	first, err := store.Stage(context.Background(), bytes.NewBufferString("教材资料"))
	require.NoError(t, err)
	require.Equal(t, "sha256:bbb93cae7b6711a4546eb6654b6db720646daee2b4cd34ba447c25939c63057b", first.Token)
	require.Equal(t, first.Token, first.ContentHash)
	require.Equal(t, int64(len([]byte("教材资料"))), first.ByteSize)

	second, err := store.Stage(context.Background(), bytes.NewBufferString("教材资料"))
	require.NoError(t, err)
	require.Equal(t, first, second)

	file, err := store.Open(first.Token)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	content, err := io.ReadAll(file)
	require.NoError(t, err)
	require.Equal(t, "教材资料", string(content))
	_, err = store.Open("../../etc/passwd")
	require.ErrorIs(t, err, ErrInvalidToken)
	_, err = store.Open(first.Token[len("sha256:"):])
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestStoreRejectsEmptyAndOversizedSources(t *testing.T) {
	store := NewStoreWithMaxBytes(t.TempDir(), 3)
	_, err := store.Stage(context.Background(), bytes.NewBuffer(nil))
	require.ErrorContains(t, err, "source is empty")
	_, err = store.Stage(context.Background(), bytes.NewBufferString("1234"))
	require.ErrorIs(t, err, ErrTooLarge)
	_, err = store.Open("sha256:not-a-digest")
	require.True(t, errors.Is(err, ErrInvalidToken))
}

package rejecteddraft

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrivateImmutableReceiptRoundTripAndIntegrity(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private")
	s, err := Open(directory)
	require.NoError(t, err)
	defer s.Close()
	data := []byte(`{"diagnostic":"原稿"}`)
	hash, err := s.Save(context.Background(), data)
	require.NoError(t, err)
	again, err := s.Save(context.Background(), data)
	require.NoError(t, err)
	require.Equal(t, hash, again)
	loaded, err := s.Load(hash)
	require.NoError(t, err)
	require.Equal(t, data, loaded)
	info, err := os.Stat(filepath.Join(directory, hash+".json"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	_, err = s.Load("../../escape")
	require.Error(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(directory, hash+".json"), []byte(`{}`), 0600))
	_, err = s.Load(hash)
	require.ErrorContains(t, err, "integrity")
	_, err = s.Save(context.Background(), data)
	require.Error(t, err, "must not overwrite damaged receipts")
	linkHash := strings.Repeat("a", 64)
	require.NoError(t, os.Symlink(filepath.Join(directory, hash+".json"), filepath.Join(directory, linkHash+".json")))
	_, err = s.Load(linkHash)
	require.Error(t, err)
}

func TestRejectedDraftQuotaAndCancellation(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "private"))
	require.NoError(t, err)
	defer s.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = s.Save(ctx, []byte(`{}`))
	require.ErrorIs(t, err, context.Canceled)
	_, err = s.Save(context.Background(), []byte(strings.Repeat(" ", MaxRecordBytes+1)))
	require.Error(t, err)
	for i := 0; i < MaxRecords; i++ {
		_, err = s.Save(context.Background(), []byte(fmt.Sprintf(`{"record":%d}`, i)))
		require.NoError(t, err)
	}
	_, err = s.Save(context.Background(), []byte(`{"extra":true}`))
	require.ErrorContains(t, err, "quota")
}

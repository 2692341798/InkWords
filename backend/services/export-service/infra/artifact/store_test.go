package artifact

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//nolint:gosec
func TestStoreSaveMovesPDFAndReturnsDownloadMetadata(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "source.pdf")
	require.NoError(t, os.WriteFile(sourcePath, []byte("pdf"), 0o644))

	store := NewStoreWithClock(t.TempDir(), 15*time.Minute, func() time.Time {
		return time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	})
	taskID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	result, err := store.Save(taskID, sourcePath, "series.pdf")
	require.NoError(t, err)
	require.Equal(t, "series.pdf", result.Filename)
	require.Equal(t, "application/pdf", result.ContentType)
	require.Equal(t, "exp_pdf_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", result.FileToken)
	require.Equal(t, time.Date(2026, 6, 3, 12, 15, 0, 0, time.UTC), result.ExpiresAt)
	require.FileExists(t, store.PathForToken(result.FileToken))
	require.NoFileExists(t, sourcePath)
}

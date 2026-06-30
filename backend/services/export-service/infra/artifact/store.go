package artifact

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	exportdomain "inkwords-backend/services/export-service/domain/export"

	"github.com/google/uuid"
)

// Store manages controlled download files in the shared export artifacts directory.
type Store struct {
	rootDir     string
	ttl         time.Duration
	nowProvider func() time.Time
}

// NewStore creates the export artifact store with export-service defaults.
func NewStore(dir string) *Store {
	return NewStoreWithClock(dir, 15*time.Minute, time.Now)
}

// NewStoreWithClock builds a store with explicit TTL and clock for tests.
func NewStoreWithClock(dir string, ttl time.Duration, nowProvider func() time.Time) *Store {
	if nowProvider == nil {
		nowProvider = time.Now
	}
	return &Store{
		rootDir:     dir,
		ttl:         ttl,
		nowProvider: nowProvider,
	}
}

// Save moves a generated PDF into the shared artifact directory.
//nolint:gosec
func (s *Store) Save(taskID uuid.UUID, sourcePath string, filename string) (exportdomain.TaskResult, error) {
	token := fmt.Sprintf("exp_pdf_%s", taskID.String())
	targetPath := s.PathForToken(token)
	if err := os.MkdirAll(s.rootDir, 0o755); err != nil {
		return exportdomain.TaskResult{}, err
	}

	if err := moveFile(sourcePath, targetPath); err != nil {
		return exportdomain.TaskResult{}, err
	}

	expiresAt := s.nowProvider().Add(s.ttl).UTC()
	return exportdomain.TaskResult{
		FileToken:   token,
		Filename:    filename,
		ContentType: "application/pdf",
		ExpiresAt:   expiresAt,
	}, nil
}

// PathForToken returns the absolute PDF path for a controlled download token.
func (s *Store) PathForToken(token string) string {
	return filepath.Join(s.rootDir, token+".pdf")
}

//nolint:gosec
func moveFile(sourcePath string, targetPath string) error {
	if err := os.Rename(sourcePath, targetPath); err == nil {
		return nil
	}

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		_ = targetFile.Close()
		_ = os.Remove(targetPath)
		return err
	}
	if err := targetFile.Close(); err != nil {
		_ = os.Remove(targetPath)
		return err
	}
	return os.Remove(sourcePath)
}

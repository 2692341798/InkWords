package bookrender

import (
	"fmt"
	"io"
	"os"

	"inkwords-backend/shared/platform/visualasset"
)

// ImageSource reads the immutable visual store through its read-only mount.
// The domain renderer verifies bytes again after reading to cover replacement
// between store resolution and the read, and validates raster decoding bounds.
type ImageSource struct{ store *visualasset.Store }

// NewImageSource accepts only the deployment-owned store, never a caller path.
func NewImageSource(store *visualasset.Store) *ImageSource { return &ImageSource{store: store} }

// ReadBookImage returns bounded bytes for the exact frozen hash.
func (s *ImageSource) ReadBookImage(hash string) ([]byte, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("frozen image store is not configured")
	}
	path, _, err := s.store.Resolve(hash)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path) //nolint:gosec // path was constrained by the content-addressed store.
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, visualasset.MaxImageBytes+1))
}

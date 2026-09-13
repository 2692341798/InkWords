// Package sourceartifact stores raw local textbook sources outside task
// payloads. Core-api writes immutable bytes; parser-service receives only a
// content-addressed token and may open the matching file read-only.
package sourceartifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MaxTextbookSourceBytes is the V1 local-file boundary from the product
// contract. Streaming keeps this limit from turning into a JSON/Base64 memory
// allocation in the browser, HTTP handler, or task broker.
const MaxTextbookSourceBytes int64 = 888 << 20

var (
	ErrTooLarge     = errors.New("textbook source exceeds the local file limit")
	ErrInvalidToken = errors.New("invalid textbook source artifact token")
)

// Artifact identifies immutable raw source bytes without exposing a local
// filesystem path to a task payload or API response.
type Artifact struct {
	Token       string
	ContentHash string
	ByteSize    int64
}

// Store is safe to share between the local core-api writer and parser-service
// reader through a dedicated volume. Files are content-addressed and never
// overwritten after the first successful write.
type Store struct {
	rootDir  string
	maxBytes int64
}

func NewStore(rootDir string) *Store {
	return NewStoreWithMaxBytes(rootDir, MaxTextbookSourceBytes)
}

// NewStoreWithMaxBytes exists for deterministic tests and local diagnostics;
// production uses NewStore and the product-level 888 MiB boundary.
func NewStoreWithMaxBytes(rootDir string, maxBytes int64) *Store {
	return &Store{rootDir: strings.TrimSpace(rootDir), maxBytes: maxBytes}
}

// Stage streams one raw source to a temporary file while hashing and bounding
// it. A completed hash becomes the only artifact token, so identical retries
// reuse the same immutable file rather than multiplying large uploads.
func (store *Store) Stage(ctx context.Context, input io.Reader) (Artifact, error) {
	if store == nil || strings.TrimSpace(store.rootDir) == "" || store.maxBytes < 1 || input == nil {
		return Artifact{}, fmt.Errorf("stage textbook source: invalid artifact store")
	}
	if err := os.MkdirAll(store.rootDir, 0o750); err != nil {
		return Artifact{}, fmt.Errorf("create source artifact directory: %w", err)
	}
	temporary, err := os.CreateTemp(store.rootDir, ".source-upload-*")
	if err != nil {
		return Artifact{}, fmt.Errorf("create source artifact: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()

	hasher := sha256.New()
	buffer := make([]byte, 64*1024)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			_ = temporary.Close()
			return Artifact{}, err
		}
		read, readErr := input.Read(buffer)
		if read > 0 {
			total += int64(read)
			if total > store.maxBytes {
				_ = temporary.Close()
				return Artifact{}, fmt.Errorf("%w: limit is %d bytes", ErrTooLarge, store.maxBytes)
			}
			if _, err := hasher.Write(buffer[:read]); err != nil {
				_ = temporary.Close()
				return Artifact{}, fmt.Errorf("hash source artifact: %w", err)
			}
			if _, err := temporary.Write(buffer[:read]); err != nil {
				_ = temporary.Close()
				return Artifact{}, fmt.Errorf("write source artifact: %w", err)
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			_ = temporary.Close()
			return Artifact{}, fmt.Errorf("read source artifact: %w", readErr)
		}
	}
	if total == 0 {
		_ = temporary.Close()
		return Artifact{}, fmt.Errorf("stage textbook source: source is empty")
	}
	if err := temporary.Chmod(0o640); err != nil {
		_ = temporary.Close()
		return Artifact{}, fmt.Errorf("set source artifact mode: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return Artifact{}, fmt.Errorf("sync source artifact: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return Artifact{}, fmt.Errorf("close source artifact: %w", err)
	}

	digest := hex.EncodeToString(hasher.Sum(nil))
	contentHash := "sha256:" + digest
	finalPath := store.pathForDigest(digest)
	if err := os.Link(temporaryPath, finalPath); err != nil && !errors.Is(err, os.ErrExist) {
		return Artifact{}, fmt.Errorf("finalize source artifact: %w", err)
	}
	return Artifact{Token: contentHash, ContentHash: contentHash, ByteSize: total}, nil
}

// Open resolves only a SHA-256 content token to a file under the configured
// root. No task-provided filename or path is ever joined into the filesystem.
func (store *Store) Open(token string) (io.ReadCloser, error) {
	if store == nil || strings.TrimSpace(store.rootDir) == "" {
		return nil, fmt.Errorf("open textbook source: invalid artifact store")
	}
	digest, err := digestFromToken(token)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(store.pathForDigest(digest))
	if err != nil {
		return nil, fmt.Errorf("open textbook source artifact: %w", err)
	}
	return file, nil
}

func (store *Store) pathForDigest(digest string) string {
	return filepath.Join(store.rootDir, digest+".source")
}

func digestFromToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, "sha256:") {
		return "", ErrInvalidToken
	}
	digest := strings.TrimPrefix(token, "sha256:")
	if len(digest) != sha256.Size*2 {
		return "", ErrInvalidToken
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", ErrInvalidToken
	}
	return strings.ToLower(digest), nil
}

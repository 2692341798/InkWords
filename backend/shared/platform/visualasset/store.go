// Package visualasset stores manually captured teaching visuals as immutable,
// content-addressed local files. It accepts bytes only; callers never choose a
// destination path or filename.
package visualasset

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

const MaxImageBytes int64 = 10 << 20

var (
	ErrTooLarge     = errors.New("teaching visual exceeds the 10 MiB limit")
	ErrUnsupported  = errors.New("unsupported teaching visual type")
	ErrInvalidToken = errors.New("invalid teaching visual token")
)

type Artifact struct {
	Token, ContentHash, MediaType string
	ByteSize                      int64
}
type Store struct {
	root      string
	maxBytes  int64
	readGroup *int
}

func NewStore(root string) *Store {
	return &Store{root: strings.TrimSpace(root), maxBytes: MaxImageBytes}
}

func (store *Store) Stage(ctx context.Context, mediaType string, source io.Reader) (Artifact, error) {
	if store == nil || store.root == "" || source == nil {
		return Artifact{}, fmt.Errorf("stage teaching visual: invalid store")
	}
	ext := extension(mediaType)
	if ext == "" {
		return Artifact{}, ErrUnsupported
	}
	if err := os.MkdirAll(store.root, 0o750); err != nil {
		return Artifact{}, err
	}
	if store.readGroup != nil {
		if err := setVisualReadPermission(store.root, true, *store.readGroup); err != nil {
			return Artifact{}, err
		}
	}
	temporary, err := os.CreateTemp(store.root, ".visual-upload-*")
	if err != nil {
		return Artifact{}, err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	hasher := sha256.New()
	reader := io.LimitReader(source, store.maxBytes+1)
	written, err := io.Copy(io.MultiWriter(temporary, hasher), reader)
	if err != nil {
		_ = temporary.Close()
		return Artifact{}, err
	}
	if written == 0 || written > store.maxBytes {
		_ = temporary.Close()
		return Artifact{}, ErrTooLarge
	}
	if err := ctx.Err(); err != nil {
		_ = temporary.Close()
		return Artifact{}, err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return Artifact{}, err
	}
	if err := temporary.Close(); err != nil {
		return Artifact{}, err
	}
	digest := hex.EncodeToString(hasher.Sum(nil))
	token := "sha256:" + digest
	if err := os.Link(temporaryPath, filepath.Join(store.root, digest+ext)); err != nil && !errors.Is(err, os.ErrExist) {
		return Artifact{}, err
	}
	if store.readGroup != nil {
		// Restaging the same bytes repairs legacy private files. Prove their
		// content before changing only the explicit publisher/reader access.
		if _, _, err := store.Resolve(token); err != nil {
			return Artifact{}, err
		}
		if err := setVisualReadPermission(filepath.Join(store.root, digest+ext), false, *store.readGroup); err != nil {
			return Artifact{}, err
		}
	}
	return Artifact{Token: token, ContentHash: token, MediaType: strings.ToLower(strings.TrimSpace(mediaType)), ByteSize: written}, nil
}

// Resolve returns a locally stored visual only after proving that its bytes
// still match the content-addressed token. It is intended for trusted local
// adapters such as the export service; HTTP handlers must never turn its
// result into a caller-controlled file path.
func (store *Store) Resolve(token string) (string, string, error) {
	if store == nil || strings.TrimSpace(store.root) == "" {
		return "", "", fmt.Errorf("resolve teaching visual: invalid store")
	}
	digest, err := digestFromToken(token)
	if err != nil {
		return "", "", err
	}
	root, err := filepath.Abs(store.root)
	if err != nil {
		return "", "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve teaching visual root: %w", err)
	}
	for extension, mediaType := range map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".webp": "image/webp"} {
		candidate := filepath.Join(resolvedRoot, digest+extension)
		resolved, resolveErr := filepath.EvalSymlinks(candidate)
		if resolveErr != nil {
			continue
		}
		relative, relativeErr := filepath.Rel(resolvedRoot, resolved)
		if relativeErr != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return "", "", ErrInvalidToken
		}
		info, statErr := os.Stat(resolved)
		if statErr != nil || !info.Mode().IsRegular() {
			return "", "", ErrInvalidToken
		}
		file, openErr := os.Open(resolved) //nolint:gosec // resolved is constrained below the content-addressed root.
		if openErr != nil {
			return "", "", openErr
		}
		hasher := sha256.New()
		_, copyErr := io.Copy(hasher, file)
		closeErr := file.Close()
		if copyErr != nil {
			return "", "", copyErr
		}
		if closeErr != nil {
			return "", "", closeErr
		}
		if hex.EncodeToString(hasher.Sum(nil)) != digest {
			return "", "", ErrInvalidToken
		}
		return resolved, mediaType, nil
	}
	return "", "", ErrInvalidToken
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

func extension(mediaType string) string {
	switch strings.ToLower(strings.TrimSpace(strings.Split(mediaType, ";")[0])) {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}

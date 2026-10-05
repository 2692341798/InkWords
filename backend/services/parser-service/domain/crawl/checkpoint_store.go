package crawl

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// CheckpointStore persists only crawl metadata. Raw fetched documents stay out
// of checkpoints, so resuming a crawl cannot turn untrusted source bytes into
// executable input or a second source-of-truth.
type CheckpointStore interface {
	Save(context.Context, checkpoint) (string, error)
	Load(context.Context, string) (checkpoint, error)
}

type checkpoint struct {
	Version  int               `json:"version"`
	Policy   Policy            `json:"policy"`
	Manifest Manifest          `json:"manifest"`
	Queue    []checkpointQueue `json:"queue"`
	Seen     []string          `json:"seen"`
	SavedAt  time.Time         `json:"saved_at"`
}

type checkpointQueue struct {
	URL    string `json:"url"`
	Depth  int    `json:"depth"`
	Parent string `json:"parent,omitempty"`
}

var checkpointIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

// FilesystemCheckpointStore provides durable local recovery without adding a
// database table to parser-service's deliberately narrow preview boundary.
type FilesystemCheckpointStore struct{ root string }

func NewFilesystemCheckpointStore(root string) (*FilesystemCheckpointStore, error) {
	if root == "" {
		return nil, fmt.Errorf("crawl checkpoint directory is required")
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("create crawl checkpoint directory: %w", err)
	}
	return &FilesystemCheckpointStore{root: root}, nil
}

func (store *FilesystemCheckpointStore) Save(ctx context.Context, value checkpoint) (string, error) {
	if store == nil {
		return "", fmt.Errorf("crawl checkpoint store is not configured")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	id, err := newCheckpointID()
	if err != nil {
		return "", err
	}
	value.Version = 1
	value.SavedAt = time.Now().UTC()
	payload, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode crawl checkpoint: %w", err)
	}
	temporary, err := os.CreateTemp(store.root, ".checkpoint-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create crawl checkpoint: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return "", fmt.Errorf("protect crawl checkpoint: %w", err)
	}
	if _, err := temporary.Write(payload); err != nil {
		temporary.Close()
		return "", fmt.Errorf("write crawl checkpoint: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return "", fmt.Errorf("sync crawl checkpoint: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close crawl checkpoint: %w", err)
	}
	if err := os.Rename(temporaryName, store.path(id)); err != nil {
		return "", fmt.Errorf("finalize crawl checkpoint: %w", err)
	}
	return id, nil
}

func (store *FilesystemCheckpointStore) Load(ctx context.Context, id string) (checkpoint, error) {
	if store == nil {
		return checkpoint{}, fmt.Errorf("crawl checkpoint store is not configured")
	}
	if err := ctx.Err(); err != nil {
		return checkpoint{}, err
	}
	if !checkpointIDPattern.MatchString(id) {
		return checkpoint{}, fmt.Errorf("invalid crawl checkpoint ID")
	}
	payload, err := os.ReadFile(store.path(id))
	if err != nil {
		return checkpoint{}, fmt.Errorf("read crawl checkpoint: %w", err)
	}
	var value checkpoint
	if err := json.Unmarshal(payload, &value); err != nil {
		return checkpoint{}, fmt.Errorf("decode crawl checkpoint: %w", err)
	}
	if value.Version != 1 {
		return checkpoint{}, fmt.Errorf("unsupported crawl checkpoint version")
	}
	if err := value.Policy.Validate(); err != nil {
		return checkpoint{}, fmt.Errorf("invalid crawl checkpoint policy: %w", err)
	}
	return value, nil
}

func (store *FilesystemCheckpointStore) path(id string) string {
	return filepath.Join(store.root, id+".json")
}

func newCheckpointID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate crawl checkpoint ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

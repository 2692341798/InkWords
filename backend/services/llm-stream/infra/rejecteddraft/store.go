// Package rejecteddraft stores private diagnostic JSON separately from caches
// and executable artifacts. Records are immutable and addressed by their bytes.
package rejecteddraft

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sync"
)

const MaxRecordBytes = 2 << 20
const MaxRecords = 32

var receiptHash = regexp.MustCompile(`^[a-f0-9]{64}$`)

// Store is owned by one local generation worker; reaching its quota refuses
// new records rather than deleting existing diagnostic content.
type Store struct {
	root *os.Root
	mu   sync.Mutex
}

func Open(directory string) (*Store, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, fmt.Errorf("create private rejected draft directory")
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return nil, fmt.Errorf("rejected draft directory must be private and not a symlink")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open rejected draft directory")
	}
	return &Store{root: root}, nil
}

func (s *Store) Close() error { return s.root.Close() }

func (s *Store) Save(ctx context.Context, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(data) == 0 || len(data) > MaxRecordBytes || !json.Valid(data) {
		return "", fmt.Errorf("invalid rejected record size or JSON")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if existing, err := s.Load(hash); err == nil && string(existing) == string(data) {
		return hash, nil
	}
	directory, err := s.root.Open(".")
	if err != nil {
		return "", fmt.Errorf("read rejected draft quota")
	}
	entries, readErr := directory.ReadDir(MaxRecords + 1)
	_ = directory.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return "", fmt.Errorf("read rejected draft quota")
	}
	if len(entries) >= MaxRecords {
		return "", fmt.Errorf("rejected draft quota reached")
	}
	f, err := s.root.OpenFile(hash+".json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("create immutable rejected draft")
	}
	_, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return "", fmt.Errorf("persist rejected draft")
	}
	return hash, nil
}

func (s *Store) Load(hash string) ([]byte, error) {
	if !receiptHash.MatchString(hash) {
		return nil, fmt.Errorf("invalid rejected draft hash")
	}
	name := hash + ".json"
	info, err := s.root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > MaxRecordBytes {
		return nil, fmt.Errorf("rejected draft unavailable")
	}
	f, err := s.root.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open rejected draft")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxRecordBytes+1))
	if err != nil || len(data) > MaxRecordBytes {
		return nil, fmt.Errorf("read rejected draft")
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != hash || !json.Valid(data) {
		return nil, fmt.Errorf("rejected draft integrity mismatch")
	}
	return data, nil
}

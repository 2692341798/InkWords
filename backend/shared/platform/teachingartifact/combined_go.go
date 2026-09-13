package teachingartifact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	dependencyLockName          = "inkwords-dependencies.json"
	dependencyLockLimit   int64 = 2 << 20
	combinedArtifactLimit       = MaxArtifactBytes + MaxGoDependencyBytes + dependencyLockLimit
	combinedArtifactFiles       = maxGoDependencyFiles + 3
)

// StageWithGoDependencies accepts only a pinned dependency inventory plus the
// two manuscript files. It never borrows unused dependency space for prose/code.
func (store *Store) StageWithGoDependencies(ctx context.Context, files []File, dependencyManifestHash string) (Artifact, error) {
	if store == nil || dependencyManifestHash == "" {
		return Artifact{}, fmt.Errorf("pinned dependency staging is not configured")
	}
	if err := validateArtifactFiles(files, dependencyManifestHash, store.maxBytes); err != nil {
		return Artifact{}, err
	}
	return store.stage(ctx, files, combinedArtifactLimit)
}

func validateArtifactFiles(files []File, dependencyHash string, teachingLimit int64) error {
	if teachingLimit < 1 || len(files) > combinedArtifactFiles {
		return ErrTooLarge
	}
	limit := teachingLimit
	if dependencyHash != "" {
		limit = combinedArtifactLimit
	}
	if _, _, err := normalizeFiles(files, limit); err != nil {
		return err
	}
	if dependencyHash == "" {
		for _, file := range files {
			if file.Path == dependencyLockName || strings.HasPrefix(file.Path, "vendor/") {
				return fmt.Errorf("dependency inventory pin is required")
			}
		}
		return nil
	}
	var lock []byte
	for _, file := range files {
		if file.Path == dependencyLockName {
			lock = file.Content
		}
	}
	if len(lock) == 0 || int64(len(lock)) > dependencyLockLimit || dependencyDigest(lock) != dependencyHash {
		return fmt.Errorf("dependency inventory pin mismatch")
	}
	var manifest GoDependencyManifest
	decoder := json.NewDecoder(bytes.NewReader(lock))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return fmt.Errorf("invalid dependency inventory: %w", err)
	}
	if decoder.Decode(new(any)) != io.EOF {
		return fmt.Errorf("trailing dependency inventory data")
	}
	index, err := dependencyFileIndex(manifest)
	if err != nil {
		return err
	}
	dependencies := make([]File, 0, len(index))
	var teachingBytes int64
	mainSeen, testSeen := false, false
	for _, file := range files {
		if file.Path == dependencyLockName {
			continue
		}
		if pinned, ok := index[file.Path]; ok {
			if int64(len(file.Content)) != pinned.Bytes || dependencyDigest(file.Content) != pinned.SHA256 {
				return fmt.Errorf("dependency file does not match pinned inventory")
			}
			dependencies = append(dependencies, file)
			continue
		}
		switch file.Path {
		case "main.go":
			mainSeen = true
		case "main_test.go":
			testSeen = true
		default:
			return fmt.Errorf("unexpected file outside dependency inventory")
		}
		teachingBytes += int64(len(file.Content))
		if teachingBytes > teachingLimit || teachingBytes > MaxArtifactBytes {
			return ErrTooLarge
		}
	}
	if !mainSeen || !testSeen || len(dependencies) != len(index) {
		return fmt.Errorf("combined teaching files are missing")
	}
	return validateDependencyMetadata(manifest, dependencies)
}

func readBoundedArtifactFiles(ctx context.Context, root string, limit int64) ([]File, error) {
	info, err := os.Lstat(root)
	if err != nil || !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrUnsafeFile
	}
	files := make([]File, 0)
	var total int64
	entries := 0
	err = filepath.WalkDir(root, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		entries++
		if entries > combinedArtifactFiles*2 {
			return ErrTooLarge
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return ErrUnsafeFile
		}
		if len(files) >= combinedArtifactFiles {
			return ErrTooLarge
		}
		file, err := os.Open(name)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(io.LimitReader(file, limit-total+1))
		closeErr := file.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		total += int64(len(data))
		if total > limit {
			return ErrTooLarge
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		files = append(files, File{Path: filepath.ToSlash(relative), Content: data})
		return nil
	})
	return files, err
}

// VerifiedSnapshot detaches bounded, hash-checked bytes into a private temporary
// tree. The caller must clean it up after all command executions complete.
func VerifiedSnapshot(ctx context.Context, root, artifactHash, dependencyHash string) (string, func(), error) {
	limit := MaxArtifactBytes
	if dependencyHash != "" {
		limit = combinedArtifactLimit
	}
	files, err := readBoundedArtifactFiles(ctx, root, limit)
	if err != nil {
		return "", nil, err
	}
	if err := validateArtifactFiles(files, dependencyHash, MaxArtifactBytes); err != nil {
		return "", nil, err
	}
	temporary, err := os.MkdirTemp("", "inkwords-verified-teaching-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(temporary) }
	// The staging copy is private and no generated process has started yet.
	staged, err := NewStore(temporary).stage(ctx, files, limit)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	if staged.ArtifactHash != artifactHash {
		cleanup()
		return "", nil, fmt.Errorf("teaching snapshot hash does not match manifest")
	}
	digest, err := tokenDigest(staged.Token)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return filepath.Join(temporary, digest), cleanup, nil
}

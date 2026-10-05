package teachingartifact

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// CopyBoundedTree copies an already-authorized snapshot into an empty temporary
// workspace and verifies identical bytes. Reading its embedded pin is an
// integrity check, not authorization to run an arbitrary source directory.
func CopyBoundedTree(ctx context.Context, source, destination string) error {
	expected, err := TreeHash(source)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) != 0 {
		return fmt.Errorf("snapshot destination must be an empty directory")
	}
	files, err := readBoundedArtifactFiles(ctx, source, combinedArtifactLimit)
	if err != nil {
		return err
	}
	dependencyHash := ""
	for _, file := range files {
		if file.Path == dependencyLockName {
			dependencyHash = dependencyDigest(file.Content)
		}
	}
	if err := validateArtifactFiles(files, dependencyHash, MaxArtifactBytes); err != nil {
		return err
	}
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		target := filepath.Join(destination, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, file.Content, 0o644); err != nil {
			return err
		}
	}
	actual, err := TreeHash(destination)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("copied artifact differs from the selected snapshot")
	}
	return nil
}

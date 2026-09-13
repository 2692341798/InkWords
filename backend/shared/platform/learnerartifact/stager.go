// Package learnerartifact builds one ephemeral execution tree from frozen
// learner bytes. It never accepts a source directory and never persists a tree.
package learnerartifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type Workspace struct {
	RootDir  string
	TreeHash string
	Cleanup  func() error
}

// Stage writes only the source and runner-derived files already bound by the
// plan. The caller owns Cleanup and must call it before returning a report.
func Stage(ctx context.Context, artifact sharedtextbook.LearnerArtifact, plan sharedtextbook.LearnerVerificationPlan) (Workspace, error) {
	if err := artifact.Validate(); err != nil || artifact.SnapshotHash != plan.SnapshotHash {
		return Workspace{}, fmt.Errorf("stage learner artifact: frozen snapshot mismatch")
	}
	files := append(append([]sharedtextbook.LearnerCodeFile(nil), artifact.Files...), plan.DerivedFiles...)
	hash, err := sharedtextbook.LearnerFilesHash(files)
	if err != nil || hash != plan.ExecutionFilesHash {
		return Workspace{}, fmt.Errorf("stage learner artifact: execution files mismatch")
	}
	normalized, err := sharedtextbook.NormalizeLearnerFiles(files)
	if err != nil {
		return Workspace{}, err
	}
	root, err := os.MkdirTemp("", "inkwords-learner-run-")
	if err != nil {
		return Workspace{}, fmt.Errorf("create learner workspace: %w", err)
	}
	cleanup := func() error { return os.RemoveAll(root) }
	for _, file := range normalized {
		if err := ctx.Err(); err != nil {
			_ = cleanup()
			return Workspace{}, err
		}
		target := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			_ = cleanup()
			return Workspace{}, fmt.Errorf("create learner directory: %w", err)
		}
		if err := os.WriteFile(target, []byte(file.Content), 0o600); err != nil {
			_ = cleanup()
			return Workspace{}, fmt.Errorf("write learner file: %w", err)
		}
	}
	treeHash, err := TreeHash(root)
	if err != nil {
		_ = cleanup()
		return Workspace{}, err
	}
	return Workspace{RootDir: root, TreeHash: treeHash, Cleanup: cleanup}, nil
}

// TreeHash binds directories, paths and exact bytes. It is intentionally kept
// separate from the persistent generated-teaching-artifact store.
func TreeHash(root string) (string, error) {
	if strings.TrimSpace(root) == "" || !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) {
		return "", fmt.Errorf("invalid learner execution tree")
	}
	entries := make([]string, 0)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported learner execution tree entry")
		}
		entries = append(entries, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(entries)
	hasher := sha256.New()
	fileCount := 0
	for _, relative := range entries {
		path := filepath.Join(root, filepath.FromSlash(relative))
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			_, _ = io.WriteString(hasher, "dir\x00"+relative+"\x00")
			continue
		}
		input, err := os.Open(path) //nolint:gosec // path was enumerated under the fresh temporary root.
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(hasher, "file\x00"+relative+"\x00")
		_, copyErr := io.Copy(hasher, input)
		closeErr := input.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		_, _ = io.WriteString(hasher, "\x00")
		fileCount++
	}
	if fileCount == 0 {
		return "", fmt.Errorf("empty learner execution tree")
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

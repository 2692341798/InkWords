package teachingartifact

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// VerifiedManifestSnapshot also binds the dependency toolchain to the persisted
// execution manifest. No mismatched profile reaches a command executor.
func VerifiedManifestSnapshot(ctx context.Context, root string, manifest sharedtextbook.TeachingArtifactManifest) (string, func(), error) {
	if err := manifest.Validate(); err != nil {
		return "", nil, err
	}
	snapshot, cleanup, err := VerifiedSnapshot(ctx, root, manifest.ArtifactHash, manifest.DependencyManifestHash)
	if err != nil {
		return "", nil, err
	}
	if manifest.DependencyManifestHash != "" {
		file, err := os.Open(filepath.Join(snapshot, dependencyLockName))
		if err != nil {
			cleanup()
			return "", nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(file, dependencyLockLimit+1))
		closeErr := file.Close()
		if readErr != nil {
			cleanup()
			return "", nil, readErr
		}
		if closeErr != nil {
			cleanup()
			return "", nil, closeErr
		}
		var dependency GoDependencyManifest
		if json.Unmarshal(data, &dependency) != nil || dependency.Toolchain != manifest.ToolchainVersion {
			cleanup()
			return "", nil, fmt.Errorf("dependency toolchain does not match execution manifest")
		}
		if s := manifest.DependencySelection; s != nil && dependency.Origin != (GoDependencyOrigin{Module: s.Module, Version: s.Version, Commit: s.Snapshot.ResolvedVersion}) {
			cleanup()
			return "", nil, fmt.Errorf("dependency origin does not match selected source")
		}
	}
	return snapshot, cleanup, nil
}

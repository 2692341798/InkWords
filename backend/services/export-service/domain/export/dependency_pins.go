package export

import (
	"encoding/json"
	"fmt"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func artifactDependencyPin(raw json.RawMessage, id, artifactHash, manifestHash string) (string, error) {
	var manifest sharedtextbook.TeachingArtifactManifest
	// Legacy unverified artifacts remain exportable under their original small
	// budget. A larger tree always needs a complete, matching manifest below.
	if json.Unmarshal(raw, &manifest) != nil || manifest.DependencyManifestHash == "" {
		return "", nil
	}
	hash, err := sharedtextbook.TeachingArtifactManifestHash(manifest)
	if err != nil || hash != manifestHash || manifest.ArtifactID != id || manifest.ArtifactHash != artifactHash {
		return "", fmt.Errorf("offline dependency artifact manifest binding mismatch")
	}
	return manifest.DependencyManifestHash, nil
}

func frozenDependencyPins(raw json.RawMessage) (map[string]string, error) {
	result := map[string]string{}
	snapshot, err := sharedtextbook.ReadBookVerificationSnapshot(raw)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return result, nil
	}
	for _, artifact := range snapshot.Artifacts {
		pin, err := artifactDependencyPin(artifact.Manifest, artifact.ID, artifact.ArtifactHash, artifact.ManifestHash)
		if err != nil {
			return nil, err
		}
		result[artifact.ID] = pin
	}
	return result, nil
}

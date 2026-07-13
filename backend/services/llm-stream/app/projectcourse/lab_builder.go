package projectcourse

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

type LabPatch struct {
	Path    string
	Content string
}

// BuildLabManifest builds structure only. Verification is a separate, sandboxed concern.
func BuildLabManifest(language, toolchain string, starter []LabPatch, checkpointPatches [][]LabPatch, tests []LabPatch) (sharedkernel.LabManifest, error) {
	manifest := sharedkernel.LabManifest{Language: language, ToolchainVersion: toolchain, AllowedCommands: []string{"test"}, ResourceLimits: map[string]string{"timeout_seconds": "30", "memory_mb": "256"}, Tests: toLabFiles(tests), DependencyGraph: map[string][]string{}}
	manifest.Starter = toLabFiles(starter)
	if len(checkpointPatches) == 0 {
		return sharedkernel.LabManifest{}, fmt.Errorf("at least one checkpoint is required")
	}
	previous := ""
	for index, patches := range checkpointPatches {
		id := fmt.Sprintf("checkpoint-%02d", index+1)
		files := toLabFiles(patches)
		manifest.Checkpoints = append(manifest.Checkpoints, sharedkernel.LabCheckpoint{ID: id, PreviousID: previous, Files: files})
		if previous != "" {
			manifest.DependencyGraph[id] = []string{previous}
		}
		previous = id
	}
	manifest.Solution = append([]sharedkernel.LabFile(nil), manifest.Checkpoints[len(manifest.Checkpoints)-1].Files...)
	if err := manifest.Validate(); err != nil {
		return sharedkernel.LabManifest{}, err
	}
	return manifest, nil
}

func toLabFiles(files []LabPatch) []sharedkernel.LabFile {
	result := make([]sharedkernel.LabFile, 0, len(files))
	for _, file := range files {
		sum := sha256.Sum256([]byte(file.Content))
		result = append(result, sharedkernel.LabFile{Path: file.Path, Content: file.Content, Hash: "sha256:" + hex.EncodeToString(sum[:])})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

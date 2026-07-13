package projectcourse

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

type LabPatch struct {
	Path    string
	Content string
}

// BuildLabManifest builds structure only. Verification is a separate, sandboxed concern.
func BuildLabManifest(language, toolchain string, starter []LabPatch, checkpointPatches [][]LabPatch, tests []LabPatch) (sharedkernel.LabManifest, error) {
	manifest := sharedkernel.LabManifest{Language: language, ToolchainVersion: toolchain, CoreTechnologies: []string{language}, ExcludedScope: []string{"生产环境部署编排、外部凭据和非核心集成"}, AllowedCommands: []string{"test"}, ResourceLimits: map[string]string{"timeout_seconds": "30", "memory_mb": "256", "pids": "64", "output_bytes": "1048576"}, DependencyGraph: map[string][]string{}}
	var err error
	manifest.Tests, err = toLabFiles(tests)
	if err != nil {
		return sharedkernel.LabManifest{}, fmt.Errorf("tests: %w", err)
	}
	manifest.Starter, err = toLabFiles(starter)
	if err != nil {
		return sharedkernel.LabManifest{}, fmt.Errorf("starter: %w", err)
	}
	if len(manifest.Starter) == 0 || len(manifest.Tests) == 0 {
		return sharedkernel.LabManifest{}, fmt.Errorf("starter and tests are required")
	}
	if len(checkpointPatches) == 0 {
		return sharedkernel.LabManifest{}, fmt.Errorf("at least one checkpoint is required")
	}
	workspace := make(map[string]LabPatch, len(starter))
	for _, file := range starter {
		workspace[file.Path] = file
	}
	previous := ""
	for index, patches := range checkpointPatches {
		id := fmt.Sprintf("checkpoint-%02d", index+1)
		for _, patch := range patches {
			workspace[patch.Path] = patch
		}
		current := make([]LabPatch, 0, len(workspace))
		for _, file := range workspace {
			current = append(current, file)
		}
		files, err := toLabFiles(current)
		if err != nil {
			return sharedkernel.LabManifest{}, fmt.Errorf("checkpoint %s: %w", id, err)
		}
		manifest.Checkpoints = append(manifest.Checkpoints, sharedkernel.LabCheckpoint{ID: id, PreviousID: previous, Files: files})
		exercise := sharedkernel.LabExercise{ExerciseID: "exercise-" + id, CheckpointBefore: previous, CheckpointAfter: id, Task: "完成并验证 " + id, AcceptanceTests: []string{"test"}, SolutionRef: "solution/" + id, Hints: []sharedkernel.LabHint{{Level: 1, Content: "先确认验收测试观察什么行为。"}, {Level: 2, Content: "定位本检查点涉及的模块或接口。"}, {Level: 3, Content: "写出最小实现骨架，再运行验收测试。"}}}
		if previous == "" {
			exercise.CheckpointBefore = id
		}
		manifest.Exercises = append(manifest.Exercises, exercise)
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

func toLabFiles(files []LabPatch) ([]sharedkernel.LabFile, error) {
	result := make([]sharedkernel.LabFile, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		clean := path.Clean(strings.ReplaceAll(strings.TrimSpace(file.Path), "\\", "/"))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
			return nil, fmt.Errorf("unsafe lab file path %q", file.Path)
		}
		if _, ok := seen[clean]; ok {
			return nil, fmt.Errorf("duplicate lab file path %q", clean)
		}
		seen[clean] = struct{}{}
		sum := sha256.Sum256([]byte(file.Content))
		result = append(result, sharedkernel.LabFile{Path: clean, Content: file.Content, Hash: "sha256:" + hex.EncodeToString(sum[:])})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}

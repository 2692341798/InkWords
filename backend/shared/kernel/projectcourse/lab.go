package projectcourse

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
)

type LabFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Hash    string `json:"hash"`
}

type LabCheckpoint struct {
	ID         string    `json:"id"`
	PreviousID string    `json:"previous_id,omitempty"`
	Files      []LabFile `json:"files"`
	Verified   bool      `json:"verified"`
}

type LabHint struct {
	Level   int    `json:"level"`
	Content string `json:"content"`
}

type LabExercise struct {
	ExerciseID       string    `json:"exercise_id"`
	CheckpointBefore string    `json:"checkpoint_before"`
	CheckpointAfter  string    `json:"checkpoint_after"`
	Task             string    `json:"task"`
	AcceptanceTests  []string  `json:"acceptance_tests"`
	Hints            []LabHint `json:"hints"`
	SolutionRef      string    `json:"solution_ref"`
}

type LabManifest struct {
	Language               string              `json:"language"`
	ToolchainVersion       string              `json:"toolchain_version"`
	CoreTechnologies       []string            `json:"core_technologies,omitempty"`
	ExcludedScope          []string            `json:"excluded_scope,omitempty"`
	StarterExpectedFailure bool                `json:"starter_expected_failure,omitempty"`
	VariantTask            string              `json:"variant_task,omitempty"`
	AllowedCommands        []string            `json:"allowed_commands"`
	ResourceLimits         map[string]string   `json:"resource_limits"`
	Starter                []LabFile           `json:"starter"`
	Checkpoints            []LabCheckpoint     `json:"checkpoints"`
	Exercises              []LabExercise       `json:"exercises"`
	Solution               []LabFile           `json:"solution"`
	Tests                  []LabFile           `json:"tests"`
	DependencyGraph        map[string][]string `json:"dependency_graph"`
}

func (m LabManifest) Validate() error {
	if strings.TrimSpace(m.Language) == "" || strings.TrimSpace(m.ToolchainVersion) == "" {
		return fmt.Errorf("lab language and toolchain version are required")
	}
	if len(m.CoreTechnologies) > 0 && len(m.ExcludedScope) == 0 {
		return fmt.Errorf("lab scope must document excluded complexity")
	}
	if len(m.CoreTechnologies) > 0 && (!m.StarterExpectedFailure || strings.TrimSpace(m.VariantTask) == "") {
		return fmt.Errorf("lab must define a starter failure and variant task")
	}
	if len(m.AllowedCommands) == 0 {
		return fmt.Errorf("lab allowed commands are required")
	}
	for _, command := range m.AllowedCommands {
		if strings.ContainsAny(command, ";&|`$\n") {
			return fmt.Errorf("unsafe lab command %q", command)
		}
	}
	if len(m.ResourceLimits) == 0 {
		return fmt.Errorf("lab resource limits are required")
	}
	if len(m.Starter) == 0 || len(m.Checkpoints) == 0 || len(m.Solution) == 0 || len(m.Tests) == 0 {
		return fmt.Errorf("lab must include starter, checkpoints, solution and tests")
	}
	seen := make(map[string]bool, len(m.Checkpoints))
	for _, checkpoint := range m.Checkpoints {
		if checkpoint.ID == "" || seen[checkpoint.ID] {
			return fmt.Errorf("duplicate or empty checkpoint id")
		}
		seen[checkpoint.ID] = true
		if checkpoint.PreviousID != "" && !seen[checkpoint.PreviousID] {
			return fmt.Errorf("checkpoint %q has a forward dependency", checkpoint.ID)
		}
		if err := validateLabFiles(checkpoint.Files); err != nil {
			return fmt.Errorf("checkpoint %q: %w", checkpoint.ID, err)
		}
	}
	if err := validateLabFiles(m.Starter); err != nil {
		return fmt.Errorf("starter: %w", err)
	}
	if err := validateLabFiles(m.Solution); err != nil {
		return fmt.Errorf("solution: %w", err)
	}
	if err := validateLabFiles(m.Tests); err != nil {
		return fmt.Errorf("tests: %w", err)
	}
	for _, exercise := range m.Exercises {
		if strings.TrimSpace(exercise.ExerciseID) == "" || strings.TrimSpace(exercise.Task) == "" || strings.TrimSpace(exercise.SolutionRef) == "" || len(exercise.AcceptanceTests) == 0 {
			return fmt.Errorf("exercise %q is incomplete", exercise.ExerciseID)
		}
		if !seen[exercise.CheckpointBefore] || !seen[exercise.CheckpointAfter] {
			return fmt.Errorf("exercise %q references unknown checkpoint", exercise.ExerciseID)
		}
		for _, hint := range exercise.Hints {
			if hint.Level < 1 || hint.Level > 3 {
				return fmt.Errorf("exercise %q has invalid hint level", exercise.ExerciseID)
			}
		}
	}
	return nil
}

func validateLabFiles(files []LabFile) error {
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		clean := path.Clean(strings.ReplaceAll(strings.TrimSpace(file.Path), "\\", "/"))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || clean != file.Path {
			return fmt.Errorf("unsafe lab file path %q", file.Path)
		}
		if _, ok := seen[clean]; ok {
			return fmt.Errorf("duplicate lab file path %q", clean)
		}
		seen[clean] = struct{}{}
		if file.Hash != "" {
			sum := sha256.Sum256([]byte(file.Content))
			if file.Hash != "sha256:"+hex.EncodeToString(sum[:]) {
				return fmt.Errorf("file %q hash does not match content", file.Path)
			}
		}
	}
	return nil
}

package projectcourse

import (
	"fmt"
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
	Language         string              `json:"language"`
	ToolchainVersion string              `json:"toolchain_version"`
	AllowedCommands  []string            `json:"allowed_commands"`
	ResourceLimits   map[string]string   `json:"resource_limits"`
	Starter          []LabFile           `json:"starter"`
	Checkpoints      []LabCheckpoint     `json:"checkpoints"`
	Exercises        []LabExercise       `json:"exercises"`
	Solution         []LabFile           `json:"solution"`
	Tests            []LabFile           `json:"tests"`
	DependencyGraph  map[string][]string `json:"dependency_graph"`
}

func (m LabManifest) Validate() error {
	if strings.TrimSpace(m.Language) == "" || strings.TrimSpace(m.ToolchainVersion) == "" {
		return fmt.Errorf("lab language and toolchain version are required")
	}
	if len(m.AllowedCommands) == 0 {
		return fmt.Errorf("lab allowed commands are required")
	}
	for _, command := range m.AllowedCommands {
		if strings.ContainsAny(command, ";&|`$\n") {
			return fmt.Errorf("unsafe lab command %q", command)
		}
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
	}
	for _, exercise := range m.Exercises {
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

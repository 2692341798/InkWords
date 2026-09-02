package verification

import (
	"time"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

type RunRequest struct {
	Manifest sharedkernel.LabManifest
	RootDir  string
}

type CommandResult struct {
	Command  string        `json:"command"`
	ExitCode int           `json:"exit_code"`
	Output   string        `json:"output,omitempty"`
	Duration time.Duration `json:"duration"`
}

type Report struct {
	Passed       bool            `json:"passed"`
	CheckpointID string          `json:"checkpoint_id,omitempty"`
	Results      []CommandResult `json:"results"`
	Error        string          `json:"error,omitempty"`
}

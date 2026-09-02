package projectcourse

import (
	"fmt"
	"strings"
)

type CourseCheckpoint struct {
	CourseID         string `json:"course_id"`
	BlueprintVersion int    `json:"blueprint_version"`
	Stage            string `json:"stage"`
	Sequence         int    `json:"sequence"`
	Checkpoint       string `json:"checkpoint"`
	InputHash        string `json:"input_hash"`
	OutputHash       string `json:"output_hash,omitempty"`
	Completed        bool   `json:"completed"`
}

func (c CourseCheckpoint) Validate() error {
	if strings.TrimSpace(c.CourseID) == "" || strings.TrimSpace(c.Stage) == "" || strings.TrimSpace(c.Checkpoint) == "" || c.Sequence < 1 {
		return fmt.Errorf("course checkpoint identity is incomplete")
	}
	if c.BlueprintVersion < 1 || !strings.HasPrefix(c.InputHash, "sha256:") {
		return fmt.Errorf("course checkpoint identity or input hash is invalid")
	}
	if c.Completed && !strings.HasPrefix(c.OutputHash, "sha256:") {
		return fmt.Errorf("completed course checkpoint requires output hash")
	}
	return nil
}

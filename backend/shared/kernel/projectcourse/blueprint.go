package projectcourse

import (
	"fmt"
	"strings"
)

type LearningOutcome struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type LabSpec struct {
	ExerciseID       string   `json:"exercise_id"`
	CheckpointBefore string   `json:"checkpoint_before"`
	CheckpointAfter  string   `json:"checkpoint_after"`
	AcceptanceTests  []string `json:"acceptance_tests"`
}

type Chapter struct {
	ID               string            `json:"chapter_id"`
	Title            string            `json:"title"`
	Sort             int               `json:"sort"`
	Enabled          bool              `json:"enabled"`
	Type             ChapterType       `json:"chapter_type"`
	PrerequisiteIDs  []string          `json:"prerequisite_ids"`
	LearningOutcomes []LearningOutcome `json:"learning_outcomes"`
	EvidenceIDs      []string          `json:"evidence_ids"`
	Lab              *LabSpec          `json:"lab,omitempty"`
}

type Volume struct {
	ID       string    `json:"volume_id"`
	Title    string    `json:"title"`
	Sort     int       `json:"sort"`
	Chapters []Chapter `json:"chapters"`
}

type Blueprint struct {
	CourseID         string        `json:"course_id"`
	BlueprintVersion int           `json:"blueprint_version"`
	CommitSHA        string        `json:"commit_sha"`
	AudienceLevel    AudienceLevel `json:"audience_level"`
	Volumes          []Volume      `json:"volumes"`
}

//nolint:gocyclo // Validation intentionally enumerates the complete persisted blueprint contract.
func (b Blueprint) Validate() error {
	if b.CourseID == "" || b.BlueprintVersion < 1 || b.CommitSHA == "" {
		return fmt.Errorf("course_id, positive blueprint_version and commit_sha are required")
	}
	if err := b.AudienceLevel.Validate(); err != nil {
		return err
	}
	knownChapters := make(map[string]bool)
	var ordered []Chapter
	for _, volume := range b.Volumes {
		if strings.TrimSpace(volume.ID) == "" || strings.TrimSpace(volume.Title) == "" || volume.Sort < 1 {
			return fmt.Errorf("invalid course volume")
		}
		for _, chapter := range volume.Chapters {
			if err := chapter.Type.Validate(); err != nil {
				return err
			}
			if strings.TrimSpace(chapter.ID) == "" || strings.TrimSpace(chapter.Title) == "" || chapter.Sort < 1 || knownChapters[chapter.ID] {
				return fmt.Errorf("invalid or duplicate chapter %q", chapter.ID)
			}
			knownChapters[chapter.ID] = true
			ordered = append(ordered, chapter)
		}
	}
	if len(ordered) > 0 && ordered[0].Type != ChapterProjectMap {
		return fmt.Errorf("project map chapter must be first")
	}
	for _, chapter := range ordered {
		for _, prerequisite := range chapter.PrerequisiteIDs {
			if !knownChapters[prerequisite] || prerequisite == chapter.ID {
				return fmt.Errorf("chapter %q has invalid prerequisite %q", chapter.ID, prerequisite)
			}
		}
	}
	if hasDependencyCycle(ordered) {
		return fmt.Errorf("blueprint chapter prerequisites contain a cycle")
	}
	return nil
}

func hasDependencyCycle(chapters []Chapter) bool {
	byID := make(map[string]Chapter, len(chapters))
	for _, chapter := range chapters {
		byID[chapter.ID] = chapter
	}
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return true
		}
		if visited[id] {
			return false
		}
		visiting[id] = true
		for _, prerequisite := range byID[id].PrerequisiteIDs {
			if visit(prerequisite) {
				return true
			}
		}
		visiting[id] = false
		visited[id] = true
		return false
	}
	for _, chapter := range chapters {
		if visit(chapter.ID) {
			return true
		}
	}
	return false
}

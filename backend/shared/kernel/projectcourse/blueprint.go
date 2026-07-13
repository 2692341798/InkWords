package projectcourse

import "fmt"

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

func (b Blueprint) Validate() error {
	if b.CourseID == "" || b.BlueprintVersion < 1 || b.CommitSHA == "" {
		return fmt.Errorf("course_id, positive blueprint_version and commit_sha are required")
	}
	if err := b.AudienceLevel.Validate(); err != nil {
		return err
	}
	for _, volume := range b.Volumes {
		for _, chapter := range volume.Chapters {
			if err := chapter.Type.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

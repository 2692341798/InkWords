package projectcourse

import "encoding/json"

type GateReport struct {
	Name    string     `json:"name"`
	Result  GateResult `json:"result"`
	Message string     `json:"message,omitempty"`
}

type CourseResult struct {
	CourseID         string       `json:"course_id"`
	BlueprintVersion int          `json:"blueprint_version"`
	CommitSHA        string       `json:"commit_sha"`
	Status           CourseStatus `json:"status"`
	Gates            []GateReport `json:"gates"`
}

type ChapterResultStatus string

const (
	ChapterSucceeded ChapterResultStatus = "succeeded"
	ChapterBlocked   ChapterResultStatus = "blocked"
	ChapterFailed    ChapterResultStatus = "failed"
)

type CourseChapterResult struct {
	ChapterID   string              `json:"chapter_id"`
	VolumeID    string              `json:"volume_id,omitempty"`
	VolumeTitle string              `json:"volume_title,omitempty"`
	Sort        int                 `json:"sort"`
	Status      ChapterResultStatus `json:"status"`
	Document    json.RawMessage     `json:"document,omitempty"`
	Error       string              `json:"error,omitempty"`
}

type CourseGenerationUsage struct {
	PromptTokens          int `json:"prompt_tokens"`
	CompletionTokens      int `json:"completion_tokens"`
	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
}

// CourseGenerationResult is result schema v1 for the task-only course flow.
type CourseGenerationResult struct {
	ResultVersion    int                   `json:"result_version"`
	TaskSubtype      string                `json:"task_subtype"`
	CourseID         string                `json:"course_id"`
	BlueprintVersion int                   `json:"blueprint_version"`
	CommitSHA        string                `json:"commit_sha"`
	Status           CourseStatus          `json:"status"`
	Chapters         []CourseChapterResult `json:"chapters"`
	QualityReport    []GateReport          `json:"quality_report,omitempty"`
	Coverage         CoverageMatrix        `json:"coverage,omitempty"`
	BlogParentID     string                `json:"blog_parent_id,omitempty"`
	LabArtifactRefs  []string              `json:"lab_artifact_refs,omitempty"`
	Usage            CourseGenerationUsage `json:"usage"`
}

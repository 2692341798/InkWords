package textbook

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	// LearnerArtifactFormat identifies code saved with a learner's original answer.
	LearnerArtifactFormat = "inkwords.learner-artifact.v1"
	// MaxLearnerFiles limits both storage and later verification admission work.
	MaxLearnerFiles = 32
	// MaxLearnerFileBytes bounds one UTF-8 source file.
	MaxLearnerFileBytes = 128 * 1024
	// MaxLearnerArtifactBytes bounds the entire submitted source set.
	MaxLearnerArtifactBytes = 256 * 1024
)

// LearnerCodeFile is data only. No caller path, command, environment or claim of
// successful execution may acquire authority by being stored in this contract.
type LearnerCodeFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// LearnerArtifact freezes files at the original practice submission time. It is
// separate from manuscript code and carries no verified/success status.
type LearnerArtifact struct {
	Format              string            `json:"format"`
	WorkspaceID         string            `json:"workspace_id"`
	ObjectiveID         string            `json:"objective_id"`
	AttemptID           string            `json:"attempt_id"`
	SessionID           string            `json:"session_id"`
	RevisionID          string            `json:"revision_id"`
	TaskID              string            `json:"task_id"`
	PracticeContentHash string            `json:"practice_content_hash"`
	Skill               LearningTaskMode  `json:"skill"`
	SubmittedAt         time.Time         `json:"submitted_at"`
	Files               []LearnerCodeFile `json:"files"`
	SnapshotHash        string            `json:"snapshot_hash,omitempty"`
}

// NormalizeLearnerFiles sorts portable relative paths without altering source
// bytes. V1 stores Go files and optional go.mod; it does not admit execution.
func NormalizeLearnerFiles(input []LearnerCodeFile) ([]LearnerCodeFile, error) {
	if len(input) == 0 || len(input) > MaxLearnerFiles {
		return nil, fmt.Errorf("learner artifact requires 1 to 32 files")
	}
	files := append([]LearnerCodeFile(nil), input...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	total, goFiles := 0, 0
	for i, file := range files {
		if !safeLearnerPath(file.Path) || i > 0 && files[i-1].Path == file.Path || strings.TrimSpace(file.Content) == "" || !utf8.ValidString(file.Content) || strings.ContainsRune(file.Content, 0) || len(file.Content) > MaxLearnerFileBytes {
			return nil, fmt.Errorf("invalid learner source file")
		}
		if strings.HasSuffix(file.Path, ".go") {
			goFiles++
		}
		for _, parent := range files[:i] {
			if strings.HasPrefix(file.Path, parent.Path+"/") {
				return nil, fmt.Errorf("learner file cannot also be a parent directory")
			}
		}
		total += len(file.Content)
	}
	if goFiles == 0 || total > MaxLearnerArtifactBytes {
		return nil, fmt.Errorf("learner artifact requires Go source within 256 KiB")
	}
	return files, nil
}

func safeLearnerPath(value string) bool {
	if value == "" || len(value) > 200 || path.Clean(value) != value || strings.HasPrefix(value, "/") || value != "go.mod" && !strings.HasSuffix(value, ".go") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || strings.HasPrefix(part, ".") || part == "vendor" || part == "node_modules" {
			return false
		}
		for _, ch := range part {
			if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' && ch != '-' && ch != '.' {
				return false
			}
		}
	}
	return true
}

// LearnerFilesHash makes HTTP retries insensitive to file selection order.
func LearnerFilesHash(files []LearnerCodeFile) (string, error) {
	normalized, err := NormalizeLearnerFiles(files)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return PracticeExcerptHash(string(encoded)), nil
}

// LearnerArtifactHash binds exact source bytes to one workspace and saved task.
// It is a submission hash, not the runner's separate content-addressed tree hash.
func LearnerArtifactHash(artifact LearnerArtifact) (string, error) {
	if artifact.Format != LearnerArtifactFormat || artifact.SubmittedAt.IsZero() || !practiceText(artifact.TaskID, 100) || !isFullSHA256Digest(artifact.PracticeContentHash) {
		return "", fmt.Errorf("incomplete learner artifact identity")
	}
	for _, id := range []string{artifact.WorkspaceID, artifact.ObjectiveID, artifact.AttemptID, artifact.SessionID, artifact.RevisionID} {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil || parsed.String() != id {
			return "", fmt.Errorf("invalid learner artifact identity")
		}
	}
	if err := artifact.Skill.Validate(); err != nil {
		return "", err
	}
	files, err := NormalizeLearnerFiles(artifact.Files)
	if err != nil {
		return "", err
	}
	artifact.Files, artifact.SnapshotHash = files, ""
	encoded, err := json.Marshal(artifact)
	if err != nil {
		return "", err
	}
	return PracticeExcerptHash(string(encoded)), nil
}

// Validate rejects a changed file or a snapshot rebound to another practice.
func (artifact LearnerArtifact) Validate() error {
	hash, err := LearnerArtifactHash(artifact)
	if err != nil {
		return err
	}
	if hash != artifact.SnapshotHash {
		return fmt.Errorf("learner artifact snapshot hash mismatch")
	}
	return nil
}

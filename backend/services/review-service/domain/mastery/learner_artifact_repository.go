package mastery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func prepareLearnerArtifact(objective Objective, record *AttemptRecord) (*sharedtextbook.LearnerArtifact, error) {
	if len(record.LearnerFiles) == 0 {
		if record.LearnerArtifactHash != "" {
			return nil, ErrPracticeMismatch
		}
		return nil, nil
	}
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	if objective.PracticeRevisionID == nil || record.PracticeSessionID == nil || record.PracticeContentHash != objective.PracticeContentHash {
		return nil, ErrPracticeMismatch
	}
	files, err := sharedtextbook.NormalizeLearnerFiles(record.LearnerFiles)
	if err != nil {
		return nil, err
	}
	record.AttemptedAt = record.AttemptedAt.UTC().Truncate(time.Microsecond)
	artifact := sharedtextbook.LearnerArtifact{Format: sharedtextbook.LearnerArtifactFormat, WorkspaceID: objective.WorkspaceID.String(), ObjectiveID: objective.ID.String(), AttemptID: record.ID.String(), SessionID: record.PracticeSessionID.String(), RevisionID: objective.PracticeRevisionID.String(), TaskID: record.PracticeTaskID, PracticeContentHash: record.PracticeContentHash, Skill: sharedtextbook.LearningTaskMode(record.Skill), SubmittedAt: record.AttemptedAt, Files: files}
	artifact.SnapshotHash, err = sharedtextbook.LearnerArtifactHash(artifact)
	if err != nil {
		return nil, err
	}
	record.LearnerArtifactHash = artifact.SnapshotHash
	return &artifact, nil
}

// GetLearnerArtifact resolves one immutable snapshot by the saved attempt key.
func (store *GormStore) GetLearnerArtifact(ctx context.Context, attemptID uuid.UUID) (*sharedtextbook.LearnerArtifact, error) {
	var row struct {
		SnapshotJSON json.RawMessage
		SnapshotHash string
		ObjectiveID  uuid.UUID
		WorkspaceID  uuid.UUID
	}
	result := store.db.WithContext(ctx).Table("mastery_learner_artifacts").Where("attempt_id = ?", attemptID).Limit(1).Find(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	var artifact sharedtextbook.LearnerArtifact
	if json.Unmarshal(row.SnapshotJSON, &artifact) != nil || artifact.Validate() != nil || artifact.AttemptID != attemptID.String() || artifact.ObjectiveID != row.ObjectiveID.String() || artifact.WorkspaceID != row.WorkspaceID.String() || artifact.SnapshotHash != row.SnapshotHash {
		return nil, fmt.Errorf("invalid stored learner code snapshot")
	}
	return &artifact, nil
}

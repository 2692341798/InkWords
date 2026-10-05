package mastery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// AssessmentApplicationRecord stores an append-only decision separately from
// the original answer and model feedback. Schema changes use goose migrations.
type AssessmentApplicationRecord struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	ObjectiveID     uuid.UUID `gorm:"type:uuid"`
	AttemptID       uuid.UUID `gorm:"type:uuid"`
	JobID           uuid.UUID `gorm:"type:uuid"`
	SequenceNo      int
	ApplicationJSON json.RawMessage `gorm:"type:jsonb"`
}

// TableName fixes the shared review-service persistence identity.
func (AssessmentApplicationRecord) TableName() string { return "mastery_assessment_applications" }

// Decode validates the column/document identities before replay or display.
func (row AssessmentApplicationRecord) Decode() (AssessmentApplication, error) {
	var event AssessmentApplication
	if json.Unmarshal(row.ApplicationJSON, &event) != nil || event.ID != row.ID || event.ObjectiveID != row.ObjectiveID || event.AttemptID != row.AttemptID || event.JobID != row.JobID || event.SequenceNo != row.SequenceNo || event.InputHash != AssessmentInputHash(event.Input) || event.FeedbackHash != AssessmentFeedbackHash(event.Feedback) || event.Feedback.Validate(event.Input) != nil {
		return event, fmt.Errorf("invalid stored assessment application")
	}
	return event, nil
}

// ListAssessmentApplications returns objective-ordered decisions for exact replay.
func (store *GormStore) ListAssessmentApplications(ctx context.Context, objectiveID uuid.UUID) ([]AssessmentApplication, error) {
	var rows []AssessmentApplicationRecord
	if err := store.db.WithContext(ctx).Where("objective_id = ?", objectiveID).Order("sequence_no ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	events := make([]AssessmentApplication, 0, len(rows))
	for _, row := range rows {
		event, err := row.Decode()
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

// ReplayAssessedSchedule reconstructs timing and task choice using all attempts
// and explicit grading decisions. The caller holds the objective write lock.
func (store *GormStore) ReplayAssessedSchedule(ctx context.Context, objective Objective) (*Schedule, error) {
	events, err := store.ListAssessmentApplications(ctx, objective.ID)
	if err != nil || len(events) == 0 {
		return nil, err
	}
	records, err := store.ListAttempts(ctx, objective.ID)
	if err != nil {
		return nil, err
	}
	attempts, err := ReplayAssessmentApplications(objective, records, events)
	if err != nil {
		return nil, err
	}
	if len(attempts) == 0 {
		return nil, fmt.Errorf("assessment application has no practice history")
	}
	_, due, err := Replay(attempts)
	if err != nil {
		return nil, err
	}
	due = enforcePracticeDue(objective, attempts[len(attempts)-1], due)
	return &Schedule{ObjectiveID: objective.ID, NextSkill: string(due.Skill), DueAt: due.DueAt, Reason: due.Reason, AlgorithmVersion: AssessmentAlgorithmVersion}, nil
}

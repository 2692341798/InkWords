package mastery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// PracticeBasis comes from core-api's approved revision projection, never a
// browser-supplied rubric. The persisted value is an immutable read model.
type PracticeBasis struct {
	WorkspaceID uuid.UUID
	Title       string
	Learning    sharedtextbook.LearningProjection
}

// PracticeSource resolves a fixed revision without importing a peer service.
type PracticeSource interface {
	LoadApprovedPractice(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (PracticeBasis, error)
}

const practiceAlgorithmVersion = "fsrs-v4-practice-delay-v1"

// ErrPracticeNotDue prevents immediate recall from becoming delayed evidence.
var ErrPracticeNotDue = errors.New("practice retention interval has not elapsed")

// ErrPracticeMismatch rejects a response attributed to a different frozen task.
var ErrPracticeMismatch = errors.New("attempt does not match approved practice task")

func (objective Objective) schedulingVersion() string {
	if objective.PracticeRevisionID != nil {
		return practiceAlgorithmVersion
	}
	return algorithmVersion
}

// WithPracticeSource enables authoritative textbook objective creation.
func (service *Service) WithPracticeSource(source PracticeSource) *Service {
	service.practiceSource = source
	return service
}

func (service *Service) resolvePractice(ctx context.Context, workspaceID uuid.UUID, input ObjectiveInput) (ObjectiveInput, *PracticeBasis, error) {
	input.ChapterID = strings.TrimSpace(input.ChapterID)
	if !strings.HasPrefix(input.ChapterID, "approved-revision:") {
		return input, nil, nil
	}
	parts := strings.Split(input.ChapterID, ":")
	if len(parts) != 3 || service.practiceSource == nil {
		return input, nil, fmt.Errorf("approved practice source is required")
	}
	chapterID, err := uuid.Parse(parts[1])
	if err != nil || chapterID == uuid.Nil {
		return input, nil, fmt.Errorf("invalid practice chapter")
	}
	revisionID, err := uuid.Parse(parts[2])
	if err != nil || revisionID == uuid.Nil {
		return input, nil, fmt.Errorf("invalid practice revision")
	}
	basis, err := service.practiceSource.LoadApprovedPractice(ctx, workspaceID, chapterID, revisionID)
	if err != nil {
		return input, nil, fmt.Errorf("load approved practice: %w", err)
	}
	if basis.WorkspaceID != workspaceID || basis.Learning.ChapterID != chapterID.String() || basis.Learning.RevisionID != revisionID.String() || basis.Learning.Format != "inkwords.learning-projection.v2" || basis.Learning.Validate() != nil || strings.TrimSpace(basis.Title) == "" {
		return input, nil, fmt.Errorf("approved practice identity or contract mismatch")
	}
	input.ChapterID = "approved-revision:" + chapterID.String() + ":" + revisionID.String()
	input.Title = basis.Title
	input.Behavior = basis.Learning.Objectives[0].Text
	input.Skills = append([]Skill(nil), Skills...)
	input.Rubric = nil
	input.KeyPoints = nil
	input.Prerequisites = nil
	for _, task := range basis.Learning.PracticeSet.Tasks {
		for _, criterion := range task.Rubric {
			input.Rubric = append(input.Rubric, criterion.Description)
		}
	}
	input.KeyPoints = append([]string(nil), input.Rubric...)
	for _, stage := range basis.Learning.LearningArc.Stages {
		input.Prerequisites = append(input.Prerequisites, stage.Prerequisites...)
	}
	input.EvidenceRefs = append([]string(nil), basis.Learning.EvidenceIDs...)
	return input, &basis, nil
}

func (objective Objective) practiceProjection() (*sharedtextbook.LearningProjection, error) {
	if objective.PracticeRevisionID == nil {
		if strings.HasPrefix(objective.ChapterID, "approved-revision:") {
			return nil, fmt.Errorf("historical objective has no authoritative practice binding")
		}
		return nil, nil
	}
	var projection sharedtextbook.LearningProjection
	if json.Unmarshal(objective.PracticeProjection, &projection) != nil || projection.Format != "inkwords.learning-projection.v2" || projection.Validate() != nil || projection.RevisionID != objective.PracticeRevisionID.String() || projection.ContentHash != objective.PracticeContentHash || objective.ChapterID != "approved-revision:"+projection.ChapterID+":"+projection.RevisionID {
		return nil, fmt.Errorf("stored practice identity mismatch")
	}
	return &projection, nil
}

func bindPracticeAttempt(objective Objective, previous []Attempt, attempt Attempt, now time.Time) (Attempt, error) {
	projection, err := objective.practiceProjection()
	if err != nil {
		return attempt, err
	}
	if projection == nil {
		if attempt.PracticeTaskID != "" || attempt.PracticeContentHash != "" {
			return attempt, fmt.Errorf("legacy attempt cannot claim an approved practice binding")
		}
		return attempt, nil
	}
	var task *sharedtextbook.PracticeTask
	for i := range projection.PracticeSet.Tasks {
		candidate := &projection.PracticeSet.Tasks[i]
		if string(candidate.Mode) == string(attempt.Skill) {
			task = candidate
			break
		}
	}
	if task == nil || task.ID != attempt.PracticeTaskID || attempt.PracticeContentHash != objective.PracticeContentHash || strings.TrimSpace(attempt.Answer) == "" {
		return attempt, ErrPracticeMismatch
	}
	// Browser timestamps cannot manufacture elapsed retention time.
	attempt.At = now.UTC()
	if len(previous) > 0 && previous[len(previous)-1].At.After(attempt.At) {
		return attempt, fmt.Errorf("attempt precedes saved evidence")
	}
	if attempt.Skill == Retain {
		if len(previous) == 0 || attempt.At.Sub(previous[len(previous)-1].At) < time.Duration(task.MinDelayHours)*time.Hour {
			return attempt, ErrPracticeNotDue
		}
	}
	return attempt, nil
}

func enforcePracticeDue(objective Objective, attempt Attempt, due DueTask) DueTask {
	projection, err := objective.practiceProjection()
	if err != nil || projection == nil || due.Skill != Retain {
		return due
	}
	for _, task := range projection.PracticeSet.Tasks {
		if task.Mode == sharedtextbook.LearningTaskRetain {
			earliest := attempt.At.Add(time.Duration(task.MinDelayHours) * time.Hour)
			if due.DueAt.Before(earliest) {
				due.DueAt = earliest
				due.Reason = "按批准题目的保持间隔安排延迟复习。"
			}
		}
	}
	return due
}

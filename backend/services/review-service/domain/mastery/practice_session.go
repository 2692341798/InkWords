package mastery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// PracticeSession fixes one attempt identity across reloads and network retries.
type PracticeSession struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	ObjectiveID         uuid.UUID      `gorm:"type:uuid;not null" json:"objective_id"`
	Skill               Skill          `gorm:"type:text;not null" json:"skill"`
	PracticeTaskID      string         `gorm:"type:text;not null" json:"practice_task_id"`
	PracticeContentHash string         `gorm:"type:text;not null" json:"practice_content_hash"`
	StartedAt           time.Time      `gorm:"not null" json:"started_at"`
	SubmittedAt         *time.Time     `json:"submitted_at,omitempty"`
	AttemptID           *uuid.UUID     `gorm:"type:uuid" json:"attempt_id,omitempty"`
	SubmissionHash      string         `gorm:"type:text;not null;default:''" json:"-"`
	SubmissionResult    datatypes.JSON `gorm:"type:jsonb" json:"-"`
}

func (PracticeSession) TableName() string { return "mastery_practice_sessions" }

// PracticeHelp is append-only evidence of an explicitly requested disclosure.
type PracticeHelp struct {
	SessionID uuid.UUID `gorm:"type:uuid;primaryKey" json:"session_id"`
	Kind      string    `gorm:"type:text;primaryKey" json:"kind"`
	Level     int       `gorm:"primaryKey" json:"level"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
}

func (PracticeHelp) TableName() string { return "mastery_practice_help" }

// PracticeSessionView exposes persisted disclosure state; counts are not client assertions.
type PracticeSessionView struct {
	PracticeSession
	HintsShown         int        `json:"hints_shown"`
	AnswerShown        bool       `json:"answer_shown"`
	HintCount          int        `json:"hint_count"`
	ObjectiveHelpCount int64      `json:"-"`
	LastOtherHelpAt    *time.Time `json:"-"`
	Result             *DueTask   `json:"result,omitempty"`
}

type practiceSessionStore interface {
	BeginPracticeSession(context.Context, uuid.UUID, Skill, string, string, time.Time) (PracticeSessionView, error)
	LoadPracticeSession(context.Context, uuid.UUID, uuid.UUID) (PracticeSessionView, error)
	RevealPracticeHelp(context.Context, uuid.UUID, uuid.UUID, string, int, time.Time) (PracticeSessionView, error)
}

// ErrPracticeSessionClosed rejects rewriting an already committed attempt.
var ErrPracticeSessionClosed = errors.New("practice session submission differs from its saved attempt")

// BeginPracticeSession resumes the one open session for this frozen task.
func (service *Service) BeginPracticeSession(ctx context.Context, workspaceID, objectiveID uuid.UUID, skill Skill) (PracticeSessionView, error) {
	if service == nil || service.store == nil || workspaceID == uuid.Nil || objectiveID == uuid.Nil {
		return PracticeSessionView{}, fmt.Errorf("invalid practice session request")
	}

	objective, err := service.store.GetObjective(ctx, workspaceID, objectiveID)
	if err != nil {
		return PracticeSessionView{}, err
	}
	if !objective.requires(skill) {
		return PracticeSessionView{}, ErrPracticeMismatch
	}
	projection, err := objective.practiceProjection()
	if err != nil || projection == nil {
		return PracticeSessionView{}, ErrPracticeMismatch
	}
	store, ok := service.store.(practiceSessionStore)
	if !ok {
		return PracticeSessionView{}, fmt.Errorf("practice session storage unavailable")
	}
	for _, task := range projection.PracticeSet.Tasks {
		if string(task.Mode) == string(skill) {
			return store.BeginPracticeSession(ctx, objectiveID, skill, task.ID, objective.PracticeContentHash, service.now().UTC())
		}
	}
	return PracticeSessionView{}, ErrPracticeMismatch
}

// LoadPracticeSession checks ownership before exposing recorded help or completion.
func (service *Service) LoadPracticeSession(ctx context.Context, workspaceID, objectiveID, sessionID uuid.UUID) (PracticeSessionView, error) {
	if service == nil || service.store == nil || workspaceID == uuid.Nil || objectiveID == uuid.Nil {
		return PracticeSessionView{}, fmt.Errorf("invalid practice session request")
	}

	if _, err := service.store.GetObjective(ctx, workspaceID, objectiveID); err != nil {
		return PracticeSessionView{}, err
	}
	store, ok := service.store.(practiceSessionStore)
	if !ok {
		return PracticeSessionView{}, fmt.Errorf("practice session storage unavailable")
	}
	return store.LoadPracticeSession(ctx, objectiveID, sessionID)
}

// RevealPracticeHelp commits disclosure before the UI shows its contents.
func (service *Service) RevealPracticeHelp(ctx context.Context, workspaceID, objectiveID, sessionID uuid.UUID, kind string, level int) (PracticeSessionView, error) {
	if service == nil || service.store == nil || workspaceID == uuid.Nil || objectiveID == uuid.Nil {
		return PracticeSessionView{}, fmt.Errorf("invalid practice session request")
	}

	if _, err := service.store.GetObjective(ctx, workspaceID, objectiveID); err != nil {
		return PracticeSessionView{}, err
	}
	if kind != "hint" && kind != "answer" || kind == "hint" && (level < 1 || level > 3) || kind == "answer" && level != 0 {
		return PracticeSessionView{}, fmt.Errorf("invalid practice help")
	}
	store, ok := service.store.(practiceSessionStore)
	if !ok {
		return PracticeSessionView{}, fmt.Errorf("practice session storage unavailable")
	}
	return store.RevealPracticeHelp(ctx, objectiveID, sessionID, kind, level, service.now().UTC())
}

// submissionHash excludes clocks and elapsed duration: both are server derived
// and therefore cannot make a network retry into a different learner answer.
func submissionHash(attempt Attempt) string {
	if len(attempt.ErrorKinds) == 0 {
		attempt.ErrorKinds = nil
	}
	encoded, _ := json.Marshal(struct {
		SessionID                   uuid.UUID
		Skill                       Skill
		TaskID, ContentHash, Answer string
		Correct, Independent        bool
		HintCount, Confidence       int
		ErrorKinds                  []string
	}{attempt.PracticeSessionID, attempt.Skill, attempt.PracticeTaskID, attempt.PracticeContentHash, attempt.Answer, attempt.Correct, attempt.Independent, attempt.HintCount, attempt.Confidence, attempt.ErrorKinds})
	digest := sha256.Sum256(encoded)
	if len(attempt.LearnerFiles) > 0 {
		filesHash, err := sharedtextbook.LearnerFilesHash(attempt.LearnerFiles)
		if err != nil {
			return ""
		}
		digest = sha256.Sum256([]byte("inkwords.practice-submission.v2\n" + hex.EncodeToString(digest[:]) + "\n" + filesHash))
	}
	return "sha256:" + hex.EncodeToString(digest[:])
}

func savedSessionResult(view PracticeSessionView, hash string) (DueTask, error) {
	if view.SubmissionHash != hash {
		return DueTask{}, ErrPracticeSessionClosed
	}
	var due DueTask
	if json.Unmarshal(view.SubmissionResult, &due) != nil || !known(due.Skill) || due.DueAt.IsZero() {
		return DueTask{}, fmt.Errorf("saved practice response is invalid")
	}
	return due, nil
}

func (service *Service) prepareSessionAttempt(ctx context.Context, objective Objective, attempt Attempt) (Attempt, *PracticeSessionView, string, error) {
	if attempt.HintCount < 0 {
		return attempt, nil, "", fmt.Errorf("invalid hint count")
	}
	if objective.PracticeRevisionID == nil {
		if attempt.PracticeSessionID != uuid.Nil {
			return attempt, nil, "", ErrPracticeMismatch
		}
		return attempt, nil, "", nil
	}
	store, ok := service.store.(practiceSessionStore)
	if !ok || attempt.PracticeSessionID == uuid.Nil {
		return attempt, nil, "", ErrPracticeMismatch
	}
	view, err := store.LoadPracticeSession(ctx, objective.ID, attempt.PracticeSessionID)
	if err != nil {
		return attempt, nil, "", err
	}
	if view.Skill != attempt.Skill || view.PracticeTaskID != attempt.PracticeTaskID || view.PracticeContentHash != attempt.PracticeContentHash {
		return attempt, nil, "", ErrPracticeMismatch
	}
	if attempt.HintCount < view.HintCount {
		attempt.HintCount = view.HintCount
	}
	if view.AnswerShown {
		attempt.Independent = false
	}
	attempt.Took = service.now().UTC().Sub(view.StartedAt)
	return attempt, &view, submissionHash(attempt), nil
}

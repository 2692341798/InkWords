package masteryassessment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"inkwords-backend/services/review-service/domain/mastery"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

var (
	ErrNotFound = errors.New("评分任务不存在或不可访问")
	ErrConflict = errors.New("评分依据或任务状态已变化，请重新读取后操作")
)

// Job freezes input and model identity. A nil result means no known completed
// provider response; interrupted work must never be reported as a zero-cost call.
type Job struct {
	ID                uuid.UUID                      `json:"id"`
	WorkspaceID       uuid.UUID                      `json:"-"`
	ObjectiveID       uuid.UUID                      `json:"objective_id"`
	AttemptID         uuid.UUID                      `json:"attempt_id"`
	RequestID         uuid.UUID                      `json:"request_id"`
	RetryOf           *uuid.UUID                     `json:"retry_of,omitempty"`
	Status            string                         `json:"status"`
	Preview           Preview                        `json:"preview"`
	Input             mastery.AssessmentInput        `json:"input"`
	Result            *Result                        `json:"result,omitempty"`
	ErrorCode         string                         `json:"error_code,omitempty"`
	CreatedAt         time.Time                      `json:"created_at"`
	CompletedAt       *time.Time                     `json:"completed_at,omitempty"`
	Corrections       []mastery.AssessmentCorrection `json:"corrections"`
	EffectiveFeedback *mastery.AssessmentFeedback    `json:"effective_feedback,omitempty"`
	EffectiveHash     string                         `json:"effective_hash,omitempty"`
	AppliedAssessment *mastery.AssessmentApplication `json:"applied_assessment,omitempty"`
	Schedule          *mastery.DueTask               `json:"schedule,omitempty"`
}

// StartInput carries consent for one exact preview and a retry-stable request identity.
type StartInput struct {
	RequestID           uuid.UUID  `json:"request_id"`
	ExpectedInputHash   string     `json:"expected_input_hash"`
	ExpectedRequestHash string     `json:"expected_request_hash"`
	RetryOf             *uuid.UUID `json:"retry_of,omitempty"`
}

// CorrectionInput cannot supply its reviewer, timestamp or authoritative input hash.
type CorrectionInput struct {
	ID           uuid.UUID                            `json:"id"`
	PreviousHash string                               `json:"previous_hash"`
	Reason       string                               `json:"reason"`
	Changes      []mastery.CriterionAssessment        `json:"changes"`
	Findings     *mastery.AssessmentFindingCorrection `json:"findings,omitempty"`
}

// ApplyInput names an exact feedback view and the previously applied decision.
// It cannot supply a score, actor, practice time or next review date.
type ApplyInput struct {
	ID                   uuid.UUID  `json:"id"`
	ExpectedFeedbackHash string     `json:"expected_feedback_hash"`
	PreviousID           *uuid.UUID `json:"previous_id,omitempty"`
}

// Store persists request deduplication, terminal responses and correction CAS.
type Store interface {
	FindRequest(context.Context, uuid.UUID, uuid.UUID) (*Job, error)
	Create(context.Context, Job) (Job, bool, error)
	Read(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Job, error)
	Latest(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*Job, error)
	Finish(context.Context, uuid.UUID, Result, string, string) error
	Cancel(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Job, error)
	Correct(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, CorrectionInput) (Job, error)
	Apply(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, ApplyInput) (Job, error)
	InterruptRunning(context.Context) error
	Interrupt(context.Context, uuid.UUID) error
}

// InputSource resolves a saved attempt; request bodies never supply learner text.
type InputSource interface {
	PrepareAssessmentInput(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (mastery.AssessmentInput, error)
}

// VerificationSource exposes owner-validated stored facts. It cannot start a
// learner run and returns nil when no usable execution report exists.
type VerificationSource interface {
	LatestAssessmentEvidence(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*sharedtextbook.LearnerVerificationInput, *sharedtextbook.LearnerVerificationReport, error)
}

// Engine is the provider-neutral single-call assessment executor.
type Engine interface {
	Preview(mastery.AssessmentInput) (Preview, error)
	Assess(context.Context, mastery.AssessmentInput) (Result, error)
}

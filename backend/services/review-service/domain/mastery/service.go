package mastery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

const algorithmVersion = "fsrs-v4-default-parameters"

type Store interface {
	CreateObjectiveAndInitialSchedule(context.Context, *Objective, *Schedule) error
	FindObjectiveByChapter(context.Context, uuid.UUID, string) (Objective, bool, error)
	GetObjective(context.Context, uuid.UUID, uuid.UUID) (Objective, error)
	ListAttempts(context.Context, uuid.UUID) ([]AttemptRecord, error)
	GetAttempt(context.Context, uuid.UUID, uuid.UUID) (AttemptRecord, error)
	AppendAttemptAndSchedule(context.Context, *AttemptRecord, *Schedule) error
	ListDue(context.Context, uuid.UUID, time.Time, int) ([]DueObjective, error)
}

type DueObjective struct {
	ObjectiveID uuid.UUID `json:"objective_id"`
	ChapterID   string    `json:"chapter_id"`
	Title       string    `json:"title"`
	Skill       Skill     `json:"skill"`
	DueAt       time.Time `json:"due_at"`
	Reason      string    `json:"reason"`
}

// ObjectiveInput freezes the teaching evidence used to judge one learnable behavior.
type ObjectiveInput struct {
	ChapterID     string
	Title         string
	Behavior      string
	Skills        []Skill
	Rubric        []string
	KeyPoints     []string
	Prerequisites []string
	EvidenceRefs  []string
}

type Service struct {
	store          Store
	practiceSource PracticeSource
	now            func() time.Time
}

func NewService(store Store) *Service { return &Service{store: store, now: time.Now} }

func (service *Service) CreateObjective(ctx context.Context, workspaceID uuid.UUID, input ObjectiveInput) (Objective, error) {
	if service == nil || service.store == nil || workspaceID == uuid.Nil {
		return Objective{}, fmt.Errorf("invalid mastery objective")
	}
	input, basis, resolveErr := service.resolvePractice(ctx, workspaceID, input)
	if resolveErr != nil {
		return Objective{}, resolveErr
	}
	if service == nil || service.store == nil || workspaceID == uuid.Nil || strings.TrimSpace(input.ChapterID) == "" || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Behavior) == "" || len(input.Skills) == 0 || !nonEmpty(input.Rubric) || !nonEmpty(input.KeyPoints) || !nonEmpty(input.EvidenceRefs) {
		return Objective{}, fmt.Errorf("invalid mastery objective")
	}
	seen := map[Skill]bool{}
	for _, skill := range input.Skills {
		if !known(skill) || seen[skill] {
			return Objective{}, fmt.Errorf("invalid mastery objective skills")
		}
		seen[skill] = true
	}
	// The client can safely retry an approved-projection sync. A chapter identity
	// identifies the frozen revision that supplied its rubric and evidence.
	existing, found, err := service.store.FindObjectiveByChapter(ctx, workspaceID, input.ChapterID)
	if err != nil {
		return Objective{}, fmt.Errorf("find existing mastery objective: %w", err)
	}
	if found {
		if basis != nil && (existing.PracticeRevisionID == nil || existing.PracticeContentHash != basis.Learning.ContentHash) {
			return Objective{}, fmt.Errorf("existing practice objective has no matching approved basis")
		}
		return existing, nil
	}
	encoded, err := json.Marshal(input.Skills)
	if err != nil {
		return Objective{}, fmt.Errorf("encode mastery skills: %w", err)
	}
	rubric, _ := json.Marshal(input.Rubric)
	keyPoints, _ := json.Marshal(input.KeyPoints)
	prerequisites, _ := json.Marshal(input.Prerequisites)
	evidenceRefs, _ := json.Marshal(input.EvidenceRefs)
	objective := Objective{WorkspaceID: workspaceID, ChapterID: strings.TrimSpace(input.ChapterID), Title: strings.TrimSpace(input.Title), Behavior: strings.TrimSpace(input.Behavior), RequiredSkills: encoded, Rubric: rubric, KeyPoints: keyPoints, Prerequisites: prerequisites, EvidenceRefs: evidenceRefs}
	if basis != nil {
		revisionID := uuid.MustParse(basis.Learning.RevisionID)
		objective.PracticeRevisionID = &revisionID
		objective.PracticeContentHash = basis.Learning.ContentHash
		objective.PracticeProjection, err = json.Marshal(basis.Learning)
		if err != nil {
			return Objective{}, err
		}
	}
	initial := Schedule{ObjectiveID: objective.ID, NextSkill: string(input.Skills[0]), DueAt: service.now().UTC(), Reason: "新学习目标先从首个要求的练习开始。", AlgorithmVersion: objective.schedulingVersion()}
	if err := service.store.CreateObjectiveAndInitialSchedule(ctx, &objective, &initial); err != nil {
		// A unique active objective identity makes concurrent browser retries
		// converge on the first committed frozen-revision objective.
		existing, found, lookupErr := service.store.FindObjectiveByChapter(ctx, workspaceID, input.ChapterID)
		if lookupErr == nil && found {
			if basis != nil && (existing.PracticeRevisionID == nil || existing.PracticeContentHash != basis.Learning.ContentHash) {
				return Objective{}, fmt.Errorf("concurrent objective has no matching approved basis")
			}
			return existing, nil
		}
		return Objective{}, err
	}
	return objective, nil
}

// EnsureLegacyNoteObjective migrates one eligible Obsidian note into the
// workspace-owned mastery model. It intentionally creates only an explain and
// retain starting point: a legacy free-form note cannot establish the other
// mastery dimensions without new, evidence-bound learning tasks.
func (service *Service) EnsureLegacyNoteObjective(ctx context.Context, workspaceID uuid.UUID, input ObjectiveInput) (Objective, bool, error) {
	if service == nil || service.store == nil || workspaceID == uuid.Nil || !strings.HasPrefix(strings.TrimSpace(input.ChapterID), "legacy-note:") {
		return Objective{}, false, fmt.Errorf("invalid legacy note objective")
	}
	existing, found, err := service.store.FindObjectiveByChapter(ctx, workspaceID, input.ChapterID)
	if err != nil {
		return Objective{}, false, err
	}
	if found {
		return existing, false, nil
	}
	input.Skills = []Skill{Explain, Retain}
	created, err := service.CreateObjective(ctx, workspaceID, input)
	if err == nil {
		return created, true, nil
	}
	// The partial unique index for legacy-note identities turns concurrent
	// migrations into an ordinary read-after-conflict instead of duplicate
	// objectives. Keep the original write error if no canonical row appears.
	existing, found, lookupErr := service.store.FindObjectiveByChapter(ctx, workspaceID, input.ChapterID)
	if lookupErr == nil && found {
		return existing, false, nil
	}
	return Objective{}, false, err
}

func nonEmpty(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func (service *Service) RecordAttempt(ctx context.Context, workspaceID, objectiveID uuid.UUID, attempt Attempt) (DueTask, error) {
	if attempt.AssessmentOutcome != "" {
		return DueTask{}, fmt.Errorf("grading must be explicitly applied to a saved attempt")
	}
	if service == nil || service.store == nil || workspaceID == uuid.Nil || objectiveID == uuid.Nil {
		return DueTask{}, fmt.Errorf("invalid mastery attempt request")
	}
	if !utf8.ValidString(attempt.Answer) || utf8.RuneCountInString(attempt.Answer) > 20000 || strings.ContainsRune(attempt.Answer, 0) {
		return DueTask{}, fmt.Errorf("mastery answer must be valid text of at most 20000 characters")
	}
	objective, err := service.store.GetObjective(ctx, workspaceID, objectiveID)
	if err != nil {
		return DueTask{}, err
	}
	if !objective.requires(attempt.Skill) {
		return DueTask{}, fmt.Errorf("attempt skill is not required by mastery objective")
	}
	if len(attempt.LearnerFiles) > 0 {
		if objective.PracticeRevisionID == nil || attempt.PracticeSessionID == uuid.Nil {
			return DueTask{}, ErrPracticeMismatch
		}
		attempt.LearnerFiles, err = sharedtextbook.NormalizeLearnerFiles(attempt.LearnerFiles)
		if err != nil {
			return DueTask{}, err
		}
	}
	attempt, session, requestHash, err := service.prepareSessionAttempt(ctx, objective, attempt)
	if err != nil {
		return DueTask{}, err
	}
	if session != nil && session.SubmittedAt != nil {
		return savedSessionResult(*session, requestHash)
	}
	records, err := service.store.ListAttempts(ctx, objectiveID)
	if err != nil {
		return DueTask{}, err
	}
	if session != nil {
		for _, record := range records {
			if record.PracticeSessionID != nil && *record.PracticeSessionID == session.ID {
				current, err := service.store.(practiceSessionStore).LoadPracticeSession(ctx, objectiveID, session.ID)
				if err != nil {
					return DueTask{}, err
				}
				return savedSessionResult(current, requestHash)
			}
		}
	}
	previous, err := attemptsFromRecords(records)
	if err != nil {
		return DueTask{}, err
	}
	validationHistory := previous
	if session != nil && session.LastOtherHelpAt != nil && (len(previous) == 0 || session.LastOtherHelpAt.After(previous[len(previous)-1].At)) {
		validationHistory = append(append([]Attempt(nil), previous...), Attempt{At: *session.LastOtherHelpAt})
	}
	attempt, err = bindPracticeAttempt(objective, validationHistory, attempt, service.now())
	if err != nil {
		return DueTask{}, err
	}
	_, due, err := Replay(append(previous, attempt))
	if err != nil {
		return DueTask{}, err
	}
	due = enforcePracticeDue(objective, attempt, due)
	errorKinds, err := json.Marshal(attempt.ErrorKinds)
	if err != nil {
		return DueTask{}, fmt.Errorf("encode mastery error kinds: %w", err)
	}
	record := &AttemptRecord{ObjectiveID: objectiveID, Skill: string(attempt.Skill), Answer: attempt.Answer, Correct: attempt.Correct, Independent: attempt.Independent, HintCount: attempt.HintCount, TookMillis: attempt.Took.Milliseconds(), Confidence: attempt.Confidence, ErrorKinds: errorKinds, AttemptedAt: attempt.At}
	record.PracticeTaskID = attempt.PracticeTaskID
	record.LearnerFiles = attempt.LearnerFiles
	record.ExpectedPreviousCount = len(records)
	record.PracticeContentHash = attempt.PracticeContentHash
	if session != nil {
		record.PracticeSessionID = &session.ID
		record.SubmissionHash = requestHash
		record.ExpectedObjectiveHelpCount = session.ObjectiveHelpCount
	}
	schedule := &Schedule{ObjectiveID: objectiveID, NextSkill: string(due.Skill), DueAt: due.DueAt, Reason: due.Reason, AlgorithmVersion: objective.schedulingVersion()}
	if err := service.store.AppendAttemptAndSchedule(ctx, record, schedule); err != nil {
		return DueTask{}, err
	}
	if session != nil {
		current, err := service.store.(practiceSessionStore).LoadPracticeSession(ctx, objectiveID, session.ID)
		if err != nil {
			return DueTask{}, err
		}
		return savedSessionResult(current, requestHash)
	}
	return due, nil
}

func (objective Objective) requires(skill Skill) bool {
	var skills []Skill
	if err := json.Unmarshal(objective.RequiredSkills, &skills); err != nil {
		return false
	}
	for _, required := range skills {
		if required == skill {
			return true
		}
	}
	return false
}

func (service *Service) Due(ctx context.Context, workspaceID uuid.UUID, now time.Time, limit int) ([]DueObjective, error) {
	if service == nil || service.store == nil || workspaceID == uuid.Nil || now.IsZero() {
		return nil, fmt.Errorf("invalid mastery due request")
	}
	return service.store.ListDue(ctx, workspaceID, now, limit)
}

func attemptsFromRecords(records []AttemptRecord) ([]Attempt, error) {
	attempts := make([]Attempt, 0, len(records))
	for _, record := range records {
		var errors []string
		if err := json.Unmarshal(record.ErrorKinds, &errors); err != nil {
			return nil, fmt.Errorf("decode mastery attempt errors: %w", err)
		}
		attempts = append(attempts, Attempt{Skill: Skill(record.Skill), Answer: record.Answer, PracticeTaskID: record.PracticeTaskID, PracticeContentHash: record.PracticeContentHash, Correct: record.Correct, Independent: record.Independent, HintCount: record.HintCount, Took: time.Duration(record.TookMillis) * time.Millisecond, Confidence: record.Confidence, ErrorKinds: errors, At: record.AttemptedAt})
	}
	return attempts, nil
}

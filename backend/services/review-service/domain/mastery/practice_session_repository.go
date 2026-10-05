package mastery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func lockPracticeObjective(tx *gorm.DB, objectiveID uuid.UUID) (Objective, error) {
	var objective Objective
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&objective, "id = ?", objectiveID).Error
	return objective, err
}

func sessionMatchesObjective(objective Objective, session PracticeSession) bool {
	projection, err := objective.practiceProjection()
	if err != nil || projection == nil || objective.ID != session.ObjectiveID || session.PracticeContentHash != objective.PracticeContentHash {
		return false
	}
	for _, task := range projection.PracticeSet.Tasks {
		if string(task.Mode) == string(session.Skill) && task.ID == session.PracticeTaskID {
			return true
		}
	}
	return false
}

// Callers hold the objective lock so disclosure counts and the session are one
// coherent snapshot, even while a second tab requests help.
func loadPracticeSessionView(tx *gorm.DB, objective Objective, sessionID uuid.UUID) (PracticeSessionView, error) {
	view := PracticeSessionView{}
	if err := tx.Where("id = ? AND objective_id = ?", sessionID, objective.ID).First(&view.PracticeSession).Error; err != nil {
		return view, err
	}
	if !sessionMatchesObjective(objective, view.PracticeSession) {
		return view, ErrPracticeMismatch
	}
	var helps []PracticeHelp
	if err := tx.Where("session_id = ?", sessionID).Find(&helps).Error; err != nil {
		return view, err
	}
	for _, help := range helps {
		view.HintCount++
		if help.Kind == "hint" && help.Level > view.HintsShown {
			view.HintsShown = help.Level
		}
		if help.Kind == "answer" {
			view.AnswerShown = true
		}
	}
	var aggregate struct {
		Total           int64
		LastOtherHelpAt *time.Time
	}
	err := tx.Table("mastery_practice_help h").Select("COUNT(*) AS total, MAX(CASE WHEN s.id <> ? THEN h.created_at END) AS last_other_help_at", sessionID).
		Joins("JOIN mastery_practice_sessions s ON s.id = h.session_id").Where("s.objective_id = ?", objective.ID).Scan(&aggregate).Error
	view.ObjectiveHelpCount = aggregate.Total
	view.LastOtherHelpAt = aggregate.LastOtherHelpAt
	if err == nil && view.SubmittedAt != nil {
		due, resultErr := savedSessionResult(view, view.SubmissionHash)
		if resultErr != nil {
			return view, resultErr
		}
		view.Result = &due
	}
	return view, err
}

// BeginPracticeSession makes refreshes and simultaneous opens resume one session.
func (store *GormStore) BeginPracticeSession(ctx context.Context, objectiveID uuid.UUID, skill Skill, taskID, contentHash string, now time.Time) (PracticeSessionView, error) {
	var view PracticeSessionView
	err := store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		objective, err := lockPracticeObjective(tx, objectiveID)
		if err != nil {
			return err
		}
		session := PracticeSession{ID: uuid.New(), ObjectiveID: objectiveID, Skill: skill, PracticeTaskID: taskID, PracticeContentHash: contentHash, StartedAt: now}
		if !sessionMatchesObjective(objective, session) || now.IsZero() {
			return ErrPracticeMismatch
		}
		var existing PracticeSession
		err = tx.Where("objective_id = ? AND skill = ? AND submitted_at IS NULL", objectiveID, skill).First(&existing).Error
		switch err {
		case nil:
			session = existing
		case gorm.ErrRecordNotFound:
			if err = tx.Create(&session).Error; err != nil {
				return err
			}
		default:
			return err
		}
		view, err = loadPracticeSessionView(tx, objective, session.ID)
		return err
	})
	return view, err
}

// LoadPracticeSession reads a coherent view without creating a new attempt.
func (store *GormStore) LoadPracticeSession(ctx context.Context, objectiveID, sessionID uuid.UUID) (PracticeSessionView, error) {
	var view PracticeSessionView
	err := store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		objective, err := lockPracticeObjective(tx, objectiveID)
		if err != nil {
			return err
		}
		view, err = loadPracticeSessionView(tx, objective, sessionID)
		return err
	})
	return view, err
}

// RevealPracticeHelp records each disclosure once and refuses skipped hint levels.
func (store *GormStore) RevealPracticeHelp(ctx context.Context, objectiveID, sessionID uuid.UUID, kind string, level int, now time.Time) (PracticeSessionView, error) {
	var view PracticeSessionView
	err := store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		objective, err := lockPracticeObjective(tx, objectiveID)
		if err != nil {
			return err
		}
		view, err = loadPracticeSessionView(tx, objective, sessionID)
		if err != nil {
			return err
		}
		if kind == "hint" && level >= 1 && level <= view.HintsShown || kind == "answer" && level == 0 && view.AnswerShown {
			return nil
		}
		if view.SubmittedAt != nil {
			return ErrPracticeSessionClosed
		}
		if now.Before(view.StartedAt) || kind != "hint" && kind != "answer" || kind == "hint" && (level != view.HintsShown+1 || level > 3) || kind == "answer" && level != 0 {
			return fmt.Errorf("invalid practice help progression")
		}
		if err = tx.Create(&PracticeHelp{SessionID: sessionID, Kind: kind, Level: level, CreatedAt: now}).Error; err != nil {
			return err
		}
		view, err = loadPracticeSessionView(tx, objective, sessionID)
		return err
	})
	return view, err
}

func preparePracticeSessionAppend(tx *gorm.DB, objective Objective, record *AttemptRecord) (bool, error) {
	if record.PracticeSessionID == nil {
		if record.PracticeTaskID != "" {
			return false, ErrPracticeMismatch
		}
		return false, nil
	}
	view, err := loadPracticeSessionView(tx, objective, *record.PracticeSessionID)
	if err != nil {
		return false, err
	}
	if view.Skill != Skill(record.Skill) || view.PracticeTaskID != record.PracticeTaskID || view.PracticeContentHash != record.PracticeContentHash {
		return false, ErrPracticeMismatch
	}
	if view.SubmittedAt != nil {
		if view.SubmissionHash != record.SubmissionHash {
			return false, ErrPracticeSessionClosed
		}
		return true, nil
	}
	if view.ObjectiveHelpCount != record.ExpectedObjectiveHelpCount {
		return false, ErrMasteryHistoryChanged
	}
	if record.HintCount < view.HintCount {
		return false, ErrMasteryHistoryChanged
	}
	if view.AnswerShown && record.Independent {
		return false, ErrMasteryHistoryChanged
	}
	return false, nil
}

func finishPracticeSession(tx *gorm.DB, record *AttemptRecord, schedule *Schedule) error {
	if record.PracticeSessionID == nil {
		return nil
	}
	result, err := json.Marshal(DueTask{Skill: Skill(schedule.NextSkill), DueAt: schedule.DueAt, Reason: schedule.Reason})
	if err != nil {
		return err
	}
	return tx.Model(&PracticeSession{}).Where("id = ? AND objective_id = ?", *record.PracticeSessionID, record.ObjectiveID).
		Updates(map[string]any{"submitted_at": record.AttemptedAt, "attempt_id": record.ID, "submission_hash": record.SubmissionHash, "submission_result": datatypes.JSON(result)}).Error
}

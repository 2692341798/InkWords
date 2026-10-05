package mastery

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// Objective is owned by the local installation workspace. ChapterID is opaque
// so review-service does not import core-api's textbook persistence model.
type Objective struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	WorkspaceID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"workspace_id"`
	ChapterID           string         `gorm:"type:text;not null;index" json:"chapter_id"`
	Title               string         `gorm:"type:text;not null" json:"title"`
	Behavior            string         `gorm:"type:text;not null" json:"behavior"`
	RequiredSkills      datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"required_skills"`
	Rubric              datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"rubric"`
	KeyPoints           datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"key_points"`
	Prerequisites       datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"prerequisites"`
	EvidenceRefs        datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"evidence_refs"`
	PracticeRevisionID  *uuid.UUID     `gorm:"type:uuid" json:"practice_revision_id,omitempty"`
	PracticeContentHash string         `gorm:"type:text;not null;default:''" json:"practice_content_hash"`
	PracticeProjection  datatypes.JSON `gorm:"type:jsonb" json:"practice_projection,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Objective) TableName() string { return "mastery_objectives" }
func (objective *Objective) BeforeCreate(*gorm.DB) error {
	if objective.ID == uuid.Nil {
		objective.ID = uuid.New()
	}
	return nil
}

// AttemptRecord is append-only performance evidence. Derived scores are never
// edited in place; they can always be reconstructed with Replay.
type AttemptRecord struct {
	LearnerFiles               []sharedtextbook.LearnerCodeFile `gorm:"-" json:"-"`
	LearnerArtifactHash        string                           `gorm:"type:text;not null;default:''" json:"learner_artifact_hash,omitempty"`
	ExpectedPreviousCount      int                              `gorm:"-" json:"-"`
	ExpectedObjectiveHelpCount int64                            `gorm:"-" json:"-"`
	SubmissionHash             string                           `gorm:"-" json:"-"`
	PracticeSessionID          *uuid.UUID                       `gorm:"type:uuid" json:"practice_session_id,omitempty"`
	ID                         uuid.UUID                        `gorm:"type:uuid;primaryKey" json:"id"`
	ObjectiveID                uuid.UUID                        `gorm:"type:uuid;not null;index" json:"objective_id"`
	Skill                      string                           `gorm:"type:varchar(32);not null" json:"skill"`
	Answer                     string                           `gorm:"type:text;not null;default:''" json:"answer"`
	PracticeTaskID             string                           `gorm:"type:text;not null;default:''" json:"practice_task_id"`
	PracticeContentHash        string                           `gorm:"type:text;not null;default:''" json:"practice_content_hash"`
	Correct                    bool                             `gorm:"not null" json:"correct"`
	Independent                bool                             `gorm:"not null" json:"independent"`
	HintCount                  int                              `gorm:"not null" json:"hint_count"`
	TookMillis                 int64                            `gorm:"not null" json:"took_millis"`
	Confidence                 int                              `gorm:"not null" json:"confidence"`
	ErrorKinds                 datatypes.JSON                   `gorm:"type:jsonb;not null;default:'[]'" json:"error_kinds"`
	AttemptedAt                time.Time                        `gorm:"not null;index" json:"attempted_at"`
	CreatedAt                  time.Time                        `json:"created_at"`
}

func (AttemptRecord) TableName() string { return "mastery_attempts" }
func (attempt *AttemptRecord) BeforeCreate(*gorm.DB) error {
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}
	return nil
}

// Schedule is a disposable projection of append-only attempts, retained only
// to query due work efficiently when the application opens.
type Schedule struct {
	ObjectiveID      uuid.UUID `gorm:"type:uuid;primaryKey" json:"objective_id"`
	NextSkill        string    `gorm:"type:varchar(32);not null" json:"next_skill"`
	DueAt            time.Time `gorm:"not null;index" json:"due_at"`
	Reason           string    `gorm:"type:text;not null" json:"reason"`
	AlgorithmVersion string    `gorm:"type:varchar(32);not null" json:"algorithm_version"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Schedule) TableName() string { return "mastery_schedules" }

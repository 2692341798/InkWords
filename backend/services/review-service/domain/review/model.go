package review

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	ReviewEntryTypeToday        = "today"
	ReviewEntryTypeManualRandom = "manual_random"
	ReviewEntryTypeManualSelect = "manual_select"

	ReviewModeLightRecall = "light_recall"
	ReviewModeDetailedQA  = "detailed_qa"

	ReviewStatusCreated    = "created"
	ReviewStatusInProgress = "in_progress"
	ReviewStatusCompleted  = "completed"
	ReviewStatusAbandoned  = "abandoned"

	ReviewTurnRoleSystem = "system"
	ReviewTurnRoleUser   = "user"

	ReviewTurnTypeOpening    = "opening"
	ReviewTurnTypeAnswer     = "answer"
	ReviewTurnTypeHint       = "hint"
	ReviewTurnTypeQuestion   = "question"
	ReviewTurnTypeFeedback   = "feedback"
	ReviewTurnTypeCompletion = "completion"
)

// ReviewSession records one knowledge review training session owned by review-service.
type ReviewSession struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID            uuid.UUID      `gorm:"type:uuid;index:idx_review_sessions_user_note_created;not null" json:"user_id"`
	NotePath          string         `gorm:"type:text;not null;index:idx_review_sessions_user_note_created" json:"note_path"`
	NoteTitle         string         `gorm:"type:varchar(255);not null" json:"note_title"`
	SourceTitle       string         `gorm:"type:varchar(255)" json:"source_title"`
	EntryType         string         `gorm:"type:varchar(32);not null" json:"entry_type"`
	Mode              string         `gorm:"type:varchar(32);not null" json:"mode"`
	Status            string         `gorm:"type:varchar(32);not null;index" json:"status"`
	ReviewReason      string         `gorm:"type:text" json:"review_reason"`
	EstimatedMinutes  int            `gorm:"type:integer;default:0" json:"estimated_minutes"`
	ContentDigest     string         `gorm:"type:text" json:"content_digest"`
	SummarySnapshot   string         `gorm:"type:text" json:"summary_snapshot"`
	KeyPointsSnapshot datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"key_points_snapshot"`
	MetadataSnapshot  datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata_snapshot"`
	HintUsedCount     int            `gorm:"type:integer;not null;default:0" json:"hint_used_count"`
	MaxHintCount      int            `gorm:"type:integer;not null;default:2" json:"max_hint_count"`
	TurnCount         int            `gorm:"type:integer;not null;default:0" json:"turn_count"`
	FinalSummary      string         `gorm:"type:text" json:"final_summary"`
	Strengths         datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"strengths"`
	Gaps              datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"gaps"`
	NextFocus         datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"next_focus"`
	FeedbackTags      datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"feedback_tags"`
	StartedAt         time.Time      `gorm:"not null;autoCreateTime" json:"started_at"`
	CompletedAt       *time.Time     `json:"completed_at"`
	AbandonedAt       *time.Time     `json:"abandoned_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate generates UUIDs for review-service owned sessions.
func (s *ReviewSession) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}

	return nil
}

// ReviewTurn records one system/user turn inside a review session.
type ReviewTurn struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	SessionID      uuid.UUID      `gorm:"type:uuid;not null;index:idx_review_turns_session_turn,unique" json:"session_id"`
	TurnIndex      int            `gorm:"type:integer;not null;index:idx_review_turns_session_turn,unique" json:"turn_index"`
	Role           string         `gorm:"type:varchar(16);not null" json:"role"`
	TurnType       string         `gorm:"type:varchar(32);not null" json:"turn_type"`
	Content        string         `gorm:"type:text;not null" json:"content"`
	EvaluationTags datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"evaluation_tags"`
	ExtraPayload   datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"extra_payload"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// BeforeCreate generates UUIDs for review-service owned turns.
func (t *ReviewTurn) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}

	return nil
}

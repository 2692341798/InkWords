package mastery

import (
	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"time"
)

type Skill string

const (
	Explain   Skill = "explain"
	Complete  Skill = "complete"
	Reproduce Skill = "reproduce"
	Transfer  Skill = "transfer"
	Diagnose  Skill = "diagnose"
	Retain    Skill = "retain"
)

var Skills = []Skill{Explain, Complete, Reproduce, Transfer, Diagnose, Retain}

// Attempt is append-only evidence of one learner performance.
type Attempt struct {
	LearnerFiles        []sharedtextbook.LearnerCodeFile `json:"-"`
	AssessmentOutcome   AssessmentOutcome                `json:"-"`
	Skill               Skill
	Answer              string
	PracticeTaskID      string
	PracticeContentHash string
	PracticeSessionID   uuid.UUID
	Correct             bool
	Independent         bool
	HintCount           int
	Took                time.Duration
	Confidence          int
	ErrorKinds          []string
	At                  time.Time
}
type State struct {
	Scores   map[Skill]int
	DueAt    time.Time
	Attempts []Attempt
}
type DueTask struct {
	Skill  Skill     `json:"skill"`
	DueAt  time.Time `json:"due_at"`
	Reason string    `json:"reason"`
}

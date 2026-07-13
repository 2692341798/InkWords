package projectcourse

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	StatusDraft            = "draft"
	StatusAnalyzing        = "analyzing"
	StatusAwaitingApproval = "awaiting_approval"
	StatusApproved         = "approved"
	StatusGenerating       = "generating"
	StatusCompleted        = "completed"
	StatusBlocked          = "blocked"
	StatusFailed           = "failed"
)

// ProjectCourse 持有课程蓝图及其固定源码快照；正文仍归档到现有 blogs。
type ProjectCourse struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	RepositoryURL string    `gorm:"type:text;not null" json:"repository_url"`
	RequestedRef  string    `gorm:"type:text;not null" json:"requested_ref"`
	// ResolvedCommitSHA is empty until the analysis worker captures the immutable snapshot.
	ResolvedCommitSHA string         `gorm:"type:varchar(64);index" json:"resolved_commit_sha"`
	AudienceLevel     string         `gorm:"type:varchar(32);not null" json:"audience_level"`
	Status            string         `gorm:"type:varchar(32);not null;index" json:"status"`
	BlueprintVersion  int            `gorm:"type:integer;not null" json:"blueprint_version"`
	BlueprintJSON     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"blueprint_json"`
	CoverageJSON      datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"coverage_json"`
	QualityReportJSON datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"quality_report_json"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ProjectCourse) TableName() string { return "project_courses" }

func (c *ProjectCourse) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

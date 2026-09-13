package textbook

import (
	"fmt"
	"strings"
)

// BlogProjection is a read-only blog view of one approved manuscript revision.
// Markdown remains owned by the revision; consumers must not treat this value
// as a second editable source of truth.
type BlogProjection struct {
	Format      string `json:"format"`
	RevisionID  string `json:"revision_id"`
	ChapterID   string `json:"chapter_id"`
	Title       string `json:"title"`
	Markdown    string `json:"markdown"`
	ContentHash string `json:"content_hash"`
}

func (projection BlogProjection) Validate() error {
	if projection.Format != "inkwords.blog-projection.v1" || strings.TrimSpace(projection.RevisionID) == "" || strings.TrimSpace(projection.ChapterID) == "" || strings.TrimSpace(projection.Title) == "" || strings.TrimSpace(projection.Markdown) == "" || !isSHA256Digest(projection.ContentHash) {
		return fmt.Errorf("blog projection is incomplete")
	}
	return nil
}

// LearningProjection is a read-only learning view of one approved revision.
// It captures structured practice requirements but does not own scheduling or
// mastery attempts; review-service remains responsible for both.
type LearningProjection struct {
	Format      string              `json:"format"`
	RevisionID  string              `json:"revision_id"`
	ChapterID   string              `json:"chapter_id"`
	ContentHash string              `json:"content_hash"`
	LearningArc LearningArc         `json:"learning_arc"`
	Objectives  []LearningObjective `json:"objectives"`
	PracticeSet *PracticeSet        `json:"practice_set,omitempty"`
	EvidenceIDs []string            `json:"evidence_ids,omitempty"`
}

func (projection LearningProjection) Validate() error {
	if (projection.Format != "inkwords.learning-projection.v1" && projection.Format != "inkwords.learning-projection.v2") || strings.TrimSpace(projection.RevisionID) == "" || strings.TrimSpace(projection.ChapterID) == "" || !isSHA256Digest(projection.ContentHash) || len(projection.Objectives) == 0 {
		return fmt.Errorf("learning projection is incomplete")
	}
	if projection.Format == "inkwords.learning-projection.v2" {
		if projection.PracticeSet == nil {
			return fmt.Errorf("learning projection requires concrete practice tasks")
		}
		if err := projection.PracticeSet.Validate(projection.EvidenceIDs); err != nil {
			return err
		}
	} else if projection.PracticeSet != nil {
		return fmt.Errorf("legacy learning projection cannot carry unversioned practice tasks")
	}
	if err := projection.LearningArc.Validate(); err != nil {
		return fmt.Errorf("learning projection arc: %w", err)
	}
	for _, objective := range projection.Objectives {
		if err := objective.Validate(); err != nil {
			return fmt.Errorf("learning projection objective: %w", err)
		}
	}
	return nil
}

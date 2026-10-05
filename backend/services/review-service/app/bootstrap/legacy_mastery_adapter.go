package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
	masterydomain "inkwords-backend/services/review-service/domain/mastery"
	reviewdomain "inkwords-backend/services/review-service/domain/review"
)

type legacyNoteMasteryAdapter struct{ mastery *masterydomain.Service }

func (adapter legacyNoteMasteryAdapter) MigrateLegacyNote(ctx context.Context, workspaceID uuid.UUID, note reviewdomain.ReviewNote) (uuid.UUID, bool, error) {
	if adapter.mastery == nil || strings.TrimSpace(note.NotePath) == "" || strings.TrimSpace(note.Title) == "" {
		return uuid.Nil, false, fmt.Errorf("invalid legacy note for mastery migration")
	}
	objective, created, err := adapter.mastery.EnsureLegacyNoteObjective(ctx, workspaceID, masterydomain.ObjectiveInput{
		ChapterID:    legacyNoteChapterID(note.NotePath),
		Title:        note.Title,
		Behavior:     "用自己的话解释旧笔记的核心概念；此兼容目标不替代教材的六维掌握任务。",
		Rubric:       []string{"说明核心概念、至少一个关键点及其适用边界。"},
		KeyPoints:    []string{note.Title},
		EvidenceRefs: []string{"legacy-note:" + note.NotePath},
	})
	if err != nil {
		return uuid.Nil, false, err
	}
	return objective.ID, created, nil
}

func legacyNoteChapterID(notePath string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(notePath)))
	return "legacy-note:" + hex.EncodeToString(sum[:])
}

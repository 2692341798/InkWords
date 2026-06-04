package wiki

import (
	"context"
	"log"
	"strings"

	"inkwords-backend/internal/service"
	reviewdomain "inkwords-backend/services/review-service/domain/review"
)

type unavailableReviewNoteSource struct {
	err error
}

func (s unavailableReviewNoteSource) ListEligibleNotes(context.Context) ([]reviewdomain.ReviewNote, error) {
	return nil, s.err
}

// BuildNoteSource keeps Obsidian bootstrap concerns in service-owned infra while reusing the domain note reader.
func BuildNoteSource(rootDir string) reviewdomain.NoteSource {
	store, err := service.NewObsidianStoreFromEnv()
	if err != nil {
		log.Printf("Review note source initialization failed: %v", err)
		return unavailableReviewNoteSource{err: err}
	}

	if strings.TrimSpace(rootDir) == "" {
		rootDir = "wiki"
	}

	return reviewdomain.NewReviewNoteSource(store, rootDir)
}

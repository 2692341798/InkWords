package wiki

import (
	"context"
	"log"
	"strings"

	reviewdomain "inkwords-backend/services/review-service/domain/review"
	"inkwords-backend/shared/platform/obsidian"
)

type unavailableReviewNoteSource struct {
	err error
}

func (s unavailableReviewNoteSource) ListEligibleNotes(context.Context) ([]reviewdomain.ReviewNote, error) {
	return nil, s.err
}

// BuildNoteSource keeps Obsidian bootstrap concerns in service-owned infra while reusing the domain note reader.
func BuildNoteSource(rootDir string) reviewdomain.NoteSource {
	store, err := obsidian.NewStoreFromEnv()
	if err != nil {
		log.Printf("Review note source initialization failed: %v", err)
		return unavailableReviewNoteSource{err: err}
	}

	if strings.TrimSpace(rootDir) == "" {
		rootDir = "wiki"
	}

	return reviewdomain.NewReviewNoteSource(store, rootDir)
}

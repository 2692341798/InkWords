package textbook

import (
	"encoding/json"
	"fmt"

	shared "inkwords-backend/shared/kernel/textbook"
)

// freezeChapterRunbook projects only the approved revision already selected in
// the build transaction. Missing guides stay missing; no live regeneration.
func freezeChapterRunbook(chapter *bookBuildChapterSource) error {
	var doc struct {
		VideoRunbook *shared.VideoRunbookProjection `json:"video_runbook"`
	}
	if len(chapter.DocumentJSON) > 0 {
		if err := json.Unmarshal(chapter.DocumentJSON, &doc); err != nil {
			return fmt.Errorf("%w: malformed approved chapter video runbook", ErrInvalidState)
		}
	}
	hash, err := shared.VideoRunbookHash(doc.VideoRunbook)
	if err != nil {
		return fmt.Errorf("%w: invalid approved chapter video runbook", ErrInvalidState)
	}
	chapter.VideoRunbook, chapter.VideoRunbookHash = doc.VideoRunbook, hash
	return nil
}

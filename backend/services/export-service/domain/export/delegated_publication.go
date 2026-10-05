package export

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// The export projection reads the versioned evidence without importing the
// owner service's models. Every record remains separate from human reviews.
func loadDelegatedPublicationReviews(db *gorm.DB, buildID uuid.UUID, manifestHash string) ([]sharedtextbook.DelegatedPublicationReview, error) {
	var rows []struct{ DocumentJSON json.RawMessage }
	if err := db.Table("textbook_delegated_publication_reviews").Where("build_id = ?", buildID).Order("stage ASC, revision ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	reviews := make([]sharedtextbook.DelegatedPublicationReview, 0, len(rows))
	for _, row := range rows {
		var review sharedtextbook.DelegatedPublicationReview
		if json.Unmarshal(row.DocumentJSON, &review) != nil || review.Validate() != nil || review.BuildID != buildID.String() || review.ManifestHash != manifestHash {
			return nil, fmt.Errorf("invalid frozen delegated publication evidence")
		}
		reviews = append(reviews, review)
	}
	return reviews, nil
}

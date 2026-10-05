package textbook

import (
	"encoding/json"

	shared "inkwords-backend/shared/kernel/textbook"
)

// Persist only an assessment of the bytes frozen in this build. Current asset
// rows are deliberately not consulted because a later revision can differ.
func assetFontReviewMatchesBuild(review shared.AssetFontReview, build BookBuildRow) bool {
	if review.Validate() != nil {
		return false
	}
	var manifest struct {
		Assets []bookBuildAssetSource `json:"assets"`
	}
	if json.Unmarshal(build.ManifestJSON, &manifest) != nil {
		return false
	}
	matches := 0
	for _, asset := range manifest.Assets {
		if asset.ID.String() == review.AssetID {
			if asset.ContentHash != review.ContentHash || (asset.Kind != shared.AssetKindDiagram && asset.Kind != shared.AssetKindScreenshot) {
				return false
			}
			matches++
		}
	}
	return matches == 1
}

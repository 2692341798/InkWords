package textbook

import (
	"fmt"
	"strings"
)

const AssetFontReviewFormat = "inkwords.asset-font-review.v1"

// AssetFontRightsRef pins the font file and the reviewed rights version. An
// empty AmendmentID explicitly refers to the immutable original rights item.
type AssetFontRightsRef struct {
	FileHash     string `json:"file_hash"`
	RightsItemID string `json:"rights_item_id"`
	AmendmentID  string `json:"amendment_id"`
}

// AssetFontReview is a scoped reviewer assertion about text already drawn into
// immutable raster pixels. It is not renderer telemetry or a license grant.
// Actor, authorization and evidence references come from its parent amendment.
type AssetFontReview struct {
	ContractVersion string               `json:"contract_version"`
	AssetID         string               `json:"asset_id"`
	ContentHash     string               `json:"content_hash"`
	Surface         string               `json:"surface"`
	TextStatus      string               `json:"text_status"`
	Scope           string               `json:"scope"`
	Fonts           []AssetFontRightsRef `json:"fonts"`
}

// Validate rejects unsupported carriers and ambiguous or incomplete assertions.
func (r AssetFontReview) Validate() error {
	if r.ContractVersion != AssetFontReviewFormat || strings.TrimSpace(r.AssetID) == "" || !isSHA256Digest(r.ContentHash) || r.Surface != "raster" || len([]rune(strings.TrimSpace(r.Scope))) < 8 || len(r.Scope) > 12000 || len(r.Fonts) > 64 {
		return fmt.Errorf("invalid asset font review scope")
	}
	if (r.TextStatus != "no_text" && r.TextStatus != "identified_fonts") || (r.TextStatus == "no_text") != (len(r.Fonts) == 0) {
		return fmt.Errorf("incomplete asset text assessment")
	}
	seenFiles, seenItems := map[string]bool{}, map[string]bool{}
	for _, font := range r.Fonts {
		if !isSHA256Digest(font.FileHash) || strings.TrimSpace(font.RightsItemID) == "" || len(font.RightsItemID) > 200 || len(font.AmendmentID) > 200 || seenFiles[font.FileHash] || seenItems[font.RightsItemID] {
			return fmt.Errorf("invalid or duplicate asset font rights reference")
		}
		seenFiles[font.FileHash], seenItems[font.RightsItemID] = true, true
	}
	return nil
}

// AssetFontsPublicationReady re-resolves the ledger so callers cannot substitute
// an effective snapshot. Only the latest asset amendment counts; omitted review
// data does not inherit an earlier assertion. Font rights changes stale it even
// when their new status is still ready.
func AssetFontsPublicationReady(asset CanonicalBookAsset, projectID, buildID, manifestHash string, supplied RightsLedger) bool {
	if projectID == "" || supplied.ContractVersion != RightsLedgerFormat || supplied.SelectionRule != "explicit-predecessor-chain-v1" {
		return false
	}
	ledger, err := ResolveRightsLedger(buildID, manifestHash, supplied.OriginalItems, supplied.Amendments)
	if err != nil {
		return false
	}
	items, last := map[string]RightsItem{}, map[string]RightsAmendment{}
	for _, item := range ledger.EffectiveItems {
		items[item.ID] = item
	}
	for _, a := range ledger.Amendments {
		last[a.BaseItemID] = a
	}
	var review *AssetFontReview
	for _, item := range ledger.EffectiveItems {
		if item.SubjectRef != "asset:"+asset.ID {
			continue
		}
		if item.ProjectID != projectID || item.BuildID != buildID || item.PublicationStatus != RightsStatusReady || (item.WorkType != RightsWorkTypeImage && item.WorkType != RightsWorkTypeScreenshot) {
			return false
		}
		review = last[item.ID].AssetFontReview
	}
	if review == nil || review.Validate() != nil || review.AssetID != asset.ID || review.ContentHash != asset.ContentHash {
		return false
	}
	for _, ref := range review.Fonts {
		item, exists := items[ref.RightsItemID]
		if !exists || item.ProjectID != projectID || item.BuildID != buildID || item.SubjectRef != "font-file:"+ref.FileHash || item.WorkType != RightsWorkTypeFont || item.PublicationStatus != RightsStatusReady || last[item.ID].ID != ref.AmendmentID {
			return false
		}
	}
	return true
}

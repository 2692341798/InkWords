package textbook

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const RightsAmendmentFormat = "inkwords.rights-amendment.v1"
const RightsLedgerFormat = "inkwords.rights-ledger.v1"

// RightsAmendment appends evidence without rewriting the original work identity.
type RightsAmendment struct {
	AssetFontReview     *AssetFontReview `json:"asset_font_review,omitempty"`
	ContractVersion     string           `json:"contract_version"`
	ID                  string           `json:"id"`
	BuildID             string           `json:"build_id"`
	ManifestHash        string           `json:"manifest_hash"`
	BaseItemID          string           `json:"base_item_id"`
	PreviousAmendmentID string           `json:"previous_amendment_id"`
	Revision            int              `json:"revision"`
	ReviewerKind        string           `json:"reviewer_kind"`
	Reviewer            string           `json:"reviewer"`
	DelegationNote      string           `json:"delegation_note"`
	Reason              string           `json:"reason"`
	EvidenceRefs        []string         `json:"evidence_refs"`
	EffectiveItem       RightsItem       `json:"effective_item"`
	CompletedAt         time.Time        `json:"completed_at"`
}

// Validate checks provenance, completeness and the immutable identity envelope.
func (a RightsAmendment) Validate() error {
	if a.AssetFontReview != nil && (a.AssetFontReview.Validate() != nil || a.EffectiveItem.SubjectRef != "asset:"+a.AssetFontReview.AssetID || (a.EffectiveItem.WorkType != RightsWorkTypeScreenshot && a.EffectiveItem.WorkType != RightsWorkTypeImage)) {
		return fmt.Errorf("invalid asset font amendment binding")
	}
	if a.ContractVersion != RightsAmendmentFormat || strings.TrimSpace(a.ID) == "" || strings.TrimSpace(a.BaseItemID) == "" || !isSHA256Digest(a.ManifestHash) || a.Revision < 1 || (a.Revision == 1) != (a.PreviousAmendmentID == "") || a.CompletedAt.IsZero() || a.ID == a.PreviousAmendmentID {
		return fmt.Errorf("invalid rights amendment identity")
	}
	if (a.ReviewerKind != "human" && a.ReviewerKind != "delegated_ai") || strings.TrimSpace(a.Reviewer) == "" || len([]rune(strings.TrimSpace(a.Reason))) < 8 || len(a.Reason) > 12000 || len(a.Reviewer) > 400 || len(a.DelegationNote) > 12000 || len(a.EvidenceRefs) == 0 || len(a.EvidenceRefs) > 100 || (a.ReviewerKind == "delegated_ai" && len([]rune(strings.TrimSpace(a.DelegationNote))) < 8) {
		return fmt.Errorf("incomplete rights amendment provenance")
	}
	for _, ref := range a.EvidenceRefs {
		if strings.TrimSpace(ref) == "" || len(ref) > 2000 {
			return fmt.Errorf("invalid rights evidence reference")
		}
	}
	if a.EffectiveItem.Validate() != nil || a.EffectiveItem.ID != a.BaseItemID || a.EffectiveItem.BuildID != a.BuildID || len(a.EffectiveItem.RightsBasis) > 12000 || len(a.EffectiveItem.AllowedUse) > 12000 || len(a.EffectiveItem.Attribution) > 12000 {
		return fmt.Errorf("invalid effective rights snapshot")
	}
	return nil
}

// RightsLedger exports both history and the result of the versioned chain rule.
type RightsLedger struct {
	ContractVersion string            `json:"contract_version"`
	SelectionRule   string            `json:"selection_rule"`
	OriginalItems   []RightsItem      `json:"original_items"`
	Amendments      []RightsAmendment `json:"amendments"`
	EffectiveItems  []RightsItem      `json:"effective_items"`
}

// ResolveRightsLedger fails closed on forks, gaps and identity changes. Input
// order and timestamps never choose a winner; only an unbroken explicit chain does.
func ResolveRightsLedger(buildID, manifestHash string, originals []RightsItem, amendments []RightsAmendment) (RightsLedger, error) {
	ledger := RightsLedger{ContractVersion: RightsLedgerFormat, SelectionRule: "explicit-predecessor-chain-v1", OriginalItems: append([]RightsItem{}, originals...), Amendments: append([]RightsAmendment{}, amendments...), EffectiveItems: append([]RightsItem{}, originals...)}
	byID, subjects := map[string]int{}, map[string]bool{}
	for i, item := range originals {
		if item.Validate() != nil || item.BuildID != buildID || subjects[item.SubjectRef] {
			return RightsLedger{}, fmt.Errorf("invalid or duplicate original rights item")
		}
		if _, exists := byID[item.ID]; exists {
			return RightsLedger{}, fmt.Errorf("duplicate rights item ID")
		}
		byID[item.ID], subjects[item.SubjectRef] = i, true
	}
	sort.Slice(ledger.Amendments, func(i, j int) bool {
		a, b := ledger.Amendments[i], ledger.Amendments[j]
		if a.BaseItemID != b.BaseItemID {
			return a.BaseItemID < b.BaseItemID
		}
		return a.Revision < b.Revision
	})
	last, revisions, seen := map[string]string{}, map[string]int{}, map[string]bool{}
	for _, a := range ledger.Amendments {
		i, exists := byID[a.BaseItemID]
		if a.Validate() != nil || !exists || a.BuildID != buildID || a.ManifestHash != manifestHash || seen[a.ID] || a.PreviousAmendmentID != last[a.BaseItemID] || a.Revision != revisions[a.BaseItemID]+1 {
			return RightsLedger{}, fmt.Errorf("invalid rights amendment chain")
		}
		base := originals[i]
		if a.EffectiveItem.ProjectID != base.ProjectID || a.EffectiveItem.SubjectRef != base.SubjectRef || a.EffectiveItem.WorkType != base.WorkType {
			return RightsLedger{}, fmt.Errorf("rights amendment changes work identity")
		}
		ledger.EffectiveItems[i] = a.EffectiveItem
		last[a.BaseItemID], revisions[a.BaseItemID], seen[a.ID] = a.ID, a.Revision, true
	}
	return ledger, nil
}

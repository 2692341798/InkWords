package projectcourse

import (
	"fmt"
	"strings"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

type EvidencePack struct {
	ChapterID       string                     `json:"chapter_id"`
	SourceEvidence  []sharedkernel.EvidenceRef `json:"source_evidence"`
	OfficialSources []OfficialSource           `json:"official_sources"`
}

func (p EvidencePack) Validate() error {
	if strings.TrimSpace(p.ChapterID) == "" {
		return fmt.Errorf("evidence pack chapter id is required")
	}
	for _, evidence := range p.SourceEvidence {
		if err := evidence.Validate(); err != nil {
			return err
		}
	}
	for _, source := range p.OfficialSources {
		if strings.TrimSpace(source.ContentHash) == "" {
			return fmt.Errorf("official source %q has no content hash", source.Technology)
		}
	}
	return nil
}

func ValidateClaimLedger(pack EvidencePack, claims []sharedkernel.Claim) error {
	if err := pack.Validate(); err != nil {
		return err
	}
	known := make(map[string]bool, len(pack.SourceEvidence))
	for _, evidence := range pack.SourceEvidence {
		known[evidence.EvidenceID] = true
	}
	for _, claim := range claims {
		if err := claim.Validate(); err != nil {
			return err
		}
		if claim.Status != sharedkernel.ClaimVerified {
			return fmt.Errorf("claim %q is not verified", claim.ClaimID)
		}
		for _, evidenceID := range claim.EvidenceIDs {
			if !known[evidenceID] {
				return fmt.Errorf("claim %q references unknown evidence %q", claim.ClaimID, evidenceID)
			}
		}
	}
	return nil
}

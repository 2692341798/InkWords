package projectcourse

import (
	"fmt"
	"strings"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

// ClaimPlan is the reviewable, pre-prose contract for a chapter.
type ClaimPlan struct {
	Claims      []sharedkernel.Claim `json:"claims"`
	EvidenceIDs []string             `json:"evidence_ids"`
}

func (p ClaimPlan) Validate(pack EvidencePack) error {
	if err := pack.Validate(); err != nil {
		return err
	}
	if len(p.Claims) == 0 {
		return fmt.Errorf("claim plan must contain at least one claim")
	}
	if err := ValidateClaimLedger(pack, p.Claims); err != nil {
		return err
	}
	known := make(map[string]bool, len(pack.SourceEvidence))
	for _, evidence := range pack.SourceEvidence {
		known[evidence.EvidenceID] = true
	}
	for _, evidenceID := range p.EvidenceIDs {
		if !known[evidenceID] {
			return fmt.Errorf("claim plan references unknown evidence %q", evidenceID)
		}
	}
	return nil
}

// BuildChapterDocument assembles a chapter only after its claim plan and lab
// artifact have passed deterministic contract checks. Model-generated markdown
// is treated as presentation; it cannot add unverified project facts here.
func BuildChapterDocument(chapter sharedkernel.Chapter, pack EvidencePack, plan ClaimPlan, markdown string, lab *sharedkernel.LabManifest, verifiedLab bool) (ChapterDocument, QualityGateReport, error) {
	if strings.TrimSpace(markdown) == "" {
		return ChapterDocument{}, QualityGateReport{}, fmt.Errorf("chapter markdown is required")
	}
	if err := plan.Validate(pack); err != nil {
		return ChapterDocument{}, QualityGateReport{}, fmt.Errorf("validate claim plan: %w", err)
	}
	document := ChapterDocument{ChapterID: chapter.ID, ChapterType: chapter.Type, Title: chapter.Title, Markdown: markdown, Claims: plan.Claims, EvidencePack: pack, Lab: lab}
	report := RunChapterQualityGates(document, verifiedLab)
	if report.Result == sharedkernel.GateHardFail {
		return document, report, fmt.Errorf("chapter quality gate failed")
	}
	return document, report, nil
}

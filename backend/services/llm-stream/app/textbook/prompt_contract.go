package textbook

import (
	"fmt"
	"strings"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// SampleGenerationRequest is the complete, reviewable input boundary for a sample chapter.
// Providers receive this contract rather than raw user documents or ambient project state.
type SampleGenerationRequest struct {
	ProjectID        string                                `json:"project_id"`
	GenerationTarget sharedtextbook.SampleGenerationTarget `json:"generation_target"`
	Audience         sharedtextbook.AudienceLevel          `json:"audience"`
	BookContract     sharedtextbook.BookContract           `json:"book_contract"`
	StyleSheet       sharedtextbook.StyleSheet             `json:"style_sheet"`
	BlueprintChapter sharedtextbook.BlueprintChapter       `json:"blueprint_chapter"`
	EvidencePack     EvidencePack                          `json:"evidence_pack"`
	ClaimCandidates  []ClaimCandidate                      `json:"claim_candidates"`
}

// Validate prevents a provider from silently mixing audience, contract, and evidence revisions.
func (request SampleGenerationRequest) Validate() error {
	if strings.TrimSpace(request.ProjectID) == "" {
		return fmt.Errorf("Gin sample generation requires a project id")
	}
	if err := request.Audience.Validate(); err != nil {
		return err
	}
	if err := request.GenerationTarget.Validate(); err != nil {
		return err
	}
	if err := request.BookContract.Validate(); err != nil {
		return fmt.Errorf("validate book contract: %w", err)
	}
	if request.BookContract.ProjectID != request.ProjectID || request.BookContract.Reader.Audience != request.Audience {
		return fmt.Errorf("book contract does not match sample project or audience")
	}
	if err := request.StyleSheet.Validate(); err != nil {
		return fmt.Errorf("validate style sheet: %w", err)
	}
	if request.StyleSheet.ProjectID != request.ProjectID {
		return fmt.Errorf("style sheet does not match sample project")
	}
	if err := request.BlueprintChapter.Validate(); err != nil {
		return fmt.Errorf("validate blueprint chapter: %w", err)
	}
	if request.BlueprintChapter.Profile != sharedtextbook.ChapterProfileConcept && request.BlueprintChapter.Profile != sharedtextbook.ChapterProfileHandsOn {
		return fmt.Errorf("sample generation does not support this chapter profile")
	}
	profileAllowed := false
	for _, profile := range request.BookContract.ChapterProfiles {
		profileAllowed = profileAllowed || profile == request.BlueprintChapter.Profile
	}
	if !profileAllowed {
		return fmt.Errorf("sample chapter profile is not allowed by the book contract")
	}
	if err := request.BlueprintChapter.ValidateCriticalClaimCoverage(); err != nil {
		return fmt.Errorf("validate blueprint critical claims: %w", err)
	}
	if err := request.EvidencePack.Validate(); err != nil {
		return fmt.Errorf("validate evidence pack: %w", err)
	}
	knownEvidence := make(map[string]bool, len(request.EvidencePack.Evidence))
	for _, evidence := range request.EvidencePack.Evidence {
		knownEvidence[evidence.ID] = true
	}
	for _, evidenceID := range request.BlueprintChapter.EvidenceIDs {
		if !knownEvidence[evidenceID] {
			return fmt.Errorf("blueprint chapter references evidence outside the frozen pack: %s", evidenceID)
		}
	}
	if conflicts := DetectSourceConflicts(request.ClaimCandidates); len(conflicts) > 0 {
		keys := make([]string, 0, len(conflicts))
		for _, conflict := range conflicts {
			keys = append(keys, conflict.Key)
		}
		return fmt.Errorf("resolve source conflicts before generation: %s", strings.Join(keys, ", "))
	}
	return nil
}

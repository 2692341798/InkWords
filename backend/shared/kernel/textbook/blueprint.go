package textbook

import (
	"fmt"
	"strings"
)

// BlueprintChapter is a reviewable teaching plan, not a generated manuscript.
type BlueprintChapter struct {
	ID              string             `json:"id"`
	Title           string             `json:"title"`
	Sort            int                `json:"sort"`
	Profile         ChapterProfile     `json:"profile"`
	PrerequisiteIDs []string           `json:"prerequisite_ids,omitempty"`
	OutcomeIDs      []string           `json:"outcome_ids,omitempty"`
	EvidenceIDs     []string           `json:"evidence_ids"`
	CriticalClaims  []ClaimRequirement `json:"critical_claims,omitempty"`
}

func (chapter BlueprintChapter) Validate() error {
	if strings.TrimSpace(chapter.ID) == "" || strings.TrimSpace(chapter.Title) == "" || chapter.Sort < 1 || len(chapter.EvidenceIDs) == 0 {
		return fmt.Errorf("blueprint chapter is incomplete")
	}
	return chapter.Profile.Validate()
}

// ClaimRequirement is an author-reviewed factual obligation for a chapter.
// It is deliberately a plan rather than generated prose: writing may not
// start until every required fact has a concrete evidence route.
type ClaimRequirement struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	EvidenceIDs []string `json:"evidence_ids"`
}

func (requirement ClaimRequirement) Validate() error {
	if strings.TrimSpace(requirement.ID) == "" || strings.TrimSpace(requirement.Label) == "" || len(requirement.EvidenceIDs) == 0 {
		return fmt.Errorf("critical claim identity and evidence are required")
	}
	return nil
}

// ValidateCriticalClaimCoverage is intentionally separate from Validate so an
// author may save an incomplete draft. Approval and task freezing call this
// stricter gate, where incomplete teaching evidence must stop generation.
func (chapter BlueprintChapter) ValidateCriticalClaimCoverage() error {
	if len(chapter.CriticalClaims) == 0 {
		return fmt.Errorf("chapter %q has no critical claim requirements", chapter.ID)
	}
	knownEvidence := make(map[string]bool, len(chapter.EvidenceIDs))
	for _, evidenceID := range chapter.EvidenceIDs {
		knownEvidence[evidenceID] = true
	}
	seenClaims := make(map[string]bool, len(chapter.CriticalClaims))
	for _, claim := range chapter.CriticalClaims {
		if err := claim.Validate(); err != nil {
			return fmt.Errorf("chapter %q: %w", chapter.ID, err)
		}
		if seenClaims[claim.ID] {
			return fmt.Errorf("chapter %q has duplicate critical claim %q", chapter.ID, claim.ID)
		}
		seenClaims[claim.ID] = true
		for _, evidenceID := range claim.EvidenceIDs {
			if !knownEvidence[evidenceID] {
				return fmt.Errorf("chapter %q critical claim %q references evidence outside the chapter", chapter.ID, claim.ID)
			}
		}
	}
	return nil
}

// ValidateGenerationReadiness rejects a blueprint whose teaching facts cannot
// be traced to selected evidence before a worker receives any source text.
func (blueprint Blueprint) ValidateGenerationReadiness() error {
	if err := blueprint.Validate(); err != nil {
		return err
	}
	for _, volume := range blueprint.Volumes {
		for _, chapter := range volume.Chapters {
			if err := chapter.ValidateCriticalClaimCoverage(); err != nil {
				return err
			}
		}
	}
	return nil
}

type BlueprintVolume struct {
	ID       string             `json:"id"`
	Title    string             `json:"title"`
	Sort     int                `json:"sort"`
	Chapters []BlueprintChapter `json:"chapters"`
}

func (volume BlueprintVolume) Validate() error {
	if strings.TrimSpace(volume.ID) == "" || strings.TrimSpace(volume.Title) == "" || volume.Sort < 1 || len(volume.Chapters) == 0 {
		return fmt.Errorf("blueprint volume is incomplete")
	}
	return nil
}

// Blueprint freezes the chapter DAG that generation and approval must share.
type Blueprint struct {
	RevisionID           string            `json:"revision_id"`
	ProjectID            string            `json:"project_id"`
	RevisionNumber       int               `json:"revision_number"`
	ContentHash          string            `json:"content_hash"`
	BookContractRevision string            `json:"book_contract_revision"`
	StyleSheetRevision   string            `json:"style_sheet_revision"`
	Volumes              []BlueprintVolume `json:"volumes"`
}

func (blueprint Blueprint) Validate() error {
	if strings.TrimSpace(blueprint.RevisionID) == "" || strings.TrimSpace(blueprint.ProjectID) == "" || blueprint.RevisionNumber < 1 || strings.TrimSpace(blueprint.BookContractRevision) == "" || strings.TrimSpace(blueprint.StyleSheetRevision) == "" || !isSHA256Digest(blueprint.ContentHash) {
		return fmt.Errorf("blueprint identity and contract revisions are required")
	}
	if len(blueprint.Volumes) == 0 {
		return fmt.Errorf("blueprint requires volumes")
	}

	known := make(map[string]BlueprintChapter)
	ordered := make([]BlueprintChapter, 0)
	for _, volume := range blueprint.Volumes {
		if err := volume.Validate(); err != nil {
			return err
		}
		for _, chapter := range volume.Chapters {
			if err := chapter.Validate(); err != nil {
				return err
			}
			if _, exists := known[chapter.ID]; exists {
				return fmt.Errorf("duplicate blueprint chapter %q", chapter.ID)
			}
			known[chapter.ID] = chapter
			ordered = append(ordered, chapter)
		}
	}
	if ordered[0].Profile == ChapterProfileReference {
		return fmt.Errorf("reference chapter cannot be the first teaching chapter")
	}
	for _, chapter := range ordered {
		for _, prerequisiteID := range chapter.PrerequisiteIDs {
			if prerequisiteID == chapter.ID {
				return fmt.Errorf("blueprint chapter %q cannot require itself", chapter.ID)
			}
			if _, exists := known[prerequisiteID]; !exists {
				return fmt.Errorf("blueprint chapter %q references missing prerequisite %q", chapter.ID, prerequisiteID)
			}
		}
	}
	if blueprintHasCycle(known) {
		return fmt.Errorf("blueprint prerequisites contain a cycle")
	}
	return nil
}

func blueprintHasCycle(chapters map[string]BlueprintChapter) bool {
	visiting := make(map[string]bool, len(chapters))
	visited := make(map[string]bool, len(chapters))
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return true
		}
		if visited[id] {
			return false
		}
		visiting[id] = true
		for _, prerequisiteID := range chapters[id].PrerequisiteIDs {
			if visit(prerequisiteID) {
				return true
			}
		}
		visiting[id] = false
		visited[id] = true
		return false
	}
	for id := range chapters {
		if visit(id) {
			return true
		}
	}
	return false
}

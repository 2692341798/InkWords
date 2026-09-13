package textbook

import (
	"fmt"
	"strings"
)

type AnalogyMapping struct {
	AnalogyElement   string `json:"analogy_element"`
	TechnicalElement string `json:"technical_element"`
}

func (mapping AnalogyMapping) Validate() error {
	if strings.TrimSpace(mapping.AnalogyElement) == "" || strings.TrimSpace(mapping.TechnicalElement) == "" {
		return fmt.Errorf("analogy mapping requires both elements")
	}
	return nil
}

// ScenarioFrame provides beginner-friendly context without treating an analogy as evidence.
type ScenarioFrame struct {
	Situation       string           `json:"situation"`
	Trigger         string           `json:"trigger"`
	Consequence     string           `json:"consequence"`
	Analogy         string           `json:"analogy"`
	Mapping         []AnalogyMapping `json:"mapping"`
	Mechanism       string           `json:"mechanism"`
	Boundary        string           `json:"boundary"`
	DemonstrationID string           `json:"demonstration_id"`
}

func (frame ScenarioFrame) Validate() error {
	if strings.TrimSpace(frame.Situation) == "" || strings.TrimSpace(frame.Trigger) == "" || strings.TrimSpace(frame.Consequence) == "" || strings.TrimSpace(frame.Analogy) == "" || strings.TrimSpace(frame.Mechanism) == "" || strings.TrimSpace(frame.Boundary) == "" || strings.TrimSpace(frame.DemonstrationID) == "" {
		return fmt.Errorf("scenario frame is incomplete")
	}
	if len(frame.Mapping) == 0 {
		return fmt.Errorf("scenario frame requires analogy mappings")
	}
	for _, mapping := range frame.Mapping {
		if err := mapping.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// UnderstandingChain keeps the technical explanation, observation, and assessment connected.
type UnderstandingChain struct {
	Need         string   `json:"need"`
	Problem      string   `json:"problem"`
	Alternatives string   `json:"alternatives"`
	Usage        string   `json:"usage"`
	Mechanism    string   `json:"mechanism"`
	Observation  string   `json:"observation"`
	Tradeoffs    string   `json:"tradeoffs"`
	AssessmentID []string `json:"assessment_ids"`
}

func (chain UnderstandingChain) Validate() error {
	if strings.TrimSpace(chain.Need) == "" || strings.TrimSpace(chain.Problem) == "" || strings.TrimSpace(chain.Alternatives) == "" || strings.TrimSpace(chain.Usage) == "" || strings.TrimSpace(chain.Mechanism) == "" || strings.TrimSpace(chain.Observation) == "" || strings.TrimSpace(chain.Tradeoffs) == "" {
		return fmt.Errorf("understanding chain is incomplete")
	}
	if len(chain.AssessmentID) == 0 {
		return fmt.Errorf("understanding chain requires assessment ids")
	}
	return nil
}

// ChapterRevision is immutable content. Applying it to a chapter is owned by core-api.
type ChapterRevision struct {
	ID                   string       `json:"id"`
	ChapterID            string       `json:"chapter_id"`
	Kind                 RevisionKind `json:"kind"`
	Markdown             string       `json:"markdown"`
	DocumentHash         string       `json:"document_hash"`
	ContentHash          string       `json:"content_hash"`
	BookContractRevision string       `json:"book_contract_revision"`
	StyleSheetRevision   string       `json:"style_sheet_revision"`
	EvidencePackHash     string       `json:"evidence_pack_hash"`
}

func (revision ChapterRevision) Validate() error {
	if strings.TrimSpace(revision.ID) == "" || strings.TrimSpace(revision.ChapterID) == "" || strings.TrimSpace(revision.Markdown) == "" || strings.TrimSpace(revision.BookContractRevision) == "" || strings.TrimSpace(revision.StyleSheetRevision) == "" {
		return fmt.Errorf("chapter revision identity and manuscript contracts are required")
	}
	if err := revision.Kind.Validate(); err != nil {
		return err
	}
	if !isSHA256Digest(revision.DocumentHash) || !isSHA256Digest(revision.ContentHash) || !isSHA256Digest(revision.EvidencePackHash) {
		return fmt.Errorf("chapter revision hashes are required")
	}
	return nil
}

// CodeArtifact separates a runnable teaching implementation from a production
// source walkthrough, so a simplified demonstration cannot masquerade as upstream code.
type CodeArtifact struct {
	ID             string           `json:"id"`
	RevisionID     string           `json:"revision_id"`
	Kind           CodeArtifactKind `json:"kind"`
	Language       string           `json:"language"`
	Entrypoint     string           `json:"entrypoint,omitempty"`
	SourceTreePath string           `json:"source_tree_path,omitempty"`
	SourceRef      string           `json:"source_ref,omitempty"`
	ManifestHash   string           `json:"manifest_hash"`
	ArtifactHash   string           `json:"artifact_hash"`
	Limitations    []string         `json:"limitations,omitempty"`
	Status         ArtifactStatus   `json:"status"`
}

func (artifact CodeArtifact) Validate() error {
	if strings.TrimSpace(artifact.ID) == "" || strings.TrimSpace(artifact.RevisionID) == "" || strings.TrimSpace(artifact.Language) == "" || !isSHA256Digest(artifact.ManifestHash) || !isSHA256Digest(artifact.ArtifactHash) {
		return fmt.Errorf("code artifact identity, manifest hash, and artifact hash are required")
	}
	if err := artifact.Kind.Validate(); err != nil {
		return err
	}
	if err := artifact.Status.Validate(); err != nil {
		return err
	}
	switch artifact.Kind {
	case CodeArtifactTeachingImplementation:
		if strings.TrimSpace(artifact.Entrypoint) == "" || len(artifact.Limitations) == 0 {
			return fmt.Errorf("teaching implementation requires an entrypoint and limitations")
		}
	case CodeArtifactUpstreamWalkthrough:
		if strings.TrimSpace(artifact.SourceRef) == "" {
			return fmt.Errorf("upstream walkthrough requires an evidence source reference")
		}
	}
	return nil
}

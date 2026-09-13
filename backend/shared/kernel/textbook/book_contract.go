package textbook

import (
	"fmt"
	"strings"
)

// ReaderModel states what the selected audience may and may not be assumed to know.
type ReaderModel struct {
	Audience             AudienceLevel `json:"audience"`
	KnownKnowledge       []string      `json:"known_knowledge"`
	ForbiddenAssumptions []string      `json:"forbidden_assumptions"`
	LearningOutcomes     []string      `json:"learning_outcomes"`
}

func (model ReaderModel) Validate() error {
	if err := model.Audience.Validate(); err != nil {
		return err
	}
	if len(model.LearningOutcomes) == 0 {
		return fmt.Errorf("reader model requires learning outcomes")
	}
	return nil
}

// BookContract freezes the pedagogical and publication promises before batch generation.
type BookContract struct {
	RevisionID         string           `json:"revision_id"`
	ProjectID          string           `json:"project_id"`
	RevisionNumber     int              `json:"revision_number"`
	ContentHash        string           `json:"content_hash"`
	Reader             ReaderModel      `json:"reader"`
	Promise            string           `json:"promise"`
	ChapterProfiles    []ChapterProfile `json:"chapter_profiles"`
	TerminologyVersion string           `json:"terminology_version"`
	PublicationProfile string           `json:"publication_profile"`
}

func (contract BookContract) Validate() error {
	if strings.TrimSpace(contract.RevisionID) == "" || strings.TrimSpace(contract.ProjectID) == "" || contract.RevisionNumber < 1 || strings.TrimSpace(contract.Promise) == "" || !isSHA256Digest(contract.ContentHash) {
		return fmt.Errorf("book contract identity and promise are required")
	}
	if err := contract.Reader.Validate(); err != nil {
		return err
	}
	if len(contract.ChapterProfiles) == 0 {
		return fmt.Errorf("book contract requires chapter profiles")
	}
	for _, profile := range contract.ChapterProfiles {
		if err := profile.Validate(); err != nil {
			return err
		}
	}
	return nil
}

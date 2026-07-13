package projectcourse

import (
	"fmt"
	"strings"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

type ChapterDocument struct {
	ChapterID    string                    `json:"chapter_id"`
	ChapterType  sharedkernel.ChapterType  `json:"chapter_type"`
	Title        string                    `json:"title"`
	Markdown     string                    `json:"markdown"`
	Claims       []sharedkernel.Claim      `json:"claims"`
	EvidencePack EvidencePack              `json:"evidence_pack"`
	Lab          *sharedkernel.LabManifest `json:"lab,omitempty"`
}

func (d ChapterDocument) ValidateContract() error {
	if strings.TrimSpace(d.ChapterID) == "" || strings.TrimSpace(d.Title) == "" || strings.TrimSpace(d.Markdown) == "" {
		return fmt.Errorf("chapter id, title and markdown are required")
	}
	if err := d.ChapterType.Validate(); err != nil {
		return err
	}
	if err := ValidateClaimLedger(d.EvidencePack, d.Claims); err != nil {
		return err
	}
	if d.Lab != nil {
		if err := d.Lab.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func ValidateChapterSet(chapters []ChapterDocument, dependencies map[string][]string) error {
	known := make(map[string]bool, len(chapters))
	for _, chapter := range chapters {
		if err := chapter.ValidateContract(); err != nil {
			return fmt.Errorf("chapter %s: %w", chapter.ChapterID, err)
		}
		if known[chapter.ChapterID] {
			return fmt.Errorf("duplicate chapter %q", chapter.ChapterID)
		}
		known[chapter.ChapterID] = true
	}
	for chapterID, prerequisites := range dependencies {
		if !known[chapterID] {
			return fmt.Errorf("dependency references unknown chapter %q", chapterID)
		}
		for _, prerequisite := range prerequisites {
			if !known[prerequisite] {
				return fmt.Errorf("chapter %q references unknown prerequisite %q", chapterID, prerequisite)
			}
		}
	}
	return nil
}

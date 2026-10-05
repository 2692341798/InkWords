package textbook

import (
	"fmt"
	"strings"
)

// StyleSheet freezes reusable writing and publishing rules for a manuscript revision.
type StyleSheet struct {
	RevisionID       string   `json:"revision_id"`
	ProjectID        string   `json:"project_id"`
	RevisionNumber   int      `json:"revision_number"`
	ContentHash      string   `json:"content_hash"`
	Language         string   `json:"language"`
	TerminologyRules []string `json:"terminology_rules"`
	CodeRules        []string `json:"code_rules"`
	VisualRules      []string `json:"visual_rules"`
	CitationRules    []string `json:"citation_rules"`
	ForbiddenPhrases []string `json:"forbidden_phrases"`
}

func (sheet StyleSheet) Validate() error {
	if strings.TrimSpace(sheet.RevisionID) == "" || strings.TrimSpace(sheet.ProjectID) == "" || sheet.RevisionNumber < 1 || !isSHA256Digest(sheet.ContentHash) {
		return fmt.Errorf("style sheet identity is required")
	}
	if strings.TrimSpace(sheet.Language) == "" {
		return fmt.Errorf("style sheet language is required")
	}
	if len(sheet.TerminologyRules) == 0 || len(sheet.CodeRules) == 0 || len(sheet.CitationRules) == 0 {
		return fmt.Errorf("style sheet requires terminology, code and citation rules")
	}
	return nil
}

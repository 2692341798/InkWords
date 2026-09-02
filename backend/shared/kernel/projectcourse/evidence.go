package projectcourse

import (
	"fmt"
	"strings"
)

// EvidenceRef 指向固定快照中的源码证据。
type EvidenceRef struct {
	EvidenceID  string `json:"evidence_id"`
	CommitSHA   string `json:"commit_sha"`
	Path        string `json:"path"`
	Symbol      string `json:"symbol,omitempty"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	ContentHash string `json:"content_hash"`
}

func (e EvidenceRef) Validate() error {
	if strings.TrimSpace(e.EvidenceID) == "" || strings.TrimSpace(e.CommitSHA) == "" || strings.TrimSpace(e.Path) == "" || strings.TrimSpace(e.ContentHash) == "" {
		return fmt.Errorf("evidence_id, commit_sha, path and content_hash are required")
	}
	if e.StartLine < 1 || e.EndLine < e.StartLine {
		return fmt.Errorf("invalid evidence line range")
	}
	return nil
}

// Claim 是文章中需要被证据校验的项目论断。
type Claim struct {
	ClaimID     string             `json:"claim_id"`
	Text        string             `json:"text"`
	ClaimType   string             `json:"claim_type"`
	Confidence  EvidenceConfidence `json:"confidence"`
	EvidenceIDs []string           `json:"evidence_ids"`
	Status      ClaimStatus        `json:"status"`
}

func (c Claim) Validate() error {
	if strings.TrimSpace(c.ClaimID) == "" || strings.TrimSpace(c.Text) == "" || strings.TrimSpace(c.ClaimType) == "" {
		return fmt.Errorf("claim_id, text and claim_type are required")
	}
	if err := c.Confidence.Validate(); err != nil {
		return err
	}
	if err := c.Status.Validate(); err != nil {
		return err
	}
	if len(c.EvidenceIDs) == 0 {
		return fmt.Errorf("claim must reference evidence")
	}
	return nil
}

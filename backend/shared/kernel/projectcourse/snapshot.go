package projectcourse

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// SourceSnapshot 是一次不可变的仓库输入快照。
type SourceSnapshot struct {
	RepositoryURL     string    `json:"repository_url"`
	RequestedRef      string    `json:"requested_ref"`
	ResolvedCommitSHA string    `json:"resolved_commit_sha"`
	CapturedAt        time.Time `json:"captured_at"`
	DefaultBranch     string    `json:"default_branch"`
}

func (s SourceSnapshot) Validate() error {
	if strings.TrimSpace(s.RepositoryURL) == "" {
		return fmt.Errorf("repository_url is required")
	}
	if strings.TrimSpace(s.RequestedRef) == "" {
		return fmt.Errorf("requested_ref is required")
	}
	sha := strings.TrimSpace(s.ResolvedCommitSHA)
	if len(sha) != 40 {
		return fmt.Errorf("resolved_commit_sha must be a 40-character SHA")
	}
	if _, err := hex.DecodeString(sha); err != nil {
		return fmt.Errorf("resolved_commit_sha must be hexadecimal: %w", err)
	}
	if s.CapturedAt.IsZero() {
		return fmt.Errorf("captured_at is required")
	}
	return nil
}

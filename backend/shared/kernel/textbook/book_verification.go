package textbook

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// BookVerificationSnapshotFormat versions observations pinned by a book build.
const BookVerificationSnapshotFormat = "inkwords.book-verification-snapshot.v1"

// FrozenBookVerificationArtifact keeps the exact execution inputs and receipts;
// it contains no host paths and never causes execution while exporting.
type FrozenBookVerificationArtifact struct {
	ID           string            `json:"id"`
	RevisionID   string            `json:"revision_id"`
	ArtifactHash string            `json:"artifact_hash"`
	ManifestHash string            `json:"manifest_hash"`
	Manifest     json.RawMessage   `json:"manifest"`
	Evidence     []RuntimeEvidence `json:"evidence"`
}

// BookVerificationSnapshot is immutable. A dated assessment is derived from
// these receipts, so a historical pass is never silently renewed at export.
type BookVerificationSnapshot struct {
	Format           string                           `json:"format"`
	CapturedAt       time.Time                        `json:"captured_at"`
	BookContractHash string                           `json:"book_contract_hash"`
	StyleSheetHash   string                           `json:"style_sheet_hash"`
	Artifacts        []FrozenBookVerificationArtifact `json:"artifacts"`
}

// BookVerificationAssessment describes completeness for the declared commands,
// not learner mastery, fresh installation, or unspecified runtime behavior.
type BookVerificationAssessment struct {
	EvaluatedAt time.Time                            `json:"evaluated_at"`
	Passed      bool                                 `json:"passed"`
	Artifacts   []BookArtifactVerificationAssessment `json:"artifacts"`
}

// BookArtifactVerificationAssessment makes missing and rejected evidence visible.
type BookArtifactVerificationAssessment struct {
	ArtifactID  string   `json:"artifact_id"`
	Passed      bool     `json:"passed"`
	EvidenceIDs []string `json:"accepted_evidence_ids"`
	Reasons     []string `json:"reasons"`
}

// Validate checks snapshot structure; invalid or missing execution receipts are
// retained as non-passing observations instead of preventing draft exports.
func (s BookVerificationSnapshot) Validate() error {
	if s.Format != BookVerificationSnapshotFormat || s.CapturedAt.IsZero() || !isFullSHA256Digest(s.BookContractHash) || !isFullSHA256Digest(s.StyleSheetHash) || s.Artifacts == nil {
		return fmt.Errorf("invalid book verification snapshot")
	}
	ids := map[string]bool{}
	for _, a := range s.Artifacts {
		if a.ID == "" || a.RevisionID == "" || ids[a.ID] || !isFullSHA256Digest(a.ArtifactHash) || !isFullSHA256Digest(a.ManifestHash) {
			return fmt.Errorf("invalid frozen verification artifact identity")
		}
		ids[a.ID] = true
	}
	return nil
}

// Assess evaluates only pinned receipts, requiring a matching success for every
// declared command. Later database records cannot upgrade an existing build.
func (s BookVerificationSnapshot) Assess(now time.Time) BookVerificationAssessment {
	out := BookVerificationAssessment{EvaluatedAt: now, Passed: s.Validate() == nil && !now.Before(s.CapturedAt), Artifacts: make([]BookArtifactVerificationAssessment, 0, len(s.Artifacts))}
	for _, a := range s.Artifacts {
		item := BookArtifactVerificationAssessment{ArtifactID: a.ID, EvidenceIDs: []string{}, Reasons: []string{}}
		var m TeachingArtifactManifest
		err := json.Unmarshal(a.Manifest, &m)
		mh, hashErr := TeachingArtifactManifestHash(m)
		if err != nil || hashErr != nil || mh != a.ManifestHash || m.ArtifactID != a.ID || m.RevisionID != a.RevisionID || m.ArtifactHash != a.ArtifactHash {
			item.Reasons = append(item.Reasons, "冻结教学清单无效或与工件身份不匹配。")
		} else {
			covered := map[VerificationCommand]bool{}
			for _, e := range a.Evidence {
				reason := ""
				if e.Validate() != nil || e.CodeArtifactID != a.ID || e.RevisionID != a.RevisionID || e.CommandManifestHash != a.ManifestHash || e.CapturedAt.After(s.CapturedAt) || e.StaleReason != "" {
					reason = "运行收据身份、清单或采集时间不匹配。"
				} else {
					reason = TeachingArtifactEvidenceStaleReason(m, s.BookContractHash, s.StyleSheetHash, e, now)
				}
				var observation struct {
					Command  VerificationCommand `json:"command"`
					ExitCode *int                `json:"exit_code"`
					Status   string              `json:"status"`
				}
				output, parseErr := ParseRuntimeObservationOutput(e.StructuredOutput)
				if reason == "" && parseErr == nil && e.Kind == RuntimeEvidenceBrowserPage {
					matched := false
					for _, command := range m.Commands {
						if frozenBrowserCommandObserved(output, command) {
							covered[command] = true
							matched = true
						}
					}
					if matched {
						item.EvidenceIDs = append(item.EvidenceIDs, e.ID)
						continue
					}
					reason = "浏览器页面路径或文本断言与冻结命令不匹配。"
				}
				if reason == "" && (parseErr != nil || json.Unmarshal(output.Observations, &observation) != nil || observation.ExitCode == nil || *observation.ExitCode != 0 || observation.Status != "verified") {
					reason = "缺少明确成功的命令运行观测。"
				}
				matched := false
				for _, command := range m.Commands {
					if e.Kind == RuntimeEvidenceTerminalOutput && command.Kind == "go_test" && command == observation.Command {
						matched = true
					}
				}
				if reason == "" && !matched {
					reason = "观测命令与冻结清单不匹配。"
				}
				if reason != "" {
					item.Reasons = append(item.Reasons, e.ID+": "+reason)
					continue
				}
				covered[observation.Command] = true
				item.EvidenceIDs = append(item.EvidenceIDs, e.ID)
			}
			item.Passed = true
			for _, command := range m.Commands {
				if !covered[command] {
					item.Passed = false
					item.Reasons = append(item.Reasons, "缺少当前有效的命令证据："+command.Kind)
				}
			}
		}
		if !item.Passed {
			out.Passed = false
		}
		out.Artifacts = append(out.Artifacts, item)
	}
	return out
}

func frozenBrowserCommandObserved(output RuntimeObservationOutput, command VerificationCommand) bool {
	if command.Kind != "browser_page" {
		return false
	}
	var bundle browserPageObservationBundle
	if json.Unmarshal(output.Observations, &bundle) != nil {
		return false
	}
	for _, page := range bundle.BrowserPages {
		parsed, err := url.Parse(page.URL)
		if err != nil || parsed.RequestURI() != command.BrowserPath {
			continue
		}
		for _, assertion := range page.DOMAssertions {
			if assertion.Assertion == "has_text" && assertion.Expected == command.ExpectedText && assertion.Locator == "text="+command.ExpectedText {
				return true
			}
		}
	}
	return false
}

// ReadBookVerificationSnapshot binds the optional versioned snapshot to the
// existing manifest artifact list. Absence means a legacy build, never success.
func ReadBookVerificationSnapshot(raw json.RawMessage) (*BookVerificationSnapshot, error) {
	var envelope struct {
		ToolVersions map[string]string                `json:"tool_versions"`
		Artifacts    []FrozenBookVerificationArtifact `json:"code_artifacts"`
		Snapshot     json.RawMessage                  `json:"verification_snapshot"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Snapshot) == 0 {
		if envelope.ToolVersions["verification_snapshot"] != "" {
			return nil, fmt.Errorf("declared verification snapshot is missing")
		}
		return nil, nil
	}
	var s BookVerificationSnapshot
	if err := json.Unmarshal(envelope.Snapshot, &s); err != nil {
		return nil, err
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if len(s.Artifacts) != len(envelope.Artifacts) {
		return nil, fmt.Errorf("verification snapshot does not cover frozen artifacts")
	}
	byID := map[string]FrozenBookVerificationArtifact{}
	for _, a := range envelope.Artifacts {
		if _, exists := byID[a.ID]; exists {
			return nil, fmt.Errorf("duplicate frozen artifact")
		}
		byID[a.ID] = a
	}
	for _, a := range s.Artifacts {
		p, ok := byID[a.ID]
		if !ok || p.RevisionID != a.RevisionID || p.ArtifactHash != a.ArtifactHash || p.ManifestHash != a.ManifestHash {
			return nil, fmt.Errorf("verification snapshot artifact binding mismatch")
		}
	}
	return &s, nil
}

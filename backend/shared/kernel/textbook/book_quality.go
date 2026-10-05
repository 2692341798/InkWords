package textbook

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// BookQualitySnapshotFormat identifies immutable chapter detector reports. It
// does not assert whole-book editorial, rights, or reader-trial completion.
const BookQualitySnapshotFormat = "inkwords.book-quality-snapshot.v1"

// FrozenChapterQuality binds the original automatic report to approved bytes
// and writing contracts, keeping advisories and manual-review requirements.
type FrozenChapterQuality struct {
	ChapterID              string          `json:"chapter_id"`
	RevisionID             string          `json:"revision_id"`
	ContentHash            string          `json:"content_hash"`
	BookContractRevisionID string          `json:"book_contract_revision_id"`
	StyleSheetRevisionID   string          `json:"style_sheet_revision_id"`
	ReportHash             string          `json:"report_hash"`
	Report                 json.RawMessage `json:"report"`
}

// BookQualitySnapshot freezes available chapter reports at build creation.
type BookQualitySnapshot struct {
	Format                 string                 `json:"format"`
	CapturedAt             time.Time              `json:"captured_at"`
	BookContractRevisionID string                 `json:"book_contract_revision_id"`
	StyleSheetRevisionID   string                 `json:"style_sheet_revision_id"`
	Chapters               []FrozenChapterQuality `json:"chapters"`
}

// BookQualityAssessment reports automatic chapter gate validity under today's
// contract version, never an average score or an editorial approval.
type BookQualityAssessment struct {
	ReviewerKind     string                     `json:"reviewer_kind"`
	RequiredContract string                     `json:"required_contract"`
	Scope            string                     `json:"scope"`
	Passed           bool                       `json:"passed"`
	Chapters         []ChapterQualityAssessment `json:"chapters"`
}

// ChapterQualityAssessment keeps the rejected report reference and reason.
type ChapterQualityAssessment struct {
	RevisionID string `json:"revision_id"`
	ReportHash string `json:"report_hash"`
	Passed     bool   `json:"passed"`
	Reason     string `json:"reason,omitempty"`
}

// BookQualityReportHash hashes the JSON value with Go's sorted-key encoding,
// so PostgreSQL JSONB whitespace/key ordering cannot invalidate the binding.
func BookQualityReportHash(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		raw = json.RawMessage(`null`)
	}
	var value any
	if !json.Valid(raw) {
		return "", fmt.Errorf("invalid chapter quality report JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	// Preserve numeric precision: a future detector may include counters or
	// identifiers larger than float64 can represent exactly.
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Validate checks structural integrity while retaining missing/failed reports
// for review instead of refusing to create an inspectable draft build.
func (s BookQualitySnapshot) Validate() error {
	if s.Format != BookQualitySnapshotFormat || s.CapturedAt.IsZero() || s.BookContractRevisionID == "" || s.StyleSheetRevisionID == "" || len(s.Chapters) == 0 {
		return fmt.Errorf("invalid book quality snapshot")
	}
	chapters, revisions := map[string]bool{}, map[string]bool{}
	for _, c := range s.Chapters {
		if c.ChapterID == "" || c.RevisionID == "" || chapters[c.ChapterID] || revisions[c.RevisionID] || !isFullSHA256Digest(c.ContentHash) {
			return fmt.Errorf("invalid frozen chapter quality identity")
		}
		chapters[c.ChapterID], revisions[c.RevisionID] = true, true
		hash, err := BookQualityReportHash(c.Report)
		if err != nil || hash != c.ReportHash {
			return fmt.Errorf("frozen chapter quality report hash mismatch")
		}
	}
	return nil
}

// Assess keeps chapter detector success separate from review completion.
func (s BookQualitySnapshot) Assess() BookQualityAssessment {
	out := BookQualityAssessment{ReviewerKind: "automated", RequiredContract: SampleQualityContractVersion, Scope: "仅已批准章节的自动生成质量门禁；不代表整书审阅、运行验证、权利或读者掌握。", Passed: s.Validate() == nil, Chapters: []ChapterQualityAssessment{}}
	for _, c := range s.Chapters {
		item := ChapterQualityAssessment{RevisionID: c.RevisionID, ReportHash: c.ReportHash, Passed: true}
		var report struct {
			Failures []json.RawMessage `json:"failures"`
		}
		switch {
		case c.BookContractRevisionID != s.BookContractRevisionID || c.StyleSheetRevisionID != s.StyleSheetRevisionID:
			item.Reason = "章节质量报告所属的写作合同与冻结构建不匹配。"
		case json.Unmarshal(c.Report, &report) != nil || ValidateCurrentSampleQualityReport(c.Report) != nil:
			item.Reason = "章节自动质量报告缺失、未通过或合同版本已过期。"
		case len(report.Failures) > 0:
			item.Reason = "章节自动质量报告仍包含失败项。"
		}
		item.Passed = item.Reason == ""
		if !item.Passed {
			out.Passed = false
		}
		out.Chapters = append(out.Chapters, item)
	}
	return out
}

// ReadBookQualitySnapshot rejects a partial/mismatched snapshot. Legacy builds
// without a declared snapshot retain explicit unavailable semantics.
func ReadBookQualitySnapshot(raw json.RawMessage) (*BookQualitySnapshot, error) {
	var envelope struct {
		Snapshot               json.RawMessage        `json:"quality_snapshot"`
		Chapters               []FrozenChapterQuality `json:"chapters"`
		BookContractRevisionID string                 `json:"book_contract_revision_id"`
		StyleSheetRevisionID   string                 `json:"style_sheet_revision_id"`
		ToolVersions           map[string]string      `json:"tool_versions"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Snapshot) == 0 {
		if envelope.ToolVersions["quality_snapshot"] != "" {
			return nil, fmt.Errorf("declared quality snapshot is missing")
		}
		return nil, nil
	}
	var s BookQualitySnapshot
	if err := json.Unmarshal(envelope.Snapshot, &s); err != nil {
		return nil, err
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if declared := envelope.ToolVersions["quality_snapshot"]; declared != "" && declared != s.Format {
		return nil, fmt.Errorf("quality snapshot version declaration mismatch")
	}
	if len(s.Chapters) != len(envelope.Chapters) || s.BookContractRevisionID != envelope.BookContractRevisionID || s.StyleSheetRevisionID != envelope.StyleSheetRevisionID {
		return nil, fmt.Errorf("quality snapshot chapter or contract binding mismatch")
	}
	byID := map[string]FrozenChapterQuality{}
	for _, c := range envelope.Chapters {
		if _, ok := byID[c.ChapterID]; ok {
			return nil, fmt.Errorf("duplicate frozen chapter")
		}
		byID[c.ChapterID] = c
	}
	for _, c := range s.Chapters {
		p, ok := byID[c.ChapterID]
		if !ok || p.RevisionID != c.RevisionID || "sha256:"+strings.TrimPrefix(p.ContentHash, "sha256:") != c.ContentHash {
			return nil, fmt.Errorf("quality snapshot does not match frozen chapter bytes")
		}
	}
	return &s, nil
}

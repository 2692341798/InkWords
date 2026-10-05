package textbook

import (
	"fmt"
	"sort"
	"strings"
)

const (
	SampleHumanReviewContractVersion = "inkwords.sample-human-review.v1"
	SampleHumanReviewMinimumScore    = 3
	SampleHumanReviewMaximumScore    = 4
	SampleHumanReviewNoteMinRunes    = 8
	SampleHumanReviewNoteMaxRunes    = 2000
)

var sampleManualReviewDimensions = []string{
	"句子与段落负担",
	"标题承诺",
	"重复",
	"语气",
	"例子相关性",
	"章节节奏",
	"图示机会",
	"练习梯度",
}

// SampleManualReviewDimensions returns the complete human rubric for a sample candidate.
func SampleManualReviewDimensions() []string {
	return append([]string(nil), sampleManualReviewDimensions...)
}

// SampleHumanReviewContract describes the exact author input required to
// approve a candidate under the current server rules.
type SampleHumanReviewContract struct {
	ReviewerKinds   []string `json:"reviewer_kinds,omitempty"`
	ContractVersion string   `json:"contract_version"`
	Dimensions      []string `json:"dimensions"`
	MinimumScore    int      `json:"minimum_score"`
	MaximumScore    int      `json:"maximum_score"`
	NoteMinRunes    int      `json:"note_min_runes"`
	NoteMaxRunes    int      `json:"note_max_runes"`
}

// CurrentSampleHumanReviewContract returns a detached value safe for transport.
func CurrentSampleHumanReviewContract() SampleHumanReviewContract {
	return SampleHumanReviewContract{
		ReviewerKinds:   []string{"human", "delegated_ai"},
		ContractVersion: SampleHumanReviewContractVersion,
		Dimensions:      SampleManualReviewDimensions(),
		MinimumScore:    SampleHumanReviewMinimumScore,
		MaximumScore:    SampleHumanReviewMaximumScore,
		NoteMinRunes:    SampleHumanReviewNoteMinRunes,
		NoteMaxRunes:    SampleHumanReviewNoteMaxRunes,
	}
}

// SampleHumanReview retains the legacy storage name while distinguishing human
// review from a separately versioned, explicitly delegated AI decision.
type SampleHumanReview struct {
	ReviewerKind    string           `json:"reviewer_kind,omitempty"`
	DelegationNote  string           `json:"delegation_note,omitempty"`
	ContractVersion string           `json:"contract_version"`
	DimensionScores []DimensionScore `json:"dimension_scores"`
}

func (review SampleHumanReview) Validate() error {
	if err := review.validateReviewer(); err != nil {
		return err
	}
	if len(review.DimensionScores) != len(sampleManualReviewDimensions) {
		return fmt.Errorf("sample human review contract is incomplete")
	}
	want := SampleManualReviewDimensions()
	got := make([]string, 0, len(review.DimensionScores))
	seen := make(map[string]bool, len(review.DimensionScores))
	for _, score := range review.DimensionScores {
		if err := score.Validate(); err != nil || score.Score < SampleHumanReviewMinimumScore || seen[score.Dimension] {
			return fmt.Errorf("sample human review requires every dimension at least 3/4")
		}
		seen[score.Dimension] = true
		got = append(got, score.Dimension)
	}
	sort.Strings(want)
	sort.Strings(got)
	if strings.Join(want, "\n") != strings.Join(got, "\n") {
		return fmt.Errorf("sample human review dimensions do not match the contract")
	}
	return nil
}

type DimensionScore struct {
	Dimension string `json:"dimension"`
	Score     int    `json:"score"`
}

func (score DimensionScore) Validate() error {
	if strings.TrimSpace(score.Dimension) == "" || score.Score < 0 || score.Score > SampleHumanReviewMaximumScore {
		return fmt.Errorf("quality dimension score is invalid")
	}
	return nil
}

type QualityFinding struct {
	Code       string        `json:"code"`
	Severity   QualityStatus `json:"severity"`
	Message    string        `json:"message"`
	EvidenceID []string      `json:"evidence_ids,omitempty"`
}

func (finding QualityFinding) Validate() error {
	if strings.TrimSpace(finding.Code) == "" || strings.TrimSpace(finding.Message) == "" {
		return fmt.Errorf("quality finding is incomplete")
	}
	return finding.Severity.Validate()
}

// QualityAssessment records automated or human findings without claiming that automation is human review.
type QualityAssessment struct {
	ID              string           `json:"id"`
	ScopeType       QualityScope     `json:"scope_type"`
	ScopeID         string           `json:"scope_id"`
	ContractVersion string           `json:"contract_version"`
	Detector        string           `json:"detector"`
	Status          QualityStatus    `json:"status"`
	DimensionScores []DimensionScore `json:"dimension_scores"`
	Findings        []QualityFinding `json:"findings,omitempty"`
}

func (assessment QualityAssessment) Validate() error {
	if strings.TrimSpace(assessment.ID) == "" || strings.TrimSpace(assessment.ScopeID) == "" || strings.TrimSpace(assessment.ContractVersion) == "" || strings.TrimSpace(assessment.Detector) == "" {
		return fmt.Errorf("quality assessment identity is incomplete")
	}
	if err := assessment.ScopeType.Validate(); err != nil {
		return err
	}
	if err := assessment.Status.Validate(); err != nil {
		return err
	}
	if len(assessment.DimensionScores) == 0 {
		return fmt.Errorf("quality assessment requires dimension scores")
	}
	for _, score := range assessment.DimensionScores {
		if err := score.Validate(); err != nil {
			return err
		}
	}
	for _, finding := range assessment.Findings {
		if err := finding.Validate(); err != nil {
			return err
		}
	}
	return nil
}

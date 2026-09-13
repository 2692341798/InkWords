package mastery

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAssessmentCanCorrectUnsupportedFindingsWithoutChangingScores(t *testing.T) {
	input, original := assessmentFixture(t, Explain)
	original.MissingPoints = []AssessmentFinding{{Text: "遗漏了题目未要求的细节。", EvidenceIDs: []string{"source-1"}}}
	before, err := json.Marshal(original)
	require.NoError(t, err)
	data, err := json.Marshal(map[string]any{
		"id": "findings-correction", "reviewer_id": "workspace-1", "corrected_at": time.Unix(1, 0),
		"reason":     "对照冻结要求移除无依据的遗漏，并改为相关提示。",
		"input_hash": AssessmentInputHash(input), "previous_hash": AssessmentFeedbackHash(original),
		"changes": []CriterionAssessment{},
		"findings": map[string]any{
			"correct_points": []AssessmentFinding{}, "missing_points": []AssessmentFinding{}, "misconceptions": []AssessmentFinding{},
			"next_hint":   AssessmentFinding{Text: "方法改变后会选择哪棵树？", EvidenceIDs: []string{"source-1"}},
			"remediation": original.Remediation,
		},
	})
	require.NoError(t, err)
	var correction AssessmentCorrection
	require.NoError(t, json.Unmarshal(data, &correction))
	effective, err := ReplayAssessmentCorrections(input, original, []AssessmentCorrection{correction})
	require.NoError(t, err)
	require.Empty(t, effective.MissingPoints)
	require.NotNil(t, effective.MissingPoints)
	require.Equal(t, original.Criteria, effective.Criteria)
	require.Equal(t, "方法改变后会选择哪棵树？", effective.NextHint.Text)
	require.NotEqual(t, AssessmentFeedbackHash(original), AssessmentFeedbackHash(effective))
	after, err := json.Marshal(original)
	require.NoError(t, err)
	require.Equal(t, before, after)
	correction.Findings.NextHint.EvidenceIDs[0] = "mutated"
	require.Equal(t, []string{"source-1"}, effective.NextHint.EvidenceIDs)
}

func TestAssessmentFindingCorrectionsRejectIncompleteOrInventedAdvice(t *testing.T) {
	for _, name := range []string{"missing section", "unknown source", "duplicate source", "empty hint", "empty remediation", "too many findings", "stale", "other input"} {
		t.Run(name, func(t *testing.T) {
			input, original := assessmentFixture(t, Explain)
			correction := AssessmentCorrection{ID: "correction", ReviewerID: "workspace", CorrectedAt: time.Unix(1, 0), Reason: "核对原题与来源后修改反馈", InputHash: AssessmentInputHash(input), PreviousHash: AssessmentFeedbackHash(original), Findings: &AssessmentFindingCorrection{CorrectPoints: []AssessmentFinding{}, MissingPoints: []AssessmentFinding{}, Misconceptions: []AssessmentFinding{}, NextHint: original.NextHint, Remediation: original.Remediation}}
			switch name {
			case "missing section":
				correction.Findings.MissingPoints = nil
			case "unknown source":
				correction.Findings.NextHint.EvidenceIDs = []string{"unknown"}
			case "duplicate source":
				correction.Findings.NextHint.EvidenceIDs = []string{"source-1", "source-1"}
			case "empty hint":
				correction.Findings.NextHint.Text = ""
			case "empty remediation":
				correction.Findings.Remediation = []AssessmentFinding{}
			case "too many findings":
				correction.Findings.CorrectPoints = make([]AssessmentFinding, 21)
			case "stale":
				correction.PreviousHash = assessmentDigest("stale")
			case "other input":
				correction.InputHash = assessmentDigest("other input")
			}
			_, err := ReplayAssessmentCorrections(input, original, []AssessmentCorrection{correction})
			require.Error(t, err)
		})
	}
}

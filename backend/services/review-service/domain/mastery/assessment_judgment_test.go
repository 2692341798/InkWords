package mastery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func judgmentFixture(t *testing.T) (AssessmentInput, []AssessmentJudgment, AssessmentFeedback) {
	t.Helper()
	input, feedback, _ := decisionFixture(t)
	input.DecisionPolicy = AssessmentJudgmentPolicy
	judgments := make([]AssessmentJudgment, len(input.Rubric))
	for index, item := range feedback.Criteria {
		judgments[index] = AssessmentJudgment{CriterionID: item.ID, Score: item.Score, Status: "satisfied", AnswerQuote: item.AnswerQuote, EvidenceIDs: append([]string(nil), item.EvidenceIDs...), Gaps: []AssessmentJudgmentGap{}}
	}
	return input, judgments, feedback
}

func TestCanonicalJudgmentProjectsReasonsAndFindingsFromTheSameGaps(t *testing.T) {
	input, judgments, advice := judgmentFixture(t)
	two := 2
	judgments[2].Score, judgments[2].Status = &two, "unsatisfied"
	judgments[2].Gaps = []AssessmentJudgmentGap{{Kind: "omission", RequirementQuote: input.Rubric[2].Description, Reason: "只给出步骤，没有说明本项要求的因果关系。"}}
	feedback, decisions, err := ProjectAssessmentJudgments(input, judgments, advice.NextHint, advice.Remediation)
	require.NoError(t, err)
	require.Equal(t, "本项尚未满足："+judgments[2].Gaps[0].Reason, feedback.Criteria[2].Reason)
	require.Equal(t, judgments[2].Gaps[0].Reason, feedback.MissingPoints[0].Text)
	require.Equal(t, judgments[2].Gaps[0].Reason, decisions[2].Gaps[0].Reason)
	require.Len(t, feedback.CorrectPoints, 4)
	require.Empty(t, feedback.Misconceptions)
	require.Equal(t, "所引作答满足本项要求。", feedback.Criteria[1].Reason)
	require.Equal(t, input.Rubric[1].Description, decisions[1].Requirement)
	judgments[2].Gaps[0].Kind = "misconception"
	feedback, _, err = ProjectAssessmentJudgments(input, judgments, advice.NextHint, advice.Remediation)
	require.NoError(t, err)
	require.Empty(t, feedback.MissingPoints)
	require.Len(t, feedback.Misconceptions, 1)
}

func TestCanonicalProjectionDoesNotAliasOriginalModelJudgmentsOrAdvice(t *testing.T) {
	input, judgments, advice := judgmentFixture(t)
	feedback, _, err := ProjectAssessmentJudgments(input, judgments, advice.NextHint, advice.Remediation)
	require.NoError(t, err)
	*feedback.Criteria[0].Score = 0
	feedback.NextHint.EvidenceIDs[0] = "changed"
	feedback.Remediation[0].EvidenceIDs[0] = "changed"
	require.Equal(t, 3, *judgments[0].Score)
	require.Equal(t, "source-1", advice.NextHint.EvidenceIDs[0])
	require.Equal(t, "source-1", advice.Remediation[0].EvidenceIDs[0])
}

func TestCanonicalJudgmentRejectsUnboundAndContradictoryFindings(t *testing.T) {
	for name, mutate := range map[string]func([]AssessmentJudgment){
		"unknown ID":             func(items []AssessmentJudgment) { items[0].CriterionID = "invented" },
		"duplicate":              func(items []AssessmentJudgment) { items[1].CriterionID = items[0].CriterionID },
		"nil gaps":               func(items []AssessmentJudgment) { items[0].Gaps = nil },
		"invented quote":         func(items []AssessmentJudgment) { items[0].AnswerQuote = "不是学习者说的话" },
		"unknown reason on met":  func(items []AssessmentJudgment) { items[0].UnknownReason = "既满足又未知" },
		"unknown without reason": func(items []AssessmentJudgment) { items[0].Status, items[0].Score = "unknown", nil },
		"unbound gap": func(items []AssessmentJudgment) {
			two := 2
			items[0].Score, items[0].Status = &two, "unsatisfied"
			items[0].Gaps = []AssessmentJudgmentGap{{Kind: "omission", RequirementQuote: "新增题外要求", Reason: "缺少"}}
		},
		"new gap kind": func(items []AssessmentJudgment) {
			items[0].Gaps = []AssessmentJudgmentGap{{Kind: "invented", RequirementQuote: "要求", Reason: "缺少"}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			input, judgments, advice := judgmentFixture(t)
			mutate(judgments)
			_, _, err := ProjectAssessmentJudgments(input, judgments, advice.NextHint, advice.Remediation)
			require.Error(t, err)
		})
	}
}

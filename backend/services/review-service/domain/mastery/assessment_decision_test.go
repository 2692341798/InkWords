package mastery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func decisionFixture(t *testing.T) (AssessmentInput, AssessmentFeedback, []AssessmentDecision) {
	t.Helper()
	input, feedback := assessmentFixture(t, Explain)
	input.TaskReference = &AssessmentTaskReference{TaskID: "explain", PracticeContentHash: assessmentDigest("practice"), ExpectedAnswer: "方法和路径共同决定查找。"}
	input.DecisionPolicy = AssessmentDecisionPolicy
	decisions := make([]AssessmentDecision, len(input.Rubric))
	for index, criterion := range input.Rubric {
		decisions[index] = AssessmentDecision{CriterionID: criterion.ID, Requirement: criterion.Description, Status: "satisfied", Gaps: []AssessmentDecisionGap{}}
	}
	return input, feedback, decisions
}

func TestAssessmentDecisionRejectsPassingScoreForUnmetFrozenRequirement(t *testing.T) {
	input, feedback, decisions := decisionFixture(t)
	require.NoError(t, ValidateAssessmentDecisions(input, feedback, decisions))
	decisions[2].Status = "unsatisfied"
	decisions[2].Gaps = []AssessmentDecisionGap{{RequirementQuote: input.Rubric[2].Description, Reason: "只有步骤，没有解释本项要求的机制依赖。"}}
	require.ErrorContains(t, ValidateAssessmentDecisions(input, feedback, decisions), "score")
	two := 2
	feedback.Criteria[2].Score = &two
	require.NoError(t, ValidateAssessmentDecisions(input, feedback, decisions))
	decisions[2].Gaps[0].RequirementQuote = "题目没有要求的中间件内部实现"
	require.ErrorContains(t, ValidateAssessmentDecisions(input, feedback, decisions), "requirement")
}

func TestAssessmentDecisionRequiresCompleteUnambiguousCoverage(t *testing.T) {
	for name, mutate := range map[string]func([]AssessmentDecision) []AssessmentDecision{
		"missing criterion":   func(items []AssessmentDecision) []AssessmentDecision { return items[:4] },
		"duplicate criterion": func(items []AssessmentDecision) []AssessmentDecision { items[1] = items[0]; return items },
		"new criterion":       func(items []AssessmentDecision) []AssessmentDecision { items[0].CriterionID = "invented"; return items },
		"changed requirement": func(items []AssessmentDecision) []AssessmentDecision {
			items[0].Requirement += "额外要求"
			return items
		},
		"missing gap list": func(items []AssessmentDecision) []AssessmentDecision { items[0].Gaps = nil; return items },
		"met with gap": func(items []AssessmentDecision) []AssessmentDecision {
			items[0].Gaps = []AssessmentDecisionGap{{RequirementQuote: items[0].Requirement, Reason: "缺少此要求"}}
			return items
		},
		"unknown with score": func(items []AssessmentDecision) []AssessmentDecision { items[0].Status = "unknown"; return items },
		"new status":         func(items []AssessmentDecision) []AssessmentDecision { items[0].Status = "mostly"; return items },
		"unmet without gap":  func(items []AssessmentDecision) []AssessmentDecision { items[0].Status = "unsatisfied"; return items },
	} {
		t.Run(name, func(t *testing.T) {
			input, feedback, decisions := decisionFixture(t)
			require.Error(t, ValidateAssessmentDecisions(input, feedback, mutate(decisions)))
		})
	}
}

func TestAssessmentDecisionUnknownRemainsUnknownAndLegacyCannotAcquirePolicy(t *testing.T) {
	input, feedback, decisions := decisionFixture(t)
	decisions[0].Status, feedback.Criteria[0].Score = "unknown", nil
	require.NoError(t, ValidateAssessmentDecisions(input, feedback, decisions))
	require.Equal(t, AssessmentDecisionContractVersion, input.ContractVersion())
	newHash := AssessmentInputHash(input)
	input.DecisionPolicy = ""
	require.NotEqual(t, newHash, AssessmentInputHash(input))
	require.NoError(t, ValidateAssessmentDecisions(input, feedback, nil))
	require.Error(t, ValidateAssessmentDecisions(input, feedback, decisions))
}

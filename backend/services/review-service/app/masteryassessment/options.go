package masteryassessment

import (
	"inkwords-backend/services/review-service/domain/mastery"
	"inkwords-backend/shared/kernel/generation"
)

// Options freezes task-scoped settings into each preview and request hash.
// Reasoning is opt-in for canonical judgments only; historical contracts keep
// their original request. Output changes are explicit and capped at 6000;
// the adapter and application deadlines remain unchanged.
type Options struct {
	GroundedIdentities bool
	ReasoningEffort    string
	MaxOutputTokens    int
}

// TaskModelPolicy keeps the evaluated text and code settings together. A code
// submission stays on the code policy when a matching runtime report is added.
type TaskModelPolicy struct {
	Text Options
	Code Options
}

// NewGeneratorWithTaskPolicy selects options from frozen task data. It does not
// select another model or change legacy request contracts.
func NewGeneratorWithTaskPolicy(port generation.Port, provider, model string, policy TaskModelPolicy) *Generator {
	g := NewGenerator(port, provider, model)
	g.taskPolicy = &policy
	return g
}

// NewGeneratorWithOptions configures canonical grading without changing the
// legacy constructor or upgrading the selected provider/model.
func NewGeneratorWithOptions(port generation.Port, provider, model string, options Options) *Generator {
	g := NewGenerator(port, provider, model)
	g.options = options
	return g
}

func (g *Generator) request(input mastery.AssessmentInput) (generation.Request, error) {
	request, err := requestForAssessment(g.model, input)
	if err != nil {
		return request, err
	}
	if input.DecisionPolicy == mastery.AssessmentJudgmentPolicy {
		options := g.options
		if g.taskPolicy != nil {
			options = g.taskPolicy.Text
			if input.LearnerArtifact != nil {
				options = g.taskPolicy.Code
			}
		}
		if options.MaxOutputTokens < 0 || options.MaxOutputTokens > 6000 {
			return request, ErrBudgetExceeded
		}
		if options.MaxOutputTokens > 0 {
			request.MaxOutputTokens = options.MaxOutputTokens
		}
		request.ReasoningEffort = options.ReasoningEffort
		if options.GroundedIdentities {
			if err := groundJudgmentRequest(&request, input); err != nil {
				return request, err
			}
		}
	}
	return request, request.Validate()
}

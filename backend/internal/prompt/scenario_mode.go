package prompt

import sharedprompt "inkwords-backend/shared/kernel/prompt"

type ScenarioMode = sharedprompt.ScenarioMode

const (
	ScenarioModeEbookInterpretation = sharedprompt.ScenarioModeEbookInterpretation
	ScenarioModeOpenBookExamReview  = sharedprompt.ScenarioModeOpenBookExamReview
	ScenarioModeBeginnerWalkthrough = sharedprompt.ScenarioModeBeginnerWalkthrough
)

func DefaultScenarioModeForSource(sourceType string) ScenarioMode {
	return sharedprompt.DefaultScenarioModeForSource(sourceType)
}

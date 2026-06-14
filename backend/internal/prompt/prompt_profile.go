package prompt

import sharedprompt "inkwords-backend/shared/kernel/prompt"

type PromptProfileKey = sharedprompt.PromptProfileKey

const (
	PromptProfileClassicTextInterpretation = sharedprompt.PromptProfileClassicTextInterpretation
	PromptProfilePsychologyCommunication   = sharedprompt.PromptProfilePsychologyCommunication
	PromptProfileHistoryThought            = sharedprompt.PromptProfileHistoryThought
	PromptProfileLiteratureCommentary      = sharedprompt.PromptProfileLiteratureCommentary
	PromptProfileTechnicalManual           = sharedprompt.PromptProfileTechnicalManual
	PromptProfileExamMaterialReview        = sharedprompt.PromptProfileExamMaterialReview
)

type PromptProfile = sharedprompt.PromptProfile
type ResolvedPromptProfile = sharedprompt.ResolvedPromptProfile

func FallbackPromptProfileForScenario(mode ScenarioMode) PromptProfile {
	return sharedprompt.FallbackPromptProfileForScenario(mode)
}

func ResolvePromptProfileKey(key string, mode ScenarioMode) PromptProfile {
	return sharedprompt.ResolvePromptProfileKey(key, mode)
}

package prompt

import sharedprompt "inkwords-backend/shared/kernel/prompt"

func DefaultRequirements(style ArticleStyle) string {
	return sharedprompt.DefaultRequirements(style)
}

func DefaultStyleRequirements(mode ScenarioMode, style ArticleStyle) string {
	return sharedprompt.DefaultStyleRequirements(mode, style)
}

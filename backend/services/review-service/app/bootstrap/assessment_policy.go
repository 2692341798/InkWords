package bootstrap

import (
	"errors"
	"strings"
	"time"

	app "inkwords-backend/services/review-service/app/masteryassessment"
)

func resolveAssessmentPolicy(profile, model string) (*app.TaskModelPolicy, time.Duration, error) {
	switch strings.TrimSpace(profile) {
	case "":
		return nil, 45 * time.Second, nil
	case "local-evaluation-v1", "local-evaluation-v2":
		if selected := strings.TrimSpace(model); selected != "" && selected != "deepseek-v4-flash" {
			return nil, 0, errors.New("本地评分配置仅支持已评测的 deepseek-v4-flash")
		}
		grounded := strings.TrimSpace(profile) == "local-evaluation-v2"
		return &app.TaskModelPolicy{Text: app.Options{ReasoningEffort: "low", MaxOutputTokens: 6000, GroundedIdentities: grounded}, Code: app.Options{MaxOutputTokens: 6000, GroundedIdentities: grounded}}, 60 * time.Second, nil
	default:
		return nil, 0, errors.New("未知的本地评分配置，请检查 MASTERY_ASSESSMENT_PROFILE")
	}
}

package bootstrap

import (
	"context"
	"os"
	"strings"

	"gorm.io/gorm"
	app "inkwords-backend/services/review-service/app/masteryassessment"
	"inkwords-backend/services/review-service/domain/mastery"
	persistence "inkwords-backend/services/review-service/infra/assessment"
	"inkwords-backend/shared/kernel/generation"
	platformllm "inkwords-backend/shared/platform/llm"
)

// BuildAssessmentService configures explicit model work and marks abandoned
// requests interrupted. It never retries a provider call during startup.
func BuildAssessmentService(db *gorm.DB, source *mastery.Service, verification ...app.VerificationSource) (*app.Service, error) {
	var port generation.Port
	key := strings.TrimSpace(os.Getenv("MASTERY_DEEPSEEK_API_KEY"))
	model := firstNonEmpty(os.Getenv("MASTERY_DEEPSEEK_MODEL"), os.Getenv("DEEPSEEK_MODEL"))
	policy, timeout, err := resolveAssessmentPolicy(os.Getenv("MASTERY_ASSESSMENT_PROFILE"), model)
	if err != nil {
		return nil, err
	}
	if key != "" && model != "" {
		port = platformllm.NewDeepSeekGenerationAdapterWithTimeout(platformllm.NewDeepSeekClient(key), timeout)
	}
	var inputSource app.InputSource = source
	if len(verification) > 0 && verification[0] != nil {
		inputSource = app.NewVerifiedInputSource(source, verification[0])
	}
	generator := app.NewGenerator(port, "deepseek", model)
	if policy != nil {
		generator = app.NewGeneratorWithTaskPolicy(port, "deepseek", model, *policy)
	}
	service := app.NewService(inputSource, generator, persistence.NewStore(db))
	if err := service.Recover(context.Background()); err != nil {
		return nil, err
	}
	return service, nil
}

package masteryassessment

import (
	"encoding/json"

	"inkwords-backend/services/review-service/domain/mastery"
)

// Preview describes the exact bounded request without contacting the provider.
type Preview struct {
	InputHash         string `json:"input_hash"`
	RequestHash       string `json:"request_hash"`
	Provider          string `json:"provider"`
	Model             string `json:"model"`
	RequestBytes      int    `json:"request_bytes"`
	InputByteLimit    int    `json:"input_byte_limit"`
	MaxOutputTokens   int    `json:"max_output_tokens"`
	VerificationRunID string `json:"verification_run_id,omitempty"`
	RuntimeStatus     string `json:"runtime_status,omitempty"`
	ReasoningEffort   string `json:"reasoning_effort,omitempty"`
}

// Preview validates the same contract and budget used by Assess.
func (generator *Generator) Preview(input mastery.AssessmentInput) (Preview, error) {
	if generator == nil || generator.port == nil || generator.provider == "" || generator.model == "" {
		return Preview{}, ErrUnavailable
	}
	if err := input.Validate(); err != nil {
		return Preview{}, err
	}
	request, err := generator.request(input)
	if err != nil {
		return Preview{}, err
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return Preview{}, ErrInvalidFeedback
	}
	if len(encoded) > 32000 {
		return Preview{}, ErrBudgetExceeded
	}
	preview := Preview{InputHash: mastery.AssessmentInputHash(input), RequestHash: digest(generator.provider + "\n" + string(encoded)), Provider: generator.provider, Model: generator.model, RequestBytes: len(encoded), InputByteLimit: 32000, MaxOutputTokens: request.MaxOutputTokens}
	preview.ReasoningEffort = request.ReasoningEffort
	for _, item := range input.Evidence {
		if item.Kind == "runtime" {
			preview.VerificationRunID = item.VerificationRunID
			var runtime struct {
				Status string `json:"status"`
			}
			if json.Unmarshal([]byte(item.Excerpt), &runtime) == nil {
				preview.RuntimeStatus = runtime.Status
			}
			break
		}
	}
	return preview, nil
}

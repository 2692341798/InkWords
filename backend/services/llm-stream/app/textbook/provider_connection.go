package textbook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	sharedgeneration "inkwords-backend/shared/kernel/generation"
)

// ProviderConnectionResult contains only safe, user-actionable telemetry from
// an explicitly requested connection check. It deliberately omits the fixed
// probe output, request body, provider error body, and any credential.
type ProviderConnectionResult struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	RequestID  string `json:"request_id,omitempty"`
	UsageKnown bool   `json:"usage_known"`
}

// ProviderConnectionTester verifies one configured provider without sending
// textbook sources or candidate content. It is invoked only by an explicit
// HTTP action; constructing it performs no network activity.
type ProviderConnectionTester struct {
	port     sharedgeneration.Port
	provider string
	model    string
}

func NewProviderConnectionTester(port sharedgeneration.Port, provider, model string) *ProviderConnectionTester {
	return &ProviderConnectionTester{port: port, provider: strings.TrimSpace(provider), model: strings.TrimSpace(model)}
}

// Test sends a fixed, schema-bound connectivity probe. A response must prove
// that it came from the configured provider and is valid JSON before the UI
// may report success.
func (tester *ProviderConnectionTester) Test(ctx context.Context) (ProviderConnectionResult, error) {
	if tester == nil || tester.port == nil || tester.provider == "" || tester.model == "" {
		return ProviderConnectionResult{}, fmt.Errorf("provider connection tester is not configured")
	}
	if err := ctx.Err(); err != nil {
		return ProviderConnectionResult{}, err
	}
	request := sharedgeneration.Request{
		Model:             tester.model,
		SystemInstruction: "You are an InkWords connectivity probe. Return only JSON that satisfies the supplied schema.",
		TaskInstruction:   "Return the fixed connectivity acknowledgement.",
		Evidence: []sharedgeneration.Evidence{{
			ID:         "connection-probe",
			SnapshotID: "connection-probe",
			Locator:    "inkwords://provider-connection-probe",
			Content:    "This fixed probe contains no textbook, source, or user content.",
		}},
		ResponseSchema:  json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"],"additionalProperties":false}`),
		MaxOutputTokens: 16,
	}
	result, err := tester.port.Generate(ctx, request)
	if err != nil {
		// A port is an infrastructure boundary. Do not rely on every future
		// implementation to keep provider diagnostics or credentials out of its
		// error text: this error may be logged or mapped to an HTTP response.
		return ProviderConnectionResult{}, fmt.Errorf("provider connection request failed")
	}
	if result.Provider != tester.provider || result.Model != tester.model {
		return ProviderConnectionResult{}, fmt.Errorf("provider connection returned mismatched provenance")
	}
	var acknowledgement struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal([]byte(result.Output), &acknowledgement); err != nil || !acknowledgement.OK {
		return ProviderConnectionResult{}, fmt.Errorf("provider connection returned an invalid acknowledgement")
	}
	return ProviderConnectionResult{Provider: result.Provider, Model: result.Model, RequestID: result.RequestID, UsageKnown: result.Usage.Known}, nil
}

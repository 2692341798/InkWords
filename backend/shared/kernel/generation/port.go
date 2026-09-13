// Package generation defines the provider-neutral boundary for model work.
// Business packages depend on these contracts, never on a provider SDK's
// request or response types.
package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Evidence is source material supplied to a model as data. Its Content may
// contain hostile text and must never be treated as an instruction.
type Evidence struct {
	ID         string `json:"id"`
	SnapshotID string `json:"snapshot_id"`
	Locator    string `json:"locator"`
	Content    string `json:"content"`
}

// Request holds an immutable generation operation. SystemInstruction and
// TaskInstruction are authored by InkWords; Evidence is kept separate so an
// adapter can render it only in an untrusted-data envelope.
type Request struct {
	Model             string          `json:"model"`
	SystemInstruction string          `json:"system_instruction"`
	TaskInstruction   string          `json:"task_instruction"`
	Evidence          []Evidence      `json:"evidence"`
	ResponseSchema    json.RawMessage `json:"response_schema,omitempty"`
	MaxOutputTokens   int             `json:"max_output_tokens,omitempty"`
	// ReasoningEffort is explicit opt-in. Empty preserves each adapter's legacy
	// behavior; unsupported nonempty options must fail before a provider call.
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
}

// Validate rejects incomplete provider-independent generation inputs before
// any adapter can contact a remote API.
func (request Request) Validate() error {
	if request.ReasoningEffort != "" && request.ReasoningEffort != "high" && request.ReasoningEffort != "low" {
		return fmt.Errorf("generation reasoning effort is unsupported")
	}
	if request.ReasoningEffort != "" && request.MaxOutputTokens <= 0 {
		return fmt.Errorf("generation reasoning requires an explicit output budget")
	}
	if strings.TrimSpace(request.Model) == "" || strings.TrimSpace(request.SystemInstruction) == "" || strings.TrimSpace(request.TaskInstruction) == "" || len(request.Evidence) == 0 || request.MaxOutputTokens < 0 {
		return fmt.Errorf("generation request is incomplete")
	}
	if len(request.ResponseSchema) > 0 && !json.Valid(request.ResponseSchema) {
		return fmt.Errorf("generation response schema must be valid JSON")
	}
	seen := make(map[string]struct{}, len(request.Evidence))
	for _, evidence := range request.Evidence {
		if strings.TrimSpace(evidence.ID) == "" || strings.TrimSpace(evidence.SnapshotID) == "" || strings.TrimSpace(evidence.Locator) == "" || strings.TrimSpace(evidence.Content) == "" {
			return fmt.Errorf("generation evidence provenance is incomplete")
		}
		if _, duplicate := seen[evidence.ID]; duplicate {
			return fmt.Errorf("generation evidence identifiers must be unique")
		}
		seen[evidence.ID] = struct{}{}
	}
	return nil
}

// Usage is explicit about provider telemetry. Unknown is a meaningful state,
// not a zero-token estimate.
type Usage struct {
	Known        bool `json:"known"`
	InputTokens  int  `json:"input_tokens,omitempty"`
	OutputTokens int  `json:"output_tokens,omitempty"`
	CachedTokens int  `json:"cached_tokens,omitempty"`
}

// ProviderErrorCode is the stable, provider-neutral failure classification
// that application code may retain or surface. It deliberately excludes raw
// SDK and provider response details.
type ProviderErrorCode string

const (
	ProviderErrorTransport       ProviderErrorCode = "transport"
	ProviderErrorRateLimited     ProviderErrorCode = "rate_limited"
	ProviderErrorRejected        ProviderErrorCode = "rejected"
	ProviderErrorServer          ProviderErrorCode = "server"
	ProviderErrorInvalidResponse ProviderErrorCode = "invalid_response"
)

// ProviderError is safe to log and persist. It carries only a normalized
// category and HTTP status, never a response body, request payload, or key.
type ProviderError struct {
	Provider   string
	Code       ProviderErrorCode
	StatusCode int
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "generation provider error"
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("generation provider %s failed: %s (status %d)", e.Provider, e.Code, e.StatusCode)
	}
	return fmt.Sprintf("generation provider %s failed: %s", e.Provider, e.Code)
}

// ProviderErrorForStatus converts a provider HTTP status into the shared
// contract without retaining the provider response body.
func ProviderErrorForStatus(provider string, statusCode int) *ProviderError {
	code := ProviderErrorRejected
	switch {
	case statusCode == 429:
		code = ProviderErrorRateLimited
	case statusCode >= 500:
		code = ProviderErrorServer
	}
	return &ProviderError{Provider: strings.TrimSpace(provider), Code: code, StatusCode: statusCode}
}

// Result is the stable result contract retained with candidate provenance.
type Result struct {
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	Output       string `json:"output"`
	FinishReason string `json:"finish_reason,omitempty"`
	RequestID    string `json:"request_id,omitempty"`
	Usage        Usage  `json:"usage"`
}

// Port executes non-streaming generation. Optional streaming adapters expose
// StreamingPort without changing the business request or result contract.
type Port interface {
	Generate(context.Context, Request) (Result, error)
}

// StreamEvent is emitted only by adapters that support streaming.
type StreamEvent struct {
	Delta string `json:"delta,omitempty"`
}

// StreamingPort retains the final Result after a sequence of deltas.
type StreamingPort interface {
	Port
	GenerateStream(context.Context, Request, func(StreamEvent) error) (Result, error)
}

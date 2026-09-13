package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sharedgeneration "inkwords-backend/shared/kernel/generation"
)

type deepSeekJSONClient interface {
	GenerateJSONWithOptions(context.Context, string, []Message, ChatOptions) (string, CompletionUsage, error)
}

// deepSeekStreamingClient is intentionally separate from the JSON request
// client so a non-streaming test double cannot accidentally claim streaming
// support. The production DeepSeekClient implements both contracts.
type deepSeekStreamingClient interface {
	GenerateStreamWithOptions(context.Context, string, []Message, chan<- string, ChatOptions) (string, CompletionUsage, error)
}

// DeepSeekGenerationAdapter keeps DeepSeek transport details outside business
// packages. It deliberately serializes source excerpts as untrusted data in a
// user message, never as system or developer instructions.
type DeepSeekGenerationAdapter struct {
	client  deepSeekJSONClient
	timeout time.Duration
}

func NewDeepSeekGenerationAdapter(client *DeepSeekClient) *DeepSeekGenerationAdapter {
	return &DeepSeekGenerationAdapter{client: client, timeout: 45 * time.Second}
}

// NewDeepSeekGenerationAdapterWithTimeout creates an adapter for call sites
// whose bounded workload needs a timeout different from the legacy default.
func NewDeepSeekGenerationAdapterWithTimeout(client *DeepSeekClient, timeout time.Duration) *DeepSeekGenerationAdapter {
	adapter := NewDeepSeekGenerationAdapter(client)
	if timeout > 0 {
		adapter.timeout = timeout
	}
	return adapter
}

func (adapter *DeepSeekGenerationAdapter) Generate(ctx context.Context, request sharedgeneration.Request) (sharedgeneration.Result, error) {
	if adapter == nil || adapter.client == nil {
		return sharedgeneration.Result{}, fmt.Errorf("DeepSeek generation adapter is not configured")
	}
	if err := request.Validate(); err != nil {
		return sharedgeneration.Result{}, err
	}
	messages, err := deepSeekMessages(request)
	if err != nil {
		return sharedgeneration.Result{}, err
	}
	requestCtx, cancel := adapter.requestContext(ctx)
	defer cancel()
	options := deepSeekGenerationOptions(request)
	output, usage, err := adapter.client.GenerateJSONWithOptions(requestCtx, request.Model, messages, options)
	result := sharedgeneration.Result{Provider: "deepseek", Model: request.Model}
	if usage != (CompletionUsage{}) {
		result.Usage = sharedgeneration.Usage{Known: true, InputTokens: usage.PromptTokens, OutputTokens: usage.CompletionTokens, CachedTokens: usage.PromptCacheHitTokens}
	}
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return result, err
		}
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return result, sharedgeneration.ProviderErrorForStatus("deepseek", apiErr.StatusCode)
		}
		return result, &sharedgeneration.ProviderError{Provider: "deepseek", Code: sharedgeneration.ProviderErrorTransport}
	}
	if !json.Valid([]byte(output)) {
		return result, &sharedgeneration.ProviderError{Provider: "deepseek", Code: sharedgeneration.ProviderErrorInvalidResponse}
	}
	result.Output = output
	return result, nil
}

// GenerateStream implements the provider-neutral streaming boundary while
// retaining the same final-result validation as Generate. Deltas are never a
// successful result on their own: the complete output must still be valid JSON
// before it can be returned to application code.
func (adapter *DeepSeekGenerationAdapter) GenerateStream(ctx context.Context, request sharedgeneration.Request, sink func(sharedgeneration.StreamEvent) error) (sharedgeneration.Result, error) {
	if adapter == nil || adapter.client == nil {
		return sharedgeneration.Result{}, fmt.Errorf("DeepSeek generation adapter is not configured")
	}
	if sink == nil {
		return sharedgeneration.Result{}, fmt.Errorf("generation stream sink is required")
	}
	streamer, ok := adapter.client.(deepSeekStreamingClient)
	if !ok {
		return sharedgeneration.Result{}, fmt.Errorf("DeepSeek generation adapter does not support streaming")
	}
	if err := request.Validate(); err != nil {
		return sharedgeneration.Result{}, err
	}
	messages, err := deepSeekMessages(request)
	if err != nil {
		return sharedgeneration.Result{}, err
	}
	timeoutCtx, cancelTimeout := adapter.requestContext(ctx)
	defer cancelTimeout()
	streamCtx, cancel := context.WithCancel(timeoutCtx)
	defer cancel()
	deltas := make(chan string, 32)
	type completion struct {
		finishReason string
		usage        CompletionUsage
		err          error
	}
	done := make(chan completion, 1)
	go func() {
		finishReason, usage, runErr := streamer.GenerateStreamWithOptions(streamCtx, request.Model, messages, deltas, deepSeekGenerationOptions(request))
		done <- completion{finishReason: finishReason, usage: usage, err: runErr}
	}()

	var output strings.Builder
	consumeDelta := func(delta string) error {
		output.WriteString(delta)
		return sink(sharedgeneration.StreamEvent{Delta: delta})
	}
	for {
		select {
		case delta, open := <-deltas:
			if !open {
				deltas = nil
				continue
			}
			if err := consumeDelta(delta); err != nil {
				cancel()
				go drainGenerationDeltas(deltas)
				return sharedgeneration.Result{}, err
			}
		case completed := <-done:
			if completed.err != nil {
				return sharedgeneration.Result{}, normalizeDeepSeekGenerationError(completed.err)
			}
			// The producer may publish its completion before this goroutine has
			// selected every buffered delta. Its channel is closed by the client,
			// so draining it here is finite and makes final validation deterministic.
			if deltas != nil {
				for delta := range deltas {
					if err := consumeDelta(delta); err != nil {
						return sharedgeneration.Result{}, err
					}
				}
			}
			if !json.Valid([]byte(output.String())) {
				return sharedgeneration.Result{}, &sharedgeneration.ProviderError{Provider: "deepseek", Code: sharedgeneration.ProviderErrorInvalidResponse}
			}
			result := sharedgeneration.Result{Provider: "deepseek", Model: request.Model, Output: output.String(), FinishReason: completed.finishReason}
			if completed.usage != (CompletionUsage{}) {
				result.Usage = sharedgeneration.Usage{Known: true, InputTokens: completed.usage.PromptTokens, OutputTokens: completed.usage.CompletionTokens, CachedTokens: completed.usage.PromptCacheHitTokens}
			}
			return result, nil
		case <-ctx.Done():
			cancel()
			go drainGenerationDeltas(deltas)
			return sharedgeneration.Result{}, ctx.Err()
		}
	}
}

func drainGenerationDeltas(deltas <-chan string) {
	for range deltas {
	}
}

func deepSeekGenerationOptions(request sharedgeneration.Request) ChatOptions {
	options := LightweightChatOptions(request.MaxOutputTokens)
	if request.ReasoningEffort != "" {
		options.ThinkingType = "enabled"
		options.ReasoningEffort = request.ReasoningEffort
	}
	return options
}

func (adapter *DeepSeekGenerationAdapter) requestContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := adapter.timeout
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	return context.WithTimeout(ctx, timeout)
}

func normalizeDeepSeekGenerationError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return sharedgeneration.ProviderErrorForStatus("deepseek", apiErr.StatusCode)
	}
	return &sharedgeneration.ProviderError{Provider: "deepseek", Code: sharedgeneration.ProviderErrorTransport}
}

func deepSeekMessages(request sharedgeneration.Request) ([]Message, error) {
	input, err := sharedgeneration.BuildInputMessages(request)
	if err != nil {
		return nil, err
	}
	messages := make([]Message, len(input))
	for i, message := range input {
		messages[i] = Message{Role: message.Role, Content: message.Content}
	}
	return messages, nil
}

var _ sharedgeneration.Port = (*DeepSeekGenerationAdapter)(nil)
var _ sharedgeneration.StreamingPort = (*DeepSeekGenerationAdapter)(nil)

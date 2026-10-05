package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	sharedgeneration "inkwords-backend/shared/kernel/generation"
)

const defaultOpenAIAPIURL = "https://api.openai.com/v1/chat/completions"

// OpenAIGenerationAdapter implements the same domain port as DeepSeek without
// exposing OpenAI transport structs to application code.
type OpenAIGenerationAdapter struct {
	apiKey  string
	apiURL  string
	client  *http.Client
	timeout time.Duration
}

func NewOpenAIGenerationAdapter(apiKey string, clients ...*http.Client) *OpenAIGenerationAdapter {
	client := http.DefaultClient
	if len(clients) > 0 && clients[0] != nil {
		client = clients[0]
	}
	return &OpenAIGenerationAdapter{apiKey: strings.TrimSpace(apiKey), apiURL: defaultOpenAIAPIURL, client: client, timeout: 45 * time.Second}
}

// NewOpenAIGenerationAdapterWithTimeout creates an adapter for call sites
// whose bounded workload needs a timeout different from the legacy default.
func NewOpenAIGenerationAdapterWithTimeout(apiKey string, timeout time.Duration, clients ...*http.Client) *OpenAIGenerationAdapter {
	adapter := NewOpenAIGenerationAdapter(apiKey, clients...)
	if timeout > 0 {
		adapter.timeout = timeout
	}
	return adapter
}

func (adapter *OpenAIGenerationAdapter) Generate(ctx context.Context, request sharedgeneration.Request) (sharedgeneration.Result, error) {
	if adapter == nil || adapter.client == nil || adapter.apiKey == "" || adapter.apiURL == "" {
		return sharedgeneration.Result{}, fmt.Errorf("OpenAI generation adapter is not configured")
	}
	if err := request.Validate(); err != nil {
		return sharedgeneration.Result{}, err
	}
	if request.ReasoningEffort != "" {
		return sharedgeneration.Result{}, fmt.Errorf("OpenAI reasoning options are not configured")
	}
	messages, err := deepSeekMessages(request)
	if err != nil {
		return sharedgeneration.Result{}, err
	}
	payload := struct {
		Model          string    `json:"model"`
		Messages       []Message `json:"messages"`
		ResponseFormat any       `json:"response_format"`
		MaxTokens      int       `json:"max_tokens,omitempty"`
	}{Model: request.Model, Messages: messages, ResponseFormat: openAIResponseFormat(request.ResponseSchema), MaxTokens: request.MaxOutputTokens}
	body, err := json.Marshal(payload)
	if err != nil {
		return sharedgeneration.Result{}, fmt.Errorf("encode OpenAI generation request: %w", err)
	}
	timeout := adapter.timeout
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(requestCtx, http.MethodPost, adapter.apiURL, bytes.NewReader(body))
	if err != nil {
		return sharedgeneration.Result{}, fmt.Errorf("create OpenAI generation request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+adapter.apiKey)
	response, err := adapter.client.Do(httpRequest)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return sharedgeneration.Result{}, err
		}
		return sharedgeneration.Result{}, &sharedgeneration.ProviderError{Provider: "openai", Code: sharedgeneration.ProviderErrorTransport}
	}
	defer func() { _ = response.Body.Close() }()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 4<<20+1))
	if err != nil || len(responseBody) > 4<<20 {
		return sharedgeneration.Result{}, &sharedgeneration.ProviderError{Provider: "openai", Code: sharedgeneration.ProviderErrorTransport}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return sharedgeneration.Result{}, sharedgeneration.ProviderErrorForStatus("openai", response.StatusCode)
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			CachedTokens     int `json:"cached_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil || len(decoded.Choices) == 0 || !json.Valid([]byte(decoded.Choices[0].Message.Content)) {
		return sharedgeneration.Result{}, &sharedgeneration.ProviderError{Provider: "openai", Code: sharedgeneration.ProviderErrorInvalidResponse}
	}
	result := sharedgeneration.Result{Provider: "openai", Model: request.Model, Output: decoded.Choices[0].Message.Content, FinishReason: decoded.Choices[0].FinishReason, RequestID: response.Header.Get("X-Request-ID")}
	if decoded.Usage != nil {
		result.Usage = sharedgeneration.Usage{Known: true, InputTokens: decoded.Usage.PromptTokens, OutputTokens: decoded.Usage.CompletionTokens, CachedTokens: decoded.Usage.CachedTokens}
	}
	return result, nil
}

// GenerateStream implements the same provider-neutral stream contract as the
// DeepSeek adapter. It validates the reconstructed response before returning;
// partial deltas never become a persisted structured result.
func (adapter *OpenAIGenerationAdapter) GenerateStream(ctx context.Context, request sharedgeneration.Request, sink func(sharedgeneration.StreamEvent) error) (sharedgeneration.Result, error) {
	if adapter == nil || adapter.client == nil || adapter.apiKey == "" || adapter.apiURL == "" {
		return sharedgeneration.Result{}, fmt.Errorf("OpenAI generation adapter is not configured")
	}
	if sink == nil {
		return sharedgeneration.Result{}, fmt.Errorf("generation stream sink is required")
	}
	if err := request.Validate(); err != nil {
		return sharedgeneration.Result{}, err
	}
	if request.ReasoningEffort != "" {
		return sharedgeneration.Result{}, fmt.Errorf("OpenAI reasoning options are not configured")
	}
	messages, err := deepSeekMessages(request)
	if err != nil {
		return sharedgeneration.Result{}, err
	}
	payload := struct {
		Model          string    `json:"model"`
		Messages       []Message `json:"messages"`
		ResponseFormat any       `json:"response_format"`
		MaxTokens      int       `json:"max_tokens,omitempty"`
		Stream         bool      `json:"stream"`
		StreamOptions  any       `json:"stream_options"`
	}{Model: request.Model, Messages: messages, ResponseFormat: openAIResponseFormat(request.ResponseSchema), MaxTokens: request.MaxOutputTokens, Stream: true, StreamOptions: map[string]bool{"include_usage": true}}
	body, err := json.Marshal(payload)
	if err != nil {
		return sharedgeneration.Result{}, fmt.Errorf("encode OpenAI generation stream request: %w", err)
	}
	timeout := adapter.timeout
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(requestCtx, http.MethodPost, adapter.apiURL, bytes.NewReader(body))
	if err != nil {
		return sharedgeneration.Result{}, fmt.Errorf("create OpenAI generation stream request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+adapter.apiKey)
	response, err := adapter.client.Do(httpRequest)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return sharedgeneration.Result{}, err
		}
		return sharedgeneration.Result{}, &sharedgeneration.ProviderError{Provider: "openai", Code: sharedgeneration.ProviderErrorTransport}
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return sharedgeneration.Result{}, sharedgeneration.ProviderErrorForStatus("openai", response.StatusCode)
	}
	var output strings.Builder
	var finishReason string
	var usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		CachedTokens     int `json:"cached_tokens"`
	}
	scanner := bufio.NewScanner(io.LimitReader(response.Body, 4<<20+1))
	scanner.Buffer(make([]byte, 1024), 4<<20+1)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				CachedTokens     int `json:"cached_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return sharedgeneration.Result{}, &sharedgeneration.ProviderError{Provider: "openai", Code: sharedgeneration.ProviderErrorInvalidResponse}
		}
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		if chunk.Choices[0].FinishReason != nil {
			finishReason = *chunk.Choices[0].FinishReason
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		output.WriteString(delta)
		if err := sink(sharedgeneration.StreamEvent{Delta: delta}); err != nil {
			return sharedgeneration.Result{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return sharedgeneration.Result{}, err
		}
		return sharedgeneration.Result{}, &sharedgeneration.ProviderError{Provider: "openai", Code: sharedgeneration.ProviderErrorTransport}
	}
	if !json.Valid([]byte(output.String())) {
		return sharedgeneration.Result{}, &sharedgeneration.ProviderError{Provider: "openai", Code: sharedgeneration.ProviderErrorInvalidResponse}
	}
	result := sharedgeneration.Result{Provider: "openai", Model: request.Model, Output: output.String(), FinishReason: finishReason, RequestID: response.Header.Get("X-Request-ID")}
	if usage != nil {
		result.Usage = sharedgeneration.Usage{Known: true, InputTokens: usage.PromptTokens, OutputTokens: usage.CompletionTokens, CachedTokens: usage.CachedTokens}
	}
	return result, nil
}

func openAIResponseFormat(schema json.RawMessage) any {
	if len(schema) == 0 {
		return map[string]string{"type": "json_object"}
	}
	return struct {
		Type       string `json:"type"`
		JSONSchema struct {
			Name   string          `json:"name"`
			Strict bool            `json:"strict"`
			Schema json.RawMessage `json:"schema"`
		} `json:"json_schema"`
	}{
		Type: "json_schema",
		JSONSchema: struct {
			Name   string          `json:"name"`
			Strict bool            `json:"strict"`
			Schema json.RawMessage `json:"schema"`
		}{Name: "inkwords_generation", Strict: true, Schema: schema},
	}
}

var _ sharedgeneration.Port = (*OpenAIGenerationAdapter)(nil)
var _ sharedgeneration.StreamingPort = (*OpenAIGenerationAdapter)(nil)

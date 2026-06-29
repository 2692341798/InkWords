package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

const (
	defaultDeepSeekAPIURL = "https://api.deepseek.com/chat/completions"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the request payload for DeepSeek API
type ChatRequest struct {
	Model           string            `json:"model"`
	Messages        []Message         `json:"messages"`
	Stream          bool              `json:"stream"`
	Temperature     *float64          `json:"temperature,omitempty"`
	MaxTokens       int               `json:"max_tokens,omitempty"`
	Thinking        map[string]string `json:"thinking,omitempty"`
	ReasoningEffort string            `json:"reasoning_effort,omitempty"`
	ResponseFormat  map[string]string `json:"response_format,omitempty"`
	StreamOptions   *StreamOptions    `json:"stream_options,omitempty"`
	UserID          string            `json:"user_id,omitempty"`
}

// StreamOptions configures DeepSeek streaming response behavior.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// ChatOptions captures optional DeepSeek request controls used by specialized call sites.
type ChatOptions struct {
	ThinkingType    string
	ReasoningEffort string
	MaxTokens       int
	UserID          string
}

// ChatCompletionChunk represents a single chunk from the stream
type ChatCompletionChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// CompletionUsage captures token usage and prompt cache telemetry returned by DeepSeek.
type CompletionUsage struct {
	PromptTokens          int `json:"prompt_tokens"`
	CompletionTokens      int `json:"completion_tokens"`
	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
}

// DeepSeekClient is the client for calling DeepSeek API
type DeepSeekClient struct {
	APIKey string
	APIURL string
	Client *http.Client
}

// APIError exposes HTTP status information for retry classification.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API request failed with status %d: %s", e.StatusCode, e.Body)
}

// NewDeepSeekClient creates a new DeepSeek client
func NewDeepSeekClient(apiKey string) *DeepSeekClient {
	return &DeepSeekClient{
		APIKey: apiKey,
		APIURL: defaultDeepSeekAPIURL,
		Client: &http.Client{},
	}
}

// DefaultChatOptions keeps the previous behavior for regular generation calls.
func DefaultChatOptions() ChatOptions {
	return ChatOptions{ThinkingType: "enabled", ReasoningEffort: "high"}
}

// LightweightChatOptions is for bounded metadata/JSON tasks where reasoning is not worth the cost.
func LightweightChatOptions(userID string, maxTokens int) ChatOptions {
	return ChatOptions{ThinkingType: "disabled", MaxTokens: maxTokens, UserID: userID}
}

func (o ChatOptions) apply(req *ChatRequest) {
	thinkingType := strings.TrimSpace(o.ThinkingType)
	if thinkingType == "" {
		thinkingType = "enabled"
	}
	req.Thinking = map[string]string{"type": thinkingType}

	if thinkingType == "enabled" {
		reasoningEffort := strings.TrimSpace(o.ReasoningEffort)
		if reasoningEffort == "" {
			reasoningEffort = "high"
		}
		req.ReasoningEffort = reasoningEffort
	}

	if o.MaxTokens > 0 {
		req.MaxTokens = o.MaxTokens
	}
	if userID := sanitizeUserID(o.UserID); userID != "" {
		req.UserID = userID
	}
}

func sanitizeUserID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var builder strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '_':
			builder.WriteRune(r)
		}
		if builder.Len() >= 512 {
			break
		}
	}
	return builder.String()
}

// IsRetryableError returns true for transient transport/API failures.
func IsRetryableError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= http.StatusInternalServerError
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

func parseCompletionUsage(body []byte) CompletionUsage {
	var payload struct {
		Usage CompletionUsage `json:"usage"`
	}
	_ = json.Unmarshal(body, &payload)
	return payload.Usage
}

func (c *DeepSeekClient) doChatCompletion(ctx context.Context, reqBody ChatRequest) ([]byte, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(bodyBytes)}
	}

	return bodyBytes, nil
}

func extractCompletionContent(bodyBytes []byte) (string, error) {
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("API returned empty choices")
	}

	content := result.Choices[0].Message.Content
	content = thinkBlockPattern.ReplaceAllString(content, "")
	return strings.TrimSpace(content), nil
}

// Generate calls the DeepSeek API with stream=false and returns the full response content
func (c *DeepSeekClient) Generate(ctx context.Context, model string, messages []Message) (string, error) {
	content, _, err := c.GenerateWithUsage(ctx, model, messages)
	return content, err
}

// GenerateWithUsage calls the DeepSeek API with stream=false and returns the full response content plus usage.
func (c *DeepSeekClient) GenerateWithUsage(ctx context.Context, model string, messages []Message) (string, CompletionUsage, error) {
	return c.GenerateWithOptions(ctx, model, messages, DefaultChatOptions())
}

// GenerateWithOptions calls the DeepSeek API with stream=false and request options.
func (c *DeepSeekClient) GenerateWithOptions(ctx context.Context, model string, messages []Message, options ChatOptions) (string, CompletionUsage, error) {
	reqBody := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}
	options.apply(&reqBody)

	bodyBytes, err := c.doChatCompletion(ctx, reqBody)
	if err != nil {
		return "", CompletionUsage{}, err
	}

	content, err := extractCompletionContent(bodyBytes)
	if err != nil {
		return "", CompletionUsage{}, err
	}

	return content, parseCompletionUsage(bodyBytes), nil
}

// GenerateJSON calls the DeepSeek API with stream=false and response_format={"type": "json_object"}
func (c *DeepSeekClient) GenerateJSON(ctx context.Context, model string, messages []Message) (string, error) {
	content, _, err := c.GenerateJSONWithUsage(ctx, model, messages)
	return content, err
}

// GenerateJSONWithUsage calls the DeepSeek API with stream=false and response_format={"type": "json_object"}.
func (c *DeepSeekClient) GenerateJSONWithUsage(ctx context.Context, model string, messages []Message) (string, CompletionUsage, error) {
	return c.GenerateJSONWithOptions(ctx, model, messages, DefaultChatOptions())
}

// GenerateJSONWithOptions calls the DeepSeek API with JSON response mode and request options.
func (c *DeepSeekClient) GenerateJSONWithOptions(ctx context.Context, model string, messages []Message, options ChatOptions) (string, CompletionUsage, error) {
	temp := 0.1 // Recommend 0.1 for stable JSON output
	reqBody := ChatRequest{
		Model:          model,
		Messages:       messages,
		Stream:         false,
		Temperature:    &temp,
		ResponseFormat: map[string]string{"type": "json_object"},
	}
	options.apply(&reqBody)

	bodyBytes, err := c.doChatCompletion(ctx, reqBody)
	if err != nil {
		return "", CompletionUsage{}, err
	}

	content, err := extractCompletionContent(bodyBytes)
	if err != nil {
		return "", CompletionUsage{}, err
	}

	return content, parseCompletionUsage(bodyBytes), nil
}

// GenerateStream calls the DeepSeek API with stream=true and parses the chunks
// It sends the content deltas to the provided channel. It returns the finish reason and any error.
func (c *DeepSeekClient) GenerateStream(ctx context.Context, model string, messages []Message, chunkChan chan<- string) (string, error) {
	finishReason, _, err := c.GenerateStreamWithUsage(ctx, model, messages, chunkChan)
	return finishReason, err
}

// GenerateStreamWithUsage calls the DeepSeek API with stream=true, emits content deltas, and captures final usage.
func (c *DeepSeekClient) GenerateStreamWithUsage(ctx context.Context, model string, messages []Message, chunkChan chan<- string) (string, CompletionUsage, error) {
	return c.GenerateStreamWithOptions(ctx, model, messages, chunkChan, DefaultChatOptions())
}

// GenerateStreamWithOptions calls the DeepSeek API with stream=true and request options.
//
//nolint:gocyclo
func (c *DeepSeekClient) GenerateStreamWithOptions(ctx context.Context, model string, messages []Message, chunkChan chan<- string, options ChatOptions) (string, CompletionUsage, error) {
	defer close(chunkChan)
	reqBody := ChatRequest{
		Model:         model,
		Messages:      messages,
		Stream:        true,
		StreamOptions: &StreamOptions{IncludeUsage: true},
	}
	options.apply(&reqBody)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", CompletionUsage{}, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", CompletionUsage{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", CompletionUsage{}, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", CompletionUsage{}, &APIError{StatusCode: resp.StatusCode, Body: string(bodyBytes)}
	}

	reader := bufio.NewReader(resp.Body)
	var finalFinishReason string
	var usage CompletionUsage
	var leadingBuffer strings.Builder
	leadingFlushed := false
	flushLeadingBuffer := func() {
		if leadingFlushed {
			return
		}
		sanitized := sanitizeLeadingGeneratedText(leadingBuffer.String())
		if sanitized != "" {
			chunkChan <- sanitized
		}
		leadingFlushed = true
		leadingBuffer.Reset()
	}

	for {
		select {
		case <-ctx.Done():
			return "", usage, ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				flushLeadingBuffer()
				return finalFinishReason, usage, nil
			}
			return "", usage, fmt.Errorf("failed to read stream: %w", err)
		}

		lineStr := strings.TrimSpace(string(line))
		if lineStr == "" {
			continue
		}

		if !strings.HasPrefix(lineStr, "data: ") {
			continue
		}

		data := strings.TrimPrefix(lineStr, "data: ")
		if data == "[DONE]" {
			flushLeadingBuffer()
			return finalFinishReason, usage, nil
		}

		parsedUsage := parseCompletionUsage([]byte(data))
		if parsedUsage != (CompletionUsage{}) {
			usage = parsedUsage
		}

		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta

			if delta.Content != "" {
				if !leadingFlushed {
					leadingBuffer.WriteString(delta.Content)
					buffer := leadingBuffer.String()
					if shouldHoldLeadingSanitization(buffer) {
						continue
					}

					sanitized := sanitizeLeadingGeneratedText(buffer)
					if sanitized != "" {
						chunkChan <- sanitized
					}
					leadingFlushed = true
					leadingBuffer.Reset()
					continue
				}

				chunkChan <- delta.Content
			}

			if chunk.Choices[0].FinishReason != nil {
				finalFinishReason = *chunk.Choices[0].FinishReason
				if finalFinishReason == "stop" || finalFinishReason == "length" {
					flushLeadingBuffer()
					continue
				}
			}
		}
	}
}

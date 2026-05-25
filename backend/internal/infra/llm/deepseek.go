package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	Model          string            `json:"model"`
	Messages       []Message         `json:"messages"`
	Stream         bool              `json:"stream"`
	Temperature    *float64          `json:"temperature,omitempty"`
	MaxTokens      int               `json:"max_tokens,omitempty"`
	Thinking       map[string]string `json:"thinking,omitempty"`
	ReasoningEffort string           `json:"reasoning_effort,omitempty"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
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

// DeepSeekClient is the client for calling DeepSeek API
type DeepSeekClient struct {
	APIKey string
	APIURL string
	Client *http.Client
}

// NewDeepSeekClient creates a new DeepSeek client
func NewDeepSeekClient(apiKey string) *DeepSeekClient {
	return &DeepSeekClient{
		APIKey: apiKey,
		APIURL: defaultDeepSeekAPIURL,
		Client: &http.Client{},
	}
}

// Generate calls the DeepSeek API with stream=false and returns the full response content
func (c *DeepSeekClient) Generate(ctx context.Context, model string, messages []Message) (string, error) {
	reqBody := ChatRequest{
		Model:           model,
		Messages:        messages,
		Stream:          false,
		Thinking:        map[string]string{"type": "enabled"},
		ReasoningEffort: "high",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

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

// GenerateJSON calls the DeepSeek API with stream=false and response_format={"type": "json_object"}
func (c *DeepSeekClient) GenerateJSON(ctx context.Context, model string, messages []Message) (string, error) {
	temp := 0.1 // Recommend 0.1 for stable JSON output
	reqBody := ChatRequest{
		Model:           model,
		Messages:        messages,
		Stream:          false,
		Temperature:     &temp,
		Thinking:        map[string]string{"type": "enabled"},
		ReasoningEffort: "high",
		ResponseFormat:  map[string]string{"type": "json_object"},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

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

// GenerateStream calls the DeepSeek API with stream=true and parses the chunks
// It sends the content deltas to the provided channel. It returns the finish reason and any error.
func (c *DeepSeekClient) GenerateStream(ctx context.Context, model string, messages []Message, chunkChan chan<- string) (string, error) {
	defer close(chunkChan)
	reqBody := ChatRequest{
		Model:           model,
		Messages:        messages,
		Stream:          true,
		Thinking:        map[string]string{"type": "enabled"},
		ReasoningEffort: "high",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	reader := bufio.NewReader(resp.Body)
	var finalFinishReason string
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
			return "", ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				flushLeadingBuffer()
				return finalFinishReason, nil
			}
			return "", fmt.Errorf("failed to read stream: %w", err)
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
			return finalFinishReason, nil
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
					return finalFinishReason, nil
				}
			}
		}
	}
}

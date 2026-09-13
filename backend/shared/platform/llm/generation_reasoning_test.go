package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/shared/kernel/generation"
)

func TestGenerationReasoningOptionsReachDeepSeekWire(t *testing.T) {
	for _, effort := range []string{"", "low", "high"} {
		for _, streaming := range []bool{false, true} {
			t.Run(effort+map[bool]string{false: "-json", true: "-stream"}[streaming], func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, float64(3000), body["max_tokens"])
					if effort == "" {
						require.Equal(t, "disabled", body["thinking"].(map[string]any)["type"])
						require.NotContains(t, body, "reasoning_effort")
					} else {
						require.Equal(t, "enabled", body["thinking"].(map[string]any)["type"])
						require.Equal(t, effort, body["reasoning_effort"])
					}
					if streaming {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"private reasoning\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"{}\"},\"finish_reason\":\"stop\"}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":500}}\n\ndata: [DONE]\n\n"))
					} else {
						_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{}","reasoning_content":"private reasoning"}}],"usage":{"prompt_tokens":100,"completion_tokens":500}}`))
					}
				}))
				defer srv.Close()
				client := NewDeepSeekClient("fixture")
				client.APIURL = srv.URL
				adapter := NewDeepSeekGenerationAdapter(client)
				request := reasoningFixture(effort)
				var result generation.Result
				var err error
				if streaming {
					var deltas string
					result, err = adapter.GenerateStream(context.Background(), request, func(event generation.StreamEvent) error { deltas += event.Delta; return nil })
					require.Equal(t, "{}", deltas)
				} else {
					result, err = adapter.Generate(context.Background(), request)
				}
				require.NoError(t, err)
				require.Equal(t, "{}", result.Output)
				require.True(t, result.Usage.Known)
				require.Equal(t, 500, result.Usage.OutputTokens)
			})
		}
	}
}

func TestUnsupportedReasoningCannotBeSilentlyIgnored(t *testing.T) {
	client := &http.Client{Transport: responseRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("unsupported option must not contact provider")
		return nil, nil
	})}
	adapter := NewOpenAIGenerationAdapter("fixture", client)
	for _, effort := range []string{"low", "high", "unbounded"} {
		request := reasoningFixture(effort)
		_, err := adapter.Generate(context.Background(), request)
		require.Error(t, err)
		_, err = adapter.GenerateStream(context.Background(), request, func(generation.StreamEvent) error { return nil })
		require.Error(t, err)
	}
	request := reasoningFixture("unbounded")
	require.ErrorContains(t, request.Validate(), "reasoning")
	request = reasoningFixture("low")
	request.MaxOutputTokens = 0
	require.ErrorContains(t, request.Validate(), "budget")
}

func reasoningFixture(effort string) generation.Request {
	return generation.Request{Model: "fixture", SystemInstruction: "JSON", TaskInstruction: "grade", Evidence: []generation.Evidence{{ID: "e", SnapshotID: "s", Locator: "l", Content: "data"}}, MaxOutputTokens: 3000, ReasoningEffort: effort}
}

func TestDeepSeekInvalidJSONPreservesUsageWithoutResponseBody(t *testing.T) {
	client := &capturedDeepSeekJSONClient{output: "truncated private output", usage: CompletionUsage{PromptTokens: 100, CompletionTokens: 3000}}
	result, err := (&DeepSeekGenerationAdapter{client: client}).Generate(context.Background(), reasoningFixture("high"))
	require.Error(t, err)
	require.Empty(t, result.Output)
	require.Equal(t, "deepseek", result.Provider)
	require.Equal(t, 3000, result.Usage.OutputTokens)
}

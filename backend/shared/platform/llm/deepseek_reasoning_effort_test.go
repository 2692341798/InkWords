package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeepSeekClient_Generate_setsReasoningEffortHigh(t *testing.T) {
	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		capturedBody = body

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	t.Cleanup(srv.Close)

	client := NewDeepSeekClient("test-key")
	client.APIURL = srv.URL

	_, err := client.Generate(context.Background(), "deepseek-v4-flash", []Message{{Role: "user", Content: "hi"}})
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(capturedBody, &payload))
	require.Equal(t, "high", payload["reasoning_effort"])
}

func TestDeepSeekClient_GenerateJSONWithOptions_DisablesThinkingAndSetsMaxTokensAndUserID(t *testing.T) {
	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		capturedBody = body

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	t.Cleanup(srv.Close)

	client := NewDeepSeekClient("test-key")
	client.APIURL = srv.URL

	_, _, err := client.GenerateJSONWithOptions(
		context.Background(),
		"deepseek-v4-flash",
		[]Message{{Role: "user", Content: "hi"}},
		LightweightChatOptions("task:abc@example.com", 256),
	)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(capturedBody, &payload))
	require.Equal(t, float64(256), payload["max_tokens"])
	require.Equal(t, "taskabcexamplecom", payload["user_id"])
	require.Nil(t, payload["reasoning_effort"])

	thinking, ok := payload["thinking"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "disabled", thinking["type"])
}

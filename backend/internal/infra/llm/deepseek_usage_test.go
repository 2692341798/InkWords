package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCompletionUsage_ReadsPromptCacheFields(t *testing.T) {
	usage := parseCompletionUsage([]byte(`{
		"usage": {
			"prompt_tokens": 1200,
			"completion_tokens": 500,
			"prompt_cache_hit_tokens": 900,
			"prompt_cache_miss_tokens": 300
		}
	}`))

	require.Equal(t, 1200, usage.PromptTokens)
	require.Equal(t, 500, usage.CompletionTokens)
	require.Equal(t, 900, usage.PromptCacheHitTokens)
	require.Equal(t, 300, usage.PromptCacheMissTokens)
}

func TestDeepSeekClient_GenerateStreamWithUsage_ReadsUsageFromFinalChunk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"最终正文\"},\"finish_reason\":null}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":1200,\"completion_tokens\":500,\"prompt_cache_hit_tokens\":900,\"prompt_cache_miss_tokens\":300}}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)

	client := NewDeepSeekClient("test-key")
	client.APIURL = srv.URL
	client.Client = srv.Client()

	chunkChan := make(chan string, 8)
	finishReason, usage, err := client.GenerateStreamWithUsage(context.Background(), "deepseek-v4-flash", []Message{{Role: "user", Content: "hi"}}, chunkChan)
	require.NoError(t, err)
	require.Equal(t, "stop", finishReason)
	require.Equal(t, CompletionUsage{
		PromptTokens:          1200,
		CompletionTokens:      500,
		PromptCacheHitTokens:  900,
		PromptCacheMissTokens: 300,
	}, usage)

	var chunks []string
	for chunk := range chunkChan {
		chunks = append(chunks, chunk)
	}

	require.Equal(t, []string{"最终正文"}, chunks)
}

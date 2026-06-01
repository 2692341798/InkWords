package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"inkwords-backend/internal/infra/parser"
)

func TestResolveFileAnalyzeChunkSize_UsesFinerSlicesForLargeDocuments(t *testing.T) {
	require.Equal(t, 120000, resolveFileAnalyzeChunkSize(1000001))
	require.Equal(t, 120000, resolveFileAnalyzeChunkSize(3000000))
	require.Equal(t, 0, resolveFileAnalyzeChunkSize(1000000))
}

func TestMapReduceAnalyzeFile_PreservesOriginalChunkOrder(t *testing.T) {
	t.Setenv("MAX_CONCURRENT_WORKERS", "10")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Len(t, req.Messages, 2)

		prompt := req.Messages[1].Content
		switch {
		case strings.Contains(prompt, "分块标识：第 1 部分"):
			time.Sleep(120 * time.Millisecond)
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"第一部分摘要"}}]}`))
		case strings.Contains(prompt, "分块标识：第 2 部分"):
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"第二部分摘要"}}]}`))
		default:
			t.Fatalf("unexpected prompt: %s", prompt)
		}
	}))
	defer server.Close()

	svc := NewDecompositionService(nil)
	svc.llmClient.APIURL = server.URL
	svc.llmClient.Client = server.Client()

	chunks := []parser.FileChunk{
		{Dir: "第 1 部分", Content: "第一部分原文"},
		{Dir: "第 2 部分", Content: "第二部分原文"},
	}

	summaries := svc.mapReduceAnalyzeFile(context.Background(), chunks, func(int, string, interface{}) {})

	require.Equal(t, []string{
		"【第 1 部分】\n第一部分摘要",
		"【第 2 部分】\n第二部分摘要",
	}, summaries)
}

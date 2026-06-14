package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeepSeekClient_GenerateStream_stripsLeadingConversationalPreamble(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"好的，收到你的需求。作为高级全栈架构师和技术博主，我将根据你提供的内容输出高质量博客。\\n\\n\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"# Python 基础语法速通\\n\\n这里是正文。\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)

	client := NewDeepSeekClient("test-key")
	client.APIURL = srv.URL

	chunkChan := make(chan string, 8)
	finishReason, err := client.GenerateStream(context.Background(), "deepseek-v4-flash", []Message{{Role: "user", Content: "hi"}}, chunkChan)
	require.NoError(t, err)
	require.Equal(t, "stop", finishReason)

	var builder strings.Builder
	for chunk := range chunkChan {
		builder.WriteString(chunk)
	}

	require.Equal(t, "# Python 基础语法速通\n\n这里是正文。", builder.String())
}

func TestSanitizeLeadingGeneratedText_StripsTextInterpreterPreamble(t *testing.T) {
	content := "我将以文本解读专家的身份，围绕原文逐章展开说明。\n\n# 非暴力沟通导读\n\n这里是正文。"

	require.Equal(t, "# 非暴力沟通导读\n\n这里是正文。", sanitizeLeadingGeneratedText(content))
}

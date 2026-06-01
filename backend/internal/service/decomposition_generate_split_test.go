package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"inkwords-backend/internal/infra/llm"
	"inkwords-backend/internal/prompt"
)

func TestBuildSeriesChapterExtraRequirements_IncludesGitURLAndNextPreview(t *testing.T) {
	outline := []Chapter{
		{Title: "第一章", Sort: 1},
		{Title: "第二章", Sort: 2},
	}

	got := buildSeriesChapterExtraRequirements("https://github.com/acme/demo", outline, 0)

	require.Contains(t, got, "https://github.com/acme/demo")
	require.Contains(t, got, "下期预告：第二章")
}

func TestResolveSeriesChapterSourceContent_TruncatesLongFallbackContent(t *testing.T) {
	original := strings.Repeat("长", 1000001)

	got := resolveSeriesChapterSourceContent("file", "", original, Chapter{
		Title: "始计第一",
		Files: []string{"ignored.go"},
	})

	require.Contains(t, got, "[Content Truncated due to length limits]")
	require.Len(t, []rune(got), seriesChapterSourceRuneLimit+len([]rune(seriesContentTruncatedSuffix)))
}

func TestStreamSeriesChapterContent_RetriesAndConcatenatesChunks(t *testing.T) {
	var attemptCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callNumber := attemptCount.Add(1)
		if callNumber == 1 {
			http.Error(w, "temporary failure", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"第一段\"},\"finish_reason\":null}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"第二段\"},\"finish_reason\":null}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
	}))
	defer server.Close()

	svc := NewDecompositionService(nil)
	svc.llmClient.APIURL = server.URL
	svc.llmClient.Client = server.Client()

	progressChan := make(chan string, 16)
	content, err := svc.streamSeriesChapterContent(
		context.Background(),
		uuid.New(),
		Chapter{Title: "第一章", Sort: 1},
		[]llm.Message{{Role: "user", Content: "测试"}},
		progressChan,
	)
	close(progressChan)

	require.NoError(t, err)
	require.Equal(t, "第一段第二段", content)
	require.EqualValues(t, 2, attemptCount.Load())

	var progressPayloads []map[string]interface{}
	for raw := range progressChan {
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(raw), &payload))
		progressPayloads = append(progressPayloads, payload)
	}

	require.Len(t, progressPayloads, 3)
	require.Equal(t, "generating", progressPayloads[0]["status"])
	require.Equal(t, "retrying", progressPayloads[1]["status"])
	require.Equal(t, float64(2), progressPayloads[1]["attempt"])
	require.Equal(t, "streaming", progressPayloads[2]["status"])
}

func TestBuildSeriesChapterMessages_UsesResolvedRequirementsAndOldContentReference(t *testing.T) {
	svc := NewDecompositionService(nil)
	profile := prompt.ResolvePromptProfileKey(
		"psychology_communication_book",
		prompt.ScenarioModeEbookInterpretation,
	)

	userID := uuid.New()
	chapter := Chapter{Title: "始计第一", Summary: "逐章精读", Sort: 1}
	outline := []Chapter{
		chapter,
		{Title: "作战第二", Summary: "承上启下", Sort: 2},
	}

	messages, modelName, err := svc.buildSeriesChapterMessages(
		context.Background(),
		userID,
		chapter,
		outline,
		0,
		"源内容",
		"file",
		"https://github.com/acme/demo",
		prompt.ScenarioModeEbookInterpretation,
		string(prompt.ArticleStyleGeneral),
		"旧内容",
		profile,
	)

	require.NoError(t, err)
	require.Equal(t, "deepseek-v4-flash", modelName)
	require.Len(t, messages, 2)
	require.Contains(t, messages[0].Content, "心理学")
	require.Contains(t, messages[0].Content, "项目源内容如下：\n源内容")
	require.NotContains(t, messages[0].Content, "高级全栈架构师")
	require.Contains(t, messages[1].Content, "下期预告：作战第二")
	require.Contains(t, messages[1].Content, "https://github.com/acme/demo")
	require.Contains(t, messages[1].Content, "旧版本内容")
	require.Contains(t, messages[1].Content, profile.GenerateRequirements)
}

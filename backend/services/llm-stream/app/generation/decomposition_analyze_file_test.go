package generation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"inkwords-backend/shared/kernel/prompt"
)

func TestAnalyzeFileStreamReturnsOutlineContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			require.NoError(t, r.Body.Close())
		}()

		var request struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.NotEmpty(t, request.Messages)

		content := `{"prompt_profile_key":"open_book_exam_review","document_kind":"course_notes","reason":"复习资料"}`
		for _, message := range request.Messages {
			if strings.Contains(message.Content, "输出必须是纯 JSON") {
				content = `{"series_title":"数据库复习","chapters":[{"title":"事务与并发","summary":"ACID 与并发控制","sort":1,"files":[],"action":"new"}]}`
				break
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":` + quoteJSON(content) + `}}]}`))
	}))
	defer server.Close()

	service := NewDecompositionService(nil, nil, nil)
	service.llmClient.APIURL = server.URL
	service.llmClient.Client = server.Client()

	progress := make(chan string, 16)
	errs := make(chan error, 1)
	service.AnalyzeFileStream(
		context.Background(),
		uuid.New(),
		"事务 ACID 与并发控制",
		prompt.ScenarioModeOpenBookExamReview,
		progress,
		errs,
	)

	require.Empty(t, collectAnalyzeErrors(errs))
	events := collectAnalyzeEvents(progress)
	require.NotEmpty(t, events)

	var finalEvent map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(events[len(events)-1]), &finalEvent))
	require.Equal(t, "complete", finalEvent["status"])
	content, ok := finalEvent["content"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "数据库复习", content["series_title"])
	require.NotEmpty(t, content["outline"])
}

func quoteJSON(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func collectAnalyzeEvents(events <-chan string) []string {
	var collected []string
	for event := range events {
		collected = append(collected, event)
	}
	return collected
}

func collectAnalyzeErrors(errs <-chan error) []error {
	var collected []error
	for err := range errs {
		collected = append(collected, err)
	}
	return collected
}

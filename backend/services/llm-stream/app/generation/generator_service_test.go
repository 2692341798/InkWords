package generation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	sharedblog "inkwords-backend/shared/kernel/blog"
	"inkwords-backend/shared/kernel/prompt"
	llm "inkwords-backend/shared/platform/llm"
)

type generatedPersistenceRecorder struct {
	calls int
	saved sharedblog.GeneratedBlogPersistenceInput
	err   error
}

func (r *generatedPersistenceRecorder) SaveGeneratedBlog(_ context.Context, input sharedblog.GeneratedBlogPersistenceInput) error {
	r.calls++
	r.saved = input
	return r.err
}

func newGenerationLLMServer(t *testing.T, streamContent string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var request struct {
			Stream bool `json:"stream"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		if request.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q},\"finish_reason\":null}]}\n\n", streamContent)
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
			fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":20,\"completion_tokens\":10,\"prompt_cache_hit_tokens\":12,\"prompt_cache_miss_tokens\":8}}\n\n")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"[\"Go\",\"Docker\"]"}}]}`)
	}))
}

func attachGeneratorServer(svc *GeneratorService, server *httptest.Server) {
	svc.llmClient = &llm.DeepSeekClient{APIKey: "test", APIURL: server.URL, Client: server.Client()}
}

func TestGeneratorSaveUsesInjectedPersistenceAndReturnsFailure(t *testing.T) {
	server := newGenerationLLMServer(t, "")
	defer server.Close()
	recorder := &generatedPersistenceRecorder{}
	svc := NewGeneratorService(nil, recorder)
	attachGeneratorServer(svc, server)
	userID := uuid.New()
	require.NoError(t, svc.saveToDB(context.Background(), userID, "file", "hello"))
	require.Equal(t, 1, recorder.calls)
	require.Equal(t, userID, recorder.saved.UserID)
	require.Equal(t, "hello", recorder.saved.Content)
	require.JSONEq(t, `["Go","Docker"]`, string(recorder.saved.TechStacks))

	recorder.err = errors.New("database unavailable")
	err := svc.saveToDB(context.Background(), userID, "file", "hello")
	require.ErrorContains(t, err, "persist generated blog")
}

func TestGenerateBlogTaskOnlyDoesNotPersistAndBuildsResultWithUsage(t *testing.T) {
	t.Setenv("INKWORDS_TASK_PERSISTENCE_MODE", "task_only")
	server := newGenerationLLMServer(t, "# 标题\n\n正文")
	defer server.Close()
	recorder := &generatedPersistenceRecorder{}
	svc := NewGeneratorService(nil, recorder)
	attachGeneratorServer(svc, server)
	chunks := make(chan string, 8)
	errs := make(chan error, 1)
	svc.GenerateBlogStreamWithProfile(context.Background(), uuid.New(), "原文", "file", prompt.ScenarioModeEbookInterpretation, string(prompt.ArticleStyleGeneral), prompt.PromptProfile{}, chunks, errs)
	var content string
	for chunk := range chunks {
		content += chunk
	}
	for err := range errs {
		require.NoError(t, err)
	}
	require.Zero(t, recorder.calls)
	result, err := svc.BuildGenerateSingleTaskResult(context.Background(), "file", content)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(result.ResultJSON, &decoded))
	require.Equal(t, "generate_single", decoded["task_subtype"])
	require.Equal(t, float64(20), decoded["usage"].(map[string]any)["prompt_tokens"])
	require.Equal(t, content, decoded["payload"].(map[string]any)["content"])
}

func TestBuildSingleGenerateMessagesIncludesPromptProfileRole(t *testing.T) {
	profile := prompt.ResolvePromptProfileKey("psychology_communication_book", prompt.ScenarioModeEbookInterpretation)
	messages := buildSingleGenerateMessages("源内容", "写作要求", profile)
	require.Len(t, messages, 2)
	require.Contains(t, messages[0].Content, "心理学")
	require.Contains(t, messages[0].Content, "源内容")
	require.Contains(t, messages[1].Content, "写作要求")
}

package generation

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	sharedblog "inkwords-backend/shared/kernel/blog"
	llm "inkwords-backend/shared/platform/llm"
)

type continuePersistenceRecorder struct {
	blog         sharedblog.ContinueBlog
	loadErr      error
	saveErr      error
	loadCalls    int
	saveCalls    int
	savedContent string
}

func (r *continuePersistenceRecorder) LoadContinueBlog(context.Context, uuid.UUID, uuid.UUID) (sharedblog.ContinueBlog, error) {
	r.loadCalls++
	return r.blog, r.loadErr
}

func (r *continuePersistenceRecorder) SaveContinuedBlog(_ context.Context, _ sharedblog.ContinueBlog, content string) error {
	r.saveCalls++
	r.savedContent = content
	return r.saveErr
}

func TestContinueGenerationUsesInjectedPersistence(t *testing.T) {
	server := newGenerationLLMServer(t, "追加内容")
	defer server.Close()
	persistence := &continuePersistenceRecorder{blog: sharedblog.ContinueBlog{ID: uuid.New(), UserID: uuid.New(), Content: "旧内容"}}
	svc := NewDecompositionService(nil, nil, persistence)
	svc.llmClient = &llm.DeepSeekClient{APIKey: "test", APIURL: server.URL, Client: server.Client()}
	chunks := make(chan string, 8)
	errs := make(chan error, 1)
	svc.ContinueGeneration(context.Background(), persistence.blog.UserID, persistence.blog.ID, chunks, errs)
	var content string
	for chunk := range chunks {
		content += chunk
	}
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, "追加内容", content)
	require.Equal(t, 1, persistence.saveCalls)
	require.Equal(t, "旧内容追加内容", persistence.savedContent)
}

func TestContinueGenerationCancellationStopsBeforeStreaming(t *testing.T) {
	persistence := &continuePersistenceRecorder{blog: sharedblog.ContinueBlog{ID: uuid.New(), UserID: uuid.New(), Content: "旧内容"}}
	svc := NewDecompositionService(nil, nil, persistence)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	chunks := make(chan string, 1)
	errs := make(chan error, 1)
	svc.ContinueGeneration(ctx, persistence.blog.UserID, persistence.blog.ID, chunks, errs)
	require.Empty(t, chunks)
	require.ErrorIs(t, <-errs, context.Canceled)
}

func TestBuildContinueTaskResultUsesPersistenceAndUsage(t *testing.T) {
	persistence := &continuePersistenceRecorder{blog: sharedblog.ContinueBlog{ID: uuid.New(), UserID: uuid.New(), Content: "旧内容"}}
	svc := NewDecompositionService(nil, nil, persistence)
	svc.storeContinueUsage(persistence.blog.ID, "追加", llm.CompletionUsage{PromptTokens: 9, CompletionTokens: 4})
	result, err := svc.BuildContinueTaskResult(context.Background(), persistence.blog.UserID, persistence.blog.ID, "追加")
	require.NoError(t, err)
	require.Equal(t, "旧内容追加", result.FinalContent)
	require.Equal(t, 9, result.Usage.PromptTokens)

	persistence.loadErr = errors.New("not found")
	_, err = svc.BuildContinueTaskResult(context.Background(), persistence.blog.UserID, persistence.blog.ID, "追加")
	require.ErrorContains(t, err, "load continue blog")
}

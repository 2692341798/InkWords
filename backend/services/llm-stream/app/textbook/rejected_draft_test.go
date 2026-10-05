package textbook

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type recordingRejectedStore struct {
	data []byte
	err  error
}

func (s *recordingRejectedStore) Save(_ context.Context, data []byte) (string, error) {
	s.data = append([]byte(nil), data...)
	return strings.Repeat("a", 64), s.err
}

func TestRejectedDraftCanBeRecheckedWithoutProviderOrPromotion(t *testing.T) {
	request := sampleGenerationRequest(t)
	request.GenerationTarget = sharedtextbook.SampleGenerationTarget{ProviderName: "fixture-provider", ModelName: "configured-model"}
	chapter, err := BuildGinRequestLifecycleSample(request.EvidencePack)
	require.NoError(t, err)
	chapter.Markdown += "\n\nABCD 未解释。ABCD 再次出现。"
	encoded, err := json.Marshal(chapter)
	require.NoError(t, err)
	port := &capturedGenerationPort{result: sharedgeneration.Result{Provider: "fixture-provider", Model: "configured-model", Output: string(encoded)}}
	store := &recordingRejectedStore{}
	generator := NewRetainingSampleGenerator(NewPortSampleGenerator(port, "configured-model"), store)
	generation, err := generator.Generate(context.Background(), request)
	require.Error(t, err)
	require.Equal(t, 1, port.calls)
	require.Equal(t, "saved", generation.RejectedDraftStorage)
	var draft RejectedDraft
	require.NoError(t, json.Unmarshal(store.data, &draft))
	require.NoError(t, draft.Validate())
	bad, err := RecheckRejectedDraft(draft, draft.Generation.Chapter.Markdown)
	require.NoError(t, err)
	require.False(t, bad.Quality.Passed)
	fixed := strings.Replace(draft.Generation.Chapter.Markdown, "ABCD 未解释。ABCD 再次出现。", "ABCD（示例缩写）已解释。ABCD 再次出现。", 1)
	checked, err := RecheckRejectedDraft(draft, fixed)
	require.NoError(t, err)
	require.True(t, checked.Quality.Passed, checked.Quality.Failures)
	require.Zero(t, checked.ProviderCalls)
	require.False(t, checked.CandidatePersisted)
	require.Equal(t, 1, port.calls)
	require.NotEqual(t, checked.OriginalContentHash, checked.CheckedContentHash)
	require.Contains(t, draft.Generation.Chapter.Markdown, "ABCD 未解释")
	broken := strings.Replace(fixed, "[evidence:gin-routergroup-get]", "[evidence:unknown-source]", 1)
	checked, err = RecheckRejectedDraft(draft, broken)
	require.NoError(t, err)
	require.False(t, checked.Quality.Passed)
	store.err = errors.New("private storage error")
	generation, err = generator.Generate(context.Background(), request)
	require.Error(t, err)
	require.Equal(t, "unavailable", generation.RejectedDraftStorage)
	require.Empty(t, generation.RejectedDraftHash)
	require.NotContains(t, err.Error(), "private storage error")
}

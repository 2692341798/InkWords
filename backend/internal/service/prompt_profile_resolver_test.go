package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	llm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/internal/prompt"
)

type fakePromptProfileLLM struct {
	payload string
	err     error
}

func (f fakePromptProfileLLM) GenerateJSON(_ context.Context, _ string, _ []llm.Message) (string, error) {
	return f.payload, f.err
}

func TestFallbackPromptProfileForScenario_FileEbookUsesClassicInterpretation(t *testing.T) {
	profile := prompt.FallbackPromptProfileForScenario(prompt.ScenarioModeEbookInterpretation)

	require.Equal(t, prompt.PromptProfileClassicTextInterpretation, profile.Key)
	require.Equal(t, "经典文本解读", profile.DisplayName)
	require.Contains(t, profile.SystemRole, "文本解读")
}

func TestResolvePromptProfileKey_FallsBackForUnknownKey(t *testing.T) {
	profile := prompt.ResolvePromptProfileKey("bad_key", prompt.ScenarioModeEbookInterpretation)

	require.Equal(t, prompt.PromptProfileClassicTextInterpretation, profile.Key)
	require.Equal(t, "classic_text_interpretation", string(profile.Key))
}

func TestPromptProfileResolver_ResolveForFile_FallsBackWhenClassifierUnavailable(t *testing.T) {
	resolver := NewPromptProfileResolver(nil)

	profile, resolved, err := resolver.ResolveForFile(
		context.Background(),
		"这是一段需要分类的原文内容",
		prompt.ScenarioModeEbookInterpretation,
	)
	require.NoError(t, err)
	require.Equal(t, prompt.PromptProfileClassicTextInterpretation, profile.Key)
	require.Equal(t, profile.Key, resolved.Key)
	require.Equal(t, profile.DocumentKind, resolved.DocumentKind)
	require.Contains(t, resolved.Reason, "回退")
}

func TestPromptProfileResolver_ResolveForFile_UsesParsedJSONResult(t *testing.T) {
	resolver := NewPromptProfileResolver(fakePromptProfileLLM{
		payload: `{"prompt_profile_key":"psychology_communication_book","document_kind":"psychology_communication","reason":"命中沟通与心理主题"}`,
	})

	profile, resolved, err := resolver.ResolveForFile(
		context.Background(),
		"书中讨论了感受、需要、表达与沟通冲突。",
		prompt.ScenarioModeEbookInterpretation,
	)
	require.NoError(t, err)
	require.Equal(t, prompt.PromptProfilePsychologyCommunication, profile.Key)
	require.Equal(t, "心理学经典解读", resolved.DisplayName)
	require.Equal(t, "psychology_communication", resolved.DocumentKind)
	require.Contains(t, resolved.Reason, "沟通")
}

func TestPromptProfileResolver_ResolveForFile_FallsBackForUnknownClassifierKey(t *testing.T) {
	resolver := NewPromptProfileResolver(fakePromptProfileLLM{
		payload: `{"prompt_profile_key":"unknown_profile","document_kind":"mystery_text","reason":"分类器给了未知 key"}`,
	})

	profile, resolved, err := resolver.ResolveForFile(
		context.Background(),
		"这是一段未知类型文本。",
		prompt.ScenarioModeEbookInterpretation,
	)
	require.NoError(t, err)
	require.Equal(t, prompt.PromptProfileClassicTextInterpretation, profile.Key)
	require.Equal(t, "mystery_text", resolved.DocumentKind)
	require.Contains(t, resolved.Reason, "未知 key")
}

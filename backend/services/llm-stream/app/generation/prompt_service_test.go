package generation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"inkwords-backend/shared/kernel/prompt"
	llm "inkwords-backend/shared/platform/llm"
)

func TestPromptRequirementsResolveFallsBackForInvalidScenario(t *testing.T) {
	svc := NewPromptRequirements()
	got, err := svc.Resolve(prompt.ScenarioMode("bad"), prompt.ArticleStyleGeneral)
	require.NoError(t, err)
	require.Contains(t, got, "电子书或长文本解读场景")
	require.NotContains(t, got, "bad")
}

func TestPromptRequirementsResolveUsesWorkspaceDefaults(t *testing.T) {
	got, err := NewPromptRequirements().Resolve(prompt.ScenarioModeBeginnerWalkthrough, prompt.ArticleStyleBeginnerTutorial)
	require.NoError(t, err)
	require.Contains(t, got, "零基础或初学者")
	require.Contains(t, got, prompt.DefaultStyleRequirements(prompt.ScenarioModeBeginnerWalkthrough, prompt.ArticleStyleBeginnerTutorial))
}

func TestPromptRequirementsResolveWithProfilePrependsProfileRequirements(t *testing.T) {
	svc := NewPromptRequirements()
	profile := prompt.ResolvePromptProfileKey("classic_text_interpretation", prompt.ScenarioModeEbookInterpretation)
	got, err := svc.ResolveWithProfile(prompt.ScenarioModeEbookInterpretation, prompt.ArticleStyleGeneral, profile)
	require.NoError(t, err)
	require.Contains(t, got, profile.GenerateRequirements)
	require.Equal(t, 0, strings.Index(got, profile.GenerateRequirements))
}

type fakePromptProfileLLM struct {
	payload string
	err     error
}

func (f fakePromptProfileLLM) GenerateJSON(context.Context, string, []llm.Message) (string, error) {
	return f.payload, f.err
}

func TestPromptProfileResolverClassifierFailureUsesFallback(t *testing.T) {
	resolver := NewPromptProfileResolver(fakePromptProfileLLM{err: errors.New("classifier unavailable")})
	profile, resolved, err := resolver.ResolveForFile(context.Background(), "待分类原文", prompt.ScenarioModeEbookInterpretation)
	require.NoError(t, err)
	require.Equal(t, prompt.PromptProfileClassicTextInterpretation, profile.Key)
	require.Contains(t, resolved.Reason, "回退")
}

func TestPromptProfileResolverValidJSONSelectsProfile(t *testing.T) {
	resolver := NewPromptProfileResolver(fakePromptProfileLLM{payload: `{"prompt_profile_key":"psychology_communication_book","document_kind":"psychology_communication","reason":"命中沟通主题"}`})
	profile, resolved, err := resolver.ResolveForFile(context.Background(), "感受、需要与沟通冲突", prompt.ScenarioModeEbookInterpretation)
	require.NoError(t, err)
	require.Equal(t, prompt.PromptProfilePsychologyCommunication, profile.Key)
	require.Equal(t, "psychology_communication", resolved.DocumentKind)
}

func TestPromptProfileResolverUnknownKeyFallsBackSafely(t *testing.T) {
	resolver := NewPromptProfileResolver(fakePromptProfileLLM{payload: `{"prompt_profile_key":"unknown","document_kind":"mystery","reason":"未知 key"}`})
	profile, resolved, err := resolver.ResolveForFile(context.Background(), "未知文本", prompt.ScenarioModeEbookInterpretation)
	require.NoError(t, err)
	require.Equal(t, prompt.PromptProfileClassicTextInterpretation, profile.Key)
	require.Equal(t, "mystery", resolved.DocumentKind)
}

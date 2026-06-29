package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	llm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/internal/prompt"
)

const promptProfileResolverModel = "deepseek-v4-flash"

type promptProfileLLM interface {
	GenerateJSON(ctx context.Context, model string, messages []llm.Message) (string, error)
}

type promptProfileOptionsLLM interface {
	GenerateJSONWithOptions(ctx context.Context, model string, messages []llm.Message, options llm.ChatOptions) (string, llm.CompletionUsage, error)
}

// PromptProfileResolver 负责把文件内容归类为稳定的 Prompt Profile。
type PromptProfileResolver struct {
	llmClient promptProfileLLM
}

// NewPromptProfileResolver 创建内容分类 resolver。
func NewPromptProfileResolver(llmClient promptProfileLLM) *PromptProfileResolver {
	return &PromptProfileResolver{llmClient: llmClient}
}

// ResolveForFile 根据文件内容解析出最终使用的 profile，并在失败时返回场景兜底。
func (r *PromptProfileResolver) ResolveForFile(
	ctx context.Context,
	sourceContent string,
	scenario prompt.ScenarioMode,
) (prompt.PromptProfile, prompt.ResolvedPromptProfile, error) {
	fallback := prompt.FallbackPromptProfileForScenario(scenario)
	if strings.TrimSpace(sourceContent) == "" || r.llmClient == nil {
		return fallback, newResolvedPromptProfile(
			fallback,
			fallback.DocumentKind,
			"文件内容为空或分类器不可用，已回退到默认提示词。",
		), nil
	}

	contentSnippet := truncatePromptProfileContent(sourceContent, 16000)
	messages := []llm.Message{
		{
			Role:    "system",
			Content: "你只负责识别文件内容类型。请返回严格 JSON，字段为 prompt_profile_key、document_kind、reason，不要输出任何额外文本。",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("scenario_mode=%s\n\n以下是文件内容：\n%s", scenario, contentSnippet),
		},
	}
	payload, err := r.generateClassifierJSON(ctx, messages)
	if err != nil {
		return fallback, newResolvedPromptProfile(
			fallback,
			fallback.DocumentKind,
			"内容分类失败，已回退到默认提示词。",
		), nil
	}

	var result struct {
		PromptProfileKey string `json:"prompt_profile_key"`
		DocumentKind     string `json:"document_kind"`
		Reason           string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		return fallback, newResolvedPromptProfile(
			fallback,
			fallback.DocumentKind,
			"分类结果无法解析，已回退到默认提示词。",
		), nil
	}

	resolvedProfile := prompt.ResolvePromptProfileKey(result.PromptProfileKey, scenario)
	return resolvedProfile, newResolvedPromptProfile(
		resolvedProfile,
		firstNonEmpty(result.DocumentKind, resolvedProfile.DocumentKind),
		firstNonEmpty(result.Reason, "已根据文件内容自动匹配提示词。"),
	), nil
}

func (r *PromptProfileResolver) generateClassifierJSON(ctx context.Context, messages []llm.Message) (string, error) {
	if client, ok := r.llmClient.(promptProfileOptionsLLM); ok {
		payload, _, err := client.GenerateJSONWithOptions(ctx, promptProfileResolverModel, messages, llm.LightweightChatOptions("", 512))
		return payload, err
	}
	return r.llmClient.GenerateJSON(ctx, promptProfileResolverModel, messages)
}

func newResolvedPromptProfile(profile prompt.PromptProfile, documentKind string, reason string) prompt.ResolvedPromptProfile {
	return prompt.ResolvedPromptProfile{
		Key:          profile.Key,
		DisplayName:  profile.DisplayName,
		DocumentKind: documentKind,
		Reason:       reason,
	}
}

func truncatePromptProfileContent(sourceContent string, maxRunes int) string {
	runes := []rune(sourceContent)
	if len(runes) <= maxRunes {
		return sourceContent
	}

	return string(runes[:maxRunes])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"inkwords-backend/shared/kernel/prompt"
	llm "inkwords-backend/shared/platform/llm"
)

// PromptRequirements 为生成链路组装最终 prompt 写作要求。
type PromptRequirements struct{}

// NewPromptRequirements 创建 PromptRequirements 实例。
func NewPromptRequirements() *PromptRequirements {
	return &PromptRequirements{}
}

// Resolve 统一合并场景层与风格层的工作区默认 Prompt 要求。
func (s *PromptRequirements) Resolve(
	scenario prompt.ScenarioMode,
	style prompt.ArticleStyle,
) (string, error) {
	if !scenario.IsValid() {
		scenario = prompt.ScenarioModeEbookInterpretation
	}
	if !style.IsValid() {
		style = prompt.ArticleStyleGeneral
	}

	return strings.TrimSpace(strings.Join([]string{
		prompt.DefaultScenarioRequirements(scenario),
		prompt.DefaultStyleRequirements(scenario, style),
	}, "\n\n")), nil
}

// ResolveWithProfile 在基础 requirements 前追加 profile 级写作要求。
func (s *PromptRequirements) ResolveWithProfile(
	scenario prompt.ScenarioMode,
	style prompt.ArticleStyle,
	profile prompt.PromptProfile,
) (string, error) {
	baseRequirements, err := s.Resolve(scenario, style)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.Join([]string{
		profile.GenerateRequirements,
		baseRequirements,
	}, "\n\n")), nil
}

const profileResolverModel = "deepseek-v4-flash"

type jsonGenerator interface {
	GenerateJSON(ctx context.Context, model string, messages []llm.Message) (string, error)
}

type jsonGeneratorWithOptions interface {
	GenerateJSONWithOptions(ctx context.Context, model string, messages []llm.Message, options llm.ChatOptions) (string, llm.CompletionUsage, error)
}

// PromptProfileResolver 负责把文件内容归类为稳定的 Prompt Profile。
type PromptProfileResolver struct {
	llmClient jsonGenerator
}

// NewPromptProfileResolver 创建内容分类 resolver。
func NewPromptProfileResolver(llmClient jsonGenerator) *PromptProfileResolver {
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
		return fallback, resolvedPromptProfile{
			Key:          fallback.Key,
			DisplayName:  fallback.DisplayName,
			DocumentKind: fallback.DocumentKind,
			Reason:       "文件内容为空或分类器不可用，已回退到默认提示词。",
		}, nil
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
		//nolint:nilerr
		return fallback, resolvedPromptProfile{
			Key:          fallback.Key,
			DisplayName:  fallback.DisplayName,
			DocumentKind: fallback.DocumentKind,
			Reason:       "内容分类失败，已回退到默认提示词。",
		}, nil
	}

	var result struct {
		PromptProfileKey string `json:"prompt_profile_key"`
		DocumentKind     string `json:"document_kind"`
		Reason           string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		//nolint:nilerr
		return fallback, resolvedPromptProfile{
			Key:          fallback.Key,
			DisplayName:  fallback.DisplayName,
			DocumentKind: fallback.DocumentKind,
			Reason:       "分类结果无法解析，已回退到默认提示词。",
		}, nil
	}

	resolvedProfile := prompt.ResolvePromptProfileKey(result.PromptProfileKey, scenario)
	return resolvedProfile, resolvedPromptProfile{
		Key:          resolvedProfile.Key,
		DisplayName:  resolvedProfile.DisplayName,
		DocumentKind: firstNonEmpty(result.DocumentKind, resolvedProfile.DocumentKind),
		Reason:       firstNonEmpty(result.Reason, "已根据文件内容自动匹配提示词。"),
	}, nil
}

func (r *PromptProfileResolver) generateClassifierJSON(ctx context.Context, messages []llm.Message) (string, error) {
	if client, ok := r.llmClient.(jsonGeneratorWithOptions); ok {
		payload, _, err := client.GenerateJSONWithOptions(ctx, profileResolverModel, messages, llm.LightweightChatOptions(512))
		return payload, err
	}
	return r.llmClient.GenerateJSON(ctx, profileResolverModel, messages)
}

type resolvedPromptProfile = prompt.ResolvedPromptProfile

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

// normalizePromptProfile 回退空 profile 到场景兜底 profile。
func normalizePromptProfile(profile prompt.PromptProfile, scenarioMode prompt.ScenarioMode) prompt.PromptProfile {
	if !scenarioMode.IsValid() {
		scenarioMode = prompt.ScenarioModeEbookInterpretation
	}
	if profile.Key == "" {
		return prompt.FallbackPromptProfileForScenario(scenarioMode)
	}
	return prompt.ResolvePromptProfileKey(string(profile.Key), scenarioMode)
}

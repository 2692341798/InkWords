package review

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	platformllm "inkwords-backend/shared/platform/llm"
)

// AIFeedbackGenerator 定义 review 领域所需的 AI 反馈能力。
type AIFeedbackGenerator interface {
	Generate(ctx context.Context, input AIFeedbackInput) (AIFeedbackResult, error)
}

// AIFeedbackInput 描述一次 AI 反馈生成的最小上下文。
type AIFeedbackInput struct {
	Title         string
	Mode          string
	SourcePreview string
	MainQuestion  string
	CurrentGoal   string
	CurrentAnswer string
	RecentTurns   []ReviewTurnResponse
}

// AIFeedbackResult 描述 AI 返回的结构化复习反馈。
type AIFeedbackResult struct {
	Judgement       string   `json:"judgement"`
	HitPoints       []string `json:"hit_points"`
	MissedPoints    []string `json:"missed_points"`
	Suggestion      string   `json:"suggestion"`
	StageFeedback   string   `json:"stage_feedback"`
	NextQuestion    string   `json:"next_question"`
	HintText        string   `json:"hint_text"`
	ExcerptText     string   `json:"excerpt_text"`
	ShouldShowQuote bool     `json:"should_show_quote"`
}

// DeepSeekAIFeedbackGenerator 使用 DeepSeek 生成结构化复习反馈。
type DeepSeekAIFeedbackGenerator struct {
	client interface {
		GenerateJSON(context.Context, string, []platformllm.Message) (string, error)
	}
	model string
}

// NewDeepSeekAIFeedbackGenerator 创建 DeepSeek 反馈生成器。
func NewDeepSeekAIFeedbackGenerator(client interface {
	GenerateJSON(context.Context, string, []platformllm.Message) (string, error)
}, model string) *DeepSeekAIFeedbackGenerator {
	return &DeepSeekAIFeedbackGenerator{
		client: client,
		model:  strings.TrimSpace(model),
	}
}

// Generate 调用 DeepSeek 生成结构化复习反馈。
func (g *DeepSeekAIFeedbackGenerator) Generate(ctx context.Context, input AIFeedbackInput) (AIFeedbackResult, error) {
	if g == nil || g.client == nil || strings.TrimSpace(g.model) == "" {
		return AIFeedbackResult{}, fmt.Errorf("ai feedback generator is not configured")
	}

	systemPrompt := strings.TrimSpace(`
你是一个中文知识复习助教。你的任务是基于原文、用户当前回答和最近会话上下文，生成严格的 JSON。
要求：
1. 所有输出必须是中文。
2. judgement 只能是：答对较多、部分答对、偏题、需要提醒。
3. 当用户明确表达“不记得/忘了/记不清”等语义时，优先返回 hint_text；如果还需要更直接帮助，再返回 excerpt_text。
4. excerpt_text 只能摘录提供的原文，不允许编造。
5. 如果信息不足，不要编造，允许字段为空字符串或空数组。`)

	payload, _ := json.Marshal(input)
	userPrompt := "请根据以下 JSON 输入输出结果 JSON，不要附加解释：\n" + string(payload)

	content, err := g.client.GenerateJSON(ctx, g.model, []platformllm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return AIFeedbackResult{}, err
	}

	var result AIFeedbackResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return AIFeedbackResult{}, fmt.Errorf("unmarshal ai feedback result: %w", err)
	}
	return result, nil
}

func indicatesMemoryGap(answer string) bool {
	normalized := strings.TrimSpace(strings.ToLower(answer))
	if normalized == "" {
		return false
	}
	keywords := []string{
		"不记得", "记不清", "忘了", "忘记了", "想不起来", "没印象", "记不得",
	}
	for _, keyword := range keywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}
	return false
}

func buildMemoryGapHint(outline SessionOutline) string {
	if strings.TrimSpace(outline.MainQuestion) != "" {
		return "先别急着回忆细节，先想想这条主问题：" + outline.MainQuestion
	}
	if len(outline.Checkpoints) > 0 {
		return "先抓住一个关键点再继续回忆：" + outline.Checkpoints[0]
	}
	return "先回忆这篇文章主要在解决什么问题，再慢慢补细节。"
}

func buildMemoryGapExcerpt(sourcePreview string, outline SessionOutline) string {
	trimmedPreview := strings.TrimSpace(sourcePreview)
	if trimmedPreview != "" {
		return "相关原文摘录：" + truncateRunes(trimmedPreview, 120)
	}
	if len(outline.Checkpoints) > 0 {
		return "相关原文摘录：" + outline.Checkpoints[0]
	}
	return ""
}

func buildAIFeedbackInput(sessionTitle string, sessionMode string, metadata sessionMetadata, turns []ReviewTurnResponse, currentGoal string, answer string) AIFeedbackInput {
	recentTurns := turns
	if len(recentTurns) > 6 {
		recentTurns = recentTurns[len(recentTurns)-6:]
	}

	return AIFeedbackInput{
		Title:         sessionTitle,
		Mode:          sessionMode,
		SourcePreview: metadata.SourcePreview,
		MainQuestion:  metadata.SessionOutline.MainQuestion,
		CurrentGoal:   currentGoal,
		CurrentAnswer: answer,
		RecentTurns:   recentTurns,
	}
}

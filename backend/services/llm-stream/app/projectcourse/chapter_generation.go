package projectcourse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
	"inkwords-backend/shared/platform/llm"
)

type ChapterGenerator interface {
	Generate(ctx context.Context, chapter sharedkernel.Chapter, pack EvidencePack, audience sharedkernel.AudienceLevel) (ChapterDocument, error)
}

type JSONChapterGenerator struct {
	Client *llm.DeepSeekClient
	Model  string
}

type generatedChapterResponse struct {
	Markdown string                    `json:"markdown"`
	Claims   []sharedkernel.Claim      `json:"claims"`
	Lab      *sharedkernel.LabManifest `json:"lab,omitempty"`
}

func (g JSONChapterGenerator) Generate(ctx context.Context, chapter sharedkernel.Chapter, pack EvidencePack, audience sharedkernel.AudienceLevel) (ChapterDocument, error) {
	if g.Client == nil {
		return ChapterDocument{}, fmt.Errorf("chapter generator client is not configured")
	}
	model := strings.TrimSpace(g.Model)
	if model == "" {
		model = "deepseek-v4-flash"
	}
	required, _, _, ok := chapterContractFor(chapter.Type)
	if !ok {
		return ChapterDocument{}, fmt.Errorf("chapter type %q has no contract", chapter.Type)
	}
	content, err := json.Marshal(struct {
		ChapterID        string       `json:"chapter_id"`
		Title            string       `json:"title"`
		Type             string       `json:"chapter_type"`
		Audience         string       `json:"audience_level"`
		RequiredSections []string     `json:"required_sections"`
		Evidence         EvidencePack `json:"evidence_pack"`
	}{chapter.ID, chapter.Title, string(chapter.Type), string(audience), required, pack})
	if err != nil {
		return ChapterDocument{}, err
	}
	messages := []llm.Message{
		{Role: "system", Content: "你是项目精通课程作者。只能根据 evidence_pack 表达项目事实；输出 JSON，不要输出 Markdown 代码围栏。claims 必须使用 evidence_pack 中已有 evidence_id，status 必须是 verified。若必须输出代码围栏，语言标记必须是 source:<evidence_id> 或 artifact:<path>，且代码必须与对应证据或实验工件逐字一致。"},
		{Role: "user", Content: "根据以下章节合同和证据生成章节：\n" + string(content)},
	}
	raw, _, err := g.Client.GenerateJSONWithOptions(ctx, model, messages, llm.LightweightChatOptions("", 2500))
	if err != nil {
		return ChapterDocument{}, err
	}
	var response generatedChapterResponse
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		return ChapterDocument{}, fmt.Errorf("decode generated chapter: %w", err)
	}
	document, _, err := BuildChapterDocument(chapter, pack, ClaimPlan{Claims: response.Claims, EvidenceIDs: chapter.EvidenceIDs}, response.Markdown, response.Lab, false)
	return document, err
}

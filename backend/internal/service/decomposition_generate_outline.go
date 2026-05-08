package service

import (
	"context"
	"encoding/json"
	"fmt"
	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/model"
	"os"
	"strings"
)

func (s *DecompositionService) GenerateOutline(ctx context.Context, sourceContent string, existingParent *model.Blog, existingChildren []model.Blog) (*OutlineResult, error) {
	runes := []rune(sourceContent)
	if len(runes) > 15000000 {
		sourceContent = string(runes[:15000000]) + "\n\n... [Content Truncated due to length limits] ..."
	}

	instruction := `你是一个高级架构师。请评估前面提供的项目文本，并生成一个系列博客的大纲。
对于大型项目、源码仓库或复杂内容，必须拆分为系列博客。
我的核心要求是：对于每个核心模块、业务逻辑或重要架构层，**都必须对应至少一篇博客进行详细说明**。不要担心生成的篇数过多！

请根据提供的项目文件数量和内容复杂度，充分且详细地规划章节数量：
- 对于普通项目，不要吝啬章节数，确保每一个独立的功能点、数据流环节、配置模块等都有专属的文章解析。
- 对于特别庞大的框架源码（如 FFmpeg 等），请大胆拆分出数十篇详细章节，做到对核心源码文件的全面覆盖。
- 每一篇博客只聚焦于**一个具体的核心技术点或模块**，并详细说明其原理与实现。`

	if existingParent != nil {
		var existingOutlineBuilder strings.Builder
		existingOutlineBuilder.WriteString(fmt.Sprintf("原系列标题: %s\n原章节列表:\n", existingParent.Title))
		for _, child := range existingChildren {
			existingOutlineBuilder.WriteString(fmt.Sprintf("- 章节ID: %s, 序号: %d, 标题: %s\n", child.ID.String(), child.ChapterSort, child.Title))
		}

		instruction += fmt.Sprintf(`

**注意：用户已经拥有该项目旧版本的博客系列大纲如下**：
%s

由于项目可能发生了更新，你需要对比最新源码与已有博客大纲，生成一个**更新后**的系列博客大纲。
- 对于内容没有发生变化的章节，你可以保留原样，并将 "action" 标记为 "skip"（跳过生成），并**务必填入对应的 "id"**。
- 对于需要根据最新代码更新的旧章节，将 "action" 标记为 "regenerate"（重新生成），并**务必填入对应的 "id"**。
- 对于根据新功能/新模块产生的新章节，将 "action" 标记为 "new"（新增），不要填 "id"。
- 对于已经废弃的功能对应的章节，直接在新的大纲中将其移除即可。
`, existingOutlineBuilder.String())
	}

	instruction += `
输出必须是纯JSON格式，包含 series_title 和 chapters 两个字段，不包含任何Markdown标记或其他文字。
JSON 格式如下：
{
  "series_title": "系列博客的标题（必须根据项目内容精准概括，例如：React 源码解析系列、Vite 配置实战等，绝不要使用通用或宽泛的占位名称）",
  "chapters": [
    {
      "id": "章节的 UUID（如果有）",
      "title": "章节标题",
      "summary": "该章节的详细摘要或内容要点（指导后续生成的具体方向）",
      "sort": 1,
      "files": ["强相关的具体文件路径或目录（必须是相对路径）"],
      "action": "new 或 regenerate 或 skip"
    }
  ]
}`

	messages := []llm.Message{
		{Role: "system", Content: "项目文本内容如下：\n" + sourceContent},
		{Role: "user", Content: instruction},
	}

	modelName := "deepseek-v4-flash"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		modelName = envModel
	}

	content, err := s.llmClient.GenerateJSON(ctx, modelName, messages)
	if err != nil {
		return nil, fmt.Errorf("llm generation failed: %w", err)
	}

	content = strings.TrimPrefix(strings.TrimSpace(content), "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var outline OutlineResult
	if err := json.Unmarshal([]byte(content), &outline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal llm output: %w, output: %s", err, content)
	}

	return &outline, nil
}

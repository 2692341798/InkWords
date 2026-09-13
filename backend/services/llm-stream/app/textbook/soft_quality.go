package textbook

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

var sampleManualReviewDimensions = sharedtextbook.SampleManualReviewDimensions()

// SoftQualityAdvisory points a human reviewer to a bounded risk. It is not a
// failure, score, or claim that the manuscript is good or bad.
type SoftQualityAdvisory struct {
	Dimension      string `json:"dimension"`
	Code           string `json:"code"`
	Message        string `json:"message"`
	Evidence       string `json:"evidence"`
	ReviewQuestion string `json:"review_question"`
}

var (
	markdownFence       = regexp.MustCompile("(?s)```.*?```")
	markdownPunctuation = regexp.MustCompile(`[\p{P}\p{S}\s]+`)
)

func manualReviewDimensions() []string {
	return sharedtextbook.SampleManualReviewDimensions()
}

// AssessSampleChapterSoftQuality produces deterministic prompts for human
// review. Findings deliberately do not affect the hard-gate Passed flag.
func AssessSampleChapterSoftQuality(markdown string) []SoftQualityAdvisory {
	plain := markdownFence.ReplaceAllString(markdown, "")
	advisories := make([]SoftQualityAdvisory, 0)

	paragraphs := proseParagraphs(plain)
	for _, paragraph := range paragraphs {
		if utf8.RuneCountInString(paragraph) > 280 {
			advisories = append(advisories, advisory(
				"句子与段落负担", "long_paragraph",
				"该段较长，零基础读者可能难以定位主句与因果链。",
				paragraph,
				"能否在不丢失因果关系的前提下拆段，并为每段保留一个中心问题？",
			))
			break
		}
	}
	for _, paragraph := range paragraphs {
		for _, sentence := range strings.FieldsFunc(paragraph, func(r rune) bool {
			return r == '。' || r == '！' || r == '？' || r == ';' || r == '；'
		}) {
			if utf8.RuneCountInString(strings.TrimSpace(sentence)) > 120 {
				advisories = append(advisories, advisory(
					"句子与段落负担", "long_sentence",
					"检测到一个包含较多信息的长句。",
					sentence,
					"这个句子是否混合了前提、机制、结果或例外，值得拆成可逐步验证的句子？",
				))
				break
			}
		}
		if containsAdvisoryCode(advisories, "long_sentence") {
			break
		}
	}

	for _, section := range markdownSections(plain) {
		if containsAny(section.heading, "为什么", "如何", "机制", "完整", "从零", "讲清") && utf8.RuneCountInString(strings.TrimSpace(section.body)) < 120 {
			advisories = append(advisories, advisory(
				"标题承诺", "thin_heading_promise",
				"标题承诺了解释或完整步骤，但本节正文较短。",
				section.heading,
				"本节是否真的回答了标题中的问题，并提供了可观察的证明或明确边界？",
			))
			break
		}
	}

	seenParagraphs := make(map[string]string)
	for _, paragraph := range paragraphs {
		normalized := normalizeProse(paragraph)
		if utf8.RuneCountInString(normalized) < 32 {
			continue
		}
		if first, exists := seenParagraphs[normalized]; exists {
			advisories = append(advisories, advisory(
				"重复", "repeated_paragraph",
				"检测到内容相同的段落，可能让章节显得原地重复。",
				first,
				"第二次出现是否承担了新的比较、证据或迁移任务；若没有，能否合并？",
			))
			break
		}
		seenParagraphs[normalized] = paragraph
	}

	for _, phrase := range []string{"你只需要", "不用担心", "轻松", "当然", "任何人都能", "一看就会"} {
		if strings.Contains(plain, phrase) {
			advisories = append(advisories, advisory(
				"语气", "over_assuring_tone",
				"这类保证性表达可能遮住真实前提或失败路径。",
				phrase,
				"能否改为具体前提、可观察结果和失败恢复动作，而不是替读者判断难度？",
			))
			break
		}
	}

	workedExample := bodyForHeading(markdown, "### 完整示范")
	if workedExample != "" && (!strings.Contains(workedExample, "```") || !containsAny(workedExample, "对应", "因为", "这一步", "因此", "可以看到")) {
		advisories = append(advisories, advisory(
			"例子相关性", "worked_example_link_weak",
			"完整示范没有同时呈现代码/工件及其与本章机制的显式对应关系。",
			"### 完整示范",
			"示例中的每个关键动作能否回指本章问题、机制和证据，而不是只展示最终用法？",
		))
	}

	sections := markdownSections(plain)
	if len(sections) >= 3 {
		lengths := make([]int, 0, len(sections))
		for _, section := range sections {
			lengths = append(lengths, utf8.RuneCountInString(strings.TrimSpace(section.body)))
		}
		sorted := append([]int(nil), lengths...)
		sort.Ints(sorted)
		if sorted[0] < 80 && sorted[len(sorted)-1] > 640 {
			advisories = append(advisories, advisory(
				"章节节奏", "section_rhythm_imbalance",
				"章节同时存在很短和很长的主节，读者节奏可能突然变化。",
				"最短主节与最长主节的正文长度差异较大",
				"短节是否缺少必要推理，长节是否应该加入小结、观察点或分段练习？",
			))
		}
	}

	mechanism := bodyForHeading(markdown, "## 跟着源码走三步")
	if mechanism == "" {
		mechanism = bodyForHeadingContaining(markdown, "机制")
	}
	if mechanism != "" && countOccurrences(mechanism, []string{"先", "然后", "最终", "进入", "调用", "返回", "查找"}) >= 5 && !containsAny(mechanism, "```mermaid", "![", "→") {
		advisories = append(advisories, advisory(
			"图示机会", "flow_visual_opportunity",
			"本节包含多步调用或数据流，但没有配套流程图。",
			"连续出现多个流程词：先、调用、进入、查找或返回",
			"一张标明观察对象与边界的调用链图，是否能降低读者在组件之间来回定位的负担？",
		))
	}

	guided := bodyForHeading(markdown, "### 引导练习")
	faded := bodyForHeading(markdown, "### 脚手架渐退")
	independent := bodyForHeading(markdown, "### 独立迁移")
	if guided != "" && faded != "" && independent != "" {
		if strings.Contains(independent, "提示：") || strings.Contains(independent, "提示:") || normalizeProse(guided) == normalizeProse(independent) {
			advisories = append(advisories, advisory(
				"练习梯度", "exercise_scaffolding_not_faded",
				"独立迁移仍保留显式提示，或与引导练习内容相同。",
				"### 独立迁移",
				"独立任务是否真正撤掉步骤提示，并改变约束以验证迁移而不是照抄？",
			))
		}
	}

	return advisories
}

type markdownSection struct {
	heading string
	body    string
}

func markdownSections(markdown string) []markdownSection {
	lines := strings.Split(markdown, "\n")
	sections := make([]markdownSection, 0)
	for index, line := range lines {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		end := len(lines)
		for next := index + 1; next < len(lines); next++ {
			if strings.HasPrefix(lines[next], "## ") {
				end = next
				break
			}
		}
		sections = append(sections, markdownSection{heading: strings.TrimSpace(line), body: strings.Join(lines[index+1:end], "\n")})
	}
	return sections
}

func proseParagraphs(markdown string) []string {
	blocks := strings.Split(markdown, "\n\n")
	paragraphs := make([]string, 0, len(blocks))
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" || strings.HasPrefix(block, "#") || strings.HasPrefix(block, "-") || strings.HasPrefix(block, "*") || strings.HasPrefix(block, ">") {
			continue
		}
		paragraphs = append(paragraphs, strings.Join(strings.Fields(block), " "))
	}
	return paragraphs
}

func normalizeProse(value string) string {
	return strings.ToLower(markdownPunctuation.ReplaceAllString(strings.TrimSpace(value), ""))
}

func advisory(dimension, code, message, evidence, question string) SoftQualityAdvisory {
	return SoftQualityAdvisory{Dimension: dimension, Code: code, Message: message, Evidence: excerpt(evidence, 120), ReviewQuestion: question}
}

func excerpt(value string, maximum int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	return string(runes[:maximum]) + "…"
}

func containsAdvisoryCode(advisories []SoftQualityAdvisory, code string) bool {
	for _, item := range advisories {
		if item.Code == code {
			return true
		}
	}
	return false
}

func bodyForHeading(markdown, heading string) string {
	index := strings.Index(markdown, heading)
	if index < 0 {
		return ""
	}
	return sectionBody(markdown, index+len(heading))
}

func bodyForHeadingContaining(markdown, needle string) string {
	for _, section := range markdownSections(markdown) {
		if strings.Contains(section.heading, needle) {
			return section.body
		}
	}
	return ""
}

func countOccurrences(value string, terms []string) int {
	total := 0
	for _, term := range terms {
		total += strings.Count(value, term)
	}
	return total
}

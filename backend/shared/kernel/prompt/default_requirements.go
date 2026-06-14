package prompt

func DefaultRequirements(style ArticleStyle) string {
	switch style {
	case ArticleStyleBeginnerTutorial:
		return `你将面向完全零基础读者输出手把手教学文章。要求：
1. 用生活化语言解释概念，但必须配合可运行的示例代码
2. 每一步都给出明确操作步骤与预期结果
3. 对关键代码进行逐段解释，指出常见踩坑与排查思路
4. 文章结构使用 H1-H4，输出必须为中文`
	case ArticleStyleExamReview:
		return `你将面向“开卷考试/备考复习”输出复习文章。要求：
1. 更强调操作性与考点清单，少解释底层原理
2. 对讲义内容做条理化重组：知识点 -> 操作步骤 -> 常见题型/易错点
3. 每个知识点尽量给出可直接照抄的步骤或命令
4. 文章结构使用 H1-H4，输出必须为中文`
	default:
		return `你将输出一篇“小白友好、图文并茂、可独立复现”的高质量技术博客。要求：
1. 单点聚焦：只讲一个核心技术点
2. 多示例代码：源码/伪代码/最佳实践用例
3. 可复现步骤：操作步骤清晰可执行
4. 抽象概念必须给代码示例或生活化比喻
5. 文章结构使用 H1-H4，输出必须为中文`
	}
}

// DefaultStyleRequirements 返回结合场景后的风格层默认 Prompt 约束。
func DefaultStyleRequirements(mode ScenarioMode, style ArticleStyle) string {
	// Why: `general` 是旧客户端最常见的默认值，但在电子书解读场景下继续沿用
	// “高质量技术博客/可独立复现”会把任务目标带偏成教程。这里仅对该组合做最小兜底。
	if mode == ScenarioModeEbookInterpretation && style == ArticleStyleGeneral {
		return `你将输出一篇面向普通读者的中文经典文本逐章解读文章。要求：
1. 按原文篇章结构逐章展开，做好概念拆解、上下文交代和历史背景说明
2. 以观点解释为主，配合代表性原文摘录，帮助读者理解原典精义
3. 如果原文概念抽象，优先用白话解释，再按需要补充少量例子
4. 文章结构使用 H1-H4，输出必须为中文`
	}

	return DefaultRequirements(style)
}

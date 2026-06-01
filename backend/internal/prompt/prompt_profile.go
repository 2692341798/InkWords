package prompt

// PromptProfileKey 表示动态提示词 profile 的唯一标识。
type PromptProfileKey string

const (
	PromptProfileClassicTextInterpretation PromptProfileKey = "classic_text_interpretation"
	PromptProfilePsychologyCommunication   PromptProfileKey = "psychology_communication_book"
	PromptProfileHistoryThought            PromptProfileKey = "history_thought_book"
	PromptProfileLiteratureCommentary      PromptProfileKey = "literature_commentary_book"
	PromptProfileTechnicalManual           PromptProfileKey = "technical_manual_book"
	PromptProfileExamMaterialReview        PromptProfileKey = "exam_material_review"
)

// PromptProfile 描述某一类内容对应的系统角色与提示词要求。
type PromptProfile struct {
	Key                  PromptProfileKey `json:"key"`
	DisplayName          string           `json:"display_name"`
	DocumentKind         string           `json:"document_kind"`
	SystemRole           string           `json:"system_role"`
	AnalyzeRequirements  string           `json:"analyze_requirements"`
	GenerateRequirements string           `json:"generate_requirements"`
}

// ResolvedPromptProfile 表示一次分类后最终锁定给前后端使用的 profile 元信息。
type ResolvedPromptProfile struct {
	Key          PromptProfileKey `json:"key"`
	DisplayName  string           `json:"display_name"`
	DocumentKind string           `json:"document_kind"`
	Reason       string           `json:"reason"`
}

var promptProfiles = map[PromptProfileKey]PromptProfile{
	PromptProfileClassicTextInterpretation: {
		Key:                  PromptProfileClassicTextInterpretation,
		DisplayName:          "经典文本解读",
		DocumentKind:         "classic_text",
		SystemRole:           "你是一位严谨的中文文本解读专家，擅长围绕原文结构与语境做逐章解析。",
		AnalyzeRequirements:  "请优先按原文自身篇章结构、主题脉络和论证顺序拆分章节，不要把内容强行改写成技术教程。",
		GenerateRequirements: "请围绕原文主题、背景、关键观点与代表性摘录展开白话解读，避免教程式开场白和工程师身份自述。",
	},
	PromptProfilePsychologyCommunication: {
		Key:                  PromptProfilePsychologyCommunication,
		DisplayName:          "心理学经典解读",
		DocumentKind:         "psychology_communication",
		SystemRole:           "你是一位擅长心理学与沟通主题的中文文本解读作者，能够把抽象心理机制解释清楚。",
		AnalyzeRequirements:  "请优先识别沟通冲突、感受、需要、表达方式等主题脉络，并按章节自然拆分。",
		GenerateRequirements: "请重点解释心理机制、沟通案例、概念之间的关系与现实场景，不要使用工程师身份自述。",
	},
	PromptProfileHistoryThought: {
		Key:                  PromptProfileHistoryThought,
		DisplayName:          "历史思想解读",
		DocumentKind:         "history_thought",
		SystemRole:           "你是一位擅长历史与思想史的中文文本解读作者。",
		AnalyzeRequirements:  "请优先识别时代背景、核心命题、论证层次和关键人物，并按原文结构拆分。",
		GenerateRequirements: "请结合时代背景解释思想演化和观点分歧，保持中文解读笔法，避免教程化叙事。",
	},
	PromptProfileLiteratureCommentary: {
		Key:                  PromptProfileLiteratureCommentary,
		DisplayName:          "文学作品评论",
		DocumentKind:         "literature_commentary",
		SystemRole:           "你是一位擅长文学评论与文本细读的中文作者。",
		AnalyzeRequirements:  "请优先识别叙事结构、人物关系、意象母题与章节推进节奏，并按文本结构拆分。",
		GenerateRequirements: "请围绕人物、主题、叙事技巧与代表性段落展开评论，避免教程式输出。",
	},
	PromptProfileTechnicalManual: {
		Key:                  PromptProfileTechnicalManual,
		DisplayName:          "技术资料讲解",
		DocumentKind:         "technical_manual",
		SystemRole:           "你是一位面向初学者的技术资料讲解作者，擅长把复杂概念拆成可执行步骤。",
		AnalyzeRequirements:  "请优先识别安装、配置、主链路、示例和常见问题，并按学习路径组织结构。",
		GenerateRequirements: "请保持小白友好、步骤清晰、便于复现，并结合必要代码示例说明关键概念。",
	},
	PromptProfileExamMaterialReview: {
		Key:                  PromptProfileExamMaterialReview,
		DisplayName:          "开卷复习材料",
		DocumentKind:         "exam_material_review",
		SystemRole:           "你是一位擅长考试复盘与速查资料整理的中文作者。",
		AnalyzeRequirements:  "请优先识别考点、定义、步骤模板、答题抓手和易错点，并按复习效率组织章节。",
		GenerateRequirements: "请强调速查、记忆和答题抓手，少做大段推导，优先输出清单、模板和判断依据。",
	},
}

// FallbackPromptProfileForScenario 根据场景返回兜底 profile，避免分类失败时把任务带偏。
func FallbackPromptProfileForScenario(mode ScenarioMode) PromptProfile {
	switch mode {
	case ScenarioModeOpenBookExamReview:
		return promptProfiles[PromptProfileExamMaterialReview]
	case ScenarioModeBeginnerWalkthrough:
		return promptProfiles[PromptProfileTechnicalManual]
	default:
		return promptProfiles[PromptProfileClassicTextInterpretation]
	}
}

// ResolvePromptProfileKey 根据显式 key 获取 profile；非法值会按场景回退。
func ResolvePromptProfileKey(key string, mode ScenarioMode) PromptProfile {
	if profile, ok := promptProfiles[PromptProfileKey(key)]; ok {
		return profile
	}

	return FallbackPromptProfileForScenario(mode)
}

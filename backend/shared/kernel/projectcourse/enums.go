package projectcourse

import "fmt"

// ScenarioMode 是项目精通课程使用的独立生成场景。
type ScenarioMode string

const ScenarioModeProjectMasteryCourse ScenarioMode = "project_mastery_course"

// AudienceLevel 描述课程面向的读者基础。
type AudienceLevel string

const (
	AudienceFoundation    AudienceLevel = "foundation"
	AudienceProgramming   AudienceLevel = "programming"
	AudienceStackFamiliar AudienceLevel = "stack_familiar"
)

// ChapterType 是蓝图支持的章节类型。
type ChapterType string

const (
	ChapterProjectMap      ChapterType = "project_map"
	ChapterTechnicalTheory ChapterType = "technical_theory"
	ChapterMainFlow        ChapterType = "main_flow"
	ChapterModuleDeepDive  ChapterType = "module_deep_dive"
	ChapterDesignTradeoff  ChapterType = "design_tradeoff"
	ChapterHandsOnLab      ChapterType = "hands_on_lab"
	ChapterTroubleshooting ChapterType = "troubleshooting"
	ChapterChallenge       ChapterType = "challenge"
)

// EvidenceConfidence 区分来源明确程度，避免把推断写成项目事实。
type EvidenceConfidence string

const (
	ConfidenceDocumented EvidenceConfidence = "documented"
	ConfidenceObserved   EvidenceConfidence = "observed"
	ConfidenceInferred   EvidenceConfidence = "inferred"
)

// ClaimStatus 是事实论断经过独立校验后的状态。
type ClaimStatus string

const (
	ClaimVerified     ClaimStatus = "verified"
	ClaimUnsupported  ClaimStatus = "unsupported"
	ClaimContradicted ClaimStatus = "contradicted"
)

// CourseStatus 表示课程生成生命周期。
type CourseStatus string

const (
	CourseDraft            CourseStatus = "draft"
	CourseAnalyzing        CourseStatus = "analyzing"
	CourseAwaitingApproval CourseStatus = "awaiting_approval"
	CourseApproved         CourseStatus = "approved"
	CourseGenerating       CourseStatus = "generating"
	CourseCompleted        CourseStatus = "completed"
	CoursePartiallyBlocked CourseStatus = "partially_blocked"
	CourseBlocked          CourseStatus = "blocked"
	CourseFailed           CourseStatus = "failed"
)

// GateResult 表示质量门禁结果。
type GateResult string

const (
	GatePass     GateResult = "pass"
	GateHardFail GateResult = "hard_fail"
	GateSoftFail GateResult = "soft_fail"
)

func validateEnum[T ~string](value T, allowed []T, name string) error {
	for _, item := range allowed {
		if value == item {
			return nil
		}
	}
	return fmt.Errorf("unknown %s %q", name, value)
}

func (a AudienceLevel) Validate() error {
	return validateEnum(a, []AudienceLevel{AudienceFoundation, AudienceProgramming, AudienceStackFamiliar}, "audience level")
}

func (t ChapterType) Validate() error {
	return validateEnum(t, []ChapterType{ChapterProjectMap, ChapterTechnicalTheory, ChapterMainFlow, ChapterModuleDeepDive, ChapterDesignTradeoff, ChapterHandsOnLab, ChapterTroubleshooting, ChapterChallenge}, "chapter type")
}

func (c EvidenceConfidence) Validate() error {
	return validateEnum(c, []EvidenceConfidence{ConfidenceDocumented, ConfidenceObserved, ConfidenceInferred}, "evidence confidence")
}

func (s ClaimStatus) Validate() error {
	return validateEnum(s, []ClaimStatus{ClaimVerified, ClaimUnsupported, ClaimContradicted}, "claim status")
}

func (s CourseStatus) Validate() error {
	return validateEnum(s, []CourseStatus{CourseDraft, CourseAnalyzing, CourseAwaitingApproval, CourseApproved, CourseGenerating, CourseCompleted, CoursePartiallyBlocked, CourseBlocked, CourseFailed}, "course status")
}

func (r GateResult) Validate() error {
	return validateEnum(r, []GateResult{GatePass, GateHardFail, GateSoftFail}, "gate result")
}

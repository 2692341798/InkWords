package prompt

// ScenarioMode 表示生成任务的业务场景。
type ScenarioMode string

const (
	ScenarioModeEbookInterpretation ScenarioMode = "ebook_interpretation"
	ScenarioModeOpenBookExamReview  ScenarioMode = "open_book_exam_review"
	ScenarioModeBeginnerWalkthrough ScenarioMode = "beginner_walkthrough"
)

// IsValid 返回当前场景是否为受支持的枚举值。
func (m ScenarioMode) IsValid() bool {
	switch m {
	case ScenarioModeEbookInterpretation, ScenarioModeOpenBookExamReview, ScenarioModeBeginnerWalkthrough:
		return true
	default:
		return false
	}
}

// DefaultScenarioModeForSource 根据来源类型给出兼容默认场景。
func DefaultScenarioModeForSource(sourceType string) ScenarioMode {
	switch sourceType {
	case "git":
		return ScenarioModeBeginnerWalkthrough
	default:
		return ScenarioModeEbookInterpretation
	}
}

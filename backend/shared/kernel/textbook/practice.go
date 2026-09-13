package textbook

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// PracticeSetVersion identifies the manuscript-owned six-mode exercise contract.
const PracticeSetVersion = "inkwords.practice-set.v1"

// PracticeCriterion describes an explicit scoring requirement for one task.
type PracticeCriterion struct {
	ID              string `json:"id"`
	Description     string `json:"description"`
	RequiresRuntime bool   `json:"requires_runtime"`
}

// PracticeHint is one explicitly requested level of help, from weak to strong.
type PracticeHint struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

// PracticeTask belongs to the manuscript revision; learning sessions only
// project it. ExpectedAnswer is an authored answer key, not execution evidence.
type PracticeTask struct {
	ID             string              `json:"id"`
	Mode           LearningTaskMode    `json:"mode"`
	Prompt         string              `json:"prompt"`
	Variation      string              `json:"variation"`
	ExpectedAnswer string              `json:"expected_answer"`
	Rubric         []PracticeCriterion `json:"rubric"`
	Hints          []PracticeHint      `json:"hints"`
	EvidenceIDs    []string            `json:"evidence_ids"`
	MinDelayHours  int                 `json:"min_delay_hours"`
}

// PracticeSet holds concrete exercises for every mastery mode.
type PracticeSet struct {
	Version string         `json:"version"`
	Tasks   []PracticeTask `json:"tasks"`
}

// Validate checks structural and source identity requirements. Human review
// still decides whether prompts, variants, rubrics and hint progression teach well.
func (set PracticeSet) Validate(evidenceIDs []string) error {
	if set.Version != PracticeSetVersion || len(set.Tasks) != 6 {
		return fmt.Errorf("practice set requires six tasks under the current version")
	}
	known := map[string]bool{}
	for _, id := range evidenceIDs {
		known[id] = true
	}
	seenIDs, seenPrompts := map[string]bool{}, map[string]bool{}
	seenModes := map[LearningTaskMode]bool{}
	for taskIndex, task := range set.Tasks {
		if err := task.Mode.Validate(); err != nil {
			return err
		}
		if seenIDs[task.ID] || seenModes[task.Mode] || seenPrompts[strings.TrimSpace(task.Prompt)] {
			return fmt.Errorf("practice tasks need distinct ids, modes and prompts")
		}
		seenIDs[task.ID], seenModes[task.Mode], seenPrompts[strings.TrimSpace(task.Prompt)] = true, true, true
		for _, field := range []struct {
			name, value string
			limit       int
		}{
			{"id", task.ID, 100}, {"prompt", task.Prompt, 2000},
			{"variation", task.Variation, 2000}, {"expected_answer", task.ExpectedAnswer, 4000},
		} {
			if reason := practiceTextFailure(field.value, field.limit); reason != "" {
				return fmt.Errorf("practice tasks[%d].%s: %s (max %d characters)", taskIndex, field.name, reason, field.limit)
			}
		}
		if task.MinDelayHours < 0 || task.MinDelayHours > 24*365 || task.Mode == LearningTaskRetain && task.MinDelayHours < 24 || task.Mode != LearningTaskRetain && task.MinDelayHours != 0 {
			return fmt.Errorf("practice delay must preserve delayed recall")
		}
		if len(task.EvidenceIDs) == 0 {
			return fmt.Errorf("practice task requires evidence")
		}
		refs := map[string]bool{}
		for _, id := range task.EvidenceIDs {
			if !known[id] || refs[id] {
				return fmt.Errorf("unknown or repeated practice evidence")
			}
			refs[id] = true
		}
		if len(task.Hints) != 3 {
			return fmt.Errorf("practice task requires three hint levels")
		}
		hints := map[string]bool{}
		for index, hint := range task.Hints {
			if hint.Level != index+1 || !practiceText(hint.Text, 1000) || hints[strings.TrimSpace(hint.Text)] {
				return fmt.Errorf("invalid practice hint progression")
			}
			hints[strings.TrimSpace(hint.Text)] = true
		}
		dimensions := PracticeRubricDimensions(task.Mode)
		if len(task.Rubric) != len(dimensions) {
			return fmt.Errorf("practice rubric must cover every scoring dimension")
		}
		criteria := map[string]PracticeCriterion{}
		for _, criterion := range task.Rubric {
			if _, exists := criteria[criterion.ID]; exists || !practiceText(criterion.Description, 2000) {
				return fmt.Errorf("invalid practice rubric criterion")
			}
			criteria[criterion.ID] = criterion
		}
		for _, dimension := range dimensions {
			criterion, exists := criteria[dimension.ID]
			if !exists || criterion.RequiresRuntime != dimension.RequiresRuntime {
				return fmt.Errorf("practice rubric dimension or runtime boundary is missing: %s", dimension.ID)
			}
		}
	}
	return nil
}

// PracticeRubricDimensions defines the shared scoring vocabulary consumed by
// manuscript generation and review-service. Descriptions must be authored per task.
func PracticeRubricDimensions(mode LearningTaskMode) []PracticeCriterion {
	ids := []string{"accuracy", "completeness", "causality", "boundaries", "clarity"}
	switch mode {
	case LearningTaskComplete, LearningTaskReproduce, LearningTaskTransfer:
		ids = []string{"correctness", "completeness", "runtime", "tests", "design"}
	case LearningTaskDiagnose:
		ids = []string{"hypothesis", "localization", "evidence", "root_cause", "fix_verification"}
	}
	criteria := make([]PracticeCriterion, 0, len(ids))
	for _, id := range ids {
		criteria = append(criteria, PracticeCriterion{ID: id, RequiresRuntime: id == "runtime" || id == "tests" || id == "fix_verification"})
	}
	return criteria
}

func practiceText(value string, limit int) bool {
	return practiceTextFailure(value, limit) == ""
}

func practiceTextFailure(value string, limit int) string {
	switch {
	case !utf8.ValidString(value) || strings.ContainsRune(value, 0):
		return "invalid_text"
	case strings.TrimSpace(value) == "":
		return "empty"
	case utf8.RuneCountInString(value) > limit:
		return "too_long"
	case strings.Contains(value, "```"):
		return "code_fence"
	default:
		return ""
	}
}

// RenderPracticeSet binds the structured task contract into the same visible
// manuscript reviewed, hashed and exported with the chapter. No executable code
// fences are added; code tasks refer to the chapter's teaching files.
func RenderPracticeSet(set PracticeSet) string {
	encoded, _ := json.Marshal(set)
	hash := sha256.Sum256(encoded)
	var out strings.Builder
	fmt.Fprintf(&out, "## 六维练习与答案\n\n<!-- %s:%s -->\n\n", PracticeSetVersion, hex.EncodeToString(hash[:]))
	labels := map[LearningTaskMode]string{LearningTaskExplain: "解释", LearningTaskComplete: "补全", LearningTaskReproduce: "复现", LearningTaskTransfer: "迁移", LearningTaskDiagnose: "诊断", LearningTaskRetain: "延迟保持"}
	for index, task := range set.Tasks {
		fmt.Fprintf(&out, "### %d. %s\n\n%s\n\n变式：%s\n\n", index+1, labels[task.Mode], task.Prompt, task.Variation)
		if task.MinDelayHours > 0 {
			fmt.Fprintf(&out, "间隔要求：至少 %d 小时后不看原文再完成。\n\n", task.MinDelayHours)
		}
		out.WriteString("评分依据：\n\n")
		for _, criterion := range task.Rubric {
			fmt.Fprintf(&out, "- %s", criterion.Description)
			if criterion.RequiresRuntime {
				out.WriteString("（需本次工件的运行证据，否则保持未知）")
			}
			out.WriteString("\n")
		}
		out.WriteString("\n分层提示（先独立作答，再按需查看）：\n\n")
		for _, hint := range task.Hints {
			fmt.Fprintf(&out, "%d. %s\n", hint.Level, hint.Text)
		}
		fmt.Fprintf(&out, "\n参考答案（不代表代码已运行）：%s\n\n来源：", task.ExpectedAnswer)
		for _, id := range task.EvidenceIDs {
			fmt.Fprintf(&out, " [evidence:%s]", id)
		}
		out.WriteString("\n\n")
	}
	return strings.TrimSpace(out.String())
}

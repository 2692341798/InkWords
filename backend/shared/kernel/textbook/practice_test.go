package textbook

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPracticeFailureIdentifiesFieldWithoutEchoingContent(t *testing.T) {
	for _, tc := range []struct{ name, value, reason string }{
		{"empty", "  ", "empty"},
		{"fenced", "```go\nprivate answer\n```", "code_fence"},
		{"oversized", strings.Repeat("中", 4001), "too_long"},
		{"nul", "private\x00answer", "invalid_text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set := practiceSetFixture()
			set.Tasks[0].ExpectedAnswer = tc.value
			err := set.Validate([]string{"evidence-1"})
			require.ErrorContains(t, err, "tasks[0].expected_answer: "+tc.reason)
			require.NotContains(t, err.Error(), "private")
		})
	}
}

func TestPracticeSetRequiresSixDistinctEvidenceBoundTasks(t *testing.T) {
	set := practiceSetFixture()
	require.NoError(t, set.Validate([]string{"evidence-1"}))
	set.Tasks[1].Prompt = set.Tasks[0].Prompt
	require.ErrorContains(t, set.Validate([]string{"evidence-1"}), "distinct")
	set = practiceSetFixture()
	set.Tasks[0].EvidenceIDs = []string{"unknown"}
	require.ErrorContains(t, set.Validate([]string{"evidence-1"}), "evidence")
	set = practiceSetFixture()
	set.Tasks = set.Tasks[:5]
	require.ErrorContains(t, set.Validate([]string{"evidence-1"}), "six")
}

func TestPracticeSetEnforcesRubricRuntimeAndDelayedRecallBoundaries(t *testing.T) {
	set := practiceSetFixture()
	set.Tasks[2].Rubric[2].RequiresRuntime = false
	require.ErrorContains(t, set.Validate([]string{"evidence-1"}), "runtime")
	set = practiceSetFixture()
	set.Tasks[5].MinDelayHours = 0
	require.ErrorContains(t, set.Validate([]string{"evidence-1"}), "delay")
	set = practiceSetFixture()
	set.Tasks[0].Hints[1].Level = 1
	require.ErrorContains(t, set.Validate([]string{"evidence-1"}), "hint")
}

func TestRenderedPracticeSetBindsAllTaskContentToTheManuscript(t *testing.T) {
	set := practiceSetFixture()
	markdown := RenderPracticeSet(set)
	require.Contains(t, markdown, set.Tasks[0].Prompt)
	require.Contains(t, markdown, set.Tasks[0].ExpectedAnswer)
	require.Contains(t, markdown, set.Tasks[0].Rubric[0].Description)
	require.Contains(t, markdown, set.Tasks[0].Hints[2].Text)
	require.Contains(t, markdown, "[evidence:evidence-1]")
	set.Tasks[0].ExpectedAnswer = "修改后的答案必须改变母稿中的绑定内容。"
	require.NotEqual(t, markdown, RenderPracticeSet(set))
}

func practiceSetFixture() PracticeSet {
	set := PracticeSet{Version: PracticeSetVersion}
	for _, mode := range []LearningTaskMode{LearningTaskExplain, LearningTaskComplete, LearningTaskReproduce, LearningTaskTransfer, LearningTaskDiagnose, LearningTaskRetain} {
		task := PracticeTask{ID: "task-" + string(mode), Mode: mode, Prompt: string(mode) + "：说明方法和路径如何共同确定处理函数。", Variation: "把路径由 /orders 换成 /health 再回答。", ExpectedAnswer: "先按方法选择路由树，然后按路径查找已登记的处理函数。", EvidenceIDs: []string{"evidence-1"}, Hints: []PracticeHint{{Level: 1, Text: "先回忆请求携带的两个条件。"}, {Level: 2, Text: "先比较方法，再比较路径。"}, {Level: 3, Text: "画出按方法分组的树及路径查找箭头。"}}}
		for _, criterion := range PracticeRubricDimensions(mode) {
			criterion.Description = "根据题目检查" + criterion.ID + "是否完整并说明边界。"
			task.Rubric = append(task.Rubric, criterion)
		}
		if mode == LearningTaskRetain {
			task.MinDelayHours = 24
		}
		set.Tasks = append(set.Tasks, task)
	}
	return set
}

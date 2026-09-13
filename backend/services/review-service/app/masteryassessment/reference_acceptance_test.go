package masteryassessment

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
)

// This key belongs only to the operator-authored evaluation task. It is not an
// approved project revision, a learner answer, or independent source evidence.
const referenceAcceptanceKey = "GET 登记先经过 RouterGroup.GET 和 handle，将方法、完整路径、处理函数交给 Engine.addRoute，保存到 GET 的方法树。处理请求时 Engine.handleHTTPRequest 先用方法选择树，再用路径经 getValue 查找并在命中后执行 handlers。查找使用登记时保存的对应关系，不会临时创建处理函数；同一路径的 GET 和 POST 可以对应不同操作，方法树将其隔离，所以只有 GET 登记不能让 POST 命中该 GET 处理函数。路径未匹配通常为 404；HandleMethodNotAllowed 开启且其他方法树有该路径时才可能为 405。"

func referenceAcceptanceInput(index int, answer string) mastery.AssessmentInput {
	input := realAcceptanceInput(index, answer)
	// Freeze task content independently of the four answer variants.
	task, _ := json.Marshal(struct {
		Prompt         string                        `json:"prompt"`
		Rubric         []mastery.AssessmentCriterion `json:"rubric"`
		ExpectedAnswer string                        `json:"expected_answer"`
	}{input.Prompt, input.Rubric, referenceAcceptanceKey})
	input.TaskReference = &mastery.AssessmentTaskReference{TaskID: "evaluation-gin-explain-reference-v1", PracticeContentHash: digest(string(task)), ExpectedAnswer: referenceAcceptanceKey}
	return input
}

func TestReferenceAcceptanceInputsAreFrozenAndBudgetedWithoutProvider(t *testing.T) {
	port := &capturedPort{}
	generator := NewGenerator(port, "deepseek", "deepseek-v4-flash")
	var taskHash string
	for index, answer := range []string{causalAcceptanceAnswer, proceduralAcceptanceAnswer, incompleteAcceptanceAnswer, misconceptionAcceptanceAnswer} {
		input := referenceAcceptanceInput(index, answer)
		require.Nil(t, realAcceptanceInput(index, answer).TaskReference, "legacy evaluation must remain unchanged")
		require.NoError(t, input.Validate())
		require.Equal(t, mastery.AssessmentReferenceContractVersion, input.ContractVersion())
		if taskHash != "" {
			require.Equal(t, taskHash, input.TaskReference.PracticeContentHash)
		}
		taskHash = input.TaskReference.PracticeContentHash
		preview, err := generator.Preview(input)
		require.NoError(t, err)
		t.Logf("case=%d input_hash=%s request_hash=%s request_bytes=%d max_output_tokens=%d", index+1, preview.InputHash, preview.RequestHash, preview.RequestBytes, preview.MaxOutputTokens)
	}
	require.Zero(t, port.calls)
}

func TestDecisionAcceptanceInputsAreBudgetedWithoutProvider(t *testing.T) {
	port := &capturedPort{}
	generator := NewGenerator(port, "deepseek", "deepseek-v4-flash")
	for index, answer := range []string{causalAcceptanceAnswer, proceduralAcceptanceAnswer, incompleteAcceptanceAnswer, misconceptionAcceptanceAnswer} {
		input := referenceAcceptanceInput(index, answer)
		input.DecisionPolicy = mastery.AssessmentDecisionPolicy
		preview, err := generator.Preview(input)
		require.NoError(t, err)
		t.Logf("case=%d input_hash=%s request_hash=%s request_bytes=%d max_output_tokens=%d", index+1, preview.InputHash, preview.RequestHash, preview.RequestBytes, preview.MaxOutputTokens)
	}
	require.Zero(t, port.calls)
}

func TestJudgmentAcceptanceInputsAreBudgetedWithoutProvider(t *testing.T) {
	port := &capturedPort{}
	generator := NewGenerator(port, "deepseek", "deepseek-v4-flash")
	for index, answer := range []string{causalAcceptanceAnswer, proceduralAcceptanceAnswer, incompleteAcceptanceAnswer, misconceptionAcceptanceAnswer} {
		input := referenceAcceptanceInput(index, answer)
		input.DecisionPolicy = mastery.AssessmentJudgmentPolicy
		preview, err := generator.Preview(input)
		require.NoError(t, err)
		t.Logf("case=%d input_hash=%s request_hash=%s request_bytes=%d max_output_tokens=%d", index+1, preview.InputHash, preview.RequestHash, preview.RequestBytes, preview.MaxOutputTokens)
	}
	require.Zero(t, port.calls)
}

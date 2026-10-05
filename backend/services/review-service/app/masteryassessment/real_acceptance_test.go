package masteryassessment

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	platformllm "inkwords-backend/shared/platform/llm"
)

const realAssessmentOptIn = "approved"

type acceptanceScore struct {
	calculated     bool
	average        float64
	minimum        int
	otherMinimum   int
	causality      int
	missing        int
	misconceptions int
}

type realAssessmentSummary struct {
	Average        float64 `json:"average"`
	Minimum        int     `json:"minimum"`
	OtherMinimum   int     `json:"other_minimum"`
	Causality      int     `json:"causality"`
	Missing        int     `json:"missing"`
	Misconceptions int     `json:"misconceptions"`
}

type realAssessmentArtifact struct {
	Case         string                   `json:"case"`
	ReviewStatus string                   `json:"review_status"`
	Acceptance   *realAssessmentSummary   `json:"acceptance,omitempty"`
	Result       Result                   `json:"result"`
	Input        *mastery.AssessmentInput `json:"input,omitempty"`
}

// TestRealAssessmentSeparatesRepresentativeAnswers is deliberately excluded
// from normal test runs. The protected real-acceptance environment must set the
// exact opt-in value before this test may make four bounded Provider calls.
func TestRealAssessmentSeparatesRepresentativeAnswers(t *testing.T) {
	if strings.TrimSpace(os.Getenv("INKWORDS_REAL_ASSESSMENT_ACCEPTANCE")) != realAssessmentOptIn {
		t.Skip("real mastery assessment acceptance requires protected-environment opt-in")
	}
	apiKey := firstConfigured(os.Getenv("MASTERY_DEEPSEEK_API_KEY"), os.Getenv("DEEPSEEK_API_KEY"))
	model := firstConfigured(os.Getenv("MASTERY_DEEPSEEK_MODEL"), os.Getenv("DEEPSEEK_MODEL"))
	outputDir := strings.TrimSpace(os.Getenv("INKWORDS_REAL_ASSESSMENT_OUTPUT_DIR"))
	require.NotEmpty(t, apiKey, "real acceptance requires a DeepSeek API key")
	require.NotEmpty(t, model, "real acceptance requires an explicit model")
	require.NotEmpty(t, outputDir, "real acceptance requires an explicit artifact directory")
	require.NoError(t, os.MkdirAll(outputDir, 0o700))
	require.NoError(t, os.Chmod(outputDir, 0o700))

	generator := NewGenerator(
		platformllm.NewDeepSeekGenerationAdapter(platformllm.NewDeepSeekClient(apiKey)),
		"deepseek",
		model,
	)
	if strings.TrimSpace(os.Getenv("INKWORDS_REAL_ASSESSMENT_REASONING")) == realAssessmentOptIn {
		require.Equal(t, realAssessmentOptIn, os.Getenv("INKWORDS_REAL_ASSESSMENT_JUDGMENT"), "reasoning evaluation requires the canonical contract")
		generator.options = Options{ReasoningEffort: firstConfigured(os.Getenv("INKWORDS_REAL_ASSESSMENT_REASONING_EFFORT"), "high")}
	}
	if strings.TrimSpace(os.Getenv("INKWORDS_REAL_ASSESSMENT_EXTENDED_BUDGET")) == realAssessmentOptIn {
		require.Equal(t, realAssessmentOptIn, os.Getenv("INKWORDS_REAL_ASSESSMENT_JUDGMENT"), "extended output budget requires canonical judgments")
		generator.options.MaxOutputTokens = 6000
	}
	cases := []struct {
		name   string
		answer string
		check  func(*testing.T, acceptanceScore)
	}{
		{
			name:   "complete-causal-answer",
			answer: causalAcceptanceAnswer,
			check: func(t *testing.T, score acceptanceScore) {
				require.GreaterOrEqual(t, score.minimum, 3, "完整答案每项都应达到基本要求，不能只检查平均分")
			},
		},
		{
			name:   "procedural-only-answer",
			answer: proceduralAcceptanceAnswer,
			check: func(t *testing.T, score acceptanceScore) {
				require.LessOrEqual(t, score.causality, 2, "正确复述顺序不等于解释为何")
				require.GreaterOrEqual(t, score.otherMinimum, 3, "因果缺失不能导致已满足的流程条目不达标")
			},
		},
		{
			name:   "incomplete-answer",
			answer: incompleteAcceptanceAnswer,
			check: func(t *testing.T, score acceptanceScore) {
				require.LessOrEqual(t, score.average, 2.5, "缺少方法树、登记路径和状态边界的答案不应高分")
				require.NotZero(t, score.missing, "不完整答案应指出遗漏")
			},
		},
		{
			name:   "misconception-answer",
			answer: misconceptionAcceptanceAnswer,
			check: func(t *testing.T, score acceptanceScore) {
				require.LessOrEqual(t, score.average, 2.0, "与来源冲突的答案不应通过")
				require.NotZero(t, score.misconceptions, "错误答案应明确指出误解")
			},
		},
	}

	names := make([]string, len(cases))
	for index, item := range cases {
		names[index] = item.name
	}
	indices, err := selectAcceptanceIndices(names, os.Getenv("INKWORDS_REAL_ASSESSMENT_CASE"))
	require.NoError(t, err)
	selectedNames := make([]string, 0, len(indices))
	for _, index := range indices {
		selectedNames = append(selectedNames, cases[index].name)
		_, statErr := os.Stat(filepath.Join(outputDir, cases[index].name+".json"))
		require.True(t, os.IsNotExist(statErr), "use a fresh output directory; never overwrite a previous receipt")
	}
	scope, err := json.MarshalIndent(struct {
		Cases            []string `json:"cases"`
		DiagnosticSubset bool     `json:"diagnostic_subset"`
	}{selectedNames, len(indices) != len(cases)}, "", "  ")
	require.NoError(t, err)
	scopeFile, err := os.OpenFile(filepath.Join(outputDir, "run-scope.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	require.NoError(t, err)
	_, err = scopeFile.Write(scope)
	require.NoError(t, scopeFile.Close())
	require.NoError(t, err)
	t.Logf("selected_cases=%v diagnostic_subset=%t", selectedNames, len(indices) != len(cases))
	for _, index := range indices {
		item := cases[index]
		input := realAcceptanceInput(index, item.answer)
		if os.Getenv("INKWORDS_REAL_ASSESSMENT_REFERENCE") == realAssessmentOptIn {
			input = referenceAcceptanceInput(index, item.answer)
		}
		if os.Getenv("INKWORDS_REAL_ASSESSMENT_DECISION") == realAssessmentOptIn {
			input = referenceAcceptanceInput(index, item.answer)
			input.DecisionPolicy = mastery.AssessmentDecisionPolicy
		}
		if os.Getenv("INKWORDS_REAL_ASSESSMENT_JUDGMENT") == realAssessmentOptIn {
			input = referenceAcceptanceInput(index, item.answer)
			input.DecisionPolicy = mastery.AssessmentJudgmentPolicy
		}
		preview, err := generator.Preview(input)
		require.NoError(t, err)
		result, err := generator.Assess(context.Background(), input)
		require.NoError(t, writeRealAssessmentArtifact(outputDir, item.name, acceptanceScore{}, result, input), "调用失败时也保留遥测，不保存原始供应商错误")
		require.NoError(t, err, "case %s failed; later scoring calls were not started", item.name)
		require.Equal(t, 1, result.ProviderCalls)
		require.True(t, result.Usage.Known, "真实验收必须保存 Provider Token 遥测")
		require.Positive(t, result.Usage.InputTokens)
		require.Positive(t, result.Usage.OutputTokens)
		require.NotNil(t, result.Feedback)

		score := summarizeAcceptanceFeedback(t, result.Feedback)
		require.NoError(t, writeRealAssessmentArtifact(outputDir, item.name, score, result, input))
		t.Logf(
			"case=%s provider=%s model=%s input_hash=%s request_hash=%s average=%.2f missing=%d misconceptions=%d input_tokens=%d output_tokens=%d latency_ms=%d request_bytes=%d",
			item.name,
			result.Provider,
			result.Model,
			result.InputHash,
			result.RequestHash,
			score.average,
			score.missing,
			score.misconceptions,
			result.Usage.InputTokens,
			result.Usage.OutputTokens,
			result.LatencyMillis,
			preview.RequestBytes,
		)
		item.check(t, score)
	}
}

func TestWriteRealAssessmentArtifactCreatesPrivateReviewableFile(t *testing.T) {
	scoreValue := 3
	result := Result{
		InputHash:     "sha256:input",
		RequestHash:   "sha256:request",
		Contract:      mastery.AssessmentContractVersion,
		Origin:        "automated",
		Provider:      "deepseek",
		Model:         "deepseek-v4-flash",
		ProviderCalls: 1,
		Feedback: &mastery.AssessmentFeedback{
			Criteria:       []mastery.CriterionAssessment{{ID: "accuracy", Score: &scoreValue, Reason: "调用顺序正确。", AnswerQuote: "先按方法选择", EvidenceIDs: []string{"gin-request-dispatch"}}},
			CorrectPoints:  []mastery.AssessmentFinding{{Text: "说明了方法树选择。", EvidenceIDs: []string{"gin-request-dispatch"}}},
			MissingPoints:  []mastery.AssessmentFinding{},
			Misconceptions: []mastery.AssessmentFinding{},
			NextHint:       mastery.AssessmentFinding{Text: "再说明路径未命中的结果。", EvidenceIDs: []string{"gin-request-dispatch"}},
			Remediation:    []mastery.AssessmentFinding{{Text: "复述 404 与 405 的边界。", EvidenceIDs: []string{"gin-request-dispatch"}}},
		},
	}
	score := acceptanceScore{calculated: true, average: 3, missing: 0, misconceptions: 0}

	outputDir := t.TempDir()
	require.NoError(t, writeRealAssessmentArtifact(outputDir, "complete-causal-answer", score, result))

	path := filepath.Join(outputDir, "complete-causal-answer.json")
	encoded, err := os.ReadFile(path)
	require.NoError(t, err)
	var decoded realAssessmentArtifact
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Equal(t, "complete-causal-answer", decoded.Case)
	require.Equal(t, "requires_semantic_review", decoded.ReviewStatus)
	require.Equal(t, score.average, decoded.Acceptance.Average)
	require.Equal(t, result.RequestHash, decoded.Result.RequestHash)
	require.Equal(t, result.Feedback, decoded.Result.Feedback)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func writeRealAssessmentArtifact(outputDir, caseName string, score acceptanceScore, result Result, inputs ...mastery.AssessmentInput) error {
	artifact := realAssessmentArtifact{
		Case:         caseName,
		ReviewStatus: "requires_semantic_review",
		Result:       result,
	}
	if score.calculated && result.Feedback != nil {
		artifact.Acceptance = &realAssessmentSummary{
			Average:        score.average,
			Minimum:        score.minimum,
			OtherMinimum:   score.otherMinimum,
			Causality:      score.causality,
			Missing:        score.missing,
			Misconceptions: score.misconceptions,
		}
	}
	if len(inputs) > 0 {
		artifact.Input = &inputs[0]
	}
	encoded, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, caseName+".json"), encoded, 0o600)
}

func TestRealAssessmentAcceptanceInputsAreValidWithoutCallingProvider(t *testing.T) {
	answers := []string{
		causalAcceptanceAnswer,
		proceduralAcceptanceAnswer,
		incompleteAcceptanceAnswer,
		misconceptionAcceptanceAnswer,
	}
	seen := map[string]bool{}
	generator := NewGenerator(&capturedPort{}, "deepseek", "deepseek-v4-flash")
	for index, answer := range answers {
		input := realAcceptanceInput(index, answer)
		require.NoError(t, input.Validate())
		hash := mastery.AssessmentInputHash(input)
		require.False(t, seen[hash], "每个代表作答必须有独立输入身份")
		seen[hash] = true
		preview, err := generator.Preview(input)
		require.NoError(t, err)
		t.Logf("case=%d input_hash=%s request_hash=%s request_bytes=%d max_output_tokens=%d", index+1, preview.InputHash, preview.RequestHash, preview.RequestBytes, preview.MaxOutputTokens)
	}
}

func realAcceptanceInput(index int, answer string) mastery.AssessmentInput {
	evidence := []mastery.AssessmentEvidence{
		{
			ID:      "gin-route-registration",
			Kind:    "source",
			Excerpt: "在 Gin 固定提交 73726dc606796a025971fe451f0aa6f1b9b847f6 中，RouterGroup.GET 调用 handle；handle 组合处理函数并把 HTTP 方法与绝对路径交给 Engine.addRoute。addRoute 按方法取得路由树并登记路径和 handlers。",
		},
		{
			ID:      "gin-request-dispatch",
			Kind:    "source",
			Excerpt: "Engine.handleHTTPRequest 按请求的 HTTP 方法选择对应路由树，再以请求路径调用 getValue。命中后执行 handlers；路径未命中通常返回 404，只有启用 HandleMethodNotAllowed 且其他方法树存在该路径时才返回 405。",
		},
		{
			ID: "gin-method-tree-identity", Kind: "source",
			Excerpt: "固定提交 73726dc606796a025971fe451f0aa6f1b9b847f6，tree.go，(methodTrees).get：\nfunc (trees methodTrees) get(method string) *node {\n\tfor _, tree := range trees {\n\t\tif tree.method == method {\n\t\t\treturn tree.root\n\t\t}\n\t}\n\treturn nil\n}",
		},
	}
	for evidenceIndex := range evidence {
		evidence[evidenceIndex].ContentHash = digest(evidence[evidenceIndex].Excerpt)
	}
	descriptions := map[string]string{
		"accuracy":     "准确说明 Gin 按 HTTP 方法和路径登记、选择并查找路由。",
		"completeness": "覆盖注册阶段、请求分派、处理函数命中及未命中的结果。",
		"causality":    "解释方法为何也是查找身份的一部分：同一路径的不同 HTTP 操作须隔离，并说明只登记 GET 为何不能让 POST 命中该 GET 处理函数。仅列调用顺序或报告 404/405 条件不满足本因果条目，最多部分达到。",
		"boundaries":   "准确区分 404 与启用 HandleMethodNotAllowed 后可能出现的 405。",
		"clarity":      "使用可复述的顺序表达完整调用过程。",
	}
	rubric := sharedtextbook.PracticeRubricDimensions(sharedtextbook.LearningTaskExplain)
	for rubricIndex := range rubric {
		rubric[rubricIndex].Description = descriptions[rubric[rubricIndex].ID]
	}
	return mastery.AssessmentInput{
		AttemptID:   "real-acceptance-attempt-" + strconv.Itoa(index+1),
		ObjectiveID: "real-acceptance-objective",
		RevisionID:  "gin-73726dc606796a025971fe451f0aa6f1b9b847f6",
		Skill:       mastery.Explain,
		Prompt:      "请解释一个 GET 请求为什么会找到对应的 Gin 处理函数，并说明未命中的边界。",
		Answer:      answer,
		Rubric:      rubric,
		Evidence:    evidence,
	}
}

func summarizeAcceptanceFeedback(t *testing.T, feedback *mastery.AssessmentFeedback) acceptanceScore {
	t.Helper()
	require.Len(t, feedback.Criteria, 5)
	total := 0
	minimum, otherMinimum, causality := 4, 4, -1
	for _, criterion := range feedback.Criteria {
		require.NotNil(t, criterion.Score, "解释题没有运行时维度，不应返回未知分数")
		total += *criterion.Score
		minimum = min(minimum, *criterion.Score)
		if criterion.ID == "causality" {
			causality = *criterion.Score
		} else {
			otherMinimum = min(otherMinimum, *criterion.Score)
		}
	}
	return acceptanceScore{
		calculated:     true,
		average:        float64(total) / float64(len(feedback.Criteria)),
		minimum:        minimum,
		otherMinimum:   otherMinimum,
		causality:      causality,
		missing:        len(feedback.MissingPoints),
		misconceptions: len(feedback.Misconceptions),
	}
}

func firstConfigured(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

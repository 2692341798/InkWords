package masteryassessment

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
	"inkwords-backend/shared/kernel/generation"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	platformllm "inkwords-backend/shared/platform/llm"
)

const correctMethodSelection = "package routing\n\ntype node struct { name string }\ntype methodTree struct { method string; root *node }\n\nfunc findRoot(trees []methodTree, method string) *node {\n\tfor _, tree := range trees {\n\t\tif tree.method == method { return tree.root }\n\t}\n\treturn nil\n}\n"
const wrongMethodSelection = "package routing\n\ntype node struct { name string }\ntype methodTree struct { method string; root *node }\n\nfunc findRoot(trees []methodTree, method string) *node {\n\tif len(trees) > 0 { return trees[0].root }\n\treturn nil\n}\n"

// codeDiagnosticPort is only used with the operator-authored fixtures below.
// It retains the response data for offline rejection diagnosis, never HTTP
// headers, API keys, error bodies, or production learner submissions.
type codeDiagnosticPort struct {
	generation.Port
	response generation.Result
}

func (p *codeDiagnosticPort) Generate(ctx context.Context, request generation.Request) (generation.Result, error) {
	response, err := p.Port.Generate(ctx, request)
	p.response = response
	return response, err
}

func codeAcceptanceOptions(t *testing.T) Options {
	t.Helper()
	effort := firstConfigured(os.Getenv("INKWORDS_REAL_CODE_ASSESSMENT_REASONING"), "low")
	require.Contains(t, []string{"low", "disabled"}, effort)
	if effort == "disabled" {
		effort = ""
	}
	return Options{ReasoningEffort: effort, MaxOutputTokens: 6000}
}

func codeAcceptanceInput(t *testing.T, name, code string) mastery.AssessmentInput {
	t.Helper()
	id := func(label string) string {
		return uuid.NewSHA1(uuid.NameSpaceOID, []byte("inkwords-code-evaluation-v1/"+label)).String()
	}
	input := realAcceptanceInput(0, "我已完成 findRoot，运行和测试均通过。")
	input.AttemptID, input.ObjectiveID, input.RevisionID = id(name), id("objective"), id("gin-fixed-revision")
	input.Skill = mastery.Reproduce
	input.Prompt = "实现 findRoot：按给定 HTTP 方法查找已有方法树并返回原 root；没有匹配则返回 nil。不得创建或修改方法树。请结合提交的源文件评阅。"
	input.Rubric = []mastery.AssessmentCriterion{
		{ID: "correctness", Description: "仅当输入 HTTP 方法与登记方法相等时返回该项 root；无匹配时返回 nil，不得无条件返回第一项。"},
		{ID: "completeness", Description: "覆盖空列表、有匹配和无匹配，包含遍历与方法匹配分支，且不修改登记列表。"},
		{ID: "runtime", Description: "仅在存在同一作答快照的 VerificationRun 时评价运行结果；文字自称运行通过不算证据。", RequiresRuntime: true},
		{ID: "tests", Description: "仅在存在同一作答快照的测试执行证据时评价测试结果；自称测试通过不算证据。", RequiresRuntime: true},
		{ID: "design", Description: "按已有数据结构清晰地选择并返回原 root，无新建树或额外共享状态。"},
	}
	input.Evidence = input.Evidence[2:3]
	key := "有界遍历登记列表，逐项比较方法；相等时返回该项保存的 root，遍历结束仍无匹配则返回 nil。空列表自然返回 nil。不得修改登记项，也不创建新 root。"
	task, err := json.Marshal(struct {
		Prompt string
		Rubric []mastery.AssessmentCriterion
		Key    string
	}{input.Prompt, input.Rubric, key})
	require.NoError(t, err)
	input.TaskReference = &mastery.AssessmentTaskReference{TaskID: "evaluation-method-selection-code-v1", PracticeContentHash: digest(string(task)), ExpectedAnswer: key}
	input.DecisionPolicy = mastery.AssessmentJudgmentPolicy
	artifact := &sharedtextbook.LearnerArtifact{Format: sharedtextbook.LearnerArtifactFormat, WorkspaceID: id("workspace"), ObjectiveID: input.ObjectiveID, AttemptID: input.AttemptID, SessionID: id("session/" + name), RevisionID: input.RevisionID, TaskID: input.TaskReference.TaskID, PracticeContentHash: input.TaskReference.PracticeContentHash, Skill: sharedtextbook.LearningTaskReproduce, SubmittedAt: time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC), Files: []sharedtextbook.LearnerCodeFile{{Path: "router.go", Content: code}}}
	artifact.SnapshotHash, err = sharedtextbook.LearnerArtifactHash(*artifact)
	require.NoError(t, err)
	input.LearnerArtifact = artifact
	require.NoError(t, input.Validate())
	return input
}

func TestCodeAcceptancePreflight(t *testing.T) {
	port := &capturedPort{}
	g := NewGeneratorWithOptions(port, "deepseek", "deepseek-v4-flash", codeAcceptanceOptions(t))
	for _, c := range []struct{ name, code string }{{"correct-method-selection", correctMethodSelection}, {"wrong-method-selection", wrongMethodSelection}} {
		input := codeAcceptanceInput(t, c.name, c.code)
		preview, err := g.Preview(input)
		require.NoError(t, err)
		data, err := json.Marshal(preview)
		require.NoError(t, err)
		t.Logf("case=%s preview=%s snapshot=%s", c.name, data, input.LearnerArtifact.SnapshotHash)
		require.Empty(t, input.ArtifactHash, "the claim in prose is not runtime evidence")
		require.Equal(t, "source", input.Evidence[0].Kind)
	}
	require.Zero(t, port.calls)
}

func TestRealCodeAssessmentSeparatesMethodMatchingAndKeepsRuntimeUnknown(t *testing.T) {
	if os.Getenv("INKWORDS_REAL_CODE_ASSESSMENT_ACCEPTANCE") != realAssessmentOptIn {
		t.Skip("requires explicit bounded real code grading opt-in")
	}
	key := firstConfigured(os.Getenv("MASTERY_DEEPSEEK_API_KEY"), os.Getenv("DEEPSEEK_API_KEY"))
	require.NotEmpty(t, key)
	model := firstConfigured(os.Getenv("MASTERY_DEEPSEEK_MODEL"), os.Getenv("DEEPSEEK_MODEL"))
	require.Equal(t, "deepseek-v4-flash", model)
	output := os.Getenv("INKWORDS_REAL_ASSESSMENT_OUTPUT_DIR")
	require.NotEmpty(t, output)
	options := codeAcceptanceOptions(t)
	require.NoError(t, os.MkdirAll(output, 0700))
	marker, err := os.OpenFile(filepath.Join(output, "run-scope.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	require.NoError(t, err)
	selected := os.Getenv("INKWORDS_REAL_CODE_ASSESSMENT_CASE")
	require.Contains(t, []string{"", "correct-method-selection", "wrong-method-selection"}, selected)
	maxCalls := 2
	if selected != "" {
		maxCalls = 1
	}
	err = json.NewEncoder(marker).Encode(map[string]any{"origin": "operator_authored_code_evaluation", "max_provider_calls": maxCalls, "case": selected, "max_output_tokens": options.MaxOutputTokens, "reasoning_effort": options.ReasoningEffort, "adapter_timeout_seconds": 60, "task_timeout_seconds": 60, "retries": 0, "code_execution": 0, "human_reviews": 0, "learning_record_writes": 0})
	require.NoError(t, err)
	require.NoError(t, marker.Close())
	port := &codeDiagnosticPort{Port: platformllm.NewDeepSeekGenerationAdapterWithTimeout(platformllm.NewDeepSeekClient(key), 60*time.Second)}
	g := NewGeneratorWithOptions(port, "deepseek", model, options)
	for _, c := range []struct{ name, code string }{{"correct-method-selection", correctMethodSelection}, {"wrong-method-selection", wrongMethodSelection}} {
		if selected != "" && c.name != selected {
			continue
		}
		input := codeAcceptanceInput(t, c.name, c.code)
		_, statErr := os.Stat(filepath.Join(output, c.name+".json"))
		require.True(t, os.IsNotExist(statErr))
		result, err := g.Assess(context.Background(), input)
		diagnostic, writeErr := os.OpenFile(filepath.Join(output, c.name+"-model-response.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		require.NoError(t, writeErr)
		writeErr = json.NewEncoder(diagnostic).Encode(struct {
			Origin      string            `json:"origin"`
			RequestHash string            `json:"request_hash"`
			Response    generation.Result `json:"untrusted_response"`
		}{"operator_authored_code_evaluation", result.RequestHash, port.response})
		require.NoError(t, diagnostic.Close())
		require.NoError(t, writeErr)
		require.NoError(t, writeRealAssessmentArtifact(output, c.name, acceptanceScore{}, result, input))
		require.NoError(t, err, "stop at this case without later calls")
		require.True(t, result.Usage.Known)
		require.Equal(t, 1, result.ProviderCalls)
		byID := map[string]mastery.CriterionAssessment{}
		for _, item := range result.Feedback.Criteria {
			byID[item.ID] = item
		}
		for _, id := range []string{"runtime", "tests"} {
			require.Nil(t, byID[id].Score, "self-reported execution must remain unknown")
		}
		if c.name == "correct-method-selection" {
			for _, id := range []string{"correctness", "completeness", "design"} {
				require.NotNil(t, byID[id].Score)
				require.GreaterOrEqual(t, *byID[id].Score, 3)
				require.Equal(t, "router.go", byID[id].AnswerPath)
			}
		} else {
			require.NotNil(t, byID["correctness"].Score)
			require.LessOrEqual(t, *byID["correctness"].Score, 2)
			require.Positive(t, len(result.Feedback.MissingPoints)+len(result.Feedback.Misconceptions), "a missing comparison may be reported as an omission or an incorrect implementation")
			require.Equal(t, "router.go", byID["correctness"].AnswerPath)
			require.NotEmpty(t, byID["correctness"].AnswerQuote)
		}
		t.Logf("case=%s input_hash=%s request_hash=%s input_tokens=%d output_tokens=%d latency_ms=%d runtime=unknown tests=unknown", c.name, result.InputHash, result.RequestHash, result.Usage.InputTokens, result.Usage.OutputTokens, result.LatencyMillis)
	}
}

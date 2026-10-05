package masteryassessment

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	shared "inkwords-backend/shared/kernel/textbook"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const methodSelectionTests = `package routing
import "testing"
func TestFindRoot(t *testing.T) {
 get,post := &node{name:"get"}, &node{name:"post"}
 trees := []methodTree{{method:"GET",root:get},{method:"POST",root:post}}
 for _, tc := range []struct{name, method string; trees []methodTree; want *node}{
  {"empty","GET",nil,nil},{"first","GET",trees,get},{"second","POST",trees,post},{"missing","DELETE",trees,nil},
 } { t.Run(tc.name,func(t *testing.T){ if got:=findRoot(tc.trees,tc.method);got!=tc.want {t.Fatalf("method %s: got %v, want %v",tc.method,got,tc.want)} }) }
 if trees[0].root!=get || trees[1].root!=post {t.Fatal("registration mutated")}
}
`

// TestPrepareRuntimeAssessmentAcceptance writes operator-authored evaluation
// data only. It never claims these synthetic IDs are approved database rows.
func TestPrepareRuntimeAssessmentAcceptance(t *testing.T) {
	if os.Getenv("INKWORDS_RUNTIME_ASSESSMENT_PREPARE") != "approved" {
		t.Skip("explicit offline preparation opt-in")
	}
	output := os.Getenv("INKWORDS_RUNTIME_ASSESSMENT_DIR")
	require.NotEmpty(t, output)
	raw, err := os.ReadFile(filepath.Join(output, "capability.json"))
	require.NoError(t, err)
	var envelope struct {
		Data shared.LearnerVerificationCapability `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &envelope))
	require.NoError(t, envelope.Data.Validate())
	require.True(t, envelope.Data.Available)
	for _, c := range []struct{ name, code string }{{"correct-method-selection", correctMethodSelection}, {"wrong-method-selection", wrongMethodSelection}} {
		input := codeAcceptanceInput(t, c.name, c.code)
		input.LearnerArtifact.Files = append(input.LearnerArtifact.Files, shared.LearnerCodeFile{Path: "router_test.go", Content: methodSelectionTests})
		input.LearnerArtifact.SnapshotHash, err = shared.LearnerArtifactHash(*input.LearnerArtifact)
		require.NoError(t, err)
		evidence := input.Evidence[0].ID
		arc := shared.DefaultLearningArc()
		for i := range arc.Stages {
			arc.Stages[i].Objective = "隔离验收样本"
			arc.Stages[i].SuccessEvidence = []string{evidence}
			arc.Stages[i].RecoveryRoute = "检查失败测试"
		}
		chapter := uuid.NewSHA1(uuid.NameSpaceOID, []byte("runtime-assessment-evaluation-chapter-v1")).String()
		set := shared.PracticeSet{Version: shared.PracticeSetVersion}
		for _, mode := range []shared.LearningTaskMode{shared.LearningTaskExplain, shared.LearningTaskComplete, shared.LearningTaskReproduce, shared.LearningTaskTransfer, shared.LearningTaskDiagnose, shared.LearningTaskRetain} {
			task := shared.PracticeTask{ID: "evaluation-" + string(mode), Mode: mode, Prompt: "仅验收结构夹具：" + string(mode), Variation: "仅用于合同验证，不是学习者任务", ExpectedAnswer: "根据冻结的方法选择说明检查结果。", EvidenceIDs: []string{evidence}, Hints: []shared.PracticeHint{{Level: 1, Text: "观察输入方法"}, {Level: 2, Text: "比较登记方法"}, {Level: 3, Text: "检查查找失败分支"}}, Rubric: shared.PracticeRubricDimensions(mode)}
			for i := range task.Rubric {
				task.Rubric[i].Description = "验收合同维度：" + task.Rubric[i].ID
			}
			if mode == shared.LearningTaskRetain {
				task.MinDelayHours = 24
			}
			if mode == shared.LearningTaskReproduce {
				task.ID = input.TaskReference.TaskID
				task.Prompt = input.Prompt
				task.ExpectedAnswer = input.TaskReference.ExpectedAnswer
				task.Rubric = input.Rubric
			}
			set.Tasks = append(set.Tasks, task)
		}
		projection := shared.LearningProjection{Format: "inkwords.learning-projection.v2", RevisionID: input.RevisionID, ChapterID: chapter, ContentHash: input.TaskReference.PracticeContentHash, LearningArc: arc, PracticeSet: &set, EvidenceIDs: []string{evidence}, Objectives: []shared.LearningObjective{{ID: input.ObjectiveID, ChapterID: chapter, Text: input.Prompt, RequiredModes: []shared.LearningTaskMode{shared.LearningTaskReproduce}}}}
		plan, err := shared.NewLearnerVerificationPlan(*input.LearnerArtifact, projection, *envelope.Data.Runner)
		require.NoError(t, err)
		resolved := shared.LearnerVerificationInput{Format: shared.LearnerVerificationInputFormat, Plan: plan, Artifact: *input.LearnerArtifact, Task: set.Tasks[2]}
		ref := shared.LearnerVerificationReference{Format: shared.LearnerVerificationReferenceFormat, RunID: uuid.NewString(), WorkspaceID: plan.WorkspaceID, ObjectiveID: plan.ObjectiveID, AttemptID: plan.AttemptID, InputHash: plan.InputHash, ClaimToken: strings.Repeat("A", 43)}
		require.NoError(t, resolved.ValidateFor(ref))
		require.NoError(t, input.Validate())
		for name, value := range map[string]any{c.name + "-assessment.json": input, c.name + "-input.json": resolved, c.name + "-reference.json": ref} {
			data, err := json.MarshalIndent(value, "", "  ")
			require.NoError(t, err)
			f, err := os.OpenFile(filepath.Join(output, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
			require.NoError(t, err)
			_, err = f.Write(data)
			require.NoError(t, err)
			require.NoError(t, f.Close())
		}
		t.Logf("case=%s input_hash=%s snapshot_hash=%s", c.name, plan.InputHash, plan.SnapshotHash)
	}
}

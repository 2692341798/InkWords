package mastery

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type fixedPracticeSource struct{ basis PracticeBasis }

func (source fixedPracticeSource) LoadApprovedPractice(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (PracticeBasis, error) {
	return source.basis, nil
}

func practiceBasisFixture() PracticeBasis {
	basis := PracticeBasis{WorkspaceID: uuid.New(), Title: "权威批准稿题目"}
	chapter, revision := uuid.New(), uuid.New()
	arc := sharedtextbook.DefaultLearningArc()
	for i := range arc.Stages {
		arc.Stages[i].Objective = "可观察目标"
		arc.Stages[i].SuccessEvidence = []string{"source-1"}
		arc.Stages[i].RecoveryRoute = "回到上一步"
	}
	set := sharedtextbook.PracticeSet{Version: sharedtextbook.PracticeSetVersion}
	modes := []sharedtextbook.LearningTaskMode{}
	for _, skill := range Skills {
		mode := sharedtextbook.LearningTaskMode(skill)
		modes = append(modes, mode)
		task := sharedtextbook.PracticeTask{ID: "task-" + string(skill), Mode: mode, Prompt: "具体题目 " + string(skill), Variation: "另一请求路径", ExpectedAnswer: "对照保存的路由条件", EvidenceIDs: []string{"source-1"}, Hints: []sharedtextbook.PracticeHint{{Level: 1, Text: "先回忆条件"}, {Level: 2, Text: "检查方法"}, {Level: 3, Text: "再检查路径"}}}
		if skill == Retain {
			task.MinDelayHours = 72
		}
		for _, criterion := range sharedtextbook.PracticeRubricDimensions(mode) {
			criterion.Description = "核对 " + criterion.ID
			task.Rubric = append(task.Rubric, criterion)
		}
		set.Tasks = append(set.Tasks, task)
	}
	basis.Learning = sharedtextbook.LearningProjection{Format: "inkwords.learning-projection.v2", ChapterID: chapter.String(), RevisionID: revision.String(), ContentHash: "sha256:" + strings.Repeat("a", 64), LearningArc: arc, PracticeSet: &set, EvidenceIDs: []string{"source-1"}, Objectives: []sharedtextbook.LearningObjective{{ID: "objective", ChapterID: chapter.String(), Text: "独立完成六维任务", RequiredModes: modes}}}
	return basis
}

func TestApprovedObjectiveUsesServerProjectionAndFailsClosedOnOtherWorkspace(t *testing.T) {
	basis := practiceBasisFixture()
	require.NoError(t, basis.Learning.Validate())
	store := newMemoryStore()
	service := NewService(store).WithPracticeSource(fixedPracticeSource{basis})
	input := ObjectiveInput{ChapterID: "approved-revision:" + basis.Learning.ChapterID + ":" + basis.Learning.RevisionID, Title: "伪造题目", Behavior: "不用看来源"}
	objective, err := service.CreateObjective(context.Background(), basis.WorkspaceID, input)
	require.NoError(t, err)
	require.Equal(t, basis.Title, objective.Title)
	require.Equal(t, basis.Learning.ContentHash, objective.PracticeContentHash)
	projection, err := objective.practiceProjection()
	require.NoError(t, err)
	require.Equal(t, basis.Learning, *projection)
	again, err := service.CreateObjective(context.Background(), basis.WorkspaceID, input)
	require.NoError(t, err)
	require.Equal(t, objective.ID, again.ID)
	_, err = service.CreateObjective(context.Background(), uuid.New(), input)
	require.ErrorContains(t, err, "mismatch")
	_, err = NewService(newMemoryStore()).CreateObjective(context.Background(), basis.WorkspaceID, input)
	require.ErrorContains(t, err, "source is required")
}

func TestPracticeBindingUsesServerTimeAndEnforces72HourDelay(t *testing.T) {
	basis := practiceBasisFixture()
	service := NewService(newMemoryStore()).WithPracticeSource(fixedPracticeSource{basis})
	now := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	objective, err := service.CreateObjective(context.Background(), basis.WorkspaceID, ObjectiveInput{ChapterID: "approved-revision:" + basis.Learning.ChapterID + ":" + basis.Learning.RevisionID})
	require.NoError(t, err)
	attempt := Attempt{Skill: Explain, Answer: "按方法再按路径查找", Correct: true, Independent: true, Confidence: 4, At: now.Add(365 * 24 * time.Hour), PracticeTaskID: "task-explain", PracticeContentHash: basis.Learning.ContentHash}
	first, err := bindPracticeAttempt(objective, nil, attempt, now)
	require.NoError(t, err)
	require.Equal(t, now, first.At)
	previous := []Attempt{first}
	attempt.Skill = Retain
	attempt.PracticeTaskID = "task-retain"
	_, err = bindPracticeAttempt(objective, previous, attempt, now.Add(time.Hour))
	require.ErrorIs(t, err, ErrPracticeNotDue)
	attempt.PracticeContentHash = "sha256:" + strings.Repeat("b", 64)
	_, err = bindPracticeAttempt(objective, previous, attempt, now.Add(72*time.Hour))
	require.ErrorIs(t, err, ErrPracticeMismatch)
	attempt.PracticeContentHash = basis.Learning.ContentHash
	attempt.PracticeTaskID = "task-explain"
	_, err = bindPracticeAttempt(objective, previous, attempt, now.Add(72*time.Hour))
	require.ErrorIs(t, err, ErrPracticeMismatch)
	attempt.PracticeTaskID = "task-retain"
	_, err = bindPracticeAttempt(objective, previous, attempt, now.Add(72*time.Hour))
	require.NoError(t, err)
	due := enforcePracticeDue(objective, first, DueTask{Skill: Retain, DueAt: first.At})
	require.Equal(t, first.At.Add(72*time.Hour), due.DueAt)
}

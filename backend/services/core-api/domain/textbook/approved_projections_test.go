package textbook

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestBuildApprovedRevisionProjectionsUsesOnlyTheApprovedRevision(t *testing.T) {
	chapterID, revisionID := uuid.New(), uuid.New()
	document, err := json.Marshal(struct {
		Format string `json:"format"`
		Sample struct {
			LearningArc sharedtextbook.LearningArc `json:"learning_arc"`
		} `json:"sample"`
		VideoRunbook sharedtextbook.VideoRunbookProjection `json:"video_runbook"`
	}{Format: "inkwords.textbook.sample.v1", Sample: struct {
		LearningArc sharedtextbook.LearningArc `json:"learning_arc"`
	}{LearningArc: validProjectionLearningArc()}, VideoRunbook: validProjectionRunbook()})
	require.NoError(t, err)
	chapter := Chapter{ID: chapterID, Title: "请求生命周期"}
	revision := ChapterRevision{ID: revisionID, ChapterID: chapterID, Kind: sharedtextbook.RevisionKindApproved, Markdown: "# 批准母稿", ContentHash: strings.Repeat("a", 64), DocumentJSON: datatypes.JSON(document)}

	projections, err := BuildApprovedRevisionProjections(chapter, revision)

	require.NoError(t, err)
	require.Equal(t, revisionID.String(), projections.Blog.RevisionID)
	require.Equal(t, revision.Markdown, projections.Blog.Markdown)
	require.Equal(t, revisionID.String(), projections.Learning.RevisionID)
	require.Equal(t, chapterID.String(), projections.Learning.Objectives[0].ChapterID)
	require.Len(t, projections.Learning.Objectives[0].RequiredModes, 6)
	require.Equal(t, "inkwords.video-runbook.v1", projections.VideoRunbook.Format)
}

func TestBuildApprovedRevisionProjectionsRejectsNonApprovedOrMalformedDataWithoutMutation(t *testing.T) {
	chapterID := uuid.New()
	chapter := Chapter{ID: chapterID, Title: "章节"}
	revision := ChapterRevision{ID: uuid.New(), ChapterID: chapterID, Kind: sharedtextbook.RevisionKindCandidate, Markdown: "# 候选", ContentHash: strings.Repeat("b", 64), DocumentJSON: datatypes.JSON(`{"format":"inkwords.textbook.sample.v1"}`)}
	before := revision

	_, err := BuildApprovedRevisionProjections(chapter, revision)

	require.ErrorIs(t, err, ErrInvalidState)
	require.Equal(t, before, revision, "projection construction is read-only")
	revision.Kind = sharedtextbook.RevisionKindApproved
	_, err = BuildApprovedRevisionProjections(chapter, revision)
	require.ErrorIs(t, err, ErrInvalidState)
}

func validProjectionLearningArc() sharedtextbook.LearningArc {
	arc := sharedtextbook.DefaultLearningArc()
	for index := range arc.Stages {
		arc.Stages[index].Objective = "完成一个可观察的学习步骤"
		arc.Stages[index].SuccessEvidence = []string{"reader-note"}
		arc.Stages[index].RecoveryRoute = "回到上一步并重新尝试"
	}
	return arc
}

func validProjectionRunbook() sharedtextbook.VideoRunbookProjection {
	return sharedtextbook.VideoRunbookProjection{
		Format: "inkwords.video-runbook.v1", Stack: "Gin", ObservationGoal: "观察路由登记", Recommendation: sharedtextbook.DemonstrationToolRecommendation{Primary: "浏览器开发者工具", Alternative: "终端", Reason: "保留人工观察", ManualCaptureRequired: true},
		Steps:            []sharedtextbook.VideoRunbookStep{{Tool: "浏览器开发者工具", ToolVersion: "当前稳定版", StartState: "教学页已打开", Action: "打开面板", ShortcutOrMenu: "F12", Input: "选择 Performance", ExpectedView: "时间线可见", Narration: "观察数据", CapturePoint: "时间线", Recovery: "从菜单打开", CompletionSignal: "面板已显示"}},
		CaptureChecklist: []string{"记录工具版本"}, ManualCapturePending: true, VerificationStatus: sharedtextbook.ArtifactStatusUnverified,
	}
}

func TestApprovedPracticeProjectionRejectsTasksChangedOutsideTheManuscript(t *testing.T) {
	chapter := Chapter{ID: uuid.New(), Title: "路由练习"}
	set := sharedtextbook.PracticeSet{Version: sharedtextbook.PracticeSetVersion}
	for _, mode := range []sharedtextbook.LearningTaskMode{sharedtextbook.LearningTaskExplain, sharedtextbook.LearningTaskComplete, sharedtextbook.LearningTaskReproduce, sharedtextbook.LearningTaskTransfer, sharedtextbook.LearningTaskDiagnose, sharedtextbook.LearningTaskRetain} {
		task := sharedtextbook.PracticeTask{ID: string(mode), Mode: mode, Prompt: string(mode) + "：解释方法与路径的作用。", Variation: "改变前缀后重新判断路径。", ExpectedAnswer: "方法选树，路径用于查找。", EvidenceIDs: []string{"source-1"}, Hints: []sharedtextbook.PracticeHint{{Level: 1, Text: "回想两个条件。"}, {Level: 2, Text: "先选方法树。"}, {Level: 3, Text: "再检查完整路径。"}}}
		for _, criterion := range sharedtextbook.PracticeRubricDimensions(mode) {
			criterion.Description = "根据题目说明" + criterion.ID
			task.Rubric = append(task.Rubric, criterion)
		}
		if mode == sharedtextbook.LearningTaskRetain {
			task.MinDelayHours = 24
		}
		set.Tasks = append(set.Tasks, task)
	}
	document := map[string]any{"format": "inkwords.textbook.sample.v1", "sample": map[string]any{"learning_arc": validProjectionLearningArc(), "practice_set": set, "evidence_ids": []string{"source-1"}}, "video_runbook": validProjectionRunbook()}
	encoded, err := json.Marshal(document)
	require.NoError(t, err)
	revision := ChapterRevision{ID: uuid.New(), ChapterID: chapter.ID, Kind: sharedtextbook.RevisionKindApproved, Markdown: "# 批准稿\n\n" + sharedtextbook.RenderPracticeSet(set), ContentHash: strings.Repeat("a", 64), DocumentJSON: encoded}
	projection, err := BuildApprovedRevisionProjections(chapter, revision)
	require.NoError(t, err)
	require.Equal(t, "inkwords.learning-projection.v2", projection.Learning.Format)
	require.Equal(t, set, *projection.Learning.PracticeSet)
	set.Tasks[0].ExpectedAnswer = "只改 JSON 的答案不能混入批准稿。"
	document["sample"].(map[string]any)["practice_set"] = set
	revision.DocumentJSON, err = json.Marshal(document)
	require.NoError(t, err)
	_, err = BuildApprovedRevisionProjections(chapter, revision)
	require.ErrorIs(t, err, ErrInvalidState)
}

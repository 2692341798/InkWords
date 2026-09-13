package mastery

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type assessmentSourceFixture struct {
	fixedPracticeSource
	projection sharedtextbook.PracticeEvidenceProjection
	calls      int
}

func (source *assessmentSourceFixture) LoadApprovedAssessmentEvidence(_ context.Context, _, _, _ uuid.UUID, _ string) (sharedtextbook.PracticeEvidenceProjection, error) {
	source.calls++
	return source.projection, nil
}

func TestPrepareAssessmentUsesOnlySavedAnswerFrozenRubricAndSelectedSources(t *testing.T) {
	basis := practiceBasisFixture()
	source := &assessmentSourceFixture{fixedPracticeSource: fixedPracticeSource{basis}}
	store := newMemoryStore()
	service := NewService(store).WithPracticeSource(source)
	ctx := context.Background()
	objective, err := service.CreateObjective(ctx, basis.WorkspaceID, ObjectiveInput{ChapterID: "approved-revision:" + basis.Learning.ChapterID + ":" + basis.Learning.RevisionID})
	require.NoError(t, err)
	task := basis.Learning.PracticeSet.Tasks[2]
	sessionID, attemptID := uuid.New(), uuid.New()
	attempt := AttemptRecord{ID: attemptID, ObjectiveID: objective.ID, PracticeSessionID: &sessionID, Skill: string(task.Mode), PracticeTaskID: task.ID, PracticeContentHash: basis.Learning.ContentHash, Answer: "我声称所有测试已通过，但没有执行记录。"}
	store.attempts[objective.ID] = []AttemptRecord{attempt}
	excerpt := "方法与路径共同决定处理函数。"
	snapshot := sharedtextbook.SourceSnapshot{ID: "snapshot-1", SourceID: "source-owner", Kind: sharedtextbook.SourceKindGitRepository, Role: sharedtextbook.SourceRolePrimary, Locator: "https://github.com/gin-gonic/gin", ResolvedVersion: strings.Repeat("a", 40), ContentHash: "sha256:snapshot", CapturedAt: time.Now().UTC()}
	ref := sharedtextbook.EvidenceRef{ID: task.EvidenceIDs[0], SnapshotID: snapshot.ID, DocumentID: "document-1", ChunkID: "chunk-1", ContentHash: "sha256:chunk", SourceRole: snapshot.Role, Confidence: sharedtextbook.EvidenceConfidenceDocumented, Locator: sharedtextbook.EvidenceLocator{Path: "gin.go", StartLine: 1, EndLine: 2}}
	source.projection = sharedtextbook.PracticeEvidenceProjection{Format: "inkwords.practice-evidence.v1", WorkspaceID: basis.WorkspaceID.String(), ChapterID: basis.Learning.ChapterID, RevisionID: basis.Learning.RevisionID, ContentHash: basis.Learning.ContentHash, TaskID: task.ID, Sources: []sharedtextbook.PracticeSourceExcerpt{{Reference: ref, Snapshot: snapshot, Excerpt: excerpt, ExcerptHash: sharedtextbook.PracticeExcerptHash(excerpt)}}}
	prepared, err := service.PrepareAssessmentInput(ctx, basis.WorkspaceID, objective.ID, attemptID)
	require.NoError(t, err)
	require.Equal(t, attempt.Answer, prepared.Answer)
	require.Equal(t, task.Rubric, prepared.Rubric)
	encoded, err := json.Marshal(prepared)
	require.NoError(t, err)
	var wire map[string]any
	require.NoError(t, json.Unmarshal(encoded, &wire))
	require.Equal(t, map[string]any{"task_id": task.ID, "practice_content_hash": basis.Learning.ContentHash, "expected_answer": task.ExpectedAnswer}, wire["task_reference"], "the grader must receive the answer key from the same frozen task")
	require.Contains(t, prepared.Prompt, task.Variation)
	require.Empty(t, prepared.ArtifactHash)
	require.Equal(t, "source", prepared.Evidence[0].Kind)
	require.Empty(t, prepared.Evidence[0].VerificationRunID)
	originalHash := AssessmentInputHash(prepared)
	repeat, err := service.PrepareAssessmentInput(ctx, basis.WorkspaceID, objective.ID, attemptID)
	require.NoError(t, err)
	require.Equal(t, originalHash, AssessmentInputHash(repeat))
	calls := source.calls
	_, err = service.PrepareAssessmentInput(ctx, uuid.New(), objective.ID, attemptID)
	require.Error(t, err)
	_, err = service.PrepareAssessmentInput(ctx, basis.WorkspaceID, objective.ID, uuid.New())
	require.Error(t, err)
	require.Equal(t, calls, source.calls, "ownership and saved attempt are checked before fetching sources")
	store.attempts[objective.ID][0].LearnerArtifactHash = "sha256:" + strings.Repeat("a", 64)
	_, err = service.PrepareAssessmentInput(ctx, basis.WorkspaceID, objective.ID, attemptID)
	require.Error(t, err, "missing code storage must prevent grading")
	require.Equal(t, calls, source.calls, "text-only input must not silently ignore submitted source files")
	store.attempts[objective.ID][0].LearnerArtifactHash = ""
	for _, change := range []func(*sharedtextbook.PracticeEvidenceProjection){
		func(p *sharedtextbook.PracticeEvidenceProjection) { p.ContentHash = "sha256:changed" },
		func(p *sharedtextbook.PracticeEvidenceProjection) { p.WorkspaceID = uuid.New().String() },
		func(p *sharedtextbook.PracticeEvidenceProjection) { p.TaskID = "other-task" },
		func(p *sharedtextbook.PracticeEvidenceProjection) { p.Sources[0].Excerpt = "被改写的原文" },
		func(p *sharedtextbook.PracticeEvidenceProjection) { p.Sources[0].Reference.ID = "unrequested-source" },
	} {
		original := source.projection
		source.projection.Sources = append([]sharedtextbook.PracticeSourceExcerpt(nil), original.Sources...)
		change(&source.projection)
		_, err = service.PrepareAssessmentInput(ctx, basis.WorkspaceID, objective.ID, attemptID)
		require.Error(t, err)
		source.projection = original
	}
}

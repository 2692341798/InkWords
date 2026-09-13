package assessment

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	app "inkwords-backend/services/review-service/app/masteryassessment"
	"inkwords-backend/services/review-service/domain/mastery"
	"inkwords-backend/shared/kernel/generation"
	textbook "inkwords-backend/shared/kernel/textbook"
	platformpostgres "inkwords-backend/shared/platform/postgres"
)

type codePracticeSource struct{ approvedPracticeFixture }

func (source codePracticeSource) LoadApprovedAssessmentEvidence(_ context.Context, owner, chapter, revision uuid.UUID, task string) (textbook.PracticeEvidenceProjection, error) {
	excerpt := "Gin 根据方法与路径选择处理函数。"
	snapshot := textbook.SourceSnapshot{ID: "snapshot-1", SourceID: "gin", Kind: textbook.SourceKindGitRepository, Role: textbook.SourceRolePrimary, Locator: "https://github.com/gin-gonic/gin", ResolvedVersion: strings.Repeat("a", 40), ContentHash: "sha256:snapshot", CapturedAt: time.Now().UTC()}
	ref := textbook.EvidenceRef{ID: "source-1", SnapshotID: snapshot.ID, DocumentID: "document-1", ChunkID: "chunk-1", ContentHash: "sha256:chunk", SourceRole: snapshot.Role, Confidence: textbook.EvidenceConfidenceDocumented, Locator: textbook.EvidenceLocator{Path: "gin.go", StartLine: 1, EndLine: 2}}
	return textbook.PracticeEvidenceProjection{Format: "inkwords.practice-evidence.v1", WorkspaceID: owner.String(), ChapterID: chapter.String(), RevisionID: revision.String(), ContentHash: source.basis.Learning.ContentHash, TaskID: task, Sources: []textbook.PracticeSourceExcerpt{{Reference: ref, Snapshot: snapshot, Excerpt: excerpt, ExcerptHash: textbook.PracticeExcerptHash(excerpt)}}}, nil
}

func TestPostgresCodeAssessmentFreezesGradesCorrectsAndAppliesUnknownRuntime(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("code_grading"), postgrescontainer.WithUsername("inkwords"), postgrescontainer.WithPassword("test-only-password"), testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := platformpostgres.InitReview(dsn)
	require.NoError(t, err)
	history, err := mastery.NewGormStore(db)
	require.NoError(t, err)
	basis := applicationPractice()
	learner := mastery.NewService(history).WithPracticeSource(codePracticeSource{approvedPracticeFixture{basis}})
	objective, err := learner.CreateObjective(ctx, basis.WorkspaceID, mastery.ObjectiveInput{ChapterID: "approved-revision:" + basis.Learning.ChapterID + ":" + basis.Learning.RevisionID})
	require.NoError(t, err)
	session, err := learner.BeginPracticeSession(ctx, basis.WorkspaceID, objective.ID, mastery.Reproduce)
	require.NoError(t, err)
	files := []textbook.LearnerCodeFile{{Path: "router.go", Content: "package main\r\nfunc Match() bool { return true }\r\n"}}
	_, err = learner.RecordAttempt(ctx, basis.WorkspaceID, objective.ID, mastery.Attempt{Skill: mastery.Reproduce, Answer: "我声称测试全部通过。", PracticeSessionID: session.ID, PracticeTaskID: session.PracticeTaskID, PracticeContentHash: session.PracticeContentHash, Correct: true, Independent: true, Confidence: 4, LearnerFiles: files})
	require.NoError(t, err)
	records, err := history.ListAttempts(ctx, objective.ID)
	require.NoError(t, err)
	require.Len(t, records, 1)
	original := records[0]
	input, err := learner.PrepareAssessmentInput(ctx, basis.WorkspaceID, objective.ID, original.ID)
	require.NoError(t, err)
	require.Equal(t, files, input.LearnerArtifact.Files)
	require.Empty(t, input.ArtifactHash)
	require.Equal(t, mastery.AssessmentJudgmentContractVersion, input.ContractVersion())
	require.NotNil(t, input.TaskReference)
	require.Equal(t, session.PracticeTaskID, input.TaskReference.TaskID)
	require.Equal(t, session.PracticeContentHash, input.TaskReference.PracticeContentHash)
	finding := mastery.AssessmentFinding{Text: "静态代码可读，缺少真实运行依据。", EvidenceIDs: []string{"source-1"}}
	feedback := mastery.AssessmentFeedback{CorrectPoints: []mastery.AssessmentFinding{}, MissingPoints: []mastery.AssessmentFinding{finding}, Misconceptions: []mastery.AssessmentFinding{}, NextHint: finding, Remediation: []mastery.AssessmentFinding{finding}}
	for _, criterion := range input.Rubric {
		score := 3
		item := mastery.CriterionAssessment{ID: criterion.ID, Score: &score, Reason: "依据原文件静态评阅", AnswerPath: "router.go", AnswerQuote: "return true", EvidenceIDs: []string{"source-1"}}
		if criterion.RequiresRuntime {
			item.Score = nil
		}
		feedback.Criteria = append(feedback.Criteria, item)
	}
	decisions := make([]mastery.AssessmentDecision, len(input.Rubric))
	for index, criterion := range input.Rubric {
		status := "satisfied"
		if criterion.RequiresRuntime {
			status = "unknown"
		}
		decisions[index] = mastery.AssessmentDecision{CriterionID: criterion.ID, Requirement: criterion.Description, Status: status, Gaps: []mastery.AssessmentDecisionGap{}}
	}
	judgments := make([]mastery.AssessmentJudgment, len(feedback.Criteria))
	for index, item := range feedback.Criteria {
		judgments[index] = mastery.AssessmentJudgment{CriterionID: item.ID, Score: item.Score, Status: decisions[index].Status, AnswerQuote: item.AnswerQuote, AnswerPath: item.AnswerPath, EvidenceIDs: item.EvidenceIDs, Gaps: []mastery.AssessmentJudgmentGap{}}
		judgments[index].AnswerQuote = ""
		if item.Score == nil {
			judgments[index].UnknownReason = "没有实际运行记录"
			judgments[index].AnswerPath = ""
			judgments[index].EvidenceIDs = []string{}
		} else {
			judgments[index].AnswerSpan = &mastery.AssessmentCodeSpan{Format: mastery.AssessmentCodeSpanFormat, StartLine: 2, EndLine: 2}
		}
	}
	payload, err := json.Marshal(struct {
		Judgments   []mastery.AssessmentJudgment `json:"judgments"`
		NextHint    mastery.AssessmentFinding    `json:"next_hint"`
		Remediation []mastery.AssessmentFinding  `json:"remediation"`
	}{judgments, feedback.NextHint, feedback.Remediation})
	require.NoError(t, err)
	// The provider schema requires an explicit empty path for unknown items;
	// persisted domain values omit that empty optional field for legacy reads.
	var wire map[string]any
	require.NoError(t, json.Unmarshal(payload, &wire))
	for index, item := range wire["judgments"].([]any) {
		item.(map[string]any)["answer_path"] = judgments[index].AnswerPath
	}
	payload, err = json.Marshal(wire)
	require.NoError(t, err)
	port := &controlledPort{started: make(chan struct{}, 1), release: make(chan struct{}), output: generation.Result{Provider: "test", Model: "test", Output: string(payload)}}
	close(port.release)
	store := NewStore(db)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	generator := app.NewGenerator(port, "test", "test")
	service := app.NewService(learner, generator, store)
	require.NoError(t, service.Recover(ctx))
	preview, err := service.Preview(ctx, basis.WorkspaceID, objective.ID, original.ID)
	require.NoError(t, err)
	require.Zero(t, port.calls.Load())
	request := app.StartInput{RequestID: uuid.New(), ExpectedInputHash: preview.InputHash, ExpectedRequestHash: preview.RequestHash}
	job, err := service.Start(ctx, basis.WorkspaceID, objective.ID, original.ID, request)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		job, err = service.Read(ctx, basis.WorkspaceID, objective.ID, job.ID)
		return err == nil && job.Status != "running"
	}, 3*time.Second, 10*time.Millisecond)
	require.Equal(t, "succeeded", job.Status, "terminal code: %s", job.ErrorCode)
	require.Equal(t, mastery.AssessmentJudgmentContractVersion, job.Result.Contract)
	require.Equal(t, judgments, job.Result.Judgments)
	require.Equal(t, "func Match() bool { return true }\r\n", job.Result.Feedback.Criteria[0].AnswerQuote)
	require.Equal(t, decisions, job.Result.Decisions)
	require.Equal(t, input.TaskReference, job.Input.TaskReference)
	require.Equal(t, files, job.Input.LearnerArtifact.Files)
	repeated, err := service.Start(ctx, basis.WorkspaceID, objective.ID, original.ID, request)
	require.NoError(t, err)
	require.Equal(t, job.ID, repeated.ID)
	change := feedback.Criteria[0]
	score := 2
	change.Score = &score
	corrected, err := store.Correct(ctx, basis.WorkspaceID, objective.ID, job.ID, app.CorrectionInput{ID: uuid.New(), PreviousHash: job.EffectiveHash, Reason: "核对原代码", Changes: []mastery.CriterionAssessment{change}})
	require.NoError(t, err)
	require.Equal(t, "router.go", corrected.EffectiveFeedback.Criteria[0].AnswerPath)
	require.Equal(t, 2, *corrected.EffectiveFeedback.Criteria[0].Score)
	require.Equal(t, decisions, corrected.Result.Decisions, "user correction does not rewrite model decisions")
	applied, err := store.Apply(ctx, basis.WorkspaceID, objective.ID, job.ID, app.ApplyInput{ID: uuid.New(), ExpectedFeedbackHash: corrected.EffectiveHash})
	require.NoError(t, err)
	require.Equal(t, files, applied.AppliedAssessment.Input.LearnerArtifact.Files)
	require.Equal(t, input.TaskReference, applied.AppliedAssessment.Input.TaskReference)
	replay, err := mastery.ReplayAssessmentApplications(objective, records, []mastery.AssessmentApplication{*applied.AppliedAssessment})
	require.NoError(t, err)
	require.Equal(t, mastery.AssessmentNeedsPractice, replay[0].AssessmentOutcome, "explicit static correction is weak even while runtime stays unknown")
	for _, criterion := range input.Rubric {
		if criterion.RequiresRuntime {
			for _, assessed := range applied.AppliedAssessment.Feedback.Criteria {
				if assessed.ID == criterion.ID {
					require.Nil(t, assessed.Score)
					require.Empty(t, assessed.EvidenceIDs)
				}
			}
		}
	}
	saved, err := history.ListAttempts(ctx, objective.ID)
	require.NoError(t, err)
	require.Equal(t, records, saved)
	require.EqualValues(t, 1, port.calls.Load(), "correction, apply and recovery never re-call provider")
	// Even an internally constructed request cannot swap or omit the saved files.
	for _, omit := range []bool{false, true} {
		changed := input
		if omit {
			changed.LearnerArtifact = nil
		} else {
			copy := *input.LearnerArtifact
			copy.Files = append([]textbook.LearnerCodeFile(nil), copy.Files...)
			copy.Files[0].Content += "// different"
			copy.SnapshotHash, err = textbook.LearnerArtifactHash(copy)
			require.NoError(t, err)
			changed.LearnerArtifact = &copy
		}
		changedPreview, err := generator.Preview(changed)
		require.NoError(t, err)
		_, _, err = store.Create(ctx, app.Job{ID: uuid.New(), WorkspaceID: basis.WorkspaceID, ObjectiveID: objective.ID, AttemptID: original.ID, RequestID: uuid.New(), Input: changed, Preview: changedPreview, CreatedAt: time.Now().UTC()})
		require.Error(t, err)
	}
	var plan string
	require.NoError(t, db.Raw("EXPLAIN (FORMAT JSON) SELECT count(*) FROM mastery_attempts a JOIN mastery_objectives o ON o.id=a.objective_id WHERE a.id=? AND o.id=? AND o.workspace_id=? AND o.deleted_at IS NULL AND a.answer=? AND a.skill=? AND a.learner_artifact_hash=?", original.ID, objective.ID, basis.WorkspaceID, original.Answer, original.Skill, original.LearnerArtifactHash).Row().Scan(&plan))
	require.Contains(t, plan, "Index Scan")
	require.Contains(t, plan, "learner_artifact_hash")
	t.Log("assessment admission", plan)
}

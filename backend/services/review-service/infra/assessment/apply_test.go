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
	"inkwords-backend/shared/platform/postgres/migrations"
)

type approvedPracticeFixture struct{ basis mastery.PracticeBasis }

func (s approvedPracticeFixture) LoadApprovedPractice(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (mastery.PracticeBasis, error) {
	return s.basis, nil
}

func applicationPractice() mastery.PracticeBasis {
	chapter, revision := uuid.NewString(), uuid.NewString()
	arc := textbook.DefaultLearningArc()
	for i := range arc.Stages {
		arc.Stages[i].Objective = "解释路由"
		arc.Stages[i].SuccessEvidence = []string{"source-1"}
		arc.Stages[i].RecoveryRoute = "回到方法和路径"
	}
	set := textbook.PracticeSet{Version: textbook.PracticeSetVersion}
	modes := []textbook.LearningTaskMode{}
	for _, skill := range mastery.Skills {
		mode := textbook.LearningTaskMode(skill)
		modes = append(modes, mode)
		task := textbook.PracticeTask{ID: "task-" + string(skill), Mode: mode, Prompt: "说明请求匹配 " + string(skill), Variation: "换一种方法", ExpectedAnswer: "按方法与路径查找", EvidenceIDs: []string{"source-1"}, Hints: []textbook.PracticeHint{{Level: 1, Text: "回忆入口"}, {Level: 2, Text: "找到方法树"}, {Level: 3, Text: "逐段匹配路径"}}}
		if skill == mastery.Retain {
			task.MinDelayHours = 72
		}
		for _, criterion := range textbook.PracticeRubricDimensions(mode) {
			criterion.Description = "检查 " + criterion.ID
			task.Rubric = append(task.Rubric, criterion)
		}
		set.Tasks = append(set.Tasks, task)
	}
	return mastery.PracticeBasis{WorkspaceID: uuid.New(), Title: "评分应用事务夹具", Learning: textbook.LearningProjection{Format: "inkwords.learning-projection.v2", ChapterID: chapter, RevisionID: revision, ContentHash: "sha256:" + strings.Repeat("a", 64), LearningArc: arc, PracticeSet: &set, EvidenceIDs: []string{"source-1"}, Objectives: []textbook.LearningObjective{{ID: "objective", ChapterID: chapter, Text: "完成六维题目", RequiredModes: modes}}}}
}

func TestPostgresExplicitAssessmentApplyReplaysAndSurvivesLaterPractice(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("apply_test"), postgrescontainer.WithUsername("inkwords"), postgrescontainer.WithPassword("test-only-password"), testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := platformpostgres.InitReview(dsn)
	require.NoError(t, err)
	historyStore, err := mastery.NewGormStore(db)
	require.NoError(t, err)
	basis := applicationPractice()
	require.NoError(t, basis.Learning.Validate())
	learner := mastery.NewService(historyStore).WithPracticeSource(approvedPracticeFixture{basis})
	objective, err := learner.CreateObjective(ctx, basis.WorkspaceID, mastery.ObjectiveInput{ChapterID: "approved-revision:" + basis.Learning.ChapterID + ":" + basis.Learning.RevisionID})
	require.NoError(t, err)
	answer := "先按方法选择树，再按路径匹配处理函数。"
	recordAnswer := func(skill mastery.Skill) {
		session, err := learner.BeginPracticeSession(ctx, basis.WorkspaceID, objective.ID, skill)
		require.NoError(t, err)
		_, err = learner.RecordAttempt(ctx, basis.WorkspaceID, objective.ID, mastery.Attempt{Skill: skill, Answer: answer, PracticeTaskID: session.PracticeTaskID, PracticeContentHash: session.PracticeContentHash, PracticeSessionID: session.ID, Correct: true, Independent: true, Confidence: 4, Took: time.Minute})
		require.NoError(t, err)
	}
	recordAnswer(mastery.Explain)
	records, err := historyStore.ListAttempts(ctx, objective.ID)
	require.NoError(t, err)
	original := records[0]
	excerpt := "Gin 按方法和路径选择处理函数。"
	input := mastery.AssessmentInput{ObjectiveID: objective.ID.String(), AttemptID: original.ID.String(), RevisionID: basis.Learning.RevisionID, Skill: mastery.Explain, Prompt: basis.Learning.PracticeSet.Tasks[0].Prompt, Answer: answer, Rubric: basis.Learning.PracticeSet.Tasks[0].Rubric, Evidence: []mastery.AssessmentEvidence{{ID: "source-1", Kind: "source", Excerpt: excerpt, ContentHash: textbook.PracticeExcerptHash(excerpt)}}}
	finding := mastery.AssessmentFinding{Text: "解释缺少匹配边界", EvidenceIDs: []string{"source-1"}}
	feedback := mastery.AssessmentFeedback{CorrectPoints: []mastery.AssessmentFinding{}, MissingPoints: []mastery.AssessmentFinding{finding}, Misconceptions: []mastery.AssessmentFinding{}, NextHint: finding, Remediation: []mastery.AssessmentFinding{finding}}
	for _, criterion := range input.Rubric {
		score := 2
		feedback.Criteria = append(feedback.Criteria, mastery.CriterionAssessment{ID: criterion.ID, Score: &score, Reason: "边界不完整", AnswerQuote: answer, EvidenceIDs: []string{"source-1"}})
	}
	data, err := json.Marshal(feedback)
	require.NoError(t, err)
	port := &controlledPort{started: make(chan struct{}, 1), release: make(chan struct{}), output: generation.Result{Provider: "test", Model: "test", Output: string(data)}}
	close(port.release)
	store := NewStore(db)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	service := app.NewService(fixedInput{input}, app.NewGenerator(port, "test", "test"), store)
	require.NoError(t, service.Recover(ctx))
	preview, err := service.Preview(ctx, basis.WorkspaceID, objective.ID, original.ID)
	require.NoError(t, err)
	job, err := service.Start(ctx, basis.WorkspaceID, objective.ID, original.ID, app.StartInput{RequestID: uuid.New(), ExpectedInputHash: preview.InputHash, ExpectedRequestHash: preview.RequestHash})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		job, err = service.Read(ctx, basis.WorkspaceID, objective.ID, job.ID)
		return err == nil && job.Status == "succeeded"
	}, time.Second, 10*time.Millisecond)
	var before mastery.Schedule
	require.NoError(t, db.First(&before, "objective_id = ?", objective.ID).Error)
	require.Nil(t, job.AppliedAssessment)
	request := app.ApplyInput{ID: uuid.New(), ExpectedFeedbackHash: job.EffectiveHash}
	stale := request
	stale.ExpectedFeedbackHash = "sha256:stale"
	_, err = service.Apply(ctx, basis.WorkspaceID, objective.ID, job.ID, stale)
	require.ErrorIs(t, err, app.ErrConflict)
	_, err = service.Apply(ctx, uuid.New(), objective.ID, job.ID, request)
	require.ErrorIs(t, err, app.ErrNotFound)
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { _, err := service.Apply(ctx, basis.WorkspaceID, objective.ID, job.ID, request); results <- err }()
	}
	require.NoError(t, <-results)
	require.NoError(t, <-results)
	job, err = service.Read(ctx, basis.WorkspaceID, objective.ID, job.ID)
	require.NoError(t, err)
	require.Equal(t, request.ID, job.AppliedAssessment.ID)
	require.Equal(t, 1, job.AppliedAssessment.SequenceNo)
	require.Equal(t, mastery.Explain, job.Schedule.Skill)
	require.True(t, job.Schedule.DueAt.Before(before.DueAt))
	events, err := historyStore.ListAssessmentApplications(ctx, objective.ID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	recordAnswer(mastery.Complete)
	projected, err := historyStore.ReplayAssessedSchedule(ctx, objective)
	require.NoError(t, err)
	var after mastery.Schedule
	require.NoError(t, db.First(&after, "objective_id = ?", objective.ID).Error)
	require.Equal(t, mastery.AssessmentAlgorithmVersion, after.AlgorithmVersion)
	require.Equal(t, projected.NextSkill, after.NextSkill)
	require.WithinDuration(t, projected.DueAt, after.DueAt, time.Microsecond)
	require.Equal(t, string(mastery.Explain), after.NextSkill, "later answer must keep the applied failure")
	// A correction is advisory until its exact new view is explicitly applied.
	score := 4
	changes := append([]mastery.CriterionAssessment(nil), feedback.Criteria...)
	for i := range changes {
		changes[i].Score = &score
	}
	job, err = service.Correct(ctx, basis.WorkspaceID, objective.ID, job.ID, app.CorrectionInput{ID: uuid.New(), PreviousHash: job.EffectiveHash, Reason: "核对原文后纠正", Changes: changes})
	require.NoError(t, err)
	require.NotEqual(t, job.EffectiveHash, job.AppliedAssessment.FeedbackHash)
	require.Equal(t, mastery.Explain, job.Schedule.Skill)
	next := app.ApplyInput{ID: uuid.New(), ExpectedFeedbackHash: job.EffectiveHash, PreviousID: &request.ID}
	job, err = service.Apply(ctx, basis.WorkspaceID, objective.ID, job.ID, next)
	require.NoError(t, err)
	require.Equal(t, 2, job.AppliedAssessment.SequenceNo)
	require.Equal(t, mastery.Reproduce, job.Schedule.Skill)
	job, err = service.Apply(ctx, basis.WorkspaceID, objective.ID, job.ID, request)
	require.NoError(t, err)
	require.Equal(t, next.ID, job.AppliedAssessment.ID, "retry of old response cannot restore old assessment")
	stale = next
	stale.ID = uuid.New()
	_, err = service.Apply(ctx, basis.WorkspaceID, objective.ID, job.ID, stale)
	require.ErrorIs(t, err, app.ErrConflict)
	records, err = historyStore.ListAttempts(ctx, objective.ID)
	require.NoError(t, err)
	require.Equal(t, original, records[0])
	require.Len(t, records, 2)
	require.Equal(t, 2, *job.Result.Feedback.Criteria[0].Score)
	require.EqualValues(t, 1, port.calls.Load(), "apply and correction never call provider")
	sqlDB, err := db.DB()
	require.NoError(t, err)
	var plan string
	require.NoError(t, sqlDB.QueryRowContext(ctx, "EXPLAIN (FORMAT JSON) SELECT id FROM mastery_assessment_applications WHERE objective_id=$1 AND attempt_id=$2 ORDER BY sequence_no DESC LIMIT 1", objective.ID, original.ID).Scan(&plan))
	require.Contains(t, plan, "idx_mastery_assessment_application_attempt")
	t.Log("latest application query:", plan)
	provider, err := migrations.NewReviewProvider(sqlDB)
	require.NoError(t, err)
	_, err = provider.Down(ctx) // v33 has no learner verification runs in this grading fixture.
	require.NoError(t, err)
	_, err = provider.Down(ctx) // v32 has no learner files in this grading fixture.
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.ErrorContains(t, err, "applied assessment evidence")
}

package assessment

import (
	"context"
	"encoding/json"
	"sync/atomic"
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

type fixedInput struct{ input mastery.AssessmentInput }

func (s fixedInput) PrepareAssessmentInput(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (mastery.AssessmentInput, error) {
	return s.input, nil
}

type controlledPort struct {
	calls   atomic.Int32
	started chan struct{}
	release chan struct{}
	output  generation.Result
}

func (p *controlledPort) Generate(ctx context.Context, _ generation.Request) (generation.Result, error) {
	p.calls.Add(1)
	p.started <- struct{}{}
	select {
	case <-p.release:
		return p.output, nil
	case <-ctx.Done():
		return generation.Result{}, ctx.Err()
	}
}

func TestPostgresJobsDeduplicateRecoverCancelAndAppendCorrections(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("assessment_test"), postgrescontainer.WithUsername("inkwords"), postgrescontainer.WithPassword("inkwords-test-password"), testcontainers.WithAdditionalWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := platformpostgres.InitReview(dsn)
	require.NoError(t, err)
	owner, objective, attempt := uuid.New(), uuid.New(), uuid.New()
	answer := "先按方法选择路由树，再按路径查找处理函数。"
	require.NoError(t, db.Exec(`INSERT INTO mastery_objectives(id,workspace_id,chapter_id,title,behavior,required_skills,rubric,key_points,evidence_refs) VALUES(?,?,?,'解释路由','解释路由','["explain"]','["准确"]','["方法"]','["source-1"]')`, objective, owner, "fixture-chapter").Error)
	saveAttempt := func(id uuid.UUID) {
		require.NoError(t, db.Exec(`INSERT INTO mastery_attempts(id,objective_id,skill,answer,correct,independent,hint_count,took_millis,confidence,error_kinds,attempted_at) VALUES(?,?,'explain',?,true,true,0,1000,4,'[]',CURRENT_TIMESTAMP)`, id, objective, answer).Error)
	}
	saveAttempt(attempt)
	excerpt := "Gin 根据方法和路径查找处理函数。"
	input := mastery.AssessmentInput{AttemptID: attempt.String(), ObjectiveID: objective.String(), RevisionID: uuid.New().String(), Skill: mastery.Explain, Prompt: "解释路由查找", Answer: answer, Evidence: []mastery.AssessmentEvidence{{ID: "source-1", Kind: "source", Excerpt: excerpt, ContentHash: textbook.PracticeExcerptHash(excerpt)}}}
	secondExcerpt := "方法决定查询哪棵树，路径决定树中的处理函数。"
	input.Evidence = append(input.Evidence, mastery.AssessmentEvidence{ID: "source-2", Kind: "source", Excerpt: secondExcerpt, ContentHash: textbook.PracticeExcerptHash(secondExcerpt)})
	feedback := mastery.AssessmentFeedback{CorrectPoints: []mastery.AssessmentFinding{}, MissingPoints: []mastery.AssessmentFinding{}, Misconceptions: []mastery.AssessmentFinding{}, NextHint: mastery.AssessmentFinding{Text: "再解释未匹配请求。", EvidenceIDs: []string{"source-1"}}, Remediation: []mastery.AssessmentFinding{{Text: "回看路由查找原文", EvidenceIDs: []string{"source-1"}}}}
	for _, criterion := range textbook.PracticeRubricDimensions(textbook.LearningTaskExplain) {
		criterion.Description = "解释 " + criterion.ID
		input.Rubric = append(input.Rubric, criterion)
		score := 3
		feedback.Criteria = append(feedback.Criteria, mastery.CriterionAssessment{ID: criterion.ID, Score: &score, Reason: "解释了查找步骤。", AnswerQuote: answer, EvidenceIDs: []string{"source-1"}})
	}
	payload, err := json.Marshal(feedback)
	require.NoError(t, err)
	newPort := func() *controlledPort {
		return &controlledPort{started: make(chan struct{}, 1), release: make(chan struct{}), output: generation.Result{Provider: "test-provider", Model: "test-model", Output: string(payload), Usage: generation.Usage{Known: true, InputTokens: 120, OutputTokens: 80}}}
	}
	port := newPort()
	store := NewStore(db)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	service := app.NewService(fixedInput{input}, app.NewGenerator(port, "test-provider", "test-model"), store)
	require.NoError(t, service.Recover(ctx))
	otherExecutor := NewStore(db)
	require.Error(t, otherExecutor.InterruptRunning(ctx), "another local process cannot interrupt this executor")
	require.NoError(t, otherExecutor.Close())
	preview, err := service.Preview(ctx, owner, objective, attempt)
	require.NoError(t, err)
	require.Zero(t, port.calls.Load())
	request := app.StartInput{RequestID: uuid.New(), ExpectedInputHash: preview.InputHash, ExpectedRequestHash: preview.RequestHash}
	stale := request
	stale.ExpectedRequestHash = "sha256:changed"
	_, err = service.Start(ctx, owner, objective, attempt, stale)
	require.ErrorIs(t, err, app.ErrConflict)
	require.Zero(t, port.calls.Load())
	job, err := service.Start(ctx, owner, objective, attempt, request)
	require.NoError(t, err)
	select {
	case <-port.started:
	case <-time.After(time.Second):
		t.Fatal("model call did not start")
	}
	duplicate, err := service.Start(ctx, owner, objective, attempt, request)
	require.NoError(t, err)
	require.Equal(t, job.ID, duplicate.ID)
	close(port.release)
	var completed app.Job
	require.Eventually(t, func() bool {
		completed, err = service.Read(ctx, owner, objective, job.ID)
		return err == nil && completed.Status == "succeeded"
	}, 3*time.Second, 10*time.Millisecond)
	require.EqualValues(t, 1, port.calls.Load())
	require.Equal(t, 120, completed.Result.Usage.InputTokens)
	request.RequestID = uuid.New()
	cached, err := service.Start(ctx, owner, objective, attempt, request)
	require.NoError(t, err)
	require.Equal(t, job.ID, cached.ID)
	require.EqualValues(t, 1, port.calls.Load())
	score := 4
	change := feedback.Criteria[0]
	change.Score = &score
	change.Reason = "我能补充明确的查找边界。"
	change.EvidenceIDs = []string{"source-2"}
	correction := app.CorrectionInput{ID: uuid.New(), PreviousHash: completed.EffectiveHash, Reason: "核对原作答后调整", Changes: []mastery.CriterionAssessment{change}}
	corrected, err := service.Correct(ctx, owner, objective, job.ID, correction)
	require.NoError(t, err)
	require.Len(t, corrected.Corrections, 1)
	require.Equal(t, owner.String(), corrected.Corrections[0].ReviewerID)
	require.Equal(t, 3, *corrected.Result.Feedback.Criteria[0].Score)
	require.Equal(t, 4, *corrected.EffectiveFeedback.Criteria[0].Score)
	require.Equal(t, []string{"source-1"}, corrected.Result.Feedback.Criteria[0].EvidenceIDs)
	require.Equal(t, []string{"source-2"}, corrected.EffectiveFeedback.Criteria[0].EvidenceIDs)
	readBack, err := service.Read(ctx, owner, objective, job.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"source-2"}, readBack.Corrections[0].Changes[0].EvidenceIDs)
	require.Equal(t, []string{"source-2"}, readBack.EffectiveFeedback.Criteria[0].EvidenceIDs)
	again, err := service.Correct(ctx, owner, objective, job.ID, correction)
	require.NoError(t, err)
	require.Len(t, again.Corrections, 1)
	correction.ID = uuid.New()
	_, err = service.Correct(ctx, owner, objective, job.ID, correction)
	require.ErrorIs(t, err, app.ErrConflict)
	_, err = service.Read(ctx, uuid.New(), objective, job.ID)
	require.ErrorIs(t, err, app.ErrNotFound)
	findings := &mastery.AssessmentFindingCorrection{CorrectPoints: []mastery.AssessmentFinding{}, MissingPoints: []mastery.AssessmentFinding{}, Misconceptions: []mastery.AssessmentFinding{}, NextHint: mastery.AssessmentFinding{Text: "方法改变后如何选择树？", EvidenceIDs: []string{"source-2"}}, Remediation: feedback.Remediation}
	adviceCorrection := app.CorrectionInput{ID: uuid.New(), PreviousHash: corrected.EffectiveHash, Reason: "移除超出题目范围的提示并引用匹配来源", Changes: []mastery.CriterionAssessment{}, Findings: findings}
	adviceJob, err := service.Correct(ctx, owner, objective, job.ID, adviceCorrection)
	require.NoError(t, err)
	require.Len(t, adviceJob.Corrections, 2)
	require.Equal(t, feedback.NextHint, adviceJob.Result.Feedback.NextHint)
	require.Equal(t, findings.NextHint, adviceJob.EffectiveFeedback.NextHint)
	require.Equal(t, corrected.EffectiveFeedback.Criteria, adviceJob.EffectiveFeedback.Criteria)
	require.Nil(t, adviceJob.AppliedAssessment, "editing advice never applies or schedules it")
	readAdvice, err := service.Read(ctx, owner, objective, job.ID)
	require.NoError(t, err)
	require.Equal(t, findings, readAdvice.Corrections[1].Findings)
	require.Equal(t, adviceJob.EffectiveHash, readAdvice.EffectiveHash)
	_, err = service.Correct(ctx, owner, objective, job.ID, adviceCorrection)
	require.NoError(t, err, "lost-response retry is idempotent")
	changedAdvice := adviceCorrection
	changedFindings := *findings
	changedFindings.NextHint.Text = "改变同一个请求 ID 的内容"
	changedAdvice.Findings = &changedFindings
	_, err = service.Correct(ctx, owner, objective, job.ID, changedAdvice)
	require.ErrorIs(t, err, app.ErrConflict)
	changedAdvice.ID = uuid.New()
	_, err = service.Correct(ctx, owner, objective, job.ID, changedAdvice)
	require.ErrorIs(t, err, app.ErrConflict, "stale advice snapshot cannot overwrite a newer view")
	changedAdvice.PreviousHash = adviceJob.EffectiveHash
	changedFindings.NextHint.EvidenceIDs = []string{"invented"}
	_, err = service.Correct(ctx, owner, objective, job.ID, changedAdvice)
	require.ErrorIs(t, err, app.ErrConflict)
	require.EqualValues(t, 1, port.calls.Load(), "advice correction never calls the provider")
	attempt2 := uuid.New()
	saveAttempt(attempt2)
	input.AttemptID = attempt2.String()
	port2 := newPort()
	service2 := app.NewService(fixedInput{input}, app.NewGenerator(port2, "test-provider", "test-model"), store)
	preview2, err := service2.Preview(ctx, owner, objective, attempt2)
	require.NoError(t, err)
	request2 := app.StartInput{RequestID: uuid.New(), ExpectedInputHash: preview2.InputHash, ExpectedRequestHash: preview2.RequestHash}
	running, err := service2.Start(ctx, owner, objective, attempt2, request2)
	require.NoError(t, err)
	select {
	case <-port2.started:
	case <-time.After(time.Second):
		t.Fatal("second call did not start")
	}
	cancelled, err := service2.Cancel(ctx, owner, objective, running.ID)
	require.NoError(t, err)
	require.Equal(t, "cancelled", cancelled.Status)
	require.Eventually(t, func() bool {
		current, err := service2.Read(ctx, owner, objective, running.ID)
		return err == nil && current.Result != nil
	}, 3*time.Second, 10*time.Millisecond)
	request2.RequestID = uuid.New()
	_, err = service2.Start(ctx, owner, objective, attempt2, request2)
	require.ErrorIs(t, err, app.ErrConflict)
	// An admitted request abandoned before a known response stays interrupted;
	// recovery never calls a provider or invents token usage.
	orphan := app.Job{ID: uuid.New(), WorkspaceID: owner, ObjectiveID: objective, AttemptID: attempt2, RequestID: uuid.New(), RetryOf: &running.ID, Input: input, Preview: preview2, CreatedAt: time.Now().UTC()}
	_, created, err := store.Create(ctx, orphan)
	require.NoError(t, err)
	require.True(t, created)
	require.NoError(t, service2.Recover(ctx))
	interrupted, err := service2.Read(ctx, owner, objective, orphan.ID)
	require.NoError(t, err)
	require.Equal(t, "interrupted", interrupted.Status)
	require.Nil(t, interrupted.Result)
	require.EqualValues(t, 1, port2.calls.Load())
	port3 := newPort()
	close(port3.release)
	service3 := app.NewService(fixedInput{input}, app.NewGenerator(port3, "test-provider", "test-model"), store)
	request2.RequestID, request2.RetryOf = uuid.New(), &orphan.ID
	retried, err := service3.Start(ctx, owner, objective, attempt2, request2)
	require.NoError(t, err)
	require.NotEqual(t, orphan.ID, retried.ID)
	require.Eventually(t, func() bool {
		current, err := service3.Read(ctx, owner, objective, retried.ID)
		return err == nil && current.Status == "succeeded"
	}, 3*time.Second, 10*time.Millisecond)
	require.EqualValues(t, 1, port3.calls.Load())
	require.Error(t, db.Model(&jobRow{}).Where("id = ?", retried.ID).Updates(map[string]any{"status": "failed", "completed_at": nil}).Error, "NULL completion cannot bypass the terminal-state constraint")
	// No source or original learner answer is altered by corrections or recovery.
	var savedAnswer string
	require.NoError(t, db.Raw("SELECT answer FROM mastery_attempts WHERE id = ?", attempt).Row().Scan(&savedAnswer))
	require.Equal(t, answer, savedAnswer)
	var plan string
	require.NoError(t, db.Raw("EXPLAIN (FORMAT JSON) SELECT id FROM mastery_assessment_jobs WHERE workspace_id = ? AND objective_id = ? AND attempt_id = ? ORDER BY created_at DESC,id DESC LIMIT 1", owner, objective, attempt).Row().Scan(&plan))
	t.Log("latest assessment plan:", plan)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	provider, err := migrations.NewReviewProvider(sqlDB)
	require.NoError(t, err)
	_, err = provider.Down(ctx) // v33/v32/v31 are empty; v30 must preserve jobs.
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.ErrorContains(t, err, "assessment jobs or corrections")
}

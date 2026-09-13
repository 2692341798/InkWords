package mastery

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func assertPersistentPracticeSessions(t *testing.T, ctx context.Context, service *Service, store *GormStore, objective Objective, basis PracticeBasis, now time.Time) {
	t.Helper()
	service.now = func() time.Time { return now }
	first, err := service.BeginPracticeSession(ctx, basis.WorkspaceID, objective.ID, Reproduce)
	require.NoError(t, err)
	resumed, err := service.BeginPracticeSession(ctx, basis.WorkspaceID, objective.ID, Reproduce)
	require.NoError(t, err)
	require.Equal(t, first.ID, resumed.ID)
	_, err = service.RevealPracticeHelp(ctx, basis.WorkspaceID, objective.ID, first.ID, "hint", 2)
	require.Error(t, err)
	now = now.Add(time.Minute)
	shown, err := service.RevealPracticeHelp(ctx, basis.WorkspaceID, objective.ID, first.ID, "hint", 1)
	require.NoError(t, err)
	require.Equal(t, 1, shown.HintCount)
	shown, err = service.RevealPracticeHelp(ctx, basis.WorkspaceID, objective.ID, first.ID, "hint", 1)
	require.NoError(t, err)
	require.Equal(t, 1, shown.HintCount)
	shown, err = service.RevealPracticeHelp(ctx, basis.WorkspaceID, objective.ID, first.ID, "answer", 0)
	require.NoError(t, err)
	require.True(t, shown.AnswerShown)
	require.Equal(t, 2, shown.HintCount)
	resumed, err = service.BeginPracticeSession(ctx, basis.WorkspaceID, objective.ID, Reproduce)
	require.NoError(t, err)
	require.Equal(t, first.ID, resumed.ID)
	require.True(t, resumed.AnswerShown)
	_, err = service.LoadPracticeSession(ctx, uuid.New(), objective.ID, first.ID)
	require.Error(t, err)
	now = now.Add(time.Minute)
	attempt := Attempt{Skill: Reproduce, PracticeSessionID: first.ID, PracticeTaskID: first.PracticeTaskID, PracticeContentHash: first.PracticeContentHash, Answer: "查看提示后按方法和路径实现查找", Correct: true, Independent: true, HintCount: 0, Confidence: 4, At: now.Add(365 * 24 * time.Hour)}
	result, err := service.RecordAttempt(ctx, basis.WorkspaceID, objective.ID, attempt)
	require.NoError(t, err)
	records, err := store.ListAttempts(ctx, objective.ID)
	require.NoError(t, err)
	require.Len(t, records, 2)
	last := records[len(records)-1]
	selectedAttempt, err := store.GetAttempt(ctx, objective.ID, last.ID)
	require.NoError(t, err)
	require.Equal(t, last.Answer, selectedAttempt.Answer)
	_, err = store.GetAttempt(ctx, uuid.New(), last.ID)
	require.Error(t, err)
	var attemptPlan string
	require.NoError(t, store.db.Raw("EXPLAIN (FORMAT JSON) SELECT * FROM mastery_attempts WHERE id = ? AND objective_id = ?", last.ID, objective.ID).Row().Scan(&attemptPlan))
	t.Log("assessment saved-answer lookup plan:", attemptPlan)
	require.Equal(t, &first.ID, last.PracticeSessionID)
	require.False(t, last.Independent)
	require.Equal(t, 2, last.HintCount)
	require.LessOrEqual(t, last.TookMillis, int64(120001))
	now = now.Add(10 * time.Minute)
	attempt.At = now
	attempt.Took = 24 * time.Hour
	repeated, err := service.RecordAttempt(ctx, basis.WorkspaceID, objective.ID, attempt)
	require.NoError(t, err)
	require.Equal(t, result, repeated)
	records, err = store.ListAttempts(ctx, objective.ID)
	require.NoError(t, err)
	require.Len(t, records, 2, "the lost-response retry cannot append a second answer")
	changed := attempt
	changed.Answer = "另一份作答"
	_, err = service.RecordAttempt(ctx, basis.WorkspaceID, objective.ID, changed)
	require.ErrorIs(t, err, ErrPracticeSessionClosed)
	_, err = service.RevealPracticeHelp(ctx, basis.WorkspaceID, objective.ID, first.ID, "hint", 2)
	require.ErrorIs(t, err, ErrPracticeSessionClosed)
	_, err = service.RevealPracticeHelp(ctx, basis.WorkspaceID, objective.ID, first.ID, "hint", 1)
	require.NoError(t, err, "retrying an already committed disclosure remains idempotent")
	closed, err := service.LoadPracticeSession(ctx, basis.WorkspaceID, objective.ID, first.ID)
	require.NoError(t, err)
	require.NotNil(t, closed.Result)
	require.Equal(t, result, *closed.Result)
	newSession, err := service.BeginPracticeSession(ctx, basis.WorkspaceID, objective.ID, Reproduce)
	require.NoError(t, err)
	require.NotEqual(t, first.ID, newSession.ID)
	require.Zero(t, newSession.HintCount)
	// Known help in another task is fresh learning contact, even when the last
	// submitted answer is old enough to otherwise qualify for delayed recall.
	now = now.Add(96 * time.Hour)
	_, err = service.RevealPracticeHelp(ctx, basis.WorkspaceID, objective.ID, newSession.ID, "hint", 1)
	require.NoError(t, err)
	retention, err := service.BeginPracticeSession(ctx, basis.WorkspaceID, objective.ID, Retain)
	require.NoError(t, err)
	retained := Attempt{Skill: Retain, PracticeSessionID: retention.ID, PracticeTaskID: retention.PracticeTaskID, PracticeContentHash: retention.PracticeContentHash, Answer: "不看原文复述查找与失败边界", Correct: true, Independent: true, Confidence: 4}
	_, err = service.RecordAttempt(ctx, basis.WorkspaceID, objective.ID, retained)
	require.ErrorIs(t, err, ErrPracticeNotDue)
	now = now.Add(72 * time.Hour)
	_, err = service.RecordAttempt(ctx, basis.WorkspaceID, objective.ID, retained)
	require.NoError(t, err)
	now = now.Add(time.Minute)
	concurrent, err := service.BeginPracticeSession(ctx, basis.WorkspaceID, objective.ID, Diagnose)
	require.NoError(t, err)
	sameAnswer := Attempt{Skill: Diagnose, PracticeSessionID: concurrent.ID, PracticeTaskID: concurrent.PracticeTaskID, PracticeContentHash: concurrent.PracticeContentHash, Answer: "按可观察的查找阶段定位错误", Correct: true, Independent: true, Confidence: 4}
	var wait sync.WaitGroup
	results := make([]DueTask, 2)
	errors := make([]error, 2)
	for index := range results {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			results[index], errors[index] = service.RecordAttempt(ctx, basis.WorkspaceID, objective.ID, sameAnswer)
		}(index)
	}
	wait.Wait()
	for _, err := range errors {
		require.NoError(t, err)
	}
	require.Equal(t, results[0], results[1])
	var count int64
	require.NoError(t, store.db.Model(&AttemptRecord{}).Where("practice_session_id = ?", concurrent.ID).Count(&count).Error)
	require.EqualValues(t, 1, count, "concurrent identical submissions must return one saved result")
	for _, query := range []string{
		"EXPLAIN (FORMAT JSON) SELECT id FROM mastery_practice_sessions WHERE objective_id = ? AND skill = 'reproduce' AND submitted_at IS NULL",
		"EXPLAIN (FORMAT JSON) SELECT count(*) FROM mastery_practice_help h JOIN mastery_practice_sessions s ON s.id = h.session_id WHERE s.objective_id = ?",
	} {
		var plan string
		require.NoError(t, store.db.Raw(query, objective.ID).Row().Scan(&plan))
		t.Log("session query plan:", plan)
	}
}

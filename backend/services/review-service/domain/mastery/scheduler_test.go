package mastery

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestApplyChangesNextTaskForWeakTransferAndDiagnose(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	state := State{Scores: map[Skill]int{Explain: 4, Complete: 3, Reproduce: 2, Transfer: -1, Diagnose: 0, Retain: 1}}
	next, task, err := Apply(state, Attempt{Skill: Transfer, Correct: false, Independent: true, Confidence: 2, At: now})
	require.NoError(t, err)
	require.Equal(t, Transfer, task.Skill)
	require.True(t, task.DueAt.After(now), "FSRS schedules a future review")
	require.Len(t, next.Attempts, 1)
	next, task, err = Apply(State{Scores: map[Skill]int{Explain: 4, Complete: 3, Reproduce: 2, Transfer: 5, Diagnose: -1, Retain: 1}}, Attempt{Skill: Diagnose, Correct: false, Independent: false, HintCount: 2, Confidence: 1, At: now.Add(time.Hour)})
	require.NoError(t, err)
	require.Equal(t, Diagnose, task.Skill)
	require.Equal(t, -3, next.Scores[Diagnose])
}

func TestIsMasteredRequiresDelayedRetainAndIndependentTransferOrReproduction(t *testing.T) {
	now := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	passed := func(skill Skill, at time.Time) Attempt {
		return Attempt{Skill: skill, Correct: true, Independent: true, Confidence: 4, At: at}
	}
	attempts := []Attempt{passed(Explain, now), passed(Reproduce, now), passed(Diagnose, now), passed(Retain, now)}
	require.False(t, IsMastered(attempts, now))
	require.False(t, IsMastered(attempts, now.Add(48*time.Hour)), "an immediate recall does not become delayed evidence as the record ages")
	attempts[3] = passed(Retain, now.Add(48*time.Hour))
	require.True(t, IsMastered(attempts, now.Add(48*time.Hour)))
}

func TestRetentionRequiresLearningBeforeRecallAndHonorsInterveningPractice(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	passed := func(skill Skill, at time.Time) Attempt {
		return Attempt{Skill: skill, Correct: true, Independent: true, Confidence: 4, At: at}
	}
	base := []Attempt{passed(Explain, start), passed(Reproduce, start), passed(Diagnose, start)}
	require.False(t, IsMastered(append(base, passed(Retain, start.Add(-48*time.Hour))), start))
	require.False(t, IsMastered(append(base, passed(Retain, start.Add(48*time.Hour))), start), "future answers are not evidence")
	intervening := append(append([]Attempt(nil), base...), passed(Complete, start.Add(47*time.Hour)), passed(Retain, start.Add(48*time.Hour)))
	require.False(t, IsMastered(intervening, start.Add(48*time.Hour)))
	ordered := append(append([]Attempt(nil), base...), passed(Retain, start.Add(24*time.Hour)))
	require.True(t, IsMastered(ordered, start.Add(24*time.Hour)))
	require.Equal(t, start, base[0].At, "evaluation cannot reorder or mutate historical evidence")
}

func TestApplyUsesConfidenceTimeAndErrorKindsWhenSelectingNextWork(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	state := State{Scores: map[Skill]int{Explain: 0, Complete: 2, Reproduce: 2, Transfer: 2, Diagnose: 2, Retain: 2}}
	updated, due, err := Apply(state, Attempt{Skill: Explain, Correct: true, Independent: true, Confidence: 1, Took: 20 * time.Minute, At: now})
	require.NoError(t, err)
	require.Equal(t, 0, updated.Scores[Explain], "a slow, uncertain correct answer is not treated as fluent mastery")
	require.Equal(t, Explain, due.Skill)
	require.True(t, due.DueAt.After(now), "FSRS schedules a future review")
	require.Contains(t, due.Reason, "不够稳固")

	updated, due, err = Apply(State{Scores: map[Skill]int{Explain: 3, Complete: 3, Reproduce: 3, Transfer: 3, Diagnose: 0, Retain: 3}}, Attempt{Skill: Diagnose, Correct: false, Independent: true, Confidence: 3, ErrorKinds: []string{"wrong_assumption"}, At: now})
	require.NoError(t, err)
	require.Equal(t, -3, updated.Scores[Diagnose])
	require.Equal(t, Diagnose, due.Skill)
	require.Contains(t, due.Reason, "错误类别")
}

func TestReplayRebuildsStateFromAppendOnlyAttempts(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	attempts := []Attempt{
		{Skill: Explain, Correct: true, Independent: true, Confidence: 5, At: now},
		{Skill: Transfer, Correct: false, Independent: false, HintCount: 2, Confidence: 2, ErrorKinds: []string{"missed_requirement"}, At: now.Add(time.Hour)},
	}
	replayed, due, err := Replay(attempts)
	require.NoError(t, err)
	require.Equal(t, attempts, replayed.Attempts)
	require.Equal(t, Transfer, due.Skill)
	require.True(t, due.DueAt.After(attempts[1].At))
}

func TestFSRSReplayKeepsTimingSeparateFromSixDimensionalTaskSelection(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	attempts := []Attempt{{Skill: Explain, Correct: true, Independent: true, Confidence: 5, At: now}, {Skill: Transfer, Correct: false, Independent: false, HintCount: 1, Confidence: 1, ErrorKinds: []string{"missing_requirement"}, At: now.Add(2 * time.Hour)}}
	state, due, err := Replay(attempts)
	require.NoError(t, err)
	require.Equal(t, Transfer, due.Skill, "six-dimensional weakness still selects the next task")
	require.Equal(t, state.DueAt, due.DueAt)

	replayedAt, err := fsrsDueAt(attempts)
	require.NoError(t, err)
	require.Equal(t, state.DueAt, replayedAt, "append-only history deterministically rebuilds the FSRS due time")
}

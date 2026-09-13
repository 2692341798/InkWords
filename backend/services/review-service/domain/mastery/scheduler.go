package mastery

import (
	"fmt"
	"sort"
	"strings"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v4"
)

// Apply appends evidence and derives the next due task without overwriting history.
func Apply(state State, attempt Attempt) (State, DueTask, error) {
	if err := validateAttempt(attempt); err != nil {
		return State{}, DueTask{}, err
	}
	if state.Scores == nil {
		state.Scores = map[Skill]int{}
	}
	state.Attempts = append(append([]Attempt(nil), state.Attempts...), attempt)
	state.Scores[attempt.Skill] += performanceDelta(attempt)
	next := weakest(state.Scores)
	dueAt, err := fsrsDueAt(state.Attempts)
	if err != nil {
		return State{}, DueTask{}, err
	}
	state.DueAt = dueAt
	return state, DueTask{Skill: next, DueAt: state.DueAt, Reason: reason(next, attempt)}, nil
}

func validateAttempt(attempt Attempt) error {
	if attempt.AssessmentOutcome != "" && attempt.AssessmentOutcome != AssessmentPassed && attempt.AssessmentOutcome != AssessmentNeedsPractice && attempt.AssessmentOutcome != AssessmentUnverified {
		return fmt.Errorf("unknown assessment outcome")
	}
	if !known(attempt.Skill) || attempt.At.IsZero() || attempt.Confidence < 1 || attempt.Confidence > 5 || attempt.HintCount < 0 || attempt.Took < 0 {
		return fmt.Errorf("invalid mastery attempt")
	}
	for _, kind := range attempt.ErrorKinds {
		if strings.TrimSpace(kind) == "" {
			return fmt.Errorf("invalid mastery attempt error kind")
		}
	}
	return nil
}

// performanceDelta is a deterministic, explainable task-selection policy. A
// future FSRS adapter will control timing, while this score keeps choosing the
// weakest one of explain/complete/reproduce/transfer/diagnose/retain.
func performanceDelta(attempt Attempt) int {
	if attempt.AssessmentOutcome == AssessmentUnverified {
		return 0
	}
	if !effectiveCorrect(attempt) {
		delta := -2
		if len(attempt.ErrorKinds) > 0 {
			delta--
		}
		return delta
	}
	delta := 1
	if attempt.Independent && attempt.HintCount == 0 {
		delta++
	}
	if attempt.Confidence <= 2 {
		delta--
	}
	if attempt.Took > 15*time.Minute {
		delta--
	}
	return delta
}

// fsrsDueAt is deliberately separate from weakest: FSRS decides only when to
// review, while InkWords' six-dimensional performance policy decides what the
// learner practices next. Replaying append-only attempts reconstructs the card
// without storing mutable provider or scheduler state beside a learner answer.
func fsrsDueAt(attempts []Attempt) (time.Time, error) {
	if len(attempts) == 0 {
		return time.Time{}, fmt.Errorf("FSRS requires at least one mastery attempt")
	}
	card := fsrs.NewCard(attempts[0].At)
	scheduler := fsrs.NewFSRS(fsrs.DefaultParam())
	for _, attempt := range attempts {
		info, err := scheduler.Next(card, attempt.At, fsrsRating(attempt))
		if err != nil {
			return time.Time{}, fmt.Errorf("schedule mastery review with FSRS: %w", err)
		}
		card = info.Card
	}
	return card.Due, nil
}

func fsrsRating(attempt Attempt) fsrs.Rating {
	if attempt.AssessmentOutcome == AssessmentUnverified {
		return fsrs.Hard
	}
	if !effectiveCorrect(attempt) {
		return fsrs.Again
	}
	if !attempt.Independent || attempt.HintCount > 0 || attempt.Confidence <= 2 || attempt.Took > 15*time.Minute {
		return fsrs.Hard
	}
	return fsrs.Good
}

// Replay reconstructs the exact score and due task from append-only attempts.
// It is the migration-safe source of truth; callers never need to edit history.
func Replay(attempts []Attempt) (State, DueTask, error) {
	state := State{}
	var due DueTask
	for _, attempt := range attempts {
		var err error
		state, due, err = Apply(state, attempt)
		if err != nil {
			return State{}, DueTask{}, err
		}
	}
	return state, due, nil
}

func weakest(scores map[Skill]int) Skill {
	ordered := append([]Skill(nil), Skills...)
	sort.SliceStable(ordered, func(i, j int) bool { return scores[ordered[i]] < scores[ordered[j]] })
	return ordered[0]
}

func known(skill Skill) bool {
	for _, item := range Skills {
		if item == skill {
			return true
		}
	}
	return false
}

func reason(next Skill, attempt Attempt) string {
	if attempt.AssessmentOutcome == AssessmentUnverified {
		return "已应用评分仍有未知项，安排补证与巩固；不计为通过或答错"
	}
	if !effectiveCorrect(attempt) {
		if len(attempt.ErrorKinds) > 0 {
			return "本次未通过，按错误类别优先补薄弱维度"
		}
		return "本次未通过，优先补薄弱维度"
	}
	if !attempt.Independent || attempt.HintCount > 0 {
		return "提示依赖仍高，安排低脚手架练习"
	}
	if attempt.Confidence <= 2 || attempt.Took > 15*time.Minute {
		return "答案虽通过但不够稳固，安排较早的巩固练习"
	}
	return "按六维薄弱项安排下一题"
}

// IsMastered requires complementary evidence; immediate recall alone is never long-term mastery.
func IsMastered(attempts []Attempt, now time.Time) bool {
	return IsMasteredAfter(attempts, now, 24*time.Hour)
}

// IsMasteredAfter measures the interval at recall time, never the age of the
// saved recall record. Every intervening practice restarts the delay.
func IsMasteredAfter(attempts []Attempt, now time.Time, delay time.Duration) bool {
	if now.IsZero() || delay < 24*time.Hour {
		return false
	}
	ordered := append([]Attempt(nil), attempts...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].At.Before(ordered[j].At) })
	passed := map[Skill]bool{}
	var previous time.Time
	for _, attempt := range ordered {
		if attempt.At.IsZero() || attempt.At.After(now) || !known(attempt.Skill) {
			return false
		}
		if effectiveCorrect(attempt) && attempt.Independent && attempt.HintCount == 0 {
			if attempt.Skill != Retain || !previous.IsZero() && attempt.At.Sub(previous) >= delay {
				passed[attempt.Skill] = true
			}
		}
		previous = attempt.At
	}
	return passed[Explain] && (passed[Reproduce] || passed[Transfer]) && passed[Diagnose] && passed[Retain]
}

package mastery

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	fsrs "github.com/open-spaced-repetition/go-fsrs/v4"
	"github.com/stretchr/testify/require"
)

func cloneFeedback(t *testing.T, feedback AssessmentFeedback) AssessmentFeedback {
	t.Helper()
	data, err := json.Marshal(feedback)
	require.NoError(t, err)
	var clone AssessmentFeedback
	require.NoError(t, json.Unmarshal(data, &clone))
	return clone
}

func TestAppliedAssessmentKeepsUnknownSeparateAndPreservesLearnerEvidence(t *testing.T) {
	input, feedback := assessmentFixture(t, Reproduce)
	original := Attempt{Skill: Reproduce, Answer: input.Answer, Correct: true, Independent: false, HintCount: 2, Confidence: 4, Took: time.Minute, At: time.Now().UTC()}
	assessed, err := ApplyAssessmentOutcome(original, input, feedback)
	require.NoError(t, err)
	require.Equal(t, AssessmentUnverified, assessed.AssessmentOutcome)
	require.True(t, assessed.Correct, "original self-report remains intact")
	require.False(t, assessed.Independent)
	require.Equal(t, 2, assessed.HintCount)
	require.Zero(t, performanceDelta(assessed), "missing runtime proof is neither success nor failure")
	require.Equal(t, fsrs.Hard, fsrsRating(assessed))
	low := 2
	feedback.Criteria[0].Score = &low
	assessed, err = ApplyAssessmentOutcome(original, input, feedback)
	require.NoError(t, err)
	require.Equal(t, AssessmentNeedsPractice, assessed.AssessmentOutcome)
	require.Equal(t, fsrs.Again, fsrsRating(assessed), "known failure remains actionable alongside unknown criteria")
	input, feedback = assessmentFixture(t, Explain)
	original.Skill = Explain
	assessed, err = ApplyAssessmentOutcome(original, input, feedback)
	require.NoError(t, err)
	require.Equal(t, AssessmentPassed, assessed.AssessmentOutcome)
	require.Equal(t, fsrs.Hard, fsrsRating(assessed), "grading cannot erase help dependence")
}

func TestAppliedUnknownCannotSupplyMissingMasteryOrDelayedRecallEvidence(t *testing.T) {
	now := time.Now().UTC()
	attempts := []Attempt{}
	for i, skill := range []Skill{Explain, Reproduce, Diagnose, Retain} {
		attempts = append(attempts, Attempt{Skill: skill, Correct: true, Independent: true, Confidence: 4, At: now.Add(time.Duration(i-3) * 48 * time.Hour)})
	}
	require.True(t, IsMastered(attempts, now))
	attempts[1].AssessmentOutcome = AssessmentUnverified
	require.False(t, IsMastered(attempts, now), "self-reported code success cannot override an applied unknown runtime grade")
	attempts[1].AssessmentOutcome = AssessmentPassed
	attempts[3].At = attempts[2].At.Add(time.Minute)
	attempts[3].AssessmentOutcome = AssessmentPassed
	require.False(t, IsMastered(attempts, now.Add(7*24*time.Hour)), "later application does not age immediate recall into delayed mastery")
}

func applicationFixture(t *testing.T) (Objective, []AttemptRecord, AssessmentApplication) {
	t.Helper()
	basis := practiceBasisFixture()
	obj := Objective{ID: uuid.New(), WorkspaceID: basis.WorkspaceID, ChapterID: "approved-revision:" + basis.Learning.ChapterID + ":" + basis.Learning.RevisionID, PracticeContentHash: basis.Learning.ContentHash}
	revision := uuid.MustParse(basis.Learning.RevisionID)
	obj.PracticeRevisionID = &revision
	input, feedback := assessmentFixture(t, Explain)
	input.ObjectiveID, input.RevisionID, input.AttemptID = obj.ID.String(), revision.String(), uuid.NewString()
	now := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	session := uuid.New()
	records := []AttemptRecord{{ID: uuid.MustParse(input.AttemptID), ObjectiveID: obj.ID, Skill: string(Explain), Answer: input.Answer, PracticeTaskID: "task-explain", PracticeContentHash: obj.PracticeContentHash, PracticeSessionID: &session, Correct: false, Independent: true, Confidence: 4, AttemptedAt: now, ErrorKinds: []byte(`[]`)}}
	event := AssessmentApplication{ID: uuid.New(), ObjectiveID: obj.ID, AttemptID: records[0].ID, JobID: uuid.New(), AppliedBy: obj.WorkspaceID, AppliedAt: now.Add(time.Hour), SequenceNo: 1, AlgorithmVersion: AssessmentAlgorithmVersion, Input: input, InputHash: AssessmentInputHash(input), Feedback: feedback, FeedbackHash: AssessmentFeedbackHash(feedback)}
	return obj, records, event
}

func TestAssessmentApplicationsReplayAtOriginalPracticeTimesWithoutNewAttempts(t *testing.T) {
	obj, records, event := applicationFixture(t)
	original := records[0]
	attempts, err := ReplayAssessmentApplications(obj, records, []AssessmentApplication{event})
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	require.Equal(t, original.AttemptedAt, attempts[0].At)
	require.Equal(t, AssessmentPassed, attempts[0].AssessmentOutcome)
	require.Equal(t, original, records[0])
	_, due, err := Replay(attempts)
	require.NoError(t, err)
	expected := attempts[0]
	expected.AssessmentOutcome = ""
	expected.Correct = true
	_, expectedDue, err := Replay([]Attempt{expected})
	require.NoError(t, err)
	require.Equal(t, expectedDue.DueAt, due.DueAt)
	newer := event
	newer.ID, newer.SequenceNo, newer.PreviousID = uuid.New(), 2, &event.ID
	newer.AppliedAt = event.AppliedAt.Add(time.Hour)
	newer.Feedback = cloneFeedback(t, event.Feedback)
	score := 1
	newer.Feedback.Criteria[0].Score = &score
	newer.FeedbackHash = AssessmentFeedbackHash(newer.Feedback)
	attempts, err = ReplayAssessmentApplications(obj, records, []AssessmentApplication{event, newer})
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	require.Equal(t, AssessmentNeedsPractice, attempts[0].AssessmentOutcome)
	require.Equal(t, 3, *event.Feedback.Criteria[0].Score)
	newer.PreviousID = nil
	_, err = ReplayAssessmentApplications(obj, records, []AssessmentApplication{event, newer})
	require.Error(t, err, "broken application chain must fail closed")
}

func TestAssessmentApplicationRejectsChangedAnswerIdentityAndFeedback(t *testing.T) {
	obj, records, event := applicationFixture(t)
	for _, change := range []func(*AssessmentApplication){
		func(e *AssessmentApplication) { e.AppliedBy = uuid.New() },
		func(e *AssessmentApplication) {
			e.Input.Answer = "different answer"
			e.InputHash = AssessmentInputHash(e.Input)
		},
		func(e *AssessmentApplication) { e.FeedbackHash = "sha256:wrong" },
		func(e *AssessmentApplication) { e.SequenceNo = 2 },
		func(e *AssessmentApplication) { e.AlgorithmVersion = "future-unknown" },
		func(e *AssessmentApplication) { e.AppliedAt = records[0].AttemptedAt.Add(-time.Second) },
	} {
		changed := event
		change(&changed)
		_, err := ReplayAssessmentApplications(obj, records, []AssessmentApplication{changed})
		require.Error(t, err)
	}
	records[0].PracticeSessionID = nil
	_, err := ReplayAssessmentApplications(obj, records, []AssessmentApplication{event})
	require.Error(t, err, "unbound historical answers cannot acquire grading authority")
}

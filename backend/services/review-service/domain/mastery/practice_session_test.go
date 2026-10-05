package mastery

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSubmissionIdentityIgnoresRetryTimingButNotChangedAnswers(t *testing.T) {
	first := Attempt{PracticeSessionID: uuid.New(), Skill: Explain, PracticeTaskID: "task-explain", PracticeContentHash: "sha256:fixture", Answer: "先按方法查找。", Correct: true, Independent: true, Confidence: 4, At: time.Now(), Took: time.Minute}
	copy := first
	copy.At = copy.At.Add(time.Hour)
	copy.Took += time.Hour
	require.Equal(t, submissionHash(first), submissionHash(copy))
	copy.ErrorKinds = []string{}
	require.Equal(t, submissionHash(first), submissionHash(copy))
	copy.Answer = "改写后的回答"
	require.NotEqual(t, submissionHash(first), submissionHash(copy))
	copy = first
	copy.PracticeSessionID = uuid.New()
	require.NotEqual(t, submissionHash(first), submissionHash(copy))
}

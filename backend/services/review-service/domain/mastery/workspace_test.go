package mastery

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWorkspacePreservesLearnerAnswerAndRequiresOwner(t *testing.T) {
	service := NewService(newMemoryStore())
	owner := uuid.New()
	objective, err := service.CreateObjective(context.Background(), owner, validObjectiveInput([]Skill{Explain}))
	require.NoError(t, err)
	answer := "我的复述：\n```go\nrouter.addRoute(\"GET\", \"/orders\", handler)\n```\n<script>未执行的学习文本</script>"
	_, err = service.RecordAttempt(context.Background(), owner, objective.ID, Attempt{Skill: Explain, Answer: answer, Confidence: 3, At: time.Now().UTC()})
	require.NoError(t, err)
	workspace, err := service.Workspace(context.Background(), owner, objective.ID)
	require.NoError(t, err)
	require.Equal(t, objective.Behavior, workspace.Objective.Behavior)
	require.Len(t, workspace.Attempts, 1)
	require.Equal(t, answer, workspace.Attempts[0].Answer)
	_, err = service.Workspace(context.Background(), uuid.New(), objective.ID)
	require.Error(t, err)
}

func TestRecordAttemptRejectsOversizedOrInvalidAnswerWithoutWriting(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)
	owner := uuid.New()
	objective, err := service.CreateObjective(context.Background(), owner, validObjectiveInput([]Skill{Explain}))
	require.NoError(t, err)
	for _, answer := range []string{strings.Repeat("中", 20001), "before\x00after"} {
		_, err := service.RecordAttempt(context.Background(), owner, objective.ID, Attempt{Skill: Explain, Answer: answer, Confidence: 3, At: time.Now().UTC()})
		require.Error(t, err)
	}
	require.Empty(t, store.attempts[objective.ID])
	// Historical self-assessments without an answer remain readable, and an
	// optional answer must not be fabricated from their boolean outcome.
	_, err = service.RecordAttempt(context.Background(), owner, objective.ID, Attempt{Skill: Explain, Confidence: 3, At: time.Now().UTC()})
	require.NoError(t, err)
	workspace, err := service.Workspace(context.Background(), owner, objective.ID)
	require.NoError(t, err)
	require.Empty(t, workspace.Attempts[0].Answer)
}

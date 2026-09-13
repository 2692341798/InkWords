package textbook

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskModelPolicyRestrictsHighCostTierToTeachingCriticalTasks(t *testing.T) {
	policy := TaskModelPolicy{CoreModel: "reasoning-model", StandardModel: "standard-model"}

	for _, task := range []GenerationTaskClass{GenerationTaskBlueprint, GenerationTaskCoreConcept, GenerationTaskEditorialReview} {
		model, err := policy.ModelFor(task)
		require.NoError(t, err)
		require.Equal(t, "reasoning-model", model)
	}
	model, err := policy.ModelFor(GenerationTaskSampleChapter)
	require.NoError(t, err)
	require.Equal(t, "standard-model", model)
}

func TestTaskModelPolicyFailsClosedForMissingOrDeterministicTaskModels(t *testing.T) {
	_, err := (TaskModelPolicy{StandardModel: "standard-model"}).ModelFor(GenerationTaskCoreConcept)
	require.ErrorContains(t, err, "no model")
	_, err = (TaskModelPolicy{CoreModel: "reasoning-model", StandardModel: "standard-model"}).ModelFor("source_parse")
	require.ErrorContains(t, err, "unsupported")
}

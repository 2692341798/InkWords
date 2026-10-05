package bootstrap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAssessmentProfileIsExplicitAndLimitedToTheEvaluatedModel(t *testing.T) {
	legacy, timeout, err := resolveAssessmentPolicy("", "another-model")
	require.NoError(t, err)
	require.Nil(t, legacy)
	require.Equal(t, 45*time.Second, timeout)
	policy, timeout, err := resolveAssessmentPolicy(" local-evaluation-v1 ", "deepseek-v4-flash")
	require.NoError(t, err)
	require.Equal(t, 60*time.Second, timeout)
	require.Equal(t, "low", policy.Text.ReasoningEffort)
	require.Empty(t, policy.Code.ReasoningEffort)
	require.Equal(t, 6000, policy.Text.MaxOutputTokens)
	require.Equal(t, 6000, policy.Code.MaxOutputTokens)
	require.False(t, policy.Text.GroundedIdentities)
	require.False(t, policy.Code.GroundedIdentities)
	v2, v2Timeout, err := resolveAssessmentPolicy("local-evaluation-v2", "deepseek-v4-flash")
	require.NoError(t, err)
	require.Equal(t, timeout, v2Timeout)
	require.True(t, v2.Text.GroundedIdentities)
	require.True(t, v2.Code.GroundedIdentities)
	require.Equal(t, policy.Text.ReasoningEffort, v2.Text.ReasoningEffort)
	require.Equal(t, policy.Code.MaxOutputTokens, v2.Code.MaxOutputTokens)
	for _, selection := range [][2]string{{"unknown", "deepseek-v4-flash"}, {"local-evaluation-v1", "another-model"}, {"local-evaluation-v2", "another-model"}} {
		_, _, err := resolveAssessmentPolicy(selection[0], selection[1])
		require.Error(t, err)
	}
}

func TestInvalidAssessmentProfileFailsBeforeAnyDatabaseRecovery(t *testing.T) {
	t.Setenv("MASTERY_ASSESSMENT_PROFILE", "unknown")
	_, err := BuildAssessmentService(nil, nil)
	require.ErrorContains(t, err, "MASTERY_ASSESSMENT_PROFILE")
}

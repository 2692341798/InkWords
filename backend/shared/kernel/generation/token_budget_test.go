package generation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckBudgetUsesAConservativeMultilingualEstimate(t *testing.T) {
	report, err := CheckBudget(TokenBudget{MaxInput: 20, ReservedOutput: 5}, []string{"12345678", "中文证据"})
	require.NoError(t, err)
	require.Equal(t, 6, report.EstimatedInput)
	require.Equal(t, 15, report.AllowedInput)
	require.False(t, report.RequiresCompression)

	report, err = CheckBudget(TokenBudget{MaxInput: 8, ReservedOutput: 3}, []string{"中文证据超额"})
	require.NoError(t, err)
	require.True(t, report.RequiresCompression)
	require.NotEmpty(t, report.Advice)
}

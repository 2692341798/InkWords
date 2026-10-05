package textbook

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCheckBudgetExplainsCompressionInsteadOfTruncating(t *testing.T) {
	report, err := CheckBudget(TokenBudget{MaxInput: 10, ReservedOutput: 2}, []string{"123456789012345678901234567890123456"})
	require.NoError(t, err)
	require.True(t, report.RequiresCompression)
	require.Contains(t, report.Advice, "不会静默截断")
	report, err = CheckBudget(TokenBudget{MaxInput: 10, ReservedOutput: 2}, []string{"中文资料必须保守估算"})
	require.NoError(t, err)
	require.True(t, report.RequiresCompression, "CJK 资料不能按每四个字符一个 token 低估")
	_, err = CheckBudget(TokenBudget{MaxInput: 10, ReservedOutput: 10}, nil)
	require.Error(t, err)
}

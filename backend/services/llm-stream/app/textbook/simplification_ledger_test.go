package textbook

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSimplificationLedgerRequiresExplicitCorrection(t *testing.T) {
	require.NoError(t, ValidateSimplificationLedger([]Simplification{{EarlyExplanation: "先把路由看成查表", Boundary: "不表示运行时重新建表", CorrectedIn: "路由树章节"}}))
	require.Error(t, ValidateSimplificationLedger([]Simplification{{EarlyExplanation: "简化", Boundary: "边界"}}))
}

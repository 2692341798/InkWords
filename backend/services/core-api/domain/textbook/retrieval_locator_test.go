package textbook

import (
	"testing"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestRetrievalResponsePreservesPhysicalSourceSymbolAndLines(t *testing.T) {
	locator := sharedtextbook.EvidenceLocator{Path: "tree.go", Symbol: "(*node).getValue", StartLine: 418, EndLine: 679}
	result := retrievalPlanCandidates([]sharedtextbook.RetrievalCandidate{{Chunk: sharedtextbook.SourceChunk{ID: "selected-chunk", Locator: locator}}})
	require.Len(t, result, 1)
	require.Equal(t, locator, result[0].Locator)
}

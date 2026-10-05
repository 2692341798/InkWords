package masteryassessment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A diagnostic subset retains original case indices and frozen input hashes.
// Invalid selectors must fail before a provider call, not pass an empty suite.
func selectAcceptanceIndices(names []string, selected string) ([]int, error) {
	selected = strings.TrimSpace(selected)
	indices := []int{}
	for index, name := range names {
		if selected == "" || selected == name {
			indices = append(indices, index)
		}
	}
	if len(indices) == 0 {
		return nil, fmt.Errorf("unknown real assessment case selection")
	}
	return indices, nil
}

func TestAcceptanceSelectionPreservesIdentityAndRejectsEmptyDiagnostic(t *testing.T) {
	names := []string{"complete-causal-answer", "procedural-only-answer", "incomplete-answer", "misconception-answer"}
	indices, err := selectAcceptanceIndices(names, "")
	require.NoError(t, err)
	require.Equal(t, []int{0, 1, 2, 3}, indices)
	indices, err = selectAcceptanceIndices(names, " procedural-only-answer ")
	require.NoError(t, err)
	require.Equal(t, []int{1}, indices)
	_, err = selectAcceptanceIndices(names, "unknown")
	require.Error(t, err)
}

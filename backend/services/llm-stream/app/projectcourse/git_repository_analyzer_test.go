package projectcourse

import (
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/shared/platform/parser"
)

func TestInventoryInputsFromChunksExtractsFilesWithoutExecutingContent(t *testing.T) {
	inputs := inventoryInputsFromChunks([]parser.FileChunk{{Content: "--- File: main.go ---\npackage main\n\n--- File: README.md ---\n# readme\n"}})
	require.Len(t, inputs, 2)
	require.Equal(t, "main.go", inputs[0].Path)
	require.Contains(t, string(inputs[0].Content), "package main")
	require.Equal(t, "README.md", inputs[1].Path)
}

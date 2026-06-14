package project

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"inkwords-backend/shared/platform/parser"
)

func TestAssembleSourceContent_MatchesLegacyFormat(t *testing.T) {
	tree := "TREE\n"
	chunks := []parser.FileChunk{
		{Dir: "a", Content: "A1\n"},
		{Dir: "b", Content: "B2\n"},
	}

	got := AssembleSourceContent(tree, chunks)
	want := "TREE\n\n=== Repository Content ===\nA1\nB2\n"
	assert.Equal(t, want, got)
}

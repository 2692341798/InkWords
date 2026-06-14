package project

import (
	"strings"

	"inkwords-backend/shared/platform/parser"
)

func AssembleSourceContent(treeContent string, chunks []parser.FileChunk) string {
	var b strings.Builder
	b.WriteString(treeContent)
	b.WriteString("\n=== Repository Content ===\n")
	for _, chunk := range chunks {
		b.WriteString(chunk.Content)
	}
	return b.String()
}

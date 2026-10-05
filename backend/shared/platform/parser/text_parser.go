package parser

import (
	"strings"

	"inkwords-backend/shared/kernel/textbook"
)

// buildTextChunks uses visible blank-line boundaries rather than arbitrary token windows.
// This keeps every chunk explainable to an author reviewing imported material.
func buildTextChunks(content, documentID, artifactPath, canonicalLocator string) []textbook.SourceChunk {
	lines := splitStructuredLines(content)
	chunks := make([]textbook.SourceChunk, 0)
	paragraph := make([]markdownLine, 0)
	paragraphNumber := 0
	flush := func() {
		if len(paragraph) == 0 {
			return
		}
		paragraphNumber++
		first, last := paragraph[0], paragraph[len(paragraph)-1]
		chunks = append(chunks, newStructuredChunk(documentID, len(chunks)+1, paragraphNumber, first.startByte, last.endByte, first.line, last.line, artifactPath, canonicalLocator, "", nil, strings.Join(markdownLineTexts(paragraph), "\n")))
		paragraph = paragraph[:0]
	}
	for _, line := range lines {
		if strings.TrimSpace(line.text) == "" {
			flush()
			continue
		}
		paragraph = append(paragraph, line)
	}
	flush()
	return chunks
}

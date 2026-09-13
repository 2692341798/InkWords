package parser

import (
	"regexp"
	"strings"

	"inkwords-backend/shared/kernel/textbook"
)

var markdownHeadingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*\s*$`)
var markdownLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)(?:\s+[^)]*)?\)`)

type markdownHeading struct {
	level int
	text  string
}

type markdownLine struct {
	text      string
	startByte int
	endByte   int
	line      int
}

// buildMarkdownChunks separates prose and fenced code while retaining the active heading path.
func buildMarkdownChunks(content, documentID, artifactPath, canonicalLocator string) ([]textbook.SourceChunk, []DocumentLink) {
	lines := splitStructuredLines(content)
	headings := make([]markdownHeading, 0, 6)
	chunks := make([]textbook.SourceChunk, 0)
	links := make([]DocumentLink, 0)
	paragraph := make([]markdownLine, 0)
	paragraphNumber := 0
	inFence := false
	fenceMarker := ""
	codeLanguage := ""
	codeStart := markdownLine{}
	codeLines := make([]markdownLine, 0)

	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		paragraphNumber++
		text := strings.Join(markdownLineTexts(paragraph), "\n")
		first, last := paragraph[0], paragraph[len(paragraph)-1]
		chunks = append(chunks, newStructuredChunk(documentID, len(chunks)+1, paragraphNumber, first.startByte, last.endByte, first.line, last.line, artifactPath, canonicalLocator, "", headingPath(headings), text))
		links = append(links, findMarkdownLinks(text, first.startByte)...)
		paragraph = paragraph[:0]
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line.text)
		if inFence {
			if strings.HasPrefix(trimmed, fenceMarker) {
				if len(codeLines) > 0 {
					paragraphNumber++
					codeEnd := codeLines[len(codeLines)-1]
					chunks = append(chunks, newStructuredChunk(documentID, len(chunks)+1, paragraphNumber, codeStart.startByte, codeEnd.endByte, codeStart.line, codeEnd.line, artifactPath, canonicalLocator, codeLanguage, headingPath(headings), strings.Join(markdownLineTexts(codeLines), "\n")))
				}
				inFence = false
				fenceMarker, codeLanguage = "", ""
				codeLines = codeLines[:0]
				continue
			}
			codeLines = append(codeLines, line)
			continue
		}

		if marker, language, ok := markdownFenceStart(trimmed); ok {
			flushParagraph()
			inFence, fenceMarker, codeLanguage, codeStart = true, marker, language, line
			continue
		}
		if match := markdownHeadingPattern.FindStringSubmatch(trimmed); len(match) == 3 {
			flushParagraph()
			level := len(match[1])
			for len(headings) > 0 && headings[len(headings)-1].level >= level {
				headings = headings[:len(headings)-1]
			}
			headings = append(headings, markdownHeading{level: level, text: strings.TrimSpace(match[2])})
			continue
		}
		if trimmed == "" {
			flushParagraph()
			continue
		}
		paragraph = append(paragraph, line)
	}
	if inFence {
		// An unterminated fence remains citeable, but is marked by its language and exact range;
		// generation gates can decide whether this incomplete source is acceptable.
		if len(codeLines) > 0 {
			paragraphNumber++
			codeEnd := codeLines[len(codeLines)-1]
			chunks = append(chunks, newStructuredChunk(documentID, len(chunks)+1, paragraphNumber, codeStart.startByte, codeEnd.endByte, codeStart.line, codeEnd.line, artifactPath, canonicalLocator, codeLanguage, headingPath(headings), strings.Join(markdownLineTexts(codeLines), "\n")))
		}
	}
	flushParagraph()
	return chunks, links
}

func markdownTitle(content, filename string) string {
	for _, line := range splitStructuredLines(content) {
		if match := markdownHeadingPattern.FindStringSubmatch(strings.TrimSpace(line.text)); len(match) == 3 && len(match[1]) == 1 {
			return strings.TrimSpace(match[2])
		}
	}
	return filename
}

func markdownFenceStart(line string) (marker, language string, ok bool) {
	if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
		length := 3
		for length < len(line) && line[length] == line[0] {
			length++
		}
		return line[:length], strings.TrimSpace(line[length:]), true
	}
	return "", "", false
}

func headingPath(headings []markdownHeading) []string {
	path := make([]string, 0, len(headings))
	for _, heading := range headings {
		path = append(path, heading.text)
	}
	return path
}

func findMarkdownLinks(text string, startByte int) []DocumentLink {
	matches := markdownLinkPattern.FindAllStringSubmatchIndex(text, -1)
	links := make([]DocumentLink, 0, len(matches))
	for _, match := range matches {
		links = append(links, DocumentLink{
			Text:      text[match[2]:match[3]],
			Target:    text[match[4]:match[5]],
			StartByte: startByte + match[0],
			EndByte:   startByte + match[1],
		})
	}
	return links
}

func splitStructuredLines(content string) []markdownLine {
	rawLines := strings.Split(content, "\n")
	lines := make([]markdownLine, 0, len(rawLines))
	offset := 0
	for index, line := range rawLines {
		lines = append(lines, markdownLine{text: line, startByte: offset, endByte: offset + len(line), line: index + 1})
		offset += len(line) + 1
	}
	return lines
}

func markdownLineTexts(lines []markdownLine) []string {
	values := make([]string, 0, len(lines))
	for _, line := range lines {
		values = append(values, line.text)
	}
	return values
}

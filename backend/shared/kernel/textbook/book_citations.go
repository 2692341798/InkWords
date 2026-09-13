package textbook

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// CanonicalBookCitation freezes the source locator alongside its immutable
// snapshot identity. Exporters must not resolve it against live source tables.
type CanonicalBookCitation struct {
	ID       string         `json:"id"`
	Evidence EvidenceRef    `json:"evidence"`
	Snapshot SourceSnapshot `json:"snapshot"`
}

// BookEvidenceMarker identifies an inline citation without matching code or
// escaped Markdown. Start and End are byte offsets in the original manuscript.
type BookEvidenceMarker struct {
	ID         string
	Start, End int
}

var bookEvidenceID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
var bookEvidenceNodeKind = ast.NewNodeKind("BookEvidence")

type bookEvidenceNode struct {
	ast.BaseInline
	marker BookEvidenceMarker
}

func (node *bookEvidenceNode) Kind() ast.NodeKind { return bookEvidenceNodeKind }
func (node *bookEvidenceNode) Dump(source []byte, level int) {
	ast.DumpHelper(node, source, level, map[string]string{"ID": node.marker.ID}, nil)
}

type bookEvidenceParser struct{}

func (bookEvidenceParser) Trigger() []byte { return []byte{'['} }
func (bookEvidenceParser) Parse(_ ast.Node, reader text.Reader, _ parser.Context) ast.Node {
	line, segment := reader.PeekLine()
	const prefix = "[evidence:"
	if !bytes.HasPrefix(line, []byte(prefix)) {
		return nil
	}
	end := bytes.IndexByte(line, ']')
	if end < len(prefix) || !bookEvidenceID.Match(line[len(prefix):end]) {
		return nil
	}
	reader.Advance(end + 1)
	return &bookEvidenceNode{marker: BookEvidenceMarker{ID: string(line[len(prefix):end]), Start: segment.Start, End: segment.Start + end + 1}}
}

// BookEvidenceMarkers delegates code-span, fenced-block, and escape handling
// to the existing Markdown parser instead of rewriting raw manuscript strings.
func BookEvidenceMarkers(markdown string) []BookEvidenceMarker {
	engine := goldmark.New(goldmark.WithParserOptions(parser.WithInlineParsers(util.Prioritized(bookEvidenceParser{}, 50))))
	document := engine.Parser().Parse(text.NewReader([]byte(markdown)))
	var markers []BookEvidenceMarker
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if value, ok := node.(*bookEvidenceNode); entering && ok {
			markers = append(markers, value.marker)
		}
		return ast.WalkContinue, nil
	})
	sort.Slice(markers, func(i, j int) bool { return markers[i].Start < markers[j].Start })
	return markers
}

func (chapter CanonicalBookChapter) validateCitations() error {
	seen := make(map[string]bool, len(chapter.Citations))
	for _, citation := range chapter.Citations {
		if !bookEvidenceID.MatchString(citation.ID) || seen[citation.ID] || citation.Evidence.Validate() != nil || citation.Snapshot.Validate() != nil || citation.Evidence.SnapshotID != citation.Snapshot.ID || citation.Evidence.SourceRole != citation.Snapshot.Role {
			return fmt.Errorf("invalid frozen book citation")
		}
		seen[citation.ID] = true
	}
	for _, marker := range BookEvidenceMarkers(chapter.Markdown) {
		if !seen[marker.ID] {
			return fmt.Errorf("book citation %q has no frozen source; create a new book build", marker.ID)
		}
	}
	return nil
}

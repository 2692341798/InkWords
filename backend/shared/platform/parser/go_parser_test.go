package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoSourceChunksKeepWholeDeclarationsAndPhysicalLocations(t *testing.T) {
	content := "package gin\n\n// GET 教学说明。\n//line forged.go:900\nfunc (group *RouterGroup) GET(path string) {\n\t// 空行不会把函数切断\n\n\tgroup.handle(path)\n}\n\nfunc helper() {}\n"
	request := StructuredParseRequest{SourceID: "source", SnapshotID: "snapshot", Filename: "routergroup.go", ArtifactPath: "routergroup.go", CanonicalLocator: "https://github.com/gin-gonic/gin"}
	parsed, err := NewStructuredParser().Parse(strings.NewReader(content), request)
	require.NoError(t, err)
	require.Len(t, parsed.Chunks, 3)
	method := parsed.Chunks[1]
	require.Equal(t, "(*RouterGroup).GET", method.Locator.Symbol)
	require.Equal(t, "routergroup.go", method.Locator.Path)
	require.Equal(t, 3, method.Locator.StartLine)
	require.Equal(t, 9, method.Locator.EndLine)
	require.Contains(t, method.SearchText, "group.handle(path)")
	require.Equal(t, "helper", parsed.Chunks[2].Locator.Symbol)
	for _, chunk := range parsed.Chunks {
		require.Equal(t, content[chunk.StartByte:chunk.EndByte], chunk.SearchText)
		require.Equal(t, sha256Digest(chunk.SearchText), chunk.TextHash)
		require.NoError(t, chunk.Validate())
	}
	duplicate, err := NewStructuredParser().Parse(strings.NewReader(content), request)
	require.NoError(t, err)
	require.Equal(t, parsed, duplicate)
}

func TestGoSourceParserDoesNotInventSymbolsFromStringsOrRunImports(t *testing.T) {
	content := "package demo\nimport _ \"does.not.exist/and-must-not-be-loaded\"\ntype Box[T any] struct{}\nfunc (b *Box[T]) Get() string { return \"func fake() {}\" }\n"
	request := StructuredParseRequest{SourceID: "source", SnapshotID: "snapshot", Filename: "box.go", ArtifactPath: "box.go", CanonicalLocator: "local"}
	parsed, err := NewStructuredParser().Parse(strings.NewReader(content), request)
	require.NoError(t, err)
	symbols := []string{}
	for _, chunk := range parsed.Chunks {
		symbols = append(symbols, chunk.Locator.Symbol)
	}
	require.Contains(t, symbols, "Box")
	require.Contains(t, symbols, "(*Box).Get")
	require.NotContains(t, symbols, "fake")
	_, err = NewStructuredParser().Parse(strings.NewReader("package demo\nfunc broken( {"), request)
	require.ErrorContains(t, err, "Go")
}

func TestGoSourceParserReplaysLegacyParagraphBoundaries(t *testing.T) {
	content := "package gin\n\nfunc helper() {\n\tx := 1\n\n\t_ = x\n}\n"
	request := StructuredParseRequest{SourceID: "source", SnapshotID: "snapshot", Filename: "file.go", CanonicalLocator: "local", LegacyCodeParagraphs: true}
	parsed, err := NewStructuredParser().Parse(strings.NewReader(content), request)
	require.NoError(t, err)
	require.Len(t, parsed.Chunks, 3)
	for _, chunk := range parsed.Chunks {
		require.Empty(t, chunk.Locator.Symbol)
	}
}

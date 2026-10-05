package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOfficialWebTabsPreserveInactivePlatformInstructionsAndLabels(t *testing.T) {
	// Same ARIA relationships as go.dev/doc/install; fixture prose is synthetic.
	source := `<main><h1>Setup</h1><h2>Install</h2><div role="tablist"><button role="tab" id="linux" aria-controls="linux-panel">Linux</button><button role="tab" id="mac" aria-controls="mac-panel">Mac</button></div><div role="tabpanel" id="linux-panel" aria-labelledby="linux"><p>Linux archive</p></div><div role="tabpanel" id="mac-panel" aria-labelledby="mac" hidden aria-hidden="true"><p>Mac package</p><pre>go version</pre><script>do not execute me</script></div><p hidden>unrelated hidden text</p></main>`
	request := StructuredParseRequest{SourceID: "source", SnapshotID: "snapshot", CanonicalLocator: "https://go.dev/doc/install", Filename: "official.md"}
	parser := NewStructuredParser()
	old, err := parser.ParseOfficialWebHTML(strings.NewReader(source), request)
	require.NoError(t, err)
	require.NotContains(t, strings.Join(chunkTexts(old), "\n"), "Mac package")
	request.PreserveOfficialWebTabs = true
	current, err := parser.ParseOfficialWebHTML(strings.NewReader(source), request)
	require.NoError(t, err)
	text := strings.Join(chunkTexts(current), "\n")
	require.Contains(t, text, "Linux archive")
	require.Contains(t, text, "Mac package")
	require.NotContains(t, text, "unrelated hidden text")
	require.NotContains(t, text, "do not execute me")
	for _, chunk := range current.Chunks {
		if chunk.SearchText == "Mac package" || chunk.SearchText == "go version" {
			require.Equal(t, []string{"Setup", "Install", "Mac"}, chunk.Locator.HeadingPath)
		}
	}
	request.PreserveOfficialWebTabs = false
	replay, err := parser.ParseOfficialWebHTML(strings.NewReader(source), request)
	require.NoError(t, err)
	require.Equal(t, old, replay, "v2 replay must remain byte-for-byte stable")
}

func TestOfficialWebTabsDoNotExposeUnassociatedOrAmbiguousHiddenContent(t *testing.T) {
	for name, source := range map[string]string{
		"orphan":              `<main><h1>Visible</h1><div role="tabpanel" id="panel" aria-labelledby="missing" hidden><p>must stay hidden</p></div><p>body</p></main>`,
		"mismatched controls": `<main><h1>Visible</h1><div role="tablist"><button role="tab" id="tab" aria-controls="other">Mac</button></div><div role="tabpanel" id="panel" aria-labelledby="tab" hidden><p>must stay hidden</p></div><p>body</p></main>`,
		"duplicate labels":    `<main><h1>Visible</h1><div role="tablist"><button role="tab" id="tab" aria-controls="panel">Mac</button><button role="tab" id="tab" aria-controls="panel">Other</button></div><div role="tabpanel" id="panel" aria-labelledby="tab" hidden><p>must stay hidden</p></div><p>body</p></main>`,
		"not a tablist":       `<main><h1>Visible</h1><button role="tab" id="tab" aria-controls="panel">Mac</button><div role="tabpanel" id="panel" aria-labelledby="tab" hidden><p>must stay hidden</p></div><p>body</p></main>`,
	} {
		t.Run(name, func(t *testing.T) {
			request := StructuredParseRequest{SourceID: "source", SnapshotID: "snapshot", CanonicalLocator: "https://example.com/", Filename: "page.md", PreserveOfficialWebTabs: true}
			parsed, err := NewStructuredParser().ParseOfficialWebHTML(strings.NewReader(source), request)
			require.NoError(t, err)
			require.NotContains(t, strings.Join(chunkTexts(parsed), "\n"), "must stay hidden")
		})
	}
}

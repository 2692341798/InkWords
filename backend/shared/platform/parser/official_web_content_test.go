package parser

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestOfficialContentExcludesNavigationAndPreservesHighlightedCode(t *testing.T) {
	source := `<html><head><script>bad</script></head><body><nav><ul><li>Sidebar</li></ul></nav><aside><p>On this page</p></aside><main><h1>路由</h1><nav><p>In-page links</p></nav><p>调用 <code>GET</code> 后读取。</p><pre data-language="go"><code><div class="ec-line"><div class="code"><span>package</span><span> </span><span>main</span></div></div><div class="ec-line"><div class="code">
</div></div><div class="ec-line"><div class="code"><span>func main() {</span></div></div><div class="ec-line"><div class="code"><span>  println(&quot;&amp;lt;ok&amp;gt;&quot;)</span></div></div><div class="ec-line"><div class="code">}</div></div></code></pre><h2>边界</h2><p>不执行上游代码。</p><footer><p>Footer links</p></footer></main></body></html>`
	request := StructuredParseRequest{SourceID: "source", SnapshotID: "snapshot", CanonicalLocator: "https://example.com/docs/route", Filename: "page.md"}
	parsed, err := NewStructuredParser().ParseOfficialWebHTML(strings.NewReader(source), request)
	require.NoError(t, err)
	text := strings.Join(chunkTexts(parsed), "\n")
	for _, navigation := range []string{"Sidebar", "On this page", "In-page links", "Footer links", "bad"} {
		require.NotContains(t, text, navigation)
	}
	require.Contains(t, text, "调用 GET 后读取。")
	var codeCount int
	for _, chunk := range parsed.Chunks {
		if chunk.CodeLanguage == "go" {
			codeCount++
			require.Equal(t, "package main\n\nfunc main() {\n  println(\"&lt;ok&gt;\")\n}", chunk.SearchText)
			require.Equal(t, []string{"路由"}, chunk.Locator.HeadingPath)
		}
	}
	require.Equal(t, 1, codeCount)
	request.LegacyOfficialWeb = true
	legacy, err := NewStructuredParser().ParseOfficialWebHTML(strings.NewReader(source), request)
	require.NoError(t, err)
	require.Contains(t, strings.Join(chunkTexts(legacy), "\n"), "Sidebar")
}

func TestOfficialContentHandlesBodyFallbackNestedListsAndVoidElements(t *testing.T) {
	content := officialWebMarkdown([]byte(`<body><script><img></script><h1>Fallback</h1><ul><li>Parent<ul><li>Child <em>term</em></li></ul>Tail</li><li>Sibling</li></ul><p>Keep<img alt="diagram"> after<br>break &amp;lt;</p><pre><code class="language-go">x := 1
// keep newline
x++</code></pre></body>`))
	for _, v := range []string{"# Fallback", "Parent", "Child term", "Tail", "Sibling", "Keep after", "break &lt;", "```go\nx := 1\n// keep newline\nx++\n```"} {
		require.Contains(t, content, v)
	}
	require.Equal(t, 1, strings.Count(content, "Child term"))
}

func TestOfficialCodeContainingMarkdownFenceStaysOneCodeChunk(t *testing.T) {
	request := StructuredParseRequest{SourceID: "source", SnapshotID: "snapshot", CanonicalLocator: "https://example.com/docs", Filename: "page.md"}
	parsed, err := NewStructuredParser().ParseOfficialWebHTML(strings.NewReader("<main><h1>Code</h1><pre data-language=go>// ``` example\npackage main</pre></main>"), request)
	require.NoError(t, err)
	require.Len(t, parsed.Chunks, 1)
	require.Equal(t, "go", parsed.Chunks[0].CodeLanguage)
	require.Equal(t, "// ``` example\npackage main", parsed.Chunks[0].SearchText)
}

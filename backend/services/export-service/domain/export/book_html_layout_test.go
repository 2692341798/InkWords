package export

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestBookHTMLNavigationAndCodeRemainBoundToTheManuscript(t *testing.T) {
	code := strings.Repeat("// <angle> & preserved\n\n", 35) + "func main() {}\n"
	book, err := sharedtextbook.NewCanonicalBookAST("中文书名", time.Unix(1, 0), []sharedtextbook.CanonicalBookChapter{{
		ID: "chapter", Order: 1, Title: "中文章名", ContentHash: "sha256:" + strings.Repeat("a", 64),
		Markdown: "# 中文章名\n\n## 重复标题\n\n正文\n\n## 重复标题\n\n```go\n" + code + "```\n\n<script>alert(1)</script>",
	}})
	require.NoError(t, err)
	result, err := RenderBookHTML(book)
	require.NoError(t, err)
	doc, err := html.Parse(strings.NewReader(string(result)))
	require.NoError(t, err)
	ids := map[string]bool{}
	var anchors, codeUnits []string
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		for _, attr := range node.Attr {
			if attr.Key == "id" {
				ids[attr.Val] = true
			}
			if node.Data == "a" && attr.Key == "href" && strings.HasPrefix(attr.Val, "#") {
				anchors = append(anchors, attr.Val[1:])
			}
		}
		if node.Type == html.ElementNode && node.Data == "pre" {
			var text strings.Builder
			var collect func(*html.Node)
			collect = func(n *html.Node) {
				if n.Type == html.TextNode {
					text.WriteString(n.Data)
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					collect(c)
				}
			}
			collect(node)
			codeUnits = append(codeUnits, text.String())
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(doc)
	require.Len(t, anchors, 3)
	for _, id := range anchors {
		require.True(t, ids[id], id)
	}
	require.NotEqual(t, anchors[1], anchors[2])
	require.Greater(t, len(codeUnits), 1)
	require.Equal(t, code, strings.Join(codeUnits, ""))
	require.NotContains(t, string(result), "<script>")
	require.Contains(t, string(result), "aria-label=\"目录\"")
}

func TestBookFootnoteSeparatorRespectsInterveningProse(t *testing.T) {
	for _, item := range []struct {
		input      string
		separators int
	}{
		{`<p><sup>7</sup><sup>8</sup></p>`, 1},
		{`<p><sup>9</sup>终端窗口会显示提示符。<sup>10</sup></p>`, 0},
		{`<p><sup>1</sup> <sup>2</sup></p>`, 0},
		{`<pre>&lt;sup&gt;1&lt;/sup&gt;&lt;sup&gt;2&lt;/sup&gt;</pre>`, 0},
	} {
		result, err := separateBookFootnotes([]byte(item.input))
		require.NoError(t, err)
		require.Equal(t, item.separators, strings.Count(string(result), `class="footnote-separator"`))
		if item.separators == 0 {
			require.Equal(t, item.input, string(result))
		}
	}
}

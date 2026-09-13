package parser

import (
	"bytes"
	"golang.org/x/net/html"
	"strings"
)

// Project the document's main region, falling back to article/body for sites
// without a main landmark. HTML tree construction owns nesting and void-tag
// semantics; a token depth counter cannot reliably describe the rendered tree.
func officialWebMarkdown(content []byte) string {
	return officialWebMarkdownWithTabs(content, false)
}

func officialWebMarkdownWithTabs(content []byte, preserveTabs bool) string {
	document, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		return ""
	}
	root := document
	for _, tag := range []string{"main", "article", "body"} {
		if selected := findContentElement(document, tag); selected != nil {
			root = selected
			break
		}
	}
	if preserveTabs {
		expandOfficialTabPanels(root)
	}
	var output strings.Builder
	write := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			output.WriteString(value)
			output.WriteString("\n\n")
		}
	}
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if excludedContentNode(node) {
			return
		}
		if node.Type == html.ElementNode {
			if level := headingLevel(node.Data); level > 0 {
				write(strings.Repeat("#", level) + " " + normalizedContentText(node))
				return
			}
			if node.Data == "pre" {
				code := officialCodeText(node)
				if strings.TrimSpace(code) == "" {
					return
				}
				fence := "```"
				for strings.Contains(code, fence) {
					fence += "`"
				}
				write(fence + officialCodeLanguage(node) + "\n" + strings.TrimSuffix(code, "\n") + "\n" + fence)
				return
			}
		}
		var inline strings.Builder
		flush := func() { write(strings.Join(strings.Fields(inline.String()), " ")); inline.Reset() }
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if excludedContentNode(child) {
				continue
			}
			if child.Type == html.ElementNode && contentBlock(child.Data) {
				flush()
				walk(child)
			} else {
				inline.WriteString(contentText(child))
			}
		}
		flush()
	}
	walk(root)
	return strings.TrimSpace(output.String())
}

func findContentElement(node *html.Node, tag string) *html.Node {
	if excludedContentNode(node) {
		return nil
	}
	if node.Type == html.ElementNode && (node.Data == tag || (tag == "main" && htmlAttribute(node, "role") == "main")) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findContentElement(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func htmlAttribute(node *html.Node, key string) string {
	for _, a := range node.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func contentClass(node *html.Node, class string) bool {
	for _, v := range strings.Fields(htmlAttribute(node, "class")) {
		if v == class {
			return true
		}
	}
	return false
}

func excludedContentNode(node *html.Node) bool {
	if node.Type != html.ElementNode {
		return false
	}
	if ignoredHTMLTag(node.Data) {
		return true
	}
	switch node.Data {
	case "head", "nav", "aside", "header", "footer", "form", "button", "dialog":
		return true
	}
	for _, a := range node.Attr {
		if a.Key == "hidden" || (a.Key == "aria-hidden" && strings.EqualFold(a.Val, "true")) {
			return true
		}
	}
	switch htmlAttribute(node, "role") {
	case "navigation", "complementary", "search":
		return true
	}
	return contentClass(node, "sr-only") || contentClass(node, "visually-hidden")
}

func contentBlock(tag string) bool {
	if captureHTMLTag(tag) {
		return true
	}
	switch tag {
	case "div", "section", "main", "article", "figure", "figcaption", "ul", "ol", "dl", "dt", "dd", "table", "thead", "tbody", "tr":
		return true
	}
	return false
}

func contentText(node *html.Node) string {
	if excludedContentNode(node) {
		return ""
	}
	if node.Type == html.TextNode {
		return node.Data
	} // html.Parse already decoded entities exactly once.
	if node.Type == html.ElementNode && node.Data == "br" {
		return "\n"
	}
	var text strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		text.WriteString(contentText(child))
	}
	return text.String()
}

func normalizedContentText(node *html.Node) string {
	return strings.Join(strings.Fields(contentText(node)), " ")
}

func officialCodeText(node *html.Node) string {
	// Highlighting frameworks often encode newlines as block elements instead
	// of literal text. Join only the outer line wrappers, never individual tokens.
	var lines []string
	var collect func(*html.Node)
	collect = func(n *html.Node) {
		if excludedContentNode(n) {
			return
		}
		if n.Type == html.ElementNode && (contentClass(n, "ec-line") || contentClass(n, "line")) {
			lines = append(lines, strings.TrimSuffix(contentText(n), "\n"))
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			collect(child)
		}
	}
	collect(node)
	if len(lines) > 0 {
		return strings.Join(lines, "\n")
	}
	return contentText(node)
}

func officialCodeLanguage(node *html.Node) string {
	language := htmlAttribute(node, "data-language")
	if language == "" {
		for _, class := range strings.Fields(htmlAttribute(node, "class")) {
			if strings.HasPrefix(class, "language-") {
				language = strings.TrimPrefix(class, "language-")
				break
			}
		}
	}
	if language != "" {
		for _, r := range language {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '+' || r == '-' || r == '_') {
				return ""
			}
		}
		return language
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if language := officialCodeLanguage(child); language != "" {
			return language
		}
	}
	return ""
}

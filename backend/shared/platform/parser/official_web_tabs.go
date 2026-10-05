package parser

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// Inactive platform tabs are still source content. Expand only a reciprocal
// tab/tabpanel association with unambiguous IDs and an actual tablist; never
// execute page scripts or reveal arbitrary hidden nodes. Mutation is confined
// to the temporary parse tree, leaving the downloaded snapshot untouched.
func expandOfficialTabPanels(root *html.Node) {
	ids := map[string][]*html.Node{}
	var collect func(*html.Node)
	collect = func(node *html.Node) {
		if node.Type == html.ElementNode && ignoredHTMLTag(node.Data) {
			return
		}
		if id := htmlAttribute(node, "id"); id != "" {
			ids[id] = append(ids[id], node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			collect(child)
		}
	}
	collect(root)
	type panelExpansion struct {
		panel *html.Node
		label string
		level int
	}
	var expansions []panelExpansion
	for id, matches := range ids {
		if len(matches) != 1 {
			continue
		}
		panel := matches[0]
		if htmlAttribute(panel, "role") != "tabpanel" || (panel.Data != "div" && panel.Data != "section") || contentClass(panel, "sr-only") || contentClass(panel, "visually-hidden") {
			continue
		}
		labels := ids[htmlAttribute(panel, "aria-labelledby")]
		if len(labels) != 1 {
			continue
		}
		tab := labels[0]
		if htmlAttribute(tab, "role") != "tab" || htmlAttribute(tab, "aria-controls") != id || !visibleTabInList(tab, root) {
			continue
		}
		var text strings.Builder
		for child := tab.FirstChild; child != nil; child = child.NextSibling {
			text.WriteString(contentText(child))
		}
		label := strings.Join(strings.Fields(text.String()), " ")
		if label == "" || len([]rune(label)) > 200 {
			continue
		}
		level := headingBeforeTab(root, tab) + 1
		if level > 6 {
			level = 6
		}
		expansions = append(expansions, panelExpansion{panel: panel, label: label, level: level})
	}
	for _, expansion := range expansions {
		panel := expansion.panel
		attrs := make([]html.Attribute, 0, len(panel.Attr))
		for _, attr := range panel.Attr {
			if attr.Key != "hidden" && attr.Key != "aria-hidden" {
				attrs = append(attrs, attr)
			}
		}
		panel.Attr = attrs
		heading := &html.Node{Type: html.ElementNode, Data: "h" + strconv.Itoa(expansion.level)}
		heading.AppendChild(&html.Node{Type: html.TextNode, Data: expansion.label})
		panel.InsertBefore(heading, panel.FirstChild)
	}
}

func visibleTabInList(tab, root *html.Node) bool {
	// Buttons are intentionally omitted from prose; test their visibility
	// separately instead of treating the normal button exclusion as hidden.
	for _, attr := range tab.Attr {
		if attr.Key == "hidden" || attr.Key == "aria-hidden" && strings.EqualFold(attr.Val, "true") {
			return false
		}
	}
	if contentClass(tab, "sr-only") || contentClass(tab, "visually-hidden") {
		return false
	}
	found := false
	for parent := tab.Parent; parent != nil; parent = parent.Parent {
		if excludedContentNode(parent) {
			return false
		}
		if htmlAttribute(parent, "role") == "tablist" {
			found = true
		}
		if parent == root {
			return found
		}
	}
	return false
}

func headingBeforeTab(root, target *html.Node) int {
	level := 0
	var visit func(*html.Node) bool
	visit = func(node *html.Node) bool {
		if node == target {
			return true
		}
		if excludedContentNode(node) {
			return false
		}
		if n := headingLevel(node.Data); node.Type == html.ElementNode && n > 0 {
			level = n
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if visit(child) {
				return true
			}
		}
		return false
	}
	visit(root)
	return level
}

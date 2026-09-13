package bookrender

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/css"
	"github.com/chromedp/cdproto/dom"
	exportdomain "inkwords-backend/services/export-service/domain/export"
)

type bodyFontObservation struct {
	fonts []exportdomain.ObservedPDFFont
	nodes int
	err   error
}

func observeBodyFonts(ctx context.Context, inventory []exportdomain.FontFileEvidence, validateCharacters func([]*css.PlatformFontUsage, string) error) bodyFontObservation {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	observation := bodyFontObservation{fonts: []exportdomain.ObservedPDFFont{}}
	if err := dom.Enable().Do(ctx); err != nil {
		observation.err = err
		return observation
	}
	if err := css.Enable().Do(ctx); err != nil {
		observation.err = err
		return observation
	}
	root, err := dom.GetDocument().WithDepth(-1).Do(ctx)
	if err != nil {
		observation.err = err
		return observation
	}
	var nodes []struct {
		id   cdp.NodeID
		text string
	}
	var visit func(*cdp.Node, bool)
	visit = func(node *cdp.Node, inBody bool) {
		if node == nil {
			return
		}
		inBody = inBody || node.NodeName == "BODY"
		if inBody {
			var text strings.Builder
			for _, child := range node.Children {
				if child.NodeType == 3 && strings.TrimSpace(child.NodeValue) != "" {
					text.WriteString(child.NodeValue)
				}
			}
			if text.Len() > 0 {
				nodes = append(nodes, struct {
					id   cdp.NodeID
					text string
				}{node.NodeID, text.String()})
			}
		}
		for _, child := range node.Children {
			visit(child, inBody)
		}
		for _, pseudo := range node.PseudoElements {
			visit(pseudo, inBody)
		}
	}
	visit(root, false)
	if len(nodes) > 10000 {
		observation.err = fmt.Errorf("font observation node budget exceeded")
		return observation
	}
	observed := map[string]*exportdomain.ObservedPDFFont{}
	for _, node := range nodes {
		fonts, err := css.GetPlatformFontsForNode(node.id).Do(ctx)
		if err != nil {
			observation.err = err
			break
		}
		observation.nodes++
		if validateCharacters != nil {
			if err := validateCharacters(fonts, node.text); err != nil {
				observation.err = err
				break
			}
		}
		for _, font := range fonts {
			if font.GlyphCount <= 0 {
				continue
			}
			key := fmt.Sprintf("%s\x00%s\x00%t", font.PostScriptName, font.FamilyName, font.IsCustomFont)
			if observed[key] == nil {
				observed[key] = &exportdomain.ObservedPDFFont{PostScriptName: font.PostScriptName, Family: font.FamilyName, Custom: font.IsCustomFont, CandidateFiles: []exportdomain.FontFileEvidence{}}
			}
			observed[key].GlyphCount += font.GlyphCount
		}
	}
	for _, font := range observed {
		if !font.Custom && font.PostScriptName != "" {
			for _, file := range inventory {
				if file.PostScriptName == font.PostScriptName {
					font.CandidateFiles = append(font.CandidateFiles, file)
				}
			}
		}
		switch len(font.CandidateFiles) {
		case 0:
			font.MatchStatus = "unresolved"
		case 1:
			font.MatchStatus = "unique_inventory_candidate"
		default:
			font.MatchStatus = "ambiguous_inventory_candidates"
		}
		observation.fonts = append(observation.fonts, *font)
	}
	sort.Slice(observation.fonts, func(i, j int) bool {
		a, b := observation.fonts[i], observation.fonts[j]
		if a.PostScriptName != b.PostScriptName {
			return a.PostScriptName < b.PostScriptName
		}
		return a.Family < b.Family
	})
	return observation
}

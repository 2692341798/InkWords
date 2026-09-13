package export

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/net/html"
	shared "inkwords-backend/shared/kernel/textbook"
)

// PublicationReadyWithAssetFonts joins immutable raster bytes, scoped reviewer
// assertions and exact current rights versions. Other embedded font surfaces
// remain unsupported; this does not promote a book or replace rights review.
func (e PDFFontEvidence) PublicationReadyWithAssetFonts(book shared.CanonicalBookAST, projectID, buildID, manifestHash string, supplied *shared.RightsLedger, exportedRights []shared.RightsItem, source BookImageSource) bool {
	if supplied == nil || source == nil || supplied.ContractVersion != shared.RightsLedgerFormat || supplied.SelectionRule != "explicit-predecessor-chain-v1" {
		return false
	}
	ledger, err := shared.ResolveRightsLedger(buildID, manifestHash, supplied.OriginalItems, supplied.Amendments)
	if err != nil || !e.textFontRightsReady(projectID, buildID, ledger.EffectiveItems) {
		return false
	}
	if len(exportedRights) != len(ledger.EffectiveItems) {
		return false
	}
	byID := map[string]shared.RightsItem{}
	for _, item := range exportedRights {
		byID[item.ID] = item
	}
	for _, item := range ledger.EffectiveItems {
		if byID[item.ID] != item {
			return false
		}
	}
	for _, chapter := range book.Chapters {
		for _, asset := range chapter.Assets {
			if !shared.AssetFontsPublicationReady(asset, projectID, buildID, manifestHash, ledger) {
				return false
			}
		}
	}
	// The normal renderer rejects foreign image URLs, unsupported encodings,
	// truncated images and mismatched bytes before producing data URIs.
	document, err := RenderBookHTML(book, source)
	return err == nil && hasOnlyReviewedRasterFontSurfaces(document)
}

func hasOnlyReviewedRasterFontSurfaces(document []byte) bool {
	tokens := html.NewTokenizer(bytes.NewReader(document))
	for {
		switch tokens.Next() {
		case html.ErrorToken:
			return errors.Is(tokens.Err(), io.EOF)
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokens.Token()
			switch token.Data {
			case "svg", "object", "embed", "iframe", "canvas", "video":
				return false
			case "img":
				valid := false
				for _, attr := range token.Attr {
					if attr.Key == "src" {
						valid = strings.HasPrefix(attr.Val, "data:image/png;base64,") || strings.HasPrefix(attr.Val, "data:image/jpeg;base64,")
					}
				}
				if !valid {
					return false
				}
			}
		}
	}
}

package export

import (
	"bytes"
	"fmt"
	"io"

	"golang.org/x/net/html"
	shared "inkwords-backend/shared/kernel/textbook"
)

// PDFFontSourceSet records a constrained Fontconfig input set, separately from
// DOM observations. It is neither an OS file-open trace nor license approval.
type PDFFontSourceSet struct {
	Format               string            `json:"format"`
	Scope                string            `json:"scope"`
	ConfigurationHash    string            `json:"configuration_hash"`
	ConfigurationSources map[string]string `json:"configuration_sources"`
	FileHashes           []string          `json:"file_hashes"`
	FooterCharsetStatus  string            `json:"footer_charset_status"`
	BodyCharsetStatus    string            `json:"body_charset_status"`
	InventoryStable      bool              `json:"inventory_stable_before_after_print"`
}

// PublicationReadyForBook joins the actual PDF source constraints with this
// build's effective rights. It clears only the PDF font gate, not book preflight.
func (e PDFFontEvidence) PublicationReadyForBook(book shared.CanonicalBookAST, projectID, buildID string, rights []shared.RightsItem) bool {
	document, err := RenderBookHTML(book)
	return err == nil && hasOnlyTextFontSurfaces(document) && e.textFontRightsReady(projectID, buildID, rights)
}

func (e PDFFontEvidence) textFontRightsReady(projectID, buildID string, rights []shared.RightsItem) bool {
	if projectID == "" || buildID == "" || e.SourceSet == nil || e.SourceSet.validate(e.InstalledFaces) != nil || e.BodyObservationStatus != "partial" || e.ObservedTextNodes <= 0 || len(e.Fonts) == 0 {
		return false
	}
	for _, font := range e.Fonts {
		if font.Custom || font.MatchStatus != "unique_inventory_candidate" {
			return false
		}
	}
	for _, hash := range e.SourceSet.FileHashes {
		matches := 0
		for _, item := range rights {
			if item.ProjectID != projectID || item.BuildID != buildID || item.SubjectRef != "font-file:"+hash || item.WorkType != shared.RightsWorkTypeFont {
				continue
			}
			if item.Validate() != nil || item.PublicationStatus != shared.RightsStatusReady {
				return false
			}
			matches++
		}
		if matches != 1 {
			return false
		}
	}
	return true
}

// Fontconfig does not account for fonts already drawn/embedded inside an image
// or object. Such a document needs a separate asset-font review contract.
func hasOnlyTextFontSurfaces(document []byte) bool {
	tokens := html.NewTokenizer(bytes.NewReader(document))
	for {
		switch tokens.Next() {
		case html.ErrorToken:
			return errors.Is(tokens.Err(), io.EOF)
		case html.StartTagToken, html.SelfClosingTagToken:
			name, _ := tokens.TagName()
			switch string(name) {
			case "img", "svg", "object", "embed", "iframe", "canvas", "video":
				return false
			}
		}
	}
}

func (s PDFFontSourceSet) validate(faces []FontFileEvidence) error {
	if s.Format != "inkwords.pdf-font-source-set.v1" || s.Scope != "fontconfig_text_and_print_furniture" || !validFontDigest(s.ConfigurationHash) || len(s.ConfigurationSources) == 0 || len(s.FileHashes) == 0 || s.FooterCharsetStatus != "covered_by_source_set" || s.BodyCharsetStatus != "covered_by_observed_font_candidates" || !s.InventoryStable {
		return fmt.Errorf("invalid PDF font source set")
	}
	for path, hash := range s.ConfigurationSources {
		if path == "" || !validFontDigest(hash) {
			return fmt.Errorf("invalid font configuration source")
		}
	}
	declared := map[string]bool{}
	for _, hash := range s.FileHashes {
		if !validFontDigest(hash) || declared[hash] {
			return fmt.Errorf("invalid or duplicate font source hash")
		}
		declared[hash] = true
	}
	seen := map[string]bool{}
	for _, face := range faces {
		if !declared[face.SHA256] {
			return fmt.Errorf("font inventory escapes source set")
		}
		seen[face.SHA256] = true
	}
	if len(seen) != len(declared) {
		return fmt.Errorf("font inventory omits declared source")
	}
	return nil
}

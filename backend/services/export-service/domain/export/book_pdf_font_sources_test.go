package export

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	shared "inkwords-backend/shared/kernel/textbook"
)

func TestPDFFontSourceSetRequiresCompleteStableInventory(t *testing.T) {
	hash := "sha256:" + strings.Repeat("a", 64)
	source := PDFFontSourceSet{Format: "inkwords.pdf-font-source-set.v1", Scope: "fontconfig_text_and_print_furniture", ConfigurationHash: hash, ConfigurationSources: map[string]string{"rules.conf": hash}, FileHashes: []string{hash}, FooterCharsetStatus: "covered_by_source_set", BodyCharsetStatus: "covered_by_observed_font_candidates", InventoryStable: true}
	faces := []FontFileEvidence{{SHA256: hash}}
	require.NoError(t, source.validate(faces))
	require.Error(t, source.validate(nil))
	require.Error(t, source.validate([]FontFileEvidence{{SHA256: "sha256:" + strings.Repeat("b", 64)}}))
	for _, change := range []func(*PDFFontSourceSet){
		func(s *PDFFontSourceSet) { s.InventoryStable = false },
		func(s *PDFFontSourceSet) { s.FooterCharsetStatus = "unobserved" },
		func(s *PDFFontSourceSet) { s.BodyCharsetStatus = "unobserved" },
		func(s *PDFFontSourceSet) { s.ConfigurationHash = "unknown" },
		func(s *PDFFontSourceSet) { s.FileHashes = append(s.FileHashes, hash) },
		func(s *PDFFontSourceSet) { s.ConfigurationSources = nil },
	} {
		copy := source
		change(&copy)
		require.Error(t, copy.validate(faces))
	}
}

func TestPDFFontGateJoinsExactBuildRightsAndRejectsUnreviewedSurfaces(t *testing.T) {
	hash := "sha256:" + strings.Repeat("a", 64)
	book, err := shared.NewCanonicalBookAST("字体门禁", time.Unix(1, 0), []shared.CanonicalBookChapter{{ID: "chapter", Order: 1, Title: "章", Markdown: "# 章\n\n正文", ContentHash: hash}})
	require.NoError(t, err)
	face := FontFileEvidence{SHA256: hash}
	source := PDFFontSourceSet{Format: "inkwords.pdf-font-source-set.v1", Scope: "fontconfig_text_and_print_furniture", ConfigurationHash: hash, ConfigurationSources: map[string]string{"rules.conf": hash}, FileHashes: []string{hash}, FooterCharsetStatus: "covered_by_source_set", BodyCharsetStatus: "covered_by_observed_font_candidates", InventoryStable: true}
	evidence := PDFFontEvidence{SourceSet: &source, BodyObservationStatus: "partial", ObservedTextNodes: 1, InstalledFaces: []FontFileEvidence{face}, Fonts: []ObservedPDFFont{{MatchStatus: "unique_inventory_candidate"}}}
	item := shared.RightsItem{ID: "font-rights", ProjectID: "project", BuildID: "build", SubjectRef: "font-file:" + hash, WorkType: shared.RightsWorkTypeFont, RightsBasis: "fixture license evidence", AllowedUse: "document embedding", Attribution: "fixture font author", PublicationStatus: shared.RightsStatusReady}
	require.True(t, evidence.PublicationReadyForBook(book, "project", "build", []shared.RightsItem{item}))
	require.False(t, evidence.PublicationReadyForBook(book, "other", "build", []shared.RightsItem{item}))
	require.False(t, evidence.PublicationReadyForBook(book, "project", "other", []shared.RightsItem{item}))
	require.False(t, evidence.PublicationReadyForBook(book, "project", "build", nil))
	require.False(t, evidence.PublicationReadyForBook(book, "project", "build", []shared.RightsItem{item, item}))
	for _, change := range []func(*shared.RightsItem){
		func(r *shared.RightsItem) { r.PublicationStatus = shared.RightsStatusPending },
		func(r *shared.RightsItem) { r.SubjectRef = "font-file:unknown" },
		func(r *shared.RightsItem) { r.WorkType = shared.RightsWorkTypeCode },
		func(r *shared.RightsItem) { r.AllowedUse = "" },
	} {
		copy := item
		change(&copy)
		require.False(t, evidence.PublicationReadyForBook(book, "project", "build", []shared.RightsItem{copy}))
	}
	require.True(t, hasOnlyTextFontSurfaces([]byte(`<p>文字</p>`)))
	for _, surface := range []string{`<img src="data:...">`, `<svg></svg>`, `<object></object>`, `<canvas></canvas>`} {
		require.False(t, hasOnlyTextFontSurfaces([]byte(surface)))
	}
}

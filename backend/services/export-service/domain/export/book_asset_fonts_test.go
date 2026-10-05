package export

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	shared "inkwords-backend/shared/kernel/textbook"
)

func assetFontPackageFixture(t *testing.T) BookPackageInput {
	t.Helper()
	book, images := imageBook(t)
	document, err := RenderBookHTML(book, images)
	require.NoError(t, err)
	content := []byte("%PDF-asset-font-fixture")
	e := NewPDFFontEvidence(book, document, content, time.Unix(1, 0))
	hash := "sha256:" + strings.Repeat("a", 64)
	e.RendererVersion, e.BodyObservationStatus, e.ObservedTextNodes = "fixture-browser", "partial", 1
	face := FontFileEvidence{Path: "/fixture/font.ttf", SHA256: hash, FaceIndex: "0", FontVersion: "1", PostScriptName: "Fixture", LicenseStatus: "not_reviewed"}
	e.InstalledFaces = []FontFileEvidence{face}
	e.Fonts = []ObservedPDFFont{{PostScriptName: "Fixture", GlyphCount: 1, MatchStatus: "unique_inventory_candidate", CandidateFiles: []FontFileEvidence{face}}}
	e.SourceSet = &PDFFontSourceSet{Format: "inkwords.pdf-font-source-set.v1", Scope: "fontconfig_text_and_print_furniture", ConfigurationHash: hash, ConfigurationSources: map[string]string{"rules": hash}, FileHashes: []string{hash}, FooterCharsetStatus: "covered_by_source_set", BodyCharsetStatus: "covered_by_observed_font_candidates", InventoryStable: true}
	font := shared.RightsItem{ID: "font", ProjectID: "project", BuildID: "build", SubjectRef: "font-file:" + hash, WorkType: shared.RightsWorkTypeFont, RightsBasis: "fixture font license", AllowedUse: "fixture document and raster use", Attribution: "fixture author", PublicationStatus: shared.RightsStatusReady}
	asset := book.Chapters[0].Assets[0]
	base := font
	base.ID, base.SubjectRef, base.WorkType, base.PublicationStatus = "asset-rights", "asset:"+asset.ID, shared.RightsWorkTypeScreenshot, shared.RightsStatusPending
	effective := base
	effective.PublicationStatus = shared.RightsStatusReady
	a := shared.RightsAmendment{ContractVersion: shared.RightsAmendmentFormat, ID: "asset-review", BuildID: "build", ManifestHash: hash, BaseItemID: base.ID, Revision: 1, ReviewerKind: "delegated_ai", Reviewer: "fixture", DelegationNote: "fixture explicit delegation", Reason: "fixture complete font review", EvidenceRefs: []string{"fixture:font-file-origin"}, EffectiveItem: effective, CompletedAt: time.Unix(2, 0), AssetFontReview: &shared.AssetFontReview{ContractVersion: shared.AssetFontReviewFormat, AssetID: asset.ID, ContentHash: asset.ContentHash, Surface: "raster", TextStatus: "identified_fonts", Scope: "fixture complete raster text coverage", Fonts: []shared.AssetFontRightsRef{{FileHash: hash, RightsItemID: font.ID}}}}
	ledger, err := shared.ResolveRightsLedger("build", hash, []shared.RightsItem{base, font}, []shared.RightsAmendment{a})
	require.NoError(t, err)
	input := BookPackageInput{BuildID: "build", ManifestHash: hash, Book: book, PDF: &BookPDFProjection{Content: content, FontEvidence: &e}, DOCX: &BookDOCXProjection{Content: []byte("PK\x03\x04fixture")}, RightsLedger: &ledger, RightsItems: ledger.EffectiveItems, BuildManifest: json.RawMessage(`{"project_id":"project"}`), Preflight: PublicationPreflightResult{Passed: false, Blockers: []string{"读者试学未完成"}}}
	for digest, data := range images {
		input.Assets = append(input.Assets, BookPackageFile{Path: strings.TrimPrefix(digest, "sha256:") + ".png", Content: data})
	}
	return input
}

func TestRasterFontEvidenceClearsOnlyItsOwnPackageGate(t *testing.T) {
	input := assetFontPackageFixture(t)
	var output bytes.Buffer
	require.NoError(t, NewBookPackageBuilder().Build(&output, input))
	z, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	require.NoError(t, err)
	for _, file := range z.File {
		if file.Name == "manifest.json" {
			var manifest struct {
				PublicationCandidate bool     `json:"publication_candidate"`
				Blockers             []string `json:"publication_blockers"`
			}
			require.NoError(t, json.Unmarshal([]byte(readZipFile(t, file)), &manifest))
			require.False(t, manifest.PublicationCandidate)
			require.Equal(t, []string{"读者试学未完成"}, manifest.Blockers)
		}
		if file.Name == "rights-ledger.json" {
			require.Contains(t, readZipFile(t, file), shared.AssetFontReviewFormat)
		}
	}
}

func TestRasterFontGateRejectsUnboundUnsupportedAndStaleEvidence(t *testing.T) {
	for name, change := range map[string]func(*BookPackageInput){
		"missing ledger":  func(i *BookPackageInput) { i.RightsLedger = nil },
		"wrong build":     func(i *BookPackageInput) { i.BuildID = "other" },
		"wrong manifest":  func(i *BookPackageInput) { i.ManifestHash = "sha256:" + strings.Repeat("b", 64) },
		"no image source": func(i *BookPackageInput) { i.Assets = nil },
		"tampered image":  func(i *BookPackageInput) { i.Assets[0].Content = []byte("changed") },
		"changed assessment hash": func(i *BookPackageInput) {
			i.RightsLedger.Amendments[0].AssetFontReview.ContentHash = "sha256:" + strings.Repeat("b", 64)
		},
		"missing assessment": func(i *BookPackageInput) { i.RightsLedger.Amendments[0].AssetFontReview = nil },
		"wrong exported rights": func(i *BookPackageInput) {
			i.RightsItems = append([]shared.RightsItem{}, i.RightsItems...)
			i.RightsItems[0].PublicationStatus = shared.RightsStatusPending
		},
	} {
		t.Run(name, func(t *testing.T) {
			i := assetFontPackageFixture(t)
			change(&i)
			require.False(t, i.PDF.FontEvidence.PublicationReadyWithAssetFonts(i.Book, "project", i.BuildID, i.ManifestHash, i.RightsLedger, i.RightsItems, bookPackageImageSource(i.Assets)))
		})
	}
	for _, surface := range []string{`<img src="https://example.com/x.png">`, `<img src="data:image/svg+xml;base64,aa">`, `<svg></svg>`, `<object></object>`, `<iframe></iframe>`, `<video></video>`} {
		require.False(t, hasOnlyReviewedRasterFontSurfaces([]byte(surface)))
	}
}

package export

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	shared "inkwords-backend/shared/kernel/textbook"
)

func TestPDFFontEvidenceBindsExactProjectionAndKeepsUnknownCoverageExplicit(t *testing.T) {
	book, err := shared.NewCanonicalBookAST("字体证据", time.Unix(1, 0), []shared.CanonicalBookChapter{{ID: "chapter", Order: 1, Title: "章节", Markdown: "# 章节", ContentHash: "sha256:" + strings.Repeat("a", 64)}})
	require.NoError(t, err)
	content := []byte("%PDF-1.7 fixture")
	html, err := RenderBookHTML(book)
	require.NoError(t, err)
	evidence := NewPDFFontEvidence(book, html, content, time.Unix(2, 0))
	require.NoError(t, evidence.ValidateBinding(book, content))
	require.Equal(t, "unavailable", evidence.BodyObservationStatus)
	require.Equal(t, "unobserved", evidence.PrintFurnitureStatus)
	require.False(t, evidence.PublicationReady())
	require.Error(t, evidence.ValidateBinding(book, append(content, 'x')))
	changed := book
	changed.Title = "其它稿件"
	require.Error(t, evidence.ValidateBinding(changed, content))
	evidence.BodyObservationStatus = "complete"
	require.Error(t, evidence.ValidateBinding(book, content), "unsupported completeness cannot be asserted")
}

func TestPDFPackageIncludesBoundFontEvidenceAndCannotPromoteIncompleteTrace(t *testing.T) {
	book, err := shared.NewCanonicalBookAST("字体审校", time.Unix(1, 0), []shared.CanonicalBookChapter{{ID: "chapter", Order: 1, Title: "章", Markdown: "# 章", ContentHash: "sha256:" + strings.Repeat("a", 64)}})
	require.NoError(t, err)
	content := []byte("%PDF-1.7 fixture")
	html, err := RenderBookHTML(book)
	require.NoError(t, err)
	evidence := NewPDFFontEvidence(book, html, content, time.Unix(2, 0))
	input := BookPackageInput{Book: book, PDF: &BookPDFProjection{Content: content, FontEvidence: &evidence}, DOCX: &BookDOCXProjection{Content: []byte("PK\x03\x04fixture")}, BuildManifest: []byte(`{"format":"fixture"}`), Preflight: PublicationPreflightResult{Passed: true}}
	var buffer bytes.Buffer
	require.NoError(t, NewBookPackageBuilder().Build(&buffer, input))
	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	require.NoError(t, err)
	files := map[string]*zip.File{}
	for _, file := range reader.File {
		files[file.Name] = file
	}
	require.Contains(t, readZipFile(t, files["projections/book.pdf.font-evidence.json"]), evidence.PDFHash)
	require.Contains(t, readZipFile(t, files["manifest.json"]), `"publication_candidate": false`)
	require.Contains(t, readZipFile(t, files["manifest.json"]), "PDF 字体来源与许可证据尚未完整覆盖正文及页眉页脚。")
	input.PDF.Content = []byte("%PDF-1.7 changed")
	require.Error(t, NewBookPackageBuilder().Build(&bytes.Buffer{}, input))
}

func TestPDFPackageClearsOnlyFontBlockerAfterBoundSourceAndRightsJoin(t *testing.T) {
	hash := "sha256:" + strings.Repeat("a", 64)
	book, err := shared.NewCanonicalBookAST("字体联验", time.Unix(1, 0), []shared.CanonicalBookChapter{{ID: "chapter", Order: 1, Title: "章", Markdown: "# 章", ContentHash: hash}})
	require.NoError(t, err)
	content := []byte("%PDF-1.7 fixture")
	html, err := RenderBookHTML(book)
	require.NoError(t, err)
	evidence := NewPDFFontEvidence(book, html, content, time.Unix(2, 0))
	face := FontFileEvidence{Path: "/usr/share/fonts/fixture.ttf", SHA256: hash, FaceIndex: "0", FontVersion: "1", PostScriptName: "Fixture", Family: "Fixture", LicenseStatus: "not_reviewed"}
	evidence.RendererVersion, evidence.BodyObservationStatus, evidence.ObservedTextNodes = "fixture-browser", "partial", 1
	evidence.InstalledFaces = []FontFileEvidence{face}
	evidence.Fonts = []ObservedPDFFont{{PostScriptName: "Fixture", GlyphCount: 1, MatchStatus: "unique_inventory_candidate", CandidateFiles: []FontFileEvidence{face}}}
	evidence.SourceSet = &PDFFontSourceSet{Format: "inkwords.pdf-font-source-set.v1", Scope: "fontconfig_text_and_print_furniture", ConfigurationHash: hash, ConfigurationSources: map[string]string{"rules.conf": hash}, FileHashes: []string{hash}, FooterCharsetStatus: "covered_by_source_set", BodyCharsetStatus: "covered_by_observed_font_candidates", InventoryStable: true}
	item := shared.RightsItem{ID: "font-rights", ProjectID: "project", BuildID: "build", SubjectRef: "font-file:" + hash, WorkType: shared.RightsWorkTypeFont, RightsBasis: "fixture license", AllowedUse: "document embedding", Attribution: "fixture author", PublicationStatus: shared.RightsStatusReady}
	input := BookPackageInput{BuildID: "build", Book: book, PDF: &BookPDFProjection{Content: content, FontEvidence: &evidence}, DOCX: &BookDOCXProjection{Content: []byte("PK\x03\x04fixture")}, BuildManifest: []byte(`{"project_id":"project"}`), RightsItems: []shared.RightsItem{item}, Preflight: PublicationPreflightResult{Passed: false, Blockers: []string{"正文权利待审"}}}
	check := func(wantBlocked bool) {
		var buffer bytes.Buffer
		require.NoError(t, NewBookPackageBuilder().Build(&buffer, input))
		reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
		require.NoError(t, err)
		for _, file := range reader.File {
			if file.Name != "manifest.json" {
				continue
			}
			manifest := readZipFile(t, file)
			require.Contains(t, manifest, `"publication_candidate": false`)
			require.Contains(t, manifest, "正文权利待审")
			require.Equal(t, wantBlocked, strings.Contains(manifest, "PDF 字体来源与许可证据"))
			return
		}
		t.Fatal("missing package manifest")
	}
	check(false)
	input.BuildID = "other-build"
	check(true)
}

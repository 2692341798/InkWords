package export

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	shared "inkwords-backend/shared/kernel/textbook"
)

const PDFFontEvidenceFormat = "inkwords.pdf-font-evidence.v1"

// FontFileEvidence identifies a face from the renderer's local Fontconfig
// inventory. A match is an inference by PostScript name, not a file-open trace.
type FontFileEvidence struct {
	Path           string `json:"path"`
	SHA256         string `json:"sha256"`
	FaceIndex      string `json:"face_index"`
	FontVersion    string `json:"fontconfig_version_raw"`
	PostScriptName string `json:"postscript_name"`
	Family         string `json:"family"`
	LicenseStatus  string `json:"license_status"`
}

// ObservedPDFFont comes from actual print-media DOM text-node observations.
// GlyphCount is the sum reported for those nodes, not the PDF's unique glyphs.
type ObservedPDFFont struct {
	PostScriptName string             `json:"postscript_name"`
	Family         string             `json:"family"`
	Custom         bool               `json:"custom"`
	GlyphCount     float64            `json:"observed_glyph_count"`
	MatchStatus    string             `json:"match_status"`
	CandidateFiles []FontFileEvidence `json:"candidate_files"`
}

// PDFFontEvidence distinguishes observed body fonts from installed candidates
// and explicitly uncovered browser-generated headers/footers.
type PDFFontEvidence struct {
	Format                string             `json:"format"`
	BookASTHash           string             `json:"book_ast_hash"`
	HTMLHash              string             `json:"html_hash"`
	PDFHash               string             `json:"pdf_hash"`
	RendererVersion       string             `json:"renderer_version"`
	ObservedAt            time.Time          `json:"observed_at"`
	BodyObservationStatus string             `json:"body_observation_status"`
	PrintFurnitureStatus  string             `json:"print_furniture_status"`
	ObservedTextNodes     int                `json:"observed_text_nodes"`
	Fonts                 []ObservedPDFFont  `json:"observed_fonts"`
	InstalledFaces        []FontFileEvidence `json:"installed_faces"`
	SourceSet             *PDFFontSourceSet  `json:"source_set,omitempty"`
	Limitations           []string           `json:"limitations"`
}

func fontEvidenceDigest(data []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(data)) }

// NewPDFFontEvidence binds the original AST, actual HTML and final PDF bytes.
// Observation completeness and licensing never follow from successful printing.
func NewPDFFontEvidence(book shared.CanonicalBookAST, html, pdf []byte, at time.Time) PDFFontEvidence {
	raw, _ := json.Marshal(book)
	return PDFFontEvidence{Format: PDFFontEvidenceFormat, BookASTHash: fontEvidenceDigest(raw), HTMLHash: fontEvidenceDigest(html), PDFHash: fontEvidenceDigest(pdf), ObservedAt: at, BodyObservationStatus: "unavailable", PrintFurnitureStatus: "unobserved", Fonts: []ObservedPDFFont{}, InstalledFaces: []FontFileEvidence{}, Limitations: []string{
		"正文观测来自 Chromium print media 文本节点；不等同于逐 PDF 字体对象或文件打开跟踪。",
		"浏览器生成的页眉、页脚与页码未被正文 DOM 接口覆盖，不能据此宣称所有字体已经追溯。",
		"字体文件候选按 PostScript 名称匹配 Fontconfig 清单；同名多候选、缺失或自定义字体不能推断唯一文件。",
		"字形计数是被查询节点的观测累计，不是 PDF 唯一字形或字符统计。字体许可与分发条件仍需单独核对。",
	}}
}

// ValidateBinding rejects evidence copied from another book or output file.
func (e PDFFontEvidence) ValidateBinding(book shared.CanonicalBookAST, pdf []byte, sources ...BookImageSource) error {
	raw, err := json.Marshal(book)
	if err != nil {
		return err
	}
	html, err := RenderBookHTML(book, sources...)
	if err != nil {
		return err
	}
	if e.Format != PDFFontEvidenceFormat || e.BookASTHash != fontEvidenceDigest(raw) || e.PDFHash != fontEvidenceDigest(pdf) || e.ObservedAt.IsZero() || e.HTMLHash != fontEvidenceDigest(html) || e.PrintFurnitureStatus != "unobserved" || (e.BodyObservationStatus != "partial" && e.BodyObservationStatus != "unavailable") || len(e.Limitations) == 0 || e.ObservedTextNodes < 0 {
		return fmt.Errorf("invalid PDF font evidence binding or scope")
	}
	if e.BodyObservationStatus == "partial" && (e.ObservedTextNodes <= 0 || len(e.Fonts) == 0 || e.RendererVersion == "") {
		return fmt.Errorf("missing observed body fonts")
	}
	for _, file := range e.InstalledFaces {
		if !validFontDigest(file.SHA256) || file.Path == "" || file.FaceIndex == "" || file.FontVersion == "" || file.PostScriptName == "" || file.LicenseStatus != "not_reviewed" {
			return fmt.Errorf("invalid font file evidence")
		}
	}
	for _, font := range e.Fonts {
		if font.GlyphCount <= 0 || math.IsInf(font.GlyphCount, 0) || math.IsNaN(font.GlyphCount) {
			return fmt.Errorf("invalid font glyph observation")
		}
		expected := "unresolved"
		if len(font.CandidateFiles) == 1 {
			expected = "unique_inventory_candidate"
		} else if len(font.CandidateFiles) > 1 {
			expected = "ambiguous_inventory_candidates"
		}
		if font.MatchStatus != expected || (font.Custom && len(font.CandidateFiles) > 0) {
			return fmt.Errorf("invalid font candidate attribution")
		}
		for _, candidate := range font.CandidateFiles {
			found := false
			for _, file := range e.InstalledFaces {
				if candidate == file && candidate.PostScriptName == font.PostScriptName {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("font candidate absent from inventory")
			}
		}
	}
	if e.SourceSet != nil {
		return e.SourceSet.validate(e.InstalledFaces)
	}
	return nil
}

func validFontDigest(value string) bool {
	raw, ok := strings.CutPrefix(value, "sha256:")
	if !ok || len(raw) != 64 {
		return false
	}
	_, err := hex.DecodeString(raw)
	return err == nil
}

// PublicationReady without book/rights context cannot establish eligibility.
// Package assembly uses PublicationReadyForBook for the closed-source contract.
func (e PDFFontEvidence) PublicationReady() bool { return false }

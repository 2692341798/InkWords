package export

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

const bookPackageSchemaVersion = "inkwords.book-package.v1"

// BookPackageFile is a content-addressed projection supplied by a trusted
// artifact store. Its relative path is validated before ZIP creation.
type BookPackageFile struct {
	Path    string
	Content []byte
}

type BookPackageInput struct {
	BuildID             string
	ManifestHash        string
	Book                sharedtextbook.CanonicalBookAST
	DOCX                *BookDOCXProjection
	PDF                 *BookPDFProjection
	CodeArtifacts       []BookPackageFile
	Assets              []BookPackageFile
	VerificationSummary json.RawMessage
	RightsItems         []sharedtextbook.RightsItem
	RightsLedger        *sharedtextbook.RightsLedger
	HumanReviews        []sharedtextbook.HumanPublicationReview
	DelegatedReviews    []sharedtextbook.DelegatedPublicationReview
	QualityReport       json.RawMessage
	Preflight           PublicationPreflightResult
	BuildManifest       json.RawMessage
}

// BookPackageBuilder emits the portable whole-book package. It never renders
// formats itself: every projection is supplied from the same frozen AST.
type BookPackageBuilder struct{}

func NewBookPackageBuilder() *BookPackageBuilder { return &BookPackageBuilder{} }

func (builder *BookPackageBuilder) Build(writer io.Writer, input BookPackageInput) error {
	if builder == nil {
		return fmt.Errorf("book package builder is not configured")
	}
	if err := input.Book.Validate(); err != nil {
		return fmt.Errorf("validate package book AST: %w", err)
	}
	runbookFiles, err := frozenBookRunbookFiles(input.BuildManifest, input.Book)
	if err != nil {
		return fmt.Errorf("validate frozen video runbooks: %w", err)
	}
	markdown, err := RenderBookMarkdown(input.Book, bookPackageImageSource(input.Assets))
	if err != nil {
		return err
	}
	archive := zip.NewWriter(writer)
	closed := false
	defer func() {
		if !closed {
			_ = archive.Close()
		}
	}()
	hashes := make(map[string]string)
	writeBytes := func(path string, content []byte) error {
		if err := safeBookPackagePath(path); err != nil {
			return err
		}
		if _, exists := hashes[path]; exists {
			return fmt.Errorf("duplicate book package path: %s", path)
		}
		file, err := archive.Create(path)
		if err != nil {
			return fmt.Errorf("create %s: %w", path, err)
		}
		if _, err := file.Write(content); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		sum := sha256.Sum256(content)
		hashes[path] = "sha256:" + hex.EncodeToString(sum[:])
		return nil
	}
	writeJSON := func(path string, value any) error {
		payload, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal %s: %w", path, err)
		}
		return writeBytes(path, payload)
	}
	if err := writeJSON("manuscript/book.json", input.Book); err != nil {
		return err
	}
	if err := writeBytes("projections/book.md", markdown); err != nil {
		return err
	}
	for _, file := range runbookFiles {
		if err := writeBytes(file.Path, file.Content); err != nil {
			return err
		}
	}
	if input.DOCX != nil {
		if !bytesHavePrefix(input.DOCX.Content, "PK\x03\x04") {
			return fmt.Errorf("book DOCX projection is invalid")
		}
		if err := writeBytes("projections/book.docx", input.DOCX.Content); err != nil {
			return err
		}
	}
	if input.PDF != nil {
		if !bytesHavePrefix(input.PDF.Content, "%PDF-") {
			return fmt.Errorf("book PDF projection is invalid")
		}
		if err := writeBytes("projections/book.pdf", input.PDF.Content); err != nil {
			return err
		}
		if err := writeBytes("projections/book.pdf.render.log", []byte(input.PDF.RenderLog)); err != nil {
			return err
		}
		if input.PDF.FontEvidence != nil {
			if err := input.PDF.FontEvidence.ValidateBinding(input.Book, input.PDF.Content, bookPackageImageSource(input.Assets)); err != nil {
				return err
			}
			if err := writeJSON("projections/book.pdf.font-evidence.json", input.PDF.FontEvidence); err != nil {
				return err
			}
		}
	}
	for _, file := range input.CodeArtifacts {
		if err := writeBytes("code/"+file.Path, file.Content); err != nil {
			return err
		}
	}
	for _, file := range input.Assets {
		if err := writeBytes("assets/"+file.Path, file.Content); err != nil {
			return err
		}
	}
	for _, file := range bookNoticeFiles(input.Book) {
		if err := writeBytes(file.Path, file.Content); err != nil {
			return err
		}
	}
	if err := writeJSON("verification-summary.json", input.VerificationSummary); err != nil {
		return err
	}
	if err := writeJSON("rights.json", input.RightsItems); err != nil {
		return err
	}
	if input.RightsLedger != nil {
		if err := writeJSON("rights-ledger.json", input.RightsLedger); err != nil {
			return err
		}
	}
	if err := writeJSON("human-reviews.json", input.HumanReviews); err != nil {
		return err
	}
	if err := writeJSON("delegated-ai-reviews.json", input.DelegatedReviews); err != nil {
		return err
	}
	if err := writeJSON("quality-report.json", input.QualityReport); err != nil {
		return err
	}
	if err := writeJSON("publication-preflight.json", input.Preflight); err != nil {
		return err
	}
	if err := writeJSON("book-build-manifest.json", input.BuildManifest); err != nil {
		return err
	}
	paths := make([]string, 0, len(hashes))
	for path := range hashes {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	publicationBlockers := append([]string(nil), input.Preflight.Blockers...)
	publicationCandidate := input.Preflight.Passed
	if input.DOCX == nil {
		publicationBlockers = append(publicationBlockers, "缺少整书 DOCX 投影。")
		publicationCandidate = false
	}
	if input.PDF == nil {
		publicationBlockers = append(publicationBlockers, "缺少整书 PDF 投影。")
		publicationCandidate = false
	}
	var identity struct {
		ProjectID string `json:"project_id"`
	}
	_ = json.Unmarshal(input.BuildManifest, &identity)
	fontReady := false
	if input.PDF != nil && input.PDF.FontEvidence != nil {
		fontReady = input.PDF.FontEvidence.PublicationReadyForBook(input.Book, identity.ProjectID, input.BuildID, input.RightsItems)
		if !fontReady && input.RightsLedger != nil {
			fontReady = input.PDF.FontEvidence.PublicationReadyWithAssetFonts(input.Book, identity.ProjectID, input.BuildID, input.ManifestHash, input.RightsLedger, input.RightsItems, bookPackageImageSource(input.Assets))
		}
	}
	if input.PDF != nil && !fontReady {
		publicationBlockers = append(publicationBlockers, "PDF 字体来源与许可证据尚未完整覆盖正文及页眉页脚。")
		publicationCandidate = false
	}
	if len(input.BuildManifest) == 0 {
		publicationBlockers = append(publicationBlockers, "缺少冻结 BookBuild manifest。")
		publicationCandidate = false
	}
	sort.Strings(publicationBlockers)
	// Rendering can change independently of the frozen manuscript. Keep the
	// actual projection tools here without mutating the BookBuild manifest.
	rendererVersions := make(map[string]map[string]string)
	if input.DOCX != nil && len(input.DOCX.ToolVersions) > 0 {
		rendererVersions["docx"] = input.DOCX.ToolVersions
	}
	if input.PDF != nil && len(input.PDF.ToolVersions) > 0 {
		rendererVersions["pdf"] = input.PDF.ToolVersions
	}
	manifest := struct {
		SchemaVersion        string                       `json:"schema_version"`
		BookFormat           string                       `json:"book_format"`
		PublicationCandidate bool                         `json:"publication_candidate"`
		PublicationBlockers  []string                     `json:"publication_blockers"`
		Files                []string                     `json:"files"`
		ManifestHashes       map[string]string            `json:"manifest_hashes"`
		RendererToolVersions map[string]map[string]string `json:"renderer_tool_versions,omitempty"`
	}{SchemaVersion: bookPackageSchemaVersion, BookFormat: input.Book.Format, PublicationCandidate: publicationCandidate, PublicationBlockers: publicationBlockers, Files: paths, ManifestHashes: hashes, RendererToolVersions: rendererVersions}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := writeBytes("manifest.json", manifestJSON); err != nil {
		return err
	}
	manifestHash := sha256.Sum256(manifestJSON)
	if err := writeBytes("manifest.sha256", []byte("sha256:"+hex.EncodeToString(manifestHash[:])+"  manifest.json\n")); err != nil {
		return err
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close book package: %w", err)
	}
	closed = true
	return nil
}

func safeBookPackagePath(path string) error {
	cleaned := filepath.ToSlash(filepath.Clean(path))
	if strings.TrimSpace(path) == "" || filepath.IsAbs(path) || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("unsafe book package path")
	}
	return nil
}

func bytesHavePrefix(value []byte, prefix string) bool {
	return len(value) >= len(prefix) && string(value[:len(prefix)]) == prefix
}

package export

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
	"inkwords-backend/shared/platform/visualasset"
)

const textbookPackageSchemaVersion = "inkwords.textbook-package.v1"

// TextbookChapterPackageBuilder produces a portable personal-learning package
// from approved data. It deliberately does not claim a publication-ready book:
// publication requires the later whole-book build and recorded human reviews.
type TextbookChapterPackageBuilder struct {
	teachingArtifacts *teachingartifact.Store
	visualAssets      *visualasset.Store
}

func NewTextbookChapterPackageBuilder(teachingArtifacts *teachingartifact.Store, visualAssets *visualasset.Store) *TextbookChapterPackageBuilder {
	return &TextbookChapterPackageBuilder{teachingArtifacts: teachingArtifacts, visualAssets: visualAssets}
}

// FrozenBookPackageFiles resolves only the content-addressed artifact tokens
// pinned in a BookBuild manifest. It intentionally never looks up a current
// chapter or revision, so a later edit cannot change a review bundle.
func (builder *TextbookChapterPackageBuilder) FrozenBookPackageFiles(manifestJSON json.RawMessage) ([]BookPackageFile, []BookPackageFile, error) {
	if builder == nil || builder.teachingArtifacts == nil || builder.visualAssets == nil {
		return nil, nil, fmt.Errorf("frozen book package stores are not configured")
	}
	var manifest struct {
		Format    string `json:"format"`
		Artifacts []struct {
			ID           uuid.UUID `json:"id"`
			ArtifactHash string    `json:"artifact_hash"`
		} `json:"code_artifacts"`
		Assets []struct {
			ID          uuid.UUID `json:"id"`
			ContentHash string    `json:"content_hash"`
		} `json:"assets"`
	}
	if len(manifestJSON) == 0 || json.Unmarshal(manifestJSON, &manifest) != nil || manifest.Format != "inkwords.book-build.v1" {
		return nil, nil, fmt.Errorf("frozen book manifest is invalid")
	}
	codeFiles := make([]BookPackageFile, 0)
	dependencyPins, err := frozenDependencyPins(manifestJSON)
	if err != nil {
		return nil, nil, err
	}
	for _, artifact := range manifest.Artifacts {
		if artifact.ID == uuid.Nil || strings.TrimSpace(artifact.ArtifactHash) == "" {
			return nil, nil, fmt.Errorf("frozen book manifest contains an incomplete code artifact")
		}
		root, err := builder.teachingArtifacts.ResolveWithGoDependencies(artifact.ArtifactHash, dependencyPins[artifact.ID.String()])
		if err != nil {
			return nil, nil, fmt.Errorf("resolve frozen code artifact %s: %w", artifact.ID, err)
		}
		files, err := readFrozenArtifactTree(root, artifact.ID.String())
		if err != nil {
			return nil, nil, fmt.Errorf("read frozen code artifact %s: %w", artifact.ID, err)
		}
		codeFiles = append(codeFiles, files...)
	}
	assetFiles := make([]BookPackageFile, 0, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		if asset.ID == uuid.Nil || strings.TrimSpace(asset.ContentHash) == "" {
			return nil, nil, fmt.Errorf("frozen book manifest contains an incomplete visual asset")
		}
		path, mediaType, err := builder.visualAssets.Resolve(asset.ContentHash)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve frozen visual asset %s: %w", asset.ID, err)
		}
		extension := extensionForVisualMediaType(mediaType)
		if extension == "" {
			return nil, nil, fmt.Errorf("frozen visual asset %s has unsupported media type", asset.ID)
		}
		content, err := os.ReadFile(path) //nolint:gosec // visual store resolved and hash-verified this content-addressed path.
		if err != nil {
			return nil, nil, fmt.Errorf("read frozen visual asset %s: %w", asset.ID, err)
		}
		assetFiles = append(assetFiles, BookPackageFile{Path: strings.TrimPrefix(asset.ContentHash, "sha256:") + extension, Content: content})
	}
	sort.Slice(codeFiles, func(left, right int) bool { return codeFiles[left].Path < codeFiles[right].Path })
	sort.Slice(assetFiles, func(left, right int) bool { return assetFiles[left].Path < assetFiles[right].Path })
	return codeFiles, assetFiles, nil
}

func readFrozenArtifactTree(root, prefix string) ([]BookPackageFile, error) {
	files := make([]BookPackageFile, 0)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not exportable: %s", relative)
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular file is not exportable: %s", relative)
		}
		content, err := os.ReadFile(path) //nolint:gosec // filepath.Walk is bounded under the store-resolved root.
		if err != nil {
			return err
		}
		packagePath := filepath.ToSlash(filepath.Join(prefix, relative))
		if err := safeBookPackagePath(packagePath); err != nil {
			return err
		}
		files = append(files, BookPackageFile{Path: packagePath, Content: content})
		return nil
	})
	return files, err
}

func (builder *TextbookChapterPackageBuilder) Build(w io.Writer, chapter TextbookChapterExport) error {
	if builder == nil || builder.teachingArtifacts == nil || builder.visualAssets == nil {
		return fmt.Errorf("textbook package builder is not configured")
	}
	archive := zip.NewWriter(w)
	closed := false
	defer func() {
		if !closed {
			_ = archive.Close()
		}
	}()
	artifacts := chapter.CodeArtifacts
	if artifacts == nil {
		artifacts = []TextbookCodeArtifactExport{}
	}
	evidence := chapter.RuntimeEvidence
	if evidence == nil {
		evidence = []TextbookRuntimeEvidenceExport{}
	}
	assets := chapter.Assets
	if assets == nil {
		assets = []TextbookManuscriptAssetExport{}
	}
	builtAt := time.Now().UTC()
	bookAST, err := chapterBookAST(chapter, builtAt)
	if err != nil {
		return err
	}
	paths := []string{"manuscript/book.json", "chapter.md", "projections/code-artifacts.json", "verification-summary.json", "assets.json"}
	if err := writeZipJSON(archive, "manuscript/book.json", bookAST); err != nil {
		return err
	}
	if err := writeZipText(archive, "chapter.md", bookAST.Chapters[0].Markdown); err != nil {
		return err
	}
	if err := writeZipJSON(archive, "projections/code-artifacts.json", artifacts); err != nil {
		return err
	}
	if err := writeZipJSON(archive, "verification-summary.json", evidence); err != nil {
		return err
	}
	if err := writeZipJSON(archive, "assets.json", assets); err != nil {
		return err
	}
	for _, artifact := range artifacts {
		dependencyPin, err := artifactDependencyPin(json.RawMessage(artifact.ManifestJSON), artifact.ID.String(), artifact.ArtifactHash, artifact.ManifestHash)
		if err != nil {
			return err
		}
		root, err := builder.teachingArtifacts.ResolveWithGoDependencies(artifact.ArtifactHash, dependencyPin)
		if err != nil {
			return fmt.Errorf("resolve teaching artifact %s: %w", artifact.ID, err)
		}
		artifactPaths, err := archiveRegularTree(archive, root, "code/"+artifact.ID.String())
		if err != nil {
			return fmt.Errorf("archive teaching artifact %s: %w", artifact.ID, err)
		}
		paths = append(paths, artifactPaths...)
	}
	for _, asset := range assets {
		if asset.Kind != "screenshot" {
			return fmt.Errorf("asset %s has unsupported portable kind %q", asset.ID, asset.Kind)
		}
		path, mediaType, err := builder.visualAssets.Resolve(asset.ContentHash)
		if err != nil {
			return fmt.Errorf("resolve manuscript asset %s: %w", asset.ID, err)
		}
		extension := extensionForVisualMediaType(mediaType)
		archivePath := "assets/" + strings.TrimPrefix(asset.ContentHash, "sha256:") + extension
		if err := writeZipFile(archive, archivePath, path); err != nil {
			return fmt.Errorf("archive manuscript asset %s: %w", asset.ID, err)
		}
		paths = append(paths, archivePath)
	}
	sort.Strings(paths)
	manifest := textbookPackageManifest{SchemaVersion: textbookPackageSchemaVersion, ExportProfile: "personal_learning", PublicationCandidate: false, CreatedAt: builtAt, Files: paths}
	manifest.Chapter.ID = chapter.ChapterID.String()
	manifest.Chapter.Title = chapter.Title
	manifest.Chapter.SortOrder = chapter.SortOrder
	manifest.Revision.ID = chapter.RevisionID.String()
	manifest.Revision.Number = chapter.RevisionNumber
	manifest.Revision.ContentHash = chapter.ContentHash
	manifest.Revision.BookContractRevisionID = chapter.BookContractRevisionID
	manifest.Revision.StyleSheetRevisionID = chapter.StyleSheetRevisionID
	manifest.Revision.BlueprintRevisionID = chapter.BlueprintRevisionID
	manifest.Revision.EvidencePackHash = chapter.EvidencePackHash
	manifest.Revision.PromptHash = chapter.PromptHash
	manifest.Revision.ProviderName = chapter.ProviderName
	manifest.Revision.ModelName = chapter.ModelName
	manifest.PublicationBlockers = publicationBlockers(chapter)
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal textbook package manifest: %w", err)
	}
	if err := writeZipText(archive, "manifest.json", string(manifestJSON)); err != nil {
		return err
	}
	sum := sha256.Sum256(manifestJSON)
	if err := writeZipText(archive, "manifest.sha256", "sha256:"+hex.EncodeToString(sum[:])+"  manifest.json\n"); err != nil {
		return err
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close textbook package: %w", err)
	}
	closed = true
	return nil
}

func chapterBookAST(chapter TextbookChapterExport, builtAt time.Time) (sharedtextbook.CanonicalBookAST, error) {
	contentHash, err := canonicalSHA256(chapter.ContentHash)
	if err != nil {
		return sharedtextbook.CanonicalBookAST{}, fmt.Errorf("canonicalize chapter content hash: %w", err)
	}
	assets := make([]sharedtextbook.CanonicalBookAsset, 0, len(chapter.Assets))
	for _, asset := range chapter.Assets {
		assets = append(assets, sharedtextbook.CanonicalBookAsset{ID: asset.ID.String(), StableRef: asset.StableRef, ContentHash: asset.ContentHash, AltText: asset.AltText})
	}
	order := chapter.SortOrder
	if order < 1 {
		order = 1
	}
	return sharedtextbook.NewCanonicalBookAST(chapter.Title, builtAt, []sharedtextbook.CanonicalBookChapter{{ID: chapter.ChapterID.String(), Order: order, Title: chapter.Title, Markdown: chapter.Markdown, ContentHash: contentHash, Assets: assets}})
}

func canonicalSHA256(value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "sha256:") {
		value = strings.TrimPrefix(value, "sha256:")
	}
	if len(value) != sha256.Size*2 {
		return "", fmt.Errorf("expected a 64-character SHA-256 digest")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", fmt.Errorf("decode SHA-256 digest: %w", err)
	}
	return "sha256:" + strings.ToLower(value), nil
}

type textbookPackageManifest struct {
	SchemaVersion        string    `json:"schema_version"`
	ExportProfile        string    `json:"export_profile"`
	PublicationCandidate bool      `json:"publication_candidate"`
	CreatedAt            time.Time `json:"created_at"`
	Files                []string  `json:"files"`
	PublicationBlockers  []string  `json:"publication_blockers"`
	Chapter              struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		SortOrder int    `json:"sort_order"`
	} `json:"chapter"`
	Revision struct {
		ID                     string     `json:"id"`
		Number                 int        `json:"number"`
		ContentHash            string     `json:"content_hash"`
		BookContractRevisionID *uuid.UUID `json:"book_contract_revision_id,omitempty"`
		StyleSheetRevisionID   *uuid.UUID `json:"style_sheet_revision_id,omitempty"`
		BlueprintRevisionID    *uuid.UUID `json:"blueprint_revision_id,omitempty"`
		EvidencePackHash       string     `json:"evidence_pack_hash,omitempty"`
		PromptHash             string     `json:"prompt_hash,omitempty"`
		ProviderName           string     `json:"provider_name,omitempty"`
		ModelName              string     `json:"model_name,omitempty"`
	} `json:"revision"`
}

func publicationBlockers(chapter TextbookChapterExport) []string {
	blockers := []string{"当前导出是单章个人学习包；尚未创建冻结的整书 BookBuild，也未记录人工发展性、技术、自学性、文字、版式、权利与试学审校。"}
	now := time.Now().UTC()
	for _, artifact := range chapter.CodeArtifacts {
		if !hasCurrentEvidence(chapter.RuntimeEvidence, artifact, now) {
			blockers = append(blockers, "代码工件 "+artifact.ID.String()+" 缺少同哈希且仍在有效期内的已验证运行证据。")
		}
	}
	for _, asset := range chapter.Assets {
		if asset.RightsStatus != "ready" {
			blockers = append(blockers, "资产 "+asset.StableRef+" 的权利状态未确认。")
		}
		if !hasCurrentAssetEvidence(chapter.RuntimeEvidence, asset, now) {
			blockers = append(blockers, "资产 "+asset.StableRef+" 关联的运行证据缺失、未验证或已过期。")
		}
	}
	return blockers
}

func hasCurrentAssetEvidence(evidence []TextbookRuntimeEvidenceExport, asset TextbookManuscriptAssetExport, now time.Time) bool {
	for _, item := range evidence {
		if item.ID == asset.EvidenceID && item.StaleReason == "" && item.Status == "verified" && item.ExpiresAt != nil && now.Before(*item.ExpiresAt) {
			return true
		}
	}
	return false
}

func hasCurrentEvidence(evidence []TextbookRuntimeEvidenceExport, artifact TextbookCodeArtifactExport, now time.Time) bool {
	for _, item := range evidence {
		if item.CodeArtifactID == artifact.ID && item.CodeArtifactHash == artifact.ArtifactHash && item.StaleReason == "" && item.Status == "verified" && item.ExpiresAt != nil && now.Before(*item.ExpiresAt) {
			return true
		}
	}
	return false
}

func archiveRegularTree(archive *zip.Writer, root, prefix string) ([]string, error) {
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not exportable: %s", relative)
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular file is not exportable: %s", relative)
		}
		archivePath := filepath.ToSlash(filepath.Join(prefix, relative))
		if err := writeZipFile(archive, archivePath, path); err != nil {
			return err
		}
		paths = append(paths, archivePath)
		return nil
	})
	return paths, err
}

func writeZipText(archive *zip.Writer, name, contents string) error {
	file, err := archive.Create(name)
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	if _, err := io.WriteString(file, contents); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func writeZipJSON(archive *zip.Writer, name string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", name, err)
	}
	return writeZipText(archive, name, string(payload))
}

func writeZipFile(archive *zip.Writer, name, source string) error {
	input, err := os.Open(source) //nolint:gosec // source is resolved by a trusted content-addressed store.
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	output, err := archive.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(output, input)
	return err
}

func extensionForVisualMediaType(mediaType string) string {
	switch mediaType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}

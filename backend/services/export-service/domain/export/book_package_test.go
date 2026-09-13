package export

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
	"inkwords-backend/shared/platform/visualasset"
)

func TestBookPackageBuildsEveryProjectionFromOneAST(t *testing.T) {
	hash := "sha256:" + strings.Repeat("a", 64)
	book, err := sharedtextbook.NewCanonicalBookAST("整书", time.Unix(1, 0), []sharedtextbook.CanonicalBookChapter{{ID: "chapter-1", Order: 1, Title: "第一章", Markdown: "# 第一章\n\n正文", ContentHash: hash}})
	require.NoError(t, err)

	var archive bytes.Buffer
	err = NewBookPackageBuilder().Build(&archive, BookPackageInput{Book: book, DOCX: &BookDOCXProjection{Content: []byte("PK\x03\x04docx"), MediaType: bookDOCXMediaType}, PDF: &BookPDFProjection{Content: []byte("%PDF-1.7"), MediaType: bookPDFMediaType, RenderLog: "status=success"}, CodeArtifacts: []BookPackageFile{{Path: "example/main.go", Content: []byte("package main\n")}}, Assets: []BookPackageFile{{Path: "route.png", Content: []byte("png")}}, VerificationSummary: json.RawMessage(`[{"status":"verified"}]`), RightsItems: []sharedtextbook.RightsItem{readyRightsItem()}, HumanReviews: []sharedtextbook.HumanPublicationReview{{ID: "review-1", BuildID: "build-1", Stage: sharedtextbook.PublicationReviewTechnical, Reviewer: "编辑", Notes: "已完成技术审校并记录结论。", CompletedAt: time.Unix(1, 0)}}, QualityReport: json.RawMessage(`{"status":"pass"}`), Preflight: PublicationPreflightResult{Passed: false, Blockers: []string{"缺少人工审校"}}, BuildManifest: json.RawMessage(`{"format":"inkwords.book-build.v1"}`)})
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	require.NoError(t, err)
	files := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		files[file.Name] = file
	}
	for _, name := range []string{"manuscript/book.json", "projections/book.md", "projections/book.docx", "projections/book.pdf", "projections/book.pdf.render.log", "code/example/main.go", "assets/route.png", "rights.json", "human-reviews.json", "quality-report.json", "publication-preflight.json", "book-build-manifest.json", "manifest.json", "manifest.sha256"} {
		require.Contains(t, files, name)
	}
	require.Contains(t, readZipFile(t, files["projections/book.md"]), "第一章")
	require.Contains(t, readZipFile(t, files["manifest.json"]), "manifest_hashes")
	require.Contains(t, readZipFile(t, files["manifest.json"]), `"publication_candidate": false`)
	require.Contains(t, readZipFile(t, files["manifest.json"]), "缺少人工审校")
	require.Contains(t, readZipFile(t, files["human-reviews.json"]), "已完成技术审校")
}

func TestSafeBookPackagePathRejectsTraversal(t *testing.T) {
	for _, path := range []string{"", "../manifest.json", "/tmp/manifest.json"} {
		require.Error(t, safeBookPackagePath(path))
	}
	require.NoError(t, safeBookPackagePath("code/example/main.go"))
}

func TestBookPackagePreservesDelegatedFailuresSeparatelyFromHumanEvidence(t *testing.T) {
	hash := "sha256:" + strings.Repeat("a", 64)
	book, err := sharedtextbook.NewCanonicalBookAST("审阅归属", time.Unix(1, 0), []sharedtextbook.CanonicalBookChapter{{ID: "chapter", Order: 1, Title: "章节", Markdown: "# 章节", ContentHash: hash}})
	require.NoError(t, err)
	review := sharedtextbook.DelegatedPublicationReview{ContractVersion: sharedtextbook.DelegatedPublicationReviewFormat, ID: "ai-review", BuildID: "build", ManifestHash: hash, Stage: sharedtextbook.PublicationReviewLayout, Revision: 1, ReviewerKind: "delegated_ai", Reviewer: "Codex", DelegationNote: "用户明确委托 AI 审阅并决策。", Verdict: "needs_revision", Score: 2, Scope: "逐页查看固定 DOCX 校样。", Notes: "版面仍需页码与跨页连续性修正。", EvidenceRefs: []string{"proof:fixed-docx"}, HardFailures: []string{"缺少页码"}, CompletedAt: time.Unix(1, 0)}
	require.NoError(t, review.Validate())
	var archive bytes.Buffer
	err = NewBookPackageBuilder().Build(&archive, BookPackageInput{Book: book, DOCX: &BookDOCXProjection{Content: []byte("PK\x03\x04docx"), MediaType: bookDOCXMediaType}, PDF: &BookPDFProjection{Content: []byte("%PDF-1.7"), MediaType: bookPDFMediaType}, DelegatedReviews: []sharedtextbook.DelegatedPublicationReview{review}, Preflight: PublicationPreflightResult{Blockers: []string{"缺少页码"}}})
	require.NoError(t, err)
	reader, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	require.NoError(t, err)
	files := map[string]*zip.File{}
	for _, file := range reader.File {
		files[file.Name] = file
	}
	require.Contains(t, files, "delegated-ai-reviews.json")
	raw := readZipFile(t, files["delegated-ai-reviews.json"])
	require.Contains(t, raw, "delegated_ai")
	require.Contains(t, raw, "缺少页码")
	require.NotContains(t, readZipFile(t, files["human-reviews.json"]), "Codex")
}

func TestFrozenBookPackageFilesReadsOnlyContentAddressedStores(t *testing.T) {
	ctx := context.Background()
	codeStore := teachingartifact.NewStore(t.TempDir())
	code, err := codeStore.Stage(ctx, []teachingartifact.File{{Path: "main.go", Content: []byte("package main\n")}})
	require.NoError(t, err)
	visualStore := visualasset.NewStore(t.TempDir())
	visual, err := visualStore.Stage(ctx, "image/png", strings.NewReader("png-bytes"))
	require.NoError(t, err)
	artifactID, assetID := uuid.New(), uuid.New()
	manifest, err := json.Marshal(struct {
		Format    string `json:"format"`
		Artifacts []struct {
			ID           uuid.UUID `json:"id"`
			ArtifactHash string    `json:"artifact_hash"`
		} `json:"code_artifacts"`
		Assets []struct {
			ID          uuid.UUID `json:"id"`
			ContentHash string    `json:"content_hash"`
		} `json:"assets"`
	}{Format: "inkwords.book-build.v1", Artifacts: []struct {
		ID           uuid.UUID `json:"id"`
		ArtifactHash string    `json:"artifact_hash"`
	}{{ID: artifactID, ArtifactHash: code.ArtifactHash}}, Assets: []struct {
		ID          uuid.UUID `json:"id"`
		ContentHash string    `json:"content_hash"`
	}{{ID: assetID, ContentHash: visual.ContentHash}}})
	require.NoError(t, err)

	codeFiles, assetFiles, err := NewTextbookChapterPackageBuilder(codeStore, visualStore).FrozenBookPackageFiles(manifest)

	require.NoError(t, err)
	require.Equal(t, []BookPackageFile{{Path: artifactID.String() + "/main.go", Content: []byte("package main\n")}}, codeFiles)
	require.Equal(t, []BookPackageFile{{Path: strings.TrimPrefix(visual.ContentHash, "sha256:") + ".png", Content: []byte("png-bytes")}}, assetFiles)
}

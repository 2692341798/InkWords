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
	"gorm.io/gorm"
	shared "inkwords-backend/shared/kernel/textbook"
)

func testExportHumanReviewHistory(t *testing.T, db *gorm.DB, workspaceID, projectID, contractID, styleID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	buildID := uuid.New()
	hash := "sha256:" + strings.Repeat("c", 64)
	book, err := shared.NewCanonicalBookAST("隔离真人审校导出测试", time.Unix(1, 0), []shared.CanonicalBookChapter{{ID: "chapter", Order: 1, Title: "测试章", Markdown: "# 隔离测试", ContentHash: hash}})
	require.NoError(t, err)
	manifest, err := json.Marshal(map[string]any{"format": "inkwords.book-build.v1", "input_hash": hash, "book": book})
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO textbook_book_builds (id, project_id, book_contract_revision_id, style_sheet_revision_id, approved_revision_ids, manifest_json, manifest_hash, input_hash, status) VALUES (?, ?, ?, ?, '[]', ?::jsonb, ?, ?, 'ready_for_review')", buildID, projectID, contractID, styleID, string(manifest), hash, hash).Error)
	legacy := shared.HumanPublicationReview{ID: uuid.NewString(), BuildID: buildID.String(), Stage: shared.PublicationReviewTechnical, Reviewer: "隔离测试夹具", Notes: "历史说明没有正式结论和评分。", CompletedAt: time.Unix(10, 0).UTC()}
	require.NoError(t, db.Exec("INSERT INTO textbook_publication_reviews (id,build_id,stage,reviewer,notes,completed_at) VALUES (?,?,?,?,?,?)", legacy.ID, buildID, legacy.Stage, legacy.Reviewer, legacy.Notes, legacy.CompletedAt).Error)
	review := legacy
	review.ID, review.ContractVersion, review.ReviewerKind = uuid.NewString(), shared.HumanPublicationReviewFormat, "human"
	review.ManifestHash, review.Revision, review.Verdict, review.Score = hash, 2, "needs_revision", 1
	review.Scope, review.EvidenceRefs, review.HardFailures = "隔离数据库测试中检查导出复审字段。", []string{"test:frozen-evidence"}, []string{"隔离测试中的未解决代码错误"}
	review.CompletedAt = time.Unix(20, 0).UTC()
	refs, _ := json.Marshal(review.EvidenceRefs)
	failures, _ := json.Marshal(review.HardFailures)
	require.NoError(t, db.Exec(`INSERT INTO textbook_publication_reviews
		(id,build_id,stage,reviewer,notes,completed_at,contract_version,manifest_hash,revision,reviewer_kind,verdict,score,scope,evidence_refs,hard_failures,input_hash)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?::jsonb,?::jsonb,?)`, review.ID, buildID, review.Stage, review.Reviewer, review.Notes, review.CompletedAt,
		review.ContractVersion, hash, review.Revision, review.ReviewerKind, review.Verdict, review.Score, review.Scope, string(refs), string(failures), hash).Error)
	exported, err := NewGormRepository(db).GetTextbookBookBuild(ctx, workspaceID, buildID)
	require.NoError(t, err)
	require.Len(t, exported.HumanReviews, 2)
	expected, err := json.Marshal([]shared.HumanPublicationReview{legacy, review})
	require.NoError(t, err)
	for i := range exported.HumanReviews {
		exported.HumanReviews[i].CompletedAt = exported.HumanReviews[i].CompletedAt.UTC()
	}
	actual, err := json.Marshal(exported.HumanReviews)
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(actual), "legacy metadata and every explicit decision field must survive the SQL mapper")
	require.False(t, exported.Preflight.Passed)
	require.Contains(t, strings.Join(exported.Preflight.Blockers, " "), review.HardFailures[0])
	var buffer bytes.Buffer
	require.NoError(t, NewBookPackageBuilder().Build(&buffer, BookPackageInput{Book: book, HumanReviews: exported.HumanReviews, DOCX: &BookDOCXProjection{Content: []byte("PK\x03\x04fixture"), MediaType: bookDOCXMediaType}, PDF: &BookPDFProjection{Content: []byte("%PDF-1.7"), MediaType: bookPDFMediaType}}))
	archive, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	require.NoError(t, err)
	var humanFile *zip.File
	for _, file := range archive.File {
		if file.Name == "human-reviews.json" {
			humanFile = file
		}
	}
	require.NotNil(t, humanFile)
	require.JSONEq(t, string(expected), readZipFile(t, humanFile))
}

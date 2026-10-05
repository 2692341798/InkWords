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

func testExportRightsLedger(t *testing.T, db *gorm.DB, projectID, contractID, styleID uuid.UUID) {
	t.Helper()
	buildID, baseID, amendmentID := uuid.New(), uuid.New(), uuid.New()
	hash := "sha256:" + strings.Repeat("a", 64)
	require.NoError(t, db.Exec("INSERT INTO textbook_book_builds (id, project_id, book_contract_revision_id, style_sheet_revision_id, approved_revision_ids, manifest_json, manifest_hash, input_hash, status) VALUES (?, ?, ?, ?, '[]', '{}', ?, ?, 'ready_for_review')", buildID, projectID, contractID, styleID, hash, hash).Error)
	base := shared.RightsItem{ID: baseID.String(), ProjectID: projectID.String(), BuildID: buildID.String(), SubjectRef: "chapter:r1", WorkType: shared.RightsWorkTypeProse, RightsBasis: "待补齐许可", AllowedUse: "审校", Attribution: "待确认署名", PublicationStatus: shared.RightsStatusPending}
	require.NoError(t, db.Exec("INSERT INTO textbook_rights_items (id, build_id, project_id, subject_ref, work_type, rights_basis, allowed_use, attribution, publication_status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", base.ID, base.BuildID, base.ProjectID, base.SubjectRef, base.WorkType, base.RightsBasis, base.AllowedUse, base.Attribution, base.PublicationStatus).Error)
	current := base
	current.PublicationStatus = shared.RightsStatusBlocked
	amendment := shared.RightsAmendment{ContractVersion: shared.RightsAmendmentFormat, ID: amendmentID.String(), BuildID: buildID.String(), ManifestHash: hash, BaseItemID: base.ID, Revision: 1, ReviewerKind: "delegated_ai", Reviewer: "测试代理", DelegationNote: "用户明确委托核对权利证明", Reason: "核对后仍缺少必要授权证明", EvidenceRefs: []string{"fixture:license-unresolved"}, EffectiveItem: current, CompletedAt: time.Unix(10, 0).UTC()}
	raw, err := json.Marshal(amendment)
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO textbook_rights_amendments (id, build_id, base_item_id, revision, manifest_hash, input_hash, document_json, completed_at) VALUES (?, ?, ?, 1, ?, ?, ?::jsonb, ?)", amendment.ID, buildID, baseID, hash, hash, string(raw), amendment.CompletedAt).Error)
	ledger, err := NewGormRepository(db).loadRightsLedger(context.Background(), buildID, hash, []shared.RightsItem{base})
	require.NoError(t, err)
	canonical, err := shared.ResolveRightsLedger(buildID.String(), hash, []shared.RightsItem{base}, []shared.RightsAmendment{amendment})
	require.NoError(t, err)
	require.Equal(t, canonical, ledger)
	book, err := shared.NewCanonicalBookAST("补证审校包", time.Unix(1, 0), []shared.CanonicalBookChapter{{ID: "chapter", Order: 1, Title: "章", Markdown: "# 章", ContentHash: hash}})
	require.NoError(t, err)
	var archive bytes.Buffer
	require.NoError(t, NewBookPackageBuilder().Build(&archive, BookPackageInput{Book: book, DOCX: &BookDOCXProjection{Content: []byte("PK\x03\x04docx"), MediaType: bookDOCXMediaType}, PDF: &BookPDFProjection{Content: []byte("%PDF-1.7"), MediaType: bookPDFMediaType}, RightsItems: ledger.EffectiveItems, RightsLedger: &ledger}))
	reader, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	require.NoError(t, err)
	files := map[string]*zip.File{}
	for _, f := range reader.File {
		files[f.Name] = f
	}
	var exported shared.RightsLedger
	require.NoError(t, json.Unmarshal([]byte(readZipFile(t, files["rights-ledger.json"])), &exported))
	require.Equal(t, ledger, exported)
	require.Contains(t, readZipFile(t, files["rights.json"]), "blocked")
	require.Contains(t, readZipFile(t, files["manifest.json"]), "rights-ledger.json")
	_, err = NewGormRepository(db).loadRightsLedger(context.Background(), buildID, "sha256:"+strings.Repeat("b", 64), []shared.RightsItem{base})
	require.ErrorIs(t, err, ErrTextbookBookBuildInvalid)
}

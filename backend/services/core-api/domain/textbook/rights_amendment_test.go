package textbook

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func testRightsAmendments(t *testing.T, db *gorm.DB, workspaceID uuid.UUID, build *BookBuildRow) {
	t.Helper()
	rollback := errors.New("rollback isolated rights amendment fixture")
	err := db.Transaction(func(tx *gorm.DB) error {
		ctx := context.Background()
		s := NewService(NewGormRepository(tx))
		base, err := s.AddRightsItem(ctx, workspaceID, AddRightsItemInput{BuildID: build.ID, SubjectRef: "chapter:test-rights-amendment", WorkType: sharedtextbook.RightsWorkTypeProse, RightsBasis: "尚待补齐原始许可", AllowedUse: "本地审校", Attribution: "待核对署名", PublicationStatus: sharedtextbook.RightsStatusPending})
		require.NoError(t, err)
		input := AppendRightsAmendmentInput{ID: uuid.New(), BuildID: build.ID, BaseItemID: base.ID, ManifestHash: build.ManifestHash, ReviewerKind: "delegated_ai", Reviewer: "测试代理", DelegationNote: "测试用户明确委托核对许可", Reason: "补充固定来源的许可证据", EvidenceRefs: []string{"fixture:license-proof"}, RightsBasis: "测试许可已核对", AllowedUse: "本地审校", Attribution: "保留测试署名", PublicationStatus: sharedtextbook.RightsStatusReady}
		first, err := s.AppendRightsAmendment(ctx, workspaceID, input)
		require.NoError(t, err)
		retry, err := s.AppendRightsAmendment(ctx, workspaceID, input)
		require.NoError(t, err)
		require.Equal(t, first, retry)
		_, err = s.AppendRightsAmendment(ctx, uuid.New(), input)
		require.ErrorIs(t, err, ErrNotFound)
		changed := input
		changed.RightsBasis += "changed"
		_, err = s.AppendRightsAmendment(ctx, workspaceID, changed)
		require.ErrorIs(t, err, ErrVersionConflict)
		changed = input
		changed.ID = uuid.New()
		_, err = s.AppendRightsAmendment(ctx, workspaceID, changed)
		require.ErrorIs(t, err, ErrVersionConflict, "stale root cannot fork")
		changed.PreviousAmendmentID = &input.ID
		changed.PublicationStatus = sharedtextbook.RightsStatusBlocked
		second, err := s.AppendRightsAmendment(ctx, workspaceID, changed)
		require.NoError(t, err)
		require.Equal(t, 2, second.Revision)
		orphan := changed
		orphan.ID = uuid.New()
		orphan.BaseItemID = uuid.New()
		_, err = s.AppendRightsAmendment(ctx, workspaceID, orphan)
		require.ErrorIs(t, err, ErrNotFound)
		fresh := NewService(NewGormRepository(tx))
		workspace, err := fresh.GetEditorialWorkspace(ctx, workspaceID, build.ID)
		require.NoError(t, err)
		require.Equal(t, sharedtextbook.RightsStatusPending, workspace.RightsItems[0].PublicationStatus)
		require.Equal(t, sharedtextbook.RightsStatusBlocked, workspace.RightsLedger.EffectiveItems[0].PublicationStatus)
		require.Len(t, workspace.RightsLedger.Amendments, 2)
		var unchanged RightsItemRow
		require.NoError(t, tx.First(&unchanged, "id = ?", base.ID).Error)
		require.Equal(t, *base, unchanged)
		var unchangedBuild BookBuildRow
		require.NoError(t, tx.First(&unchangedBuild, "id = ?", build.ID).Error)
		require.JSONEq(t, string(build.ManifestJSON), string(unchangedBuild.ManifestJSON))
		require.Equal(t, build.ManifestHash, unchangedBuild.ManifestHash)
		require.NoError(t, tx.Model(&BookBuildRow{}).Where("id = ?", build.ID).Update("status", sharedtextbook.BookBuildPublicationCandidate).Error)
		_, err = fresh.AppendRightsAmendment(ctx, workspaceID, input)
		require.NoError(t, err, "exact retry does not mutate a promoted build")
		changed.ID = uuid.New()
		previous := uuid.MustParse(second.ID)
		changed.PreviousAmendmentID = &previous
		_, err = fresh.AppendRightsAmendment(ctx, workspaceID, changed)
		require.ErrorIs(t, err, ErrInvalidState)
		var plan []struct {
			QueryPlan string `gorm:"column:QUERY PLAN"`
		}
		require.NoError(t, tx.Exec("SET LOCAL enable_seqscan = off").Error)
		require.NoError(t, tx.Raw("EXPLAIN SELECT * FROM textbook_rights_amendments WHERE build_id = ? ORDER BY base_item_id, revision", build.ID).Scan(&plan).Error)
		raw, err := json.Marshal(plan)
		require.NoError(t, err)
		require.Contains(t, string(raw), "Index Scan")
		t.Logf("rights amendment history EXPLAIN: %s", raw)
		return rollback
	})
	require.ErrorIs(t, err, rollback)
}

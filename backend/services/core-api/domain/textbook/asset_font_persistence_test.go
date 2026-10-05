package textbook

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	shared "inkwords-backend/shared/kernel/textbook"
)

func testAssetFontAmendmentPersistence(t *testing.T, db *gorm.DB, workspaceID uuid.UUID, existing *BookBuildRow) {
	t.Helper()
	rollback := errors.New("rollback asset font fixture")
	err := db.Transaction(func(tx *gorm.DB) error {
		ctx := context.Background()
		assetID := uuid.New()
		hash := "sha256:" + strings.Repeat("b", 64)
		build := *existing
		build.ID, build.InputHash = uuid.New(), digestJSON(uuid.NewString())
		var manifest map[string]any
		require.NoError(t, json.Unmarshal(build.ManifestJSON, &manifest))
		manifest["assets"] = []bookBuildAssetSource{{ID: assetID, Kind: shared.AssetKindScreenshot, ContentHash: hash}}
		raw, err := json.Marshal(manifest)
		require.NoError(t, err)
		build.ManifestJSON = raw
		build.ManifestHash = digestJSON(manifest)
		require.NoError(t, tx.Create(&build).Error)
		s := NewService(NewGormRepository(tx))
		base, err := s.AddRightsItem(ctx, workspaceID, AddRightsItemInput{BuildID: build.ID, SubjectRef: "asset:" + assetID.String(), WorkType: shared.RightsWorkTypeScreenshot, RightsBasis: "测试素材待补证", AllowedUse: "测试本地用途", Attribution: "测试署名", PublicationStatus: shared.RightsStatusPending})
		require.NoError(t, err)
		input := AppendRightsAmendmentInput{ID: uuid.New(), BuildID: build.ID, BaseItemID: base.ID, ManifestHash: build.ManifestHash, ReviewerKind: "delegated_ai", Reviewer: "fixture", DelegationNote: "测试用户授权核对字体", Reason: "测试完整图片无文字证据", EvidenceRefs: []string{"fixture:inspected-pixels"}, RightsBasis: "测试图片来源已核对", AllowedUse: "测试本地用途", Attribution: "测试署名", PublicationStatus: shared.RightsStatusReady, AssetFontReview: &shared.AssetFontReview{ContractVersion: shared.AssetFontReviewFormat, AssetID: assetID.String(), ContentHash: hash, Surface: "raster", TextStatus: "no_text", Scope: "测试完整空白图片所有区域均无字形"}}
		wrong := *input.AssetFontReview
		wrong.ContentHash = "sha256:" + strings.Repeat("c", 64)
		bad := input
		bad.AssetFontReview = &wrong
		_, err = s.AppendRightsAmendment(ctx, workspaceID, bad)
		require.ErrorIs(t, err, ErrInvalidState)
		first, err := s.AppendRightsAmendment(ctx, workspaceID, input)
		require.NoError(t, err)
		retry, err := s.AppendRightsAmendment(ctx, workspaceID, input)
		require.NoError(t, err)
		require.Equal(t, first, retry)
		var row RightsAmendmentRow
		require.NoError(t, tx.First(&row, "id = ?", input.ID).Error)
		read, err := row.document()
		require.NoError(t, err)
		require.Equal(t, input.AssetFontReview, read.AssetFontReview)
		bad = input
		changed := *input.AssetFontReview
		changed.Scope += "变更"
		bad.AssetFontReview = &changed
		_, err = s.AppendRightsAmendment(ctx, workspaceID, bad)
		require.ErrorIs(t, err, ErrVersionConflict)
		var old RightsItemRow
		require.NoError(t, tx.First(&old, "id = ?", base.ID).Error)
		require.Equal(t, *base, old)
		var preserved BookBuildRow
		require.NoError(t, tx.First(&preserved, "id = ?", build.ID).Error)
		require.JSONEq(t, string(build.ManifestJSON), string(preserved.ManifestJSON))
		t.Logf("asset font assessment persisted and retried once; old rights and frozen manifest unchanged")
		return rollback
	})
	require.ErrorIs(t, err, rollback)
}

package textbook

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	shared "inkwords-backend/shared/kernel/textbook"
)

func TestAssetFontReviewBindsFrozenAssetNotCurrentRow(t *testing.T) {
	id := uuid.New()
	hash := "sha256:" + strings.Repeat("a", 64)
	review := shared.AssetFontReview{ContractVersion: shared.AssetFontReviewFormat, AssetID: id.String(), ContentHash: hash, Surface: "raster", TextStatus: "no_text", Scope: "已逐区域检查完整图片中没有文字或字形"}
	asset := bookBuildAssetSource{ID: id, ContentHash: hash, Kind: shared.AssetKindScreenshot}
	manifest := func(assets []bookBuildAssetSource) BookBuildRow {
		raw, err := json.Marshal(map[string]any{"assets": assets})
		require.NoError(t, err)
		return BookBuildRow{ManifestJSON: raw}
	}
	require.True(t, assetFontReviewMatchesBuild(review, manifest([]bookBuildAssetSource{asset})))
	require.False(t, assetFontReviewMatchesBuild(review, manifest(nil)))
	require.False(t, assetFontReviewMatchesBuild(review, manifest([]bookBuildAssetSource{asset, asset})))
	asset.ContentHash = "sha256:" + strings.Repeat("b", 64)
	require.False(t, assetFontReviewMatchesBuild(review, manifest([]bookBuildAssetSource{asset})))
	asset.ContentHash = hash
	asset.Kind = shared.AssetKindRecording
	require.False(t, assetFontReviewMatchesBuild(review, manifest([]bookBuildAssetSource{asset})))
}

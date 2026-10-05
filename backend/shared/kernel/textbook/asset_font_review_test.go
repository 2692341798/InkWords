package textbook

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func assetFontFixture(t *testing.T) (CanonicalBookAsset, RightsItem, RightsItem, RightsAmendment) {
	t.Helper()
	base, amendment := rightsAmendmentFixture()
	asset := CanonicalBookAsset{ID: "asset-1", ContentHash: "sha256:" + strings.Repeat("b", 64)}
	base.SubjectRef, base.WorkType = "asset:"+asset.ID, RightsWorkTypeScreenshot
	amendment.EffectiveItem = base
	amendment.EffectiveItem.PublicationStatus = RightsStatusReady
	font := base
	font.ID, font.SubjectRef, font.WorkType, font.PublicationStatus = "font-1", "font-file:sha256:"+strings.Repeat("c", 64), RightsWorkTypeFont, RightsStatusReady
	amendment.AssetFontReview = &AssetFontReview{ContractVersion: AssetFontReviewFormat, AssetID: asset.ID, ContentHash: asset.ContentHash, Surface: "raster", TextStatus: "identified_fonts", Scope: "核对截图全部界面和代码文字的字体来源", Fonts: []AssetFontRightsRef{{FileHash: strings.TrimPrefix(font.SubjectRef, "font-file:"), RightsItemID: font.ID}}}
	return asset, base, font, amendment
}

func TestAssetFontReviewRequiresExactCurrentRightsChain(t *testing.T) {
	asset, base, font, first := assetFontFixture(t)
	ledger, err := ResolveRightsLedger(base.BuildID, first.ManifestHash, []RightsItem{base, font}, []RightsAmendment{first})
	require.NoError(t, err)
	require.True(t, AssetFontsPublicationReady(asset, base.ProjectID, base.BuildID, first.ManifestHash, ledger))
	for name, change := range map[string]func(*CanonicalBookAsset, *RightsLedger){
		"asset bytes changed": func(a *CanonicalBookAsset, _ *RightsLedger) { a.ContentHash = "sha256:" + strings.Repeat("d", 64) },
		"wrong asset":         func(a *CanonicalBookAsset, _ *RightsLedger) { a.ID = "other" },
		"font pending": func(_ *CanonicalBookAsset, l *RightsLedger) {
			l.OriginalItems[1].PublicationStatus = RightsStatusPending
		},
		"missing font":  func(_ *CanonicalBookAsset, l *RightsLedger) { l.OriginalItems = l.OriginalItems[:1] },
		"wrong project": func(_ *CanonicalBookAsset, l *RightsLedger) { l.OriginalItems[1].ProjectID = "other" },
		"asset pending": func(_ *CanonicalBookAsset, l *RightsLedger) {
			l.Amendments[0].EffectiveItem.PublicationStatus = RightsStatusPending
		},
		"new version omits review": func(_ *CanonicalBookAsset, l *RightsLedger) {
			next := first
			next.ID, next.PreviousAmendmentID, next.Revision, next.AssetFontReview = "asset-next", first.ID, 2, nil
			l.Amendments = append(l.Amendments, next)
		},
		"font version changed": func(_ *CanonicalBookAsset, l *RightsLedger) {
			next := first
			next.ID, next.BaseItemID, next.EffectiveItem, next.AssetFontReview = "font-next", font.ID, font, nil
			l.Amendments = append(l.Amendments, next)
		},
	} {
		t.Run(name, func(t *testing.T) {
			copy := ledger
			copy.OriginalItems = append([]RightsItem{}, ledger.OriginalItems...)
			copy.Amendments = append([]RightsAmendment{}, ledger.Amendments...)
			a := asset
			change(&a, &copy)
			require.False(t, AssetFontsPublicationReady(a, base.ProjectID, base.BuildID, first.ManifestHash, copy))
		})
	}
}

func TestAssetFontReviewRejectsUnsupportedOrIncompleteDeclarations(t *testing.T) {
	_, _, _, first := assetFontFixture(t)
	for name, change := range map[string]func(*AssetFontReview){
		"unknown version":    func(r *AssetFontReview) { r.ContractVersion = "future" },
		"unsupported vector": func(r *AssetFontReview) { r.Surface = "svg" },
		"no source":          func(r *AssetFontReview) { r.Fonts = nil },
		"missing scope":      func(r *AssetFontReview) { r.Scope = "" },
		"ambiguous no text":  func(r *AssetFontReview) { r.TextStatus = "no_text" },
		"duplicate font":     func(r *AssetFontReview) { r.Fonts = append(r.Fonts, r.Fonts[0]) },
	} {
		t.Run(name, func(t *testing.T) { r := *first.AssetFontReview; change(&r); require.Error(t, r.Validate()) })
	}
	r := *first.AssetFontReview
	r.TextStatus, r.Fonts = "no_text", nil
	require.NoError(t, r.Validate())
	first.AssetFontReview = &r
	first.EffectiveItem.WorkType = RightsWorkTypeProse
	require.Error(t, first.Validate(), "font declaration cannot attach to prose")
}

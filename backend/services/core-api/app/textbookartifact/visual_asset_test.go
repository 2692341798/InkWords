package textbookartifact

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/visualasset"
)

type recordingAssetRegistrar struct {
	input textbookdomain.RegisterManuscriptAssetInput
}

func (r *recordingAssetRegistrar) RegisterManuscriptAsset(_ context.Context, _ uuid.UUID, input textbookdomain.RegisterManuscriptAssetInput) (*textbookdomain.ManuscriptAssetRow, error) {
	r.input = input
	return &textbookdomain.ManuscriptAssetRow{ID: uuid.MustParse(input.Asset.ID), StableRef: input.Asset.StableRef}, nil
}

func TestStageAndRegisterVisualAssetFreezesProvenance(t *testing.T) {
	registrar := &recordingAssetRegistrar{}
	service := NewVisualAssetService(visualasset.NewStore(t.TempDir()), registrar)
	row, err := service.StageAndRegister(context.Background(), uuid.New(), VisualAssetInput{ChapterID: uuid.New(), RevisionID: uuid.New(), EvidenceID: uuid.New(), MediaType: "image/png", AltText: "路由登记调用栈截图", Source: "本地 GoLand 手工截图", GenerationMethod: "manual_capture", VisualPurpose: sharedtextbook.VisualEvidencePurposeCallStack, RightsStatus: sharedtextbook.RightsStatusPending, Content: bytes.NewBufferString("png")})
	require.NoError(t, err)
	require.NotEmpty(t, row.StableRef)
	require.Contains(t, registrar.input.Asset.ContentHash, "sha256:")
	require.Equal(t, "manual_capture", registrar.input.Asset.GenerationMethod)
	require.Equal(t, sharedtextbook.VisualEvidencePurposeCallStack, registrar.input.Asset.VisualPurpose)
	_, err = service.StageAndRegister(context.Background(), uuid.New(), VisualAssetInput{ChapterID: uuid.New(), RevisionID: uuid.New(), EvidenceID: uuid.New(), MediaType: "image/png", AltText: "截图", Source: "本地", GenerationMethod: "manual_capture", VisualPurpose: sharedtextbook.VisualEvidencePurposeLegacy, RightsStatus: sharedtextbook.RightsStatusPending, Content: bytes.NewBufferString("png")})
	require.ErrorContains(t, err, "incomplete")
}

package textbookartifact

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/visualasset"
)

type visualStager interface {
	Stage(context.Context, string, io.Reader) (visualasset.Artifact, error)
}
type manuscriptAssetRegistrar interface {
	RegisterManuscriptAsset(context.Context, uuid.UUID, textbookdomain.RegisterManuscriptAssetInput) (*textbookdomain.ManuscriptAssetRow, error)
}

type VisualAssetInput struct {
	ChapterID, RevisionID, EvidenceID            uuid.UUID
	MediaType, AltText, Source, GenerationMethod string
	VisualPurpose                                sharedtextbook.VisualEvidencePurpose
	RightsStatus                                 sharedtextbook.RightsStatus
	Content                                      io.Reader
}

type VisualAssetService struct {
	stager    visualStager
	registrar manuscriptAssetRegistrar
}

func NewVisualAssetService(stager visualStager, registrar manuscriptAssetRegistrar) *VisualAssetService {
	return &VisualAssetService{stager: stager, registrar: registrar}
}

func (service *VisualAssetService) StageAndRegister(ctx context.Context, workspaceID uuid.UUID, input VisualAssetInput) (*textbookdomain.ManuscriptAssetRow, error) {
	if service == nil || service.stager == nil || service.registrar == nil || workspaceID == uuid.Nil || input.ChapterID == uuid.Nil || input.RevisionID == uuid.Nil || input.EvidenceID == uuid.Nil || input.Content == nil || strings.TrimSpace(input.AltText) == "" || strings.TrimSpace(input.Source) == "" || strings.TrimSpace(input.GenerationMethod) == "" || input.VisualPurpose.ValidateForNewCapture() != nil || input.RightsStatus.Validate() != nil {
		return nil, fmt.Errorf("visual manuscript asset is incomplete")
	}
	staged, err := service.stager.Stage(ctx, input.MediaType, input.Content)
	if err != nil {
		return nil, fmt.Errorf("stage visual manuscript asset: %w", err)
	}
	assetID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("inkwords:visual:"+input.RevisionID.String()+":"+staged.ContentHash))
	stableRef := "inkwords-asset:" + input.RevisionID.String() + ":" + strings.TrimPrefix(staged.ContentHash, "sha256:")
	asset := sharedtextbook.ManuscriptAsset{ID: assetID.String(), RevisionID: input.RevisionID.String(), EvidenceID: input.EvidenceID.String(), StableRef: stableRef, Kind: sharedtextbook.AssetKindScreenshot, ContentHash: staged.ContentHash, AltText: strings.TrimSpace(input.AltText), Source: strings.TrimSpace(input.Source), GenerationMethod: strings.TrimSpace(input.GenerationMethod), VisualPurpose: input.VisualPurpose, RightsStatus: input.RightsStatus, Status: sharedtextbook.ArtifactStatusUnverified}
	return service.registrar.RegisterManuscriptAsset(ctx, workspaceID, textbookdomain.RegisterManuscriptAssetInput{ChapterID: input.ChapterID, Asset: asset})
}

func (service *VisualAssetService) UploadManualVisualAsset(ctx context.Context, workspaceID uuid.UUID, input textbookdomain.VisualAssetUploadInput) (*textbookdomain.ManuscriptAssetRow, error) {
	return service.StageAndRegister(ctx, workspaceID, VisualAssetInput{ChapterID: input.ChapterID, RevisionID: input.RevisionID, EvidenceID: input.EvidenceID, MediaType: input.MediaType, AltText: input.AltText, Source: input.Source, GenerationMethod: input.GenerationMethod, VisualPurpose: input.VisualPurpose, RightsStatus: input.RightsStatus, Content: input.Content})
}

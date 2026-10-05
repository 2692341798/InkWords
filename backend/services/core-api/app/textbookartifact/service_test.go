package textbookartifact

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
)

type recordingRegistrar struct {
	input textbookdomain.RegisterGeneratedCodeArtifactInput
}

func (registrar *recordingRegistrar) RegisterGeneratedCodeArtifact(_ context.Context, _ uuid.UUID, input textbookdomain.RegisterGeneratedCodeArtifactInput) (*textbookdomain.CodeArtifactRow, error) {
	registrar.input = input
	return &textbookdomain.CodeArtifactRow{ID: uuid.MustParse(input.Artifact.ID), RevisionID: input.RevisionID, ArtifactHash: input.Artifact.ArtifactHash, ManifestHash: input.Artifact.ManifestHash, Status: sharedtextbook.ArtifactStatusUnverified}, nil
}

func TestStageAndRegisterUsesContentAddressedGeneratedFiles(t *testing.T) {
	registrar := &recordingRegistrar{}
	service := NewService(teachingartifact.NewStore(t.TempDir()), registrar)
	revisionID := uuid.New()
	bookHash, styleHash := "sha256:"+strings.Repeat("a", 64), "sha256:"+strings.Repeat("b", 64)
	row, err := service.StageAndRegister(context.Background(), uuid.New(), Input{ArtifactID: uuid.New(), RevisionID: revisionID, BookContractHash: bookHash, StyleSheetHash: styleHash, Language: "go", ToolchainVersion: "go1.25.4", Entrypoint: "main.go", Files: []teachingartifact.File{{Path: "go.mod", Content: []byte("module example.com/demo\n\ngo 1.25\n")}, {Path: "main.go", Content: []byte("package demo\n")}}, Commands: []sharedtextbook.VerificationCommand{{Kind: "go_test"}}, Limitations: []string{"只演示最小路由登记。"}})
	require.NoError(t, err)
	require.Equal(t, sharedtextbook.ArtifactStatusUnverified, row.Status)
	require.Equal(t, revisionID.String(), registrar.input.Artifact.RevisionID)
	require.Contains(t, registrar.input.Artifact.ArtifactHash, "sha256:")
	var manifest sharedtextbook.TeachingArtifactManifest
	require.NoError(t, json.Unmarshal(registrar.input.ManifestJSON, &manifest))
	require.NoError(t, manifest.Validate())
	require.Equal(t, registrar.input.Artifact.ArtifactHash, manifest.ArtifactHash)
	require.Equal(t, bookHash, manifest.BookContractHash)
	require.Equal(t, styleHash, manifest.StyleSheetHash)

	_, err = service.StageAndRegister(context.Background(), uuid.New(), Input{ArtifactID: uuid.New(), RevisionID: revisionID, BookContractHash: bookHash, StyleSheetHash: styleHash, Language: "go", ToolchainVersion: "go1.25.4", Entrypoint: "main.go", Files: []teachingartifact.File{{Path: "main.go", Content: []byte("package demo\n")}}, Commands: []sharedtextbook.VerificationCommand{{Kind: "go_test", TestPattern: "TestA;curl"}}, Limitations: []string{"最小示例。"}})
	require.ErrorContains(t, err, "unsafe")
}

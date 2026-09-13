// Package textbookartifact stages generated teaching code before core-api
// persists its immutable metadata. It is not an HTTP upload API.
package textbookartifact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
)

type artifactStager interface {
	Stage(context.Context, []teachingartifact.File) (teachingartifact.Artifact, error)
}

type artifactRegistrar interface {
	RegisterGeneratedCodeArtifact(context.Context, uuid.UUID, textbookdomain.RegisterGeneratedCodeArtifactInput) (*textbookdomain.CodeArtifactRow, error)
}

type Input struct {
	ArtifactID             uuid.UUID
	RevisionID             uuid.UUID
	BookContractHash       string
	StyleSheetHash         string
	Language               string
	ToolchainVersion       string
	Entrypoint             string
	Files                  []teachingartifact.File
	Commands               []sharedtextbook.VerificationCommand
	Limitations            []string
	dependencyManifestHash string
	dependencySelection    *sharedtextbook.TeachingDependencySelection
}

type Service struct {
	stager    artifactStager
	registrar artifactRegistrar
}

func NewService(stager artifactStager, registrar artifactRegistrar) *Service {
	return &Service{stager: stager, registrar: registrar}
}

// StageAndRegister is intentionally usable only by a trusted generation
// adapter. The generated file bytes are staged before metadata is persisted;
// a failed DB write leaves at most harmless deduplicated content-addressed data.
func (service *Service) StageAndRegister(ctx context.Context, workspaceID uuid.UUID, input Input) (*textbookdomain.CodeArtifactRow, error) {
	if service == nil || service.stager == nil || service.registrar == nil || workspaceID == uuid.Nil || input.ArtifactID == uuid.Nil || input.RevisionID == uuid.Nil || !strings.HasPrefix(input.BookContractHash, "sha256:") || !strings.HasPrefix(input.StyleSheetHash, "sha256:") || input.Language != "go" || strings.TrimSpace(input.ToolchainVersion) == "" || strings.TrimSpace(input.Entrypoint) == "" || len(input.Files) == 0 || len(input.Commands) == 0 || len(input.Limitations) == 0 {
		return nil, errors.New("generated textbook artifact is not configured")
	}
	for _, command := range input.Commands {
		if err := command.Validate(); err != nil {
			return nil, fmt.Errorf("validate generated teaching command: %w", err)
		}
	}
	staged, err := service.stageInput(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("stage generated teaching artifact: %w", err)
	}
	manifest := sharedtextbook.TeachingArtifactManifest{Format: sharedtextbook.TeachingArtifactManifestFormat, ArtifactID: input.ArtifactID.String(), RevisionID: input.RevisionID.String(), ArtifactHash: staged.ArtifactHash, BookContractHash: input.BookContractHash, StyleSheetHash: input.StyleSheetHash, Language: input.Language, ToolchainVersion: input.ToolchainVersion, DependencyManifestHash: input.dependencyManifestHash, Commands: append([]sharedtextbook.VerificationCommand(nil), input.Commands...)}
	manifest.DependencySelection = input.dependencySelection
	manifestHash, err := sharedtextbook.TeachingArtifactManifestHash(manifest)
	if err != nil {
		return nil, err
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal generated teaching manifest: %w", err)
	}
	artifact := sharedtextbook.CodeArtifact{ID: input.ArtifactID.String(), RevisionID: input.RevisionID.String(), Kind: sharedtextbook.CodeArtifactTeachingImplementation, Language: input.Language, Entrypoint: input.Entrypoint, ManifestHash: manifestHash, ArtifactHash: staged.ArtifactHash, Limitations: append([]string(nil), input.Limitations...), Status: sharedtextbook.ArtifactStatusUnverified}
	return service.registrar.RegisterGeneratedCodeArtifact(ctx, workspaceID, textbookdomain.RegisterGeneratedCodeArtifactInput{RevisionID: input.RevisionID, Artifact: artifact, ManifestJSON: manifestJSON})
}

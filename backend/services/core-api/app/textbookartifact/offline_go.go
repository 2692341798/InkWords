package textbookartifact

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
)

// ManuscriptGoInputForOfflineBundle combines the saved teaching bytes with an
// operator-selected verified bundle. Selection must be bound to source evidence
// by the caller, never inferred from an import or taken from manuscript prose.
// This constructor does not stage, approve, execute, or raise storage limits.
func ManuscriptGoInputForOfflineBundle(revision textbookdomain.ChapterRevision, toolchain string, origin teachingartifact.GoDependencyOrigin, bundle *teachingartifact.OfflineGoBundle) (Input, error) {
	if bundle == nil || bundle.Hash() == "" || bundle.Toolchain() != toolchain || bundle.Origin() != origin {
		return Input{}, fmt.Errorf("offline teaching dependencies do not match the selected source and toolchain")
	}
	input, err := ManuscriptGoInputForToolchain(revision, toolchain)
	if err != nil {
		return Input{}, err
	}
	// The vetted go.mod replaces only operator build metadata. The two saved
	// teaching files retain their exact bytes; no passing example is substituted.
	input.Files = append(input.Files[1:], bundle.Files()...)
	input.dependencyManifestHash = bundle.Hash()
	input.ArtifactID = uuid.NewSHA1(uuid.NameSpaceURL, []byte("inkwords:manuscript-go:v3:"+revision.ID.String()+":"+revision.ContentHash+":"+toolchain+":"+bundle.Hash()))
	return input, nil
}

type goDependencyStager interface {
	StageWithGoDependencies(context.Context, []teachingartifact.File, string) (teachingartifact.Artifact, error)
}

func (service *Service) stageInput(ctx context.Context, input Input) (teachingartifact.Artifact, error) {
	if input.dependencyManifestHash == "" {
		return service.stager.Stage(ctx, input.Files)
	}
	stager, ok := service.stager.(goDependencyStager)
	if !ok {
		return teachingartifact.Artifact{}, fmt.Errorf("offline dependency staging is unavailable")
	}
	for _, file := range input.Files {
		if file.Path == "inkwords-dependencies.json" {
			var manifest teachingartifact.GoDependencyManifest
			if json.Unmarshal(file.Content, &manifest) != nil || manifest.Toolchain != input.ToolchainVersion {
				return teachingartifact.Artifact{}, fmt.Errorf("offline dependency toolchain does not match artifact input")
			}
		}
	}
	return stager.StageWithGoDependencies(ctx, input.Files, input.dependencyManifestHash)
}

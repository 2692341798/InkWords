package textbook

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// RegisterGeneratedCodeArtifact persists an immutable generated code tree's
// metadata only after proving the target revision belongs to the local
// workspace. It does not accept a filesystem path or mark the code verified.
func (r *GormRepository) RegisterGeneratedCodeArtifact(ctx context.Context, workspaceID uuid.UUID, input RegisterGeneratedCodeArtifactInput) (*CodeArtifactRow, error) {
	var artifact *CodeArtifactRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var revision ChapterRevision
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", input.RevisionID).First(&revision).Error; err != nil {
			return ErrNotFound
		}
		var chapter Chapter
		if err := tx.Where("id = ?", revision.ChapterID).First(&chapter).Error; err != nil {
			return ErrNotFound
		}
		var project Project
		if err := tx.Where("id = ? AND workspace_id = ?", chapter.ProjectID, workspaceID).First(&project).Error; err != nil {
			return ErrNotFound
		}
		if revision.CreatedBy != RevisionCreatorGeneration || revision.Kind != sharedtextbook.RevisionKindCandidate {
			return ErrInvalidState
		}
		var manifest sharedtextbook.TeachingArtifactManifest
		if json.Unmarshal(input.ManifestJSON, &manifest) != nil {
			return ErrInvalidState
		}
		if selection := manifest.DependencySelection; selection != nil {
			if manifest.Validate() != nil || selection.WorkspaceID != workspaceID.String() || selection.ProjectID != project.ID.String() || selection.ChapterID != chapter.ID.String() {
				return ErrInvalidState
			}
			bound := NewGormRepository(tx)
			snapshot, err := bound.GetDependencySourceSnapshot(ctx, workspaceID, project.ID, chapter.ID, uuid.MustParse(selection.Snapshot.ID))
			if err != nil || !selection.MatchesSnapshot(snapshot) {
				return ErrInvalidState
			}
			if err := bound.ValidateDependencyCandidateSource(ctx, revision, *selection); err != nil {
				return err
			}
		}

		var existing CodeArtifactRow
		err := tx.Where("revision_id = ? AND artifact_hash = ?", revision.ID, input.Artifact.ArtifactHash).First(&existing).Error
		if err == nil {
			if existing.ManifestHash != input.Artifact.ManifestHash || existing.ID.String() != input.Artifact.ID {
				return ErrInvalidState
			}
			artifact = &existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		artifactID, err := uuid.Parse(input.Artifact.ID)
		if err != nil {
			return ErrInvalidState
		}
		limitations, err := json.Marshal(input.Artifact.Limitations)
		if err != nil {
			return err
		}
		artifact = &CodeArtifactRow{ID: artifactID, RevisionID: revision.ID, Kind: input.Artifact.Kind, Language: input.Artifact.Language, Entrypoint: input.Artifact.Entrypoint, SourceTreePath: input.Artifact.SourceTreePath, SourceRef: input.Artifact.SourceRef, ManifestJSON: datatypes.JSON(input.ManifestJSON), ManifestHash: input.Artifact.ManifestHash, ArtifactHash: input.Artifact.ArtifactHash, LimitationsJSON: datatypes.JSON(limitations), Status: sharedtextbook.ArtifactStatusUnverified}
		return tx.Create(artifact).Error
	})
	return artifact, err
}

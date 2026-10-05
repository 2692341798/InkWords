package textbook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

const editorialDetector = "core-api-book-build-v1"

func loadWorkspaceBookBuild(db *gorm.DB, workspaceID, buildID uuid.UUID, lock bool) (*BookBuildRow, error) {
	query := db.Table("textbook_book_builds AS build").
		Select("build.*").
		Joins("JOIN textbook_projects AS project ON project.id = build.project_id AND project.deleted_at IS NULL").
		Where("build.id = ? AND project.workspace_id = ?", buildID, workspaceID)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "build"}})
	}
	var build BookBuildRow
	if err := query.First(&build).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &build, nil
}

func publicationAutomatedChecks(db *gorm.DB, build BookBuildRow) ([]sharedtextbook.AutomatedPublicationCheck, error) {
	manifestStatus := sharedtextbook.QualityStatusPass
	var manifest struct {
		Format                 string `json:"format"`
		InputHash              string `json:"input_hash"`
		ProjectID              string `json:"project_id"`
		BookContractRevisionID string `json:"book_contract_revision_id"`
		StyleSheetRevisionID   string `json:"style_sheet_revision_id"`
		Chapters               []struct {
			RevisionID string `json:"revision_id"`
		} `json:"chapters"`
		Artifacts []struct {
			ID uuid.UUID `json:"id"`
		} `json:"code_artifacts"`
		Assets []struct {
			ID uuid.UUID `json:"id"`
		} `json:"assets"`
	}
	if json.Unmarshal(build.ManifestJSON, &manifest) != nil ||
		manifest.Format != "inkwords.book-build.v1" ||
		!isDigest(manifest.InputHash) ||
		manifest.ProjectID != build.ProjectID.String() ||
		manifest.BookContractRevisionID != build.BookContractRevisionID.String() ||
		manifest.StyleSheetRevisionID != build.StyleSheetRevisionID.String() ||
		!isDigest(build.ManifestHash) || len(manifest.Chapters) == 0 {
		manifestStatus = sharedtextbook.QualityStatusHardFail
	}

	revisionStatus := sharedtextbook.QualityStatusPass
	var approved []string
	if json.Unmarshal(build.ApprovedRevisionIDs, &approved) != nil || len(approved) == 0 || len(approved) != len(manifest.Chapters) {
		revisionStatus = sharedtextbook.QualityStatusHardFail
	} else {
		frozen := make(map[string]struct{}, len(approved))
		for _, revisionID := range approved {
			if _, err := uuid.Parse(revisionID); err != nil {
				revisionStatus = sharedtextbook.QualityStatusHardFail
				break
			}
			frozen[revisionID] = struct{}{}
		}
		for _, chapter := range manifest.Chapters {
			if _, ok := frozen[chapter.RevisionID]; !ok {
				revisionStatus = sharedtextbook.QualityStatusHardFail
				break
			}
		}
	}
	qualityStatus := sharedtextbook.QualityStatusPass
	qualitySnapshot, qualityErr := sharedtextbook.ReadBookQualitySnapshot(json.RawMessage(build.ManifestJSON))
	if qualityErr != nil || (qualitySnapshot != nil && !qualitySnapshot.Assess().Passed) {
		qualityStatus = sharedtextbook.QualityStatusHardFail
	}
	var approvedIDs []string
	if json.Unmarshal(build.ApprovedRevisionIDs, &approvedIDs) != nil || len(approvedIDs) == 0 {
		qualityStatus = sharedtextbook.QualityStatusHardFail
	}
	var approvedRows []struct {
		ID                uuid.UUID
		QualityReportJSON datatypes.JSON
	}
	if qualityStatus == sharedtextbook.QualityStatusPass && qualitySnapshot == nil && qualityErr == nil {
		if err := db.Table("chapter_revisions").Select("id, quality_report_json").Where("id IN ? AND kind = ?", approvedIDs, sharedtextbook.RevisionKindApproved).Find(&approvedRows).Error; err != nil {
			return nil, err
		}
		if len(approvedRows) != len(approvedIDs) {
			qualityStatus = sharedtextbook.QualityStatusHardFail
		}
		for _, revision := range approvedRows {
			if sharedtextbook.ValidateCurrentSampleQualityReport(json.RawMessage(revision.QualityReportJSON)) != nil {
				qualityStatus = sharedtextbook.QualityStatusHardFail
				break
			}
		}
	}
	artifactStatus := sharedtextbook.QualityStatusPass
	verificationSnapshot, snapshotErr := sharedtextbook.ReadBookVerificationSnapshot(json.RawMessage(build.ManifestJSON))
	if snapshotErr != nil || (verificationSnapshot != nil && !verificationSnapshot.Assess(time.Now().UTC()).Passed) {
		artifactStatus = sharedtextbook.QualityStatusHardFail
	}
	artifactIDs := make([]uuid.UUID, 0, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		artifactIDs = append(artifactIDs, artifact.ID)
	}
	if len(artifactIDs) > 0 && verificationSnapshot == nil && snapshotErr == nil {
		var artifactRows []CodeArtifactRow
		if err := db.Where("id IN ?", artifactIDs).Find(&artifactRows).Error; err != nil {
			return nil, err
		}
		var evidenceRows []RuntimeEvidenceRow
		if err := db.Where("code_artifact_id IN ?", artifactIDs).Find(&evidenceRows).Error; err != nil {
			return nil, err
		}
		bookContractHash, styleSheetHash, err := frozenBookBuildContractHashes(db, build)
		if err != nil {
			return nil, err
		}
		evidenceByArtifact := make(map[uuid.UUID][]sharedtextbook.RuntimeEvidence, len(artifactIDs))
		for _, evidence := range evidenceRows {
			evidenceByArtifact[evidence.CodeArtifactID] = append(evidenceByArtifact[evidence.CodeArtifactID], runtimeEvidenceContract(evidence))
		}
		if len(artifactRows) != len(artifactIDs) {
			artifactStatus = sharedtextbook.QualityStatusHardFail
		} else {
			now := time.Now().UTC()
			for _, artifact := range artifactRows {
				var artifactManifest sharedtextbook.TeachingArtifactManifest
				if artifact.Status != sharedtextbook.ArtifactStatusVerified || json.Unmarshal(artifact.ManifestJSON, &artifactManifest) != nil || artifactManifest.ArtifactID != artifact.ID.String() || !sharedtextbook.HasCurrentTeachingArtifactEvidence(artifactManifest, bookContractHash, styleSheetHash, evidenceByArtifact[artifact.ID], now) {
					artifactStatus = sharedtextbook.QualityStatusHardFail
					break
				}
			}
		}
	}
	assetStatus := sharedtextbook.QualityStatusPass
	var assetCount, readyAssetCount int64
	assetIDs := make([]uuid.UUID, 0, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		assetIDs = append(assetIDs, asset.ID)
	}
	if len(assetIDs) > 0 {
		if err := db.Table("textbook_manuscript_assets").Where("id IN ?", assetIDs).Count(&assetCount).Error; err != nil {
			return nil, err
		}
		if err := db.Table("textbook_manuscript_assets").Where("id IN ? AND rights_status = ?", assetIDs, sharedtextbook.RightsStatusReady).Count(&readyAssetCount).Error; err != nil {
			return nil, err
		}
		if assetCount != int64(len(assetIDs)) || readyAssetCount != assetCount {
			assetStatus = sharedtextbook.QualityStatusHardFail
		}
	}
	return []sharedtextbook.AutomatedPublicationCheck{
		{ID: "frozen-manifest-identity", Detector: editorialDetector, Status: manifestStatus},
		{ID: "approved-revisions-frozen", Detector: editorialDetector, Status: revisionStatus},
		{ID: "approved-revision-quality", Detector: editorialDetector, Status: qualityStatus},
		{ID: "generated-artifacts-verified", Detector: editorialDetector, Status: artifactStatus},
		{ID: "manuscript-assets-rights-ready", Detector: editorialDetector, Status: assetStatus},
	}, nil
}

func frozenBookBuildContractHashes(db *gorm.DB, build BookBuildRow) (string, string, error) {
	var row struct {
		BookContractHash string `gorm:"column:book_contract_hash"`
		StyleSheetHash   string `gorm:"column:style_sheet_hash"`
	}
	err := db.Table("book_contract_revisions AS book_contract").
		Select("book_contract.content_hash AS book_contract_hash, style_sheet.content_hash AS style_sheet_hash").
		Joins("JOIN style_sheet_revisions AS style_sheet ON style_sheet.id = ?", build.StyleSheetRevisionID).
		Where("book_contract.id = ?", build.BookContractRevisionID).
		Take(&row).Error
	if err != nil {
		return "", "", fmt.Errorf("get frozen book build contract hashes: %w", err)
	}
	return canonicalContractHash(row.BookContractHash), canonicalContractHash(row.StyleSheetHash), nil
}

func requiredRightsSubjects(build BookBuildRow) ([]sharedtextbook.RightsSubject, error) {
	var manifest struct {
		Chapters []struct {
			RevisionID string `json:"revision_id"`
		} `json:"chapters"`
		Artifacts []struct {
			ID string `json:"id"`
		} `json:"code_artifacts"`
		Assets []struct {
			ID   string                   `json:"id"`
			Kind sharedtextbook.AssetKind `json:"kind"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(build.ManifestJSON, &manifest); err != nil {
		return nil, ErrInvalidState
	}
	subjects := make([]sharedtextbook.RightsSubject, 0, len(manifest.Chapters)+len(manifest.Artifacts)+len(manifest.Assets))
	for _, chapter := range manifest.Chapters {
		subjects = append(subjects, sharedtextbook.RightsSubject{SubjectRef: "chapter-revision:" + chapter.RevisionID, WorkType: sharedtextbook.RightsWorkTypeProse})
	}
	for _, artifact := range manifest.Artifacts {
		subjects = append(subjects, sharedtextbook.RightsSubject{SubjectRef: "code-artifact:" + artifact.ID, WorkType: sharedtextbook.RightsWorkTypeCode})
	}
	for _, asset := range manifest.Assets {
		workType := sharedtextbook.RightsWorkTypeImage
		if asset.Kind == sharedtextbook.AssetKindScreenshot {
			workType = sharedtextbook.RightsWorkTypeScreenshot
		}
		subjects = append(subjects, sharedtextbook.RightsSubject{SubjectRef: "asset:" + asset.ID, WorkType: workType})
	}
	for _, subject := range subjects {
		if err := subject.Validate(); err != nil {
			return nil, ErrInvalidState
		}
	}
	return subjects, nil
}

func loadEditorialWorkspace(db *gorm.DB, build *BookBuildRow) (*EditorialWorkspace, error) {
	requiredSubjects, err := requiredRightsSubjects(*build)
	if err != nil {
		return nil, err
	}
	automatedChecks, err := publicationAutomatedChecks(db, *build)
	if err != nil {
		return nil, err
	}
	workspace := &EditorialWorkspace{
		Build: build, RequiredRightsSubjects: requiredSubjects, RightsItems: []RightsItemRow{}, HumanReviews: []PublicationReviewRow{},
		AutomatedChecks: automatedChecks,
	}
	if err := db.Where("build_id = ?", build.ID).Order("created_at ASC, id ASC").Find(&workspace.RightsItems).Error; err != nil {
		return nil, fmt.Errorf("list publication rights: %w", err)
	}
	if err := db.Where("build_id = ?", build.ID).Order("completed_at ASC, id ASC").Find(&workspace.HumanReviews).Error; err != nil {
		return nil, fmt.Errorf("list publication reviews: %w", err)
	}
	rights := make([]sharedtextbook.RightsItem, 0, len(workspace.RightsItems))
	for _, item := range workspace.RightsItems {
		rights = append(rights, item.ToContract())
	}
	reviews := make([]sharedtextbook.HumanPublicationReview, 0, len(workspace.HumanReviews))
	for _, review := range workspace.HumanReviews {
		reviews = append(reviews, review.ToContract())
	}
	workspace.DelegatedReviewContract = sharedtextbook.DelegatedPublicationReviewFormat
	workspace.DelegatedReviews, err = loadDelegatedPublicationReviews(db, build.ID)
	if err != nil {
		return nil, err
	}
	amendments, err := loadRightsAmendments(db, build.ID)
	if err != nil {
		return nil, err
	}
	workspace.RightsLedger, err = sharedtextbook.ResolveRightsLedger(build.ID.String(), build.ManifestHash, rights, amendments)
	if err != nil {
		return nil, ErrInvalidState
	}
	workspace.Preflight = sharedtextbook.EvaluatePublicationPreflight(sharedtextbook.PublicationPreflightInput{
		BuildID: build.ID.String(), ManifestHash: build.ManifestHash, DelegatedReviews: workspace.DelegatedReviews, RequiredRightsSubjects: requiredSubjects, RightsItems: rights, RightsAmendments: amendments, AutomatedChecks: workspace.AutomatedChecks, HumanReviews: reviews,
	})
	return workspace, nil
}

func (r *GormRepository) GetEditorialWorkspace(ctx context.Context, workspaceID, buildID uuid.UUID) (*EditorialWorkspace, error) {
	build, err := loadWorkspaceBookBuild(r.db.WithContext(ctx), workspaceID, buildID, false)
	if err != nil {
		return nil, err
	}
	return loadEditorialWorkspace(r.db.WithContext(ctx), build)
}

func (r *GormRepository) AddRightsItem(ctx context.Context, workspaceID uuid.UUID, input AddRightsItemInput) (*RightsItemRow, error) {
	var result *RightsItemRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		build, err := loadWorkspaceBookBuild(tx, workspaceID, input.BuildID, true)
		if err != nil {
			return err
		}
		if build.Status != sharedtextbook.BookBuildReadyForReview {
			return ErrInvalidState
		}
		var existing RightsItemRow
		err = tx.Where("build_id = ? AND subject_ref = ?", build.ID, input.SubjectRef).First(&existing).Error
		if err == nil {
			if existing.WorkType == input.WorkType && existing.RightsBasis == input.RightsBasis && existing.AllowedUse == input.AllowedUse && existing.Attribution == input.Attribution && existing.PublicationStatus == input.PublicationStatus {
				result = &existing
				return nil
			}
			return ErrInvalidState
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		result = &RightsItemRow{
			ID: uuid.New(), BuildID: build.ID, ProjectID: build.ProjectID, SubjectRef: input.SubjectRef, WorkType: input.WorkType,
			RightsBasis: input.RightsBasis, AllowedUse: input.AllowedUse, Attribution: input.Attribution, PublicationStatus: input.PublicationStatus,
		}
		if err := result.ToContract().Validate(); err != nil {
			return ErrInvalidState
		}
		return tx.Create(result).Error
	})
	return result, err
}

func (r *GormRepository) PromoteBookBuild(ctx context.Context, workspaceID, buildID uuid.UUID) (*BookBuildRow, error) {
	var result *BookBuildRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		build, err := loadWorkspaceBookBuild(tx, workspaceID, buildID, true)
		if err != nil {
			return err
		}
		if build.Status == sharedtextbook.BookBuildPublicationCandidate {
			result = build
			return nil
		}
		if build.Status != sharedtextbook.BookBuildReadyForReview {
			return ErrInvalidState
		}
		workspace, err := loadEditorialWorkspace(tx, build)
		if err != nil {
			return err
		}
		if !workspace.Preflight.Passed {
			return ErrInvalidState
		}
		emptyBlockers, err := json.Marshal([]string{})
		if err != nil {
			return err
		}
		if err := tx.Model(build).Updates(map[string]any{
			"status": sharedtextbook.BookBuildPublicationCandidate, "blockers_json": datatypes.JSON(emptyBlockers),
		}).Error; err != nil {
			return err
		}
		build.Status = sharedtextbook.BookBuildPublicationCandidate
		build.BlockersJSON = datatypes.JSON(emptyBlockers)
		result = build
		return nil
	})
	return result, err
}

func trimEditorialInput(input AddRightsItemInput) AddRightsItemInput {
	input.SubjectRef = strings.TrimSpace(input.SubjectRef)
	input.RightsBasis = strings.TrimSpace(input.RightsBasis)
	input.AllowedUse = strings.TrimSpace(input.AllowedUse)
	input.Attribution = strings.TrimSpace(input.Attribution)
	return input
}

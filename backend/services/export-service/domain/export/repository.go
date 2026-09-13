package export

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
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type Repository interface {
	GetByID(ctx context.Context, workspaceID uuid.UUID, blogID uuid.UUID) (Blog, error)
	GetSeriesBlogs(ctx context.Context, workspaceID uuid.UUID, parentID uuid.UUID) ([]Blog, error)
}

// TextbookChapterRepository is intentionally separate from the legacy blog
// repository contract. Textbook exports have local-workspace ownership and must
// never acquire a user-owned blog dependency just to download a chapter.
type TextbookChapterRepository interface {
	GetApprovedTextbookChapter(ctx context.Context, workspaceID uuid.UUID, chapterID uuid.UUID) (TextbookChapterExport, error)
	GetTextbookBookBuild(ctx context.Context, workspaceID uuid.UUID, buildID uuid.UUID) (TextbookBookBuildExport, error)
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) GetByID(ctx context.Context, workspaceID uuid.UUID, blogID uuid.UUID) (Blog, error) {
	var blog Blog
	if err := r.db.WithContext(ctx).Where("id = ? AND workspace_id = ?", blogID, workspaceID).First(&blog).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Blog{}, ErrBlogNotFound
		}
		return Blog{}, err
	}
	return blog, nil
}

func (r *GormRepository) GetSeriesBlogs(ctx context.Context, workspaceID uuid.UUID, parentID uuid.UUID) ([]Blog, error) {
	var blogs []Blog

	var parent Blog
	if err := r.db.WithContext(ctx).Where("id = ? AND workspace_id = ?", parentID, workspaceID).First(&parent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSeriesNotFound
		}
		return nil, err
	}
	blogs = append(blogs, parent)

	var children []Blog
	if err := r.db.WithContext(ctx).
		Where("parent_id = ? AND workspace_id = ?", parentID, workspaceID).
		Order("chapter_sort ASC").
		Find(&children).Error; err != nil {
		return nil, err
	}

	blogs = append(blogs, children...)
	return blogs, nil
}

func (r *GormRepository) GetApprovedTextbookChapter(ctx context.Context, workspaceID uuid.UUID, chapterID uuid.UUID) (TextbookChapterExport, error) {
	var chapter struct {
		ID                 uuid.UUID  `gorm:"column:id"`
		ApprovedRevisionID *uuid.UUID `gorm:"column:approved_revision_id"`
	}
	query := r.db.WithContext(ctx).Table("textbook_chapters AS chapter").
		Select("chapter.id, chapter.approved_revision_id").
		Joins("JOIN textbook_projects AS project ON project.id = chapter.project_id").
		Where("chapter.id = ? AND project.workspace_id = ? AND chapter.deleted_at IS NULL AND project.deleted_at IS NULL", chapterID, workspaceID).
		Take(&chapter)
	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return TextbookChapterExport{}, ErrTextbookChapterNotFound
	}
	if query.Error != nil {
		return TextbookChapterExport{}, query.Error
	}
	if chapter.ApprovedRevisionID == nil {
		return TextbookChapterExport{}, ErrTextbookChapterNotApproved
	}

	var exported TextbookChapterExport
	query = r.db.WithContext(ctx).Table("textbook_chapters AS chapter").
		Select(`chapter.id AS chapter_id, chapter.title, chapter.sort_order,
			revision.id AS revision_id, revision.revision_number, revision.markdown,
			revision.content_hash, revision.book_contract_revision_id,
			revision.style_sheet_revision_id, revision.blueprint_revision_id,
			revision.evidence_pack_hash, revision.prompt_hash, revision.provider_name,
			revision.model_name, current_book_contract.content_hash AS current_book_contract_hash,
			current_style_sheet.content_hash AS current_style_sheet_hash,
			projection.id AS projection_revision_id`).
		Joins("JOIN chapter_revisions AS revision ON revision.id = chapter.approved_revision_id AND revision.chapter_id = chapter.id").
		Joins("LEFT JOIN chapter_revisions AS projection ON projection.id = revision.parent_revision_id AND projection.chapter_id = chapter.id AND projection.kind = ?", sharedtextbook.RevisionKindCandidate).
		Joins("JOIN textbook_projects AS project ON project.id = chapter.project_id").
		Joins("LEFT JOIN book_contract_revisions AS current_book_contract ON current_book_contract.id = project.approved_book_contract_revision_id").
		Joins("LEFT JOIN style_sheet_revisions AS current_style_sheet ON current_style_sheet.id = project.approved_style_sheet_revision_id").
		Where("chapter.id = ? AND revision.id = ? AND revision.kind = ?", chapter.ID, *chapter.ApprovedRevisionID, "approved").
		Take(&exported)
	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		// A dangling or non-approved pointer is an integrity failure, but treating it
		// as unapproved keeps the exporter fail-closed and never exposes another revision.
		return TextbookChapterExport{}, ErrTextbookChapterNotApproved
	}
	if query.Error != nil {
		return TextbookChapterExport{}, query.Error
	}
	projectionRevisionIDs := []uuid.UUID{exported.RevisionID}
	if exported.ProjectionRevisionID != nil {
		projectionRevisionIDs = append(projectionRevisionIDs, *exported.ProjectionRevisionID)
	}
	if err := r.db.WithContext(ctx).Table("textbook_code_artifacts").
		Select("id, kind, language, entrypoint, manifest_json, manifest_hash, artifact_hash, limitations_json AS limitations, status").
		Where("revision_id IN ?", projectionRevisionIDs).
		Order("created_at ASC, id ASC").
		Find(&exported.CodeArtifacts).Error; err != nil {
		return TextbookChapterExport{}, err
	}
	if err := r.db.WithContext(ctx).Table("textbook_runtime_evidence").
		Select(`id, code_artifact_id, code_artifact_hash, input_hash, kind, status,
			command_manifest_hash, runner_image_digest, toolchain_version, tool_name,
			tool_version, structured_output, output_truncated, captured_at, expires_at, stale_reason`).
		Where("revision_id IN ?", projectionRevisionIDs).
		Order("created_at ASC, id ASC").
		Find(&exported.RuntimeEvidence).Error; err != nil {
		return TextbookChapterExport{}, err
	}
	exported.CurrentBookContractHash = canonicalExportContractHash(exported.CurrentBookContractHash)
	exported.CurrentStyleSheetHash = canonicalExportContractHash(exported.CurrentStyleSheetHash)
	deriveExportRuntimeEvidenceCurrentness(exported.CodeArtifacts, exported.RuntimeEvidence, exported.CurrentBookContractHash, exported.CurrentStyleSheetHash, time.Now().UTC())
	if err := r.db.WithContext(ctx).Table("textbook_manuscript_assets").
		Select("id, evidence_id, stable_ref, kind, content_hash, alt_text, source, generation_method, visual_purpose, rights_status, status").
		Where("revision_id IN ?", projectionRevisionIDs).
		Order("created_at ASC, id ASC").
		Find(&exported.Assets).Error; err != nil {
		return TextbookChapterExport{}, err
	}
	return exported, nil
}

func canonicalExportContractHash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "sha256:") {
		return value
	}
	return "sha256:" + value
}

func deriveExportRuntimeEvidenceCurrentness(artifacts []TextbookCodeArtifactExport, evidence []TextbookRuntimeEvidenceExport, bookContractHash, styleSheetHash string, now time.Time) {
	artifactsByID := make(map[uuid.UUID]TextbookCodeArtifactExport, len(artifacts))
	for _, artifact := range artifacts {
		artifactsByID[artifact.ID] = artifact
	}
	for index := range evidence {
		item := &evidence[index]
		if item.Status != string(sharedtextbook.ArtifactStatusVerified) {
			continue
		}
		artifact, ok := artifactsByID[item.CodeArtifactID]
		if !ok {
			item.StaleReason = "关联的代码工件已不可用，需重新验证。"
			continue
		}
		var manifest sharedtextbook.TeachingArtifactManifest
		if err := json.Unmarshal(artifact.ManifestJSON, &manifest); err != nil {
			item.StaleReason = "教学工件清单无效，不能将此运行记录作为当前证据。"
			continue
		}
		item.StaleReason = sharedtextbook.TeachingArtifactEvidenceStaleReason(manifest, bookContractHash, styleSheetHash, sharedtextbook.RuntimeEvidence{ID: item.ID.String(), CodeArtifactID: item.CodeArtifactID.String(), CodeArtifactHash: item.CodeArtifactHash, InputHash: item.InputHash, Kind: sharedtextbook.RuntimeEvidenceKind(item.Kind), Status: sharedtextbook.ArtifactStatus(item.Status), CommandManifestHash: item.CommandManifestHash, RunnerImageDigest: item.RunnerImageDigest, ToolchainVersion: item.ToolchainVersion, ToolName: item.ToolName, ToolVersion: item.ToolVersion, CapturedAt: exportEvidenceCapturedAt(item.CapturedAt), ExpiresAt: item.ExpiresAt, StaleReason: item.StaleReason}, now)
	}
}

func exportEvidenceCapturedAt(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

// GetTextbookBookBuild reads only the immutable AST embedded in a Build
// manifest. It deliberately does not join chapters or revisions after the
// build was frozen.
func (r *GormRepository) GetTextbookBookBuild(ctx context.Context, workspaceID uuid.UUID, buildID uuid.UUID) (TextbookBookBuildExport, error) {
	var row struct {
		ID               uuid.UUID                      `gorm:"column:id"`
		ProjectID        uuid.UUID                      `gorm:"column:project_id"`
		ManifestHash     string                         `gorm:"column:manifest_hash"`
		ManifestJSON     json.RawMessage                `gorm:"column:manifest_json"`
		Status           sharedtextbook.BookBuildStatus `gorm:"column:status"`
		BookContractHash string                         `gorm:"column:book_contract_hash"`
		StyleSheetHash   string                         `gorm:"column:style_sheet_hash"`
	}
	query := r.db.WithContext(ctx).Table("textbook_book_builds AS build").
		Select("build.id, build.project_id, build.manifest_hash, build.manifest_json, build.status, book_contract.content_hash AS book_contract_hash, style_sheet.content_hash AS style_sheet_hash").
		Joins("JOIN textbook_projects AS project ON project.id = build.project_id").
		Joins("JOIN book_contract_revisions AS book_contract ON book_contract.id = build.book_contract_revision_id").
		Joins("JOIN style_sheet_revisions AS style_sheet ON style_sheet.id = build.style_sheet_revision_id").
		Where("build.id = ? AND project.workspace_id = ? AND project.deleted_at IS NULL", buildID, workspaceID).
		Take(&row)
	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return TextbookBookBuildExport{}, ErrTextbookBookBuildNotFound
	}
	if query.Error != nil {
		return TextbookBookBuildExport{}, query.Error
	}
	var manifest struct {
		Format    string                          `json:"format"`
		InputHash string                          `json:"input_hash"`
		Book      sharedtextbook.CanonicalBookAST `json:"book"`
		Chapters  []struct {
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
	if err := json.Unmarshal(row.ManifestJSON, &manifest); err != nil || manifest.Format != "inkwords.book-build.v1" || !strings.HasPrefix(manifest.InputHash, "sha256:") {
		return TextbookBookBuildExport{}, fmt.Errorf("%w: malformed frozen book build manifest", ErrTextbookBookBuildInvalid)
	}
	if err := manifest.Book.Validate(); err != nil {
		return TextbookBookBuildExport{}, fmt.Errorf("%w: %w", ErrTextbookBookBuildInvalid, err)
	}
	if _, err := sharedtextbook.ReadBookVideoRunbooks(row.ManifestJSON); err != nil {
		return TextbookBookBuildExport{}, fmt.Errorf("%w: invalid video runbooks", ErrTextbookBookBuildInvalid)
	}
	rightsRows := []struct {
		ID                uuid.UUID
		BuildID           uuid.UUID
		ProjectID         uuid.UUID
		SubjectRef        string
		WorkType          sharedtextbook.RightsWorkType
		RightsBasis       string
		AllowedUse        string
		Attribution       string
		PublicationStatus sharedtextbook.RightsStatus
	}{}
	if err := r.db.WithContext(ctx).Table("textbook_rights_items").Where("build_id = ?", row.ID).Order("created_at ASC, id ASC").Find(&rightsRows).Error; err != nil {
		return TextbookBookBuildExport{}, err
	}
	rights := make([]sharedtextbook.RightsItem, 0, len(rightsRows))
	for _, item := range rightsRows {
		rights = append(rights, sharedtextbook.RightsItem{ID: item.ID.String(), BuildID: item.BuildID.String(), ProjectID: item.ProjectID.String(), SubjectRef: item.SubjectRef, WorkType: item.WorkType, RightsBasis: item.RightsBasis, AllowedUse: item.AllowedUse, Attribution: item.Attribution, PublicationStatus: item.PublicationStatus})
	}
	reviewRows := []struct {
		ContractVersion string
		ManifestHash    string
		Revision        int
		ReviewerKind    string
		Verdict         string
		Score           int
		Scope           string
		EvidenceRefs    datatypes.JSON
		HardFailures    datatypes.JSON
		ID              uuid.UUID
		BuildID         uuid.UUID
		Stage           sharedtextbook.PublicationReviewStage
		Reviewer        string
		Notes           string
		Automated       bool
		CompletedAt     time.Time
	}{}
	if err := r.db.WithContext(ctx).Table("textbook_publication_reviews").Where("build_id = ?", row.ID).Order("completed_at ASC, id ASC").Find(&reviewRows).Error; err != nil {
		return TextbookBookBuildExport{}, err
	}
	reviews := make([]sharedtextbook.HumanPublicationReview, 0, len(reviewRows))
	for _, review := range reviewRows {
		item := sharedtextbook.HumanPublicationReview{ID: review.ID.String(), BuildID: review.BuildID.String(), Stage: review.Stage, Reviewer: review.Reviewer, Notes: review.Notes, Automated: review.Automated, CompletedAt: review.CompletedAt}
		if review.ContractVersion != "" {
			item.ContractVersion, item.ManifestHash, item.Revision = review.ContractVersion, review.ManifestHash, review.Revision
			item.ReviewerKind, item.Verdict, item.Score, item.Scope = review.ReviewerKind, review.Verdict, review.Score, review.Scope
			if json.Unmarshal(review.EvidenceRefs, &item.EvidenceRefs) != nil || json.Unmarshal(review.HardFailures, &item.HardFailures) != nil || item.Validate() != nil {
				return TextbookBookBuildExport{}, ErrTextbookBookBuildInvalid
			}
		}
		reviews = append(reviews, item)
	}
	requiredSubjects := make([]sharedtextbook.RightsSubject, 0, len(manifest.Chapters)+len(manifest.Artifacts)+len(manifest.Assets))
	for _, chapter := range manifest.Chapters {
		requiredSubjects = append(requiredSubjects, sharedtextbook.RightsSubject{SubjectRef: "chapter-revision:" + chapter.RevisionID, WorkType: sharedtextbook.RightsWorkTypeProse})
	}
	for _, artifact := range manifest.Artifacts {
		requiredSubjects = append(requiredSubjects, sharedtextbook.RightsSubject{SubjectRef: "code-artifact:" + artifact.ID, WorkType: sharedtextbook.RightsWorkTypeCode})
	}
	for _, asset := range manifest.Assets {
		workType := sharedtextbook.RightsWorkTypeImage
		if asset.Kind == sharedtextbook.AssetKindScreenshot {
			workType = sharedtextbook.RightsWorkTypeScreenshot
		}
		requiredSubjects = append(requiredSubjects, sharedtextbook.RightsSubject{SubjectRef: "asset:" + asset.ID, WorkType: workType})
	}
	manifestStatus := sharedtextbook.QualityStatusPass
	if strings.TrimSpace(row.ManifestHash) == "" || manifest.Format != "inkwords.book-build.v1" || !strings.HasPrefix(manifest.InputHash, "sha256:") || manifest.Book.Validate() != nil {
		manifestStatus = sharedtextbook.QualityStatusHardFail
	}
	candidateStatus := sharedtextbook.QualityStatusHardFail
	if row.Status == sharedtextbook.BookBuildPublicationCandidate {
		candidateStatus = sharedtextbook.QualityStatusPass
	}
	revisionIDs := make([]string, 0, len(manifest.Chapters))
	for _, chapter := range manifest.Chapters {
		revisionIDs = append(revisionIDs, chapter.RevisionID)
	}
	qualityStatus := sharedtextbook.QualityStatusPass
	qualitySnapshot, qualityErr := sharedtextbook.ReadBookQualitySnapshot(row.ManifestJSON)
	if qualityErr != nil {
		return TextbookBookBuildExport{}, fmt.Errorf("%w: invalid quality snapshot", ErrTextbookBookBuildInvalid)
	}
	if qualitySnapshot != nil && !qualitySnapshot.Assess().Passed {
		qualityStatus = sharedtextbook.QualityStatusHardFail
	}
	var qualityRows []struct {
		ID                uuid.UUID
		QualityReportJSON datatypes.JSON
	}
	if qualitySnapshot != nil {
		// New builds are evaluated exclusively from frozen detector reports.
	} else if len(revisionIDs) == 0 {
		qualityStatus = sharedtextbook.QualityStatusHardFail
	} else if err := r.db.WithContext(ctx).Table("chapter_revisions").Select("id, quality_report_json").Where("id IN ? AND kind = ?", revisionIDs, sharedtextbook.RevisionKindApproved).Find(&qualityRows).Error; err != nil {
		return TextbookBookBuildExport{}, err
	} else {
		if len(qualityRows) != len(revisionIDs) {
			qualityStatus = sharedtextbook.QualityStatusHardFail
		}
		for _, revision := range qualityRows {
			if sharedtextbook.ValidateCurrentSampleQualityReport(json.RawMessage(revision.QualityReportJSON)) != nil {
				qualityStatus = sharedtextbook.QualityStatusHardFail
				break
			}
		}
	}
	artifactStatus := sharedtextbook.QualityStatusPass
	verificationSnapshot, snapshotErr := sharedtextbook.ReadBookVerificationSnapshot(row.ManifestJSON)
	if snapshotErr != nil {
		return TextbookBookBuildExport{}, fmt.Errorf("%w: invalid verification snapshot", ErrTextbookBookBuildInvalid)
	}
	if verificationSnapshot != nil && !verificationSnapshot.Assess(time.Now().UTC()).Passed {
		artifactStatus = sharedtextbook.QualityStatusHardFail
	}
	artifactIDs := make([]string, 0, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		artifactIDs = append(artifactIDs, artifact.ID)
	}
	if len(artifactIDs) > 0 && verificationSnapshot == nil {
		var artifactRows []TextbookCodeArtifactExport
		if err := r.db.WithContext(ctx).Table("textbook_code_artifacts").
			Select("id, kind, language, entrypoint, manifest_json, manifest_hash, artifact_hash, limitations_json AS limitations, status").
			Where("id IN ?", artifactIDs).Find(&artifactRows).Error; err != nil {
			return TextbookBookBuildExport{}, err
		}
		var evidenceRows []TextbookRuntimeEvidenceExport
		if err := r.db.WithContext(ctx).Table("textbook_runtime_evidence").
			Select(`id, code_artifact_id, code_artifact_hash, input_hash, kind, status,
				command_manifest_hash, runner_image_digest, toolchain_version, tool_name,
				tool_version, structured_output, output_truncated, captured_at, expires_at, stale_reason`).
			Where("code_artifact_id IN ?", artifactIDs).Find(&evidenceRows).Error; err != nil {
			return TextbookBookBuildExport{}, err
		}
		now := time.Now().UTC()
		deriveExportRuntimeEvidenceCurrentness(artifactRows, evidenceRows, canonicalExportContractHash(row.BookContractHash), canonicalExportContractHash(row.StyleSheetHash), now)
		if len(artifactRows) != len(artifactIDs) {
			artifactStatus = sharedtextbook.QualityStatusHardFail
		} else {
			for _, artifact := range artifactRows {
				if artifact.Status != string(sharedtextbook.ArtifactStatusVerified) || !hasCurrentEvidence(evidenceRows, artifact, now) {
					artifactStatus = sharedtextbook.QualityStatusHardFail
					break
				}
			}
		}
	}
	assetStatus := sharedtextbook.QualityStatusPass
	var assetCount, readyAssetCount int64
	assetIDs := make([]string, 0, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		assetIDs = append(assetIDs, asset.ID)
	}
	if len(assetIDs) > 0 {
		if err := r.db.WithContext(ctx).Table("textbook_manuscript_assets").Where("id IN ?", assetIDs).Count(&assetCount).Error; err != nil {
			return TextbookBookBuildExport{}, err
		}
		if err := r.db.WithContext(ctx).Table("textbook_manuscript_assets").Where("id IN ? AND rights_status = ?", assetIDs, sharedtextbook.RightsStatusReady).Count(&readyAssetCount).Error; err != nil {
			return TextbookBookBuildExport{}, err
		}
		if assetCount != int64(len(assetIDs)) || readyAssetCount != assetCount {
			assetStatus = sharedtextbook.QualityStatusHardFail
		}
	}
	checks := []sharedtextbook.AutomatedPublicationCheck{
		{ID: "frozen-manifest-readable", Detector: "export-service-book-build-v1", Status: manifestStatus},
		{ID: "explicit-publication-candidate", Detector: "core-api-book-build-state", Status: candidateStatus},
		{ID: "approved-revision-quality", Detector: "export-service-book-build-v1", Status: qualityStatus},
		{ID: "generated-artifacts-verified", Detector: "export-service-book-build-v1", Status: artifactStatus},
		{ID: "manuscript-assets-rights-ready", Detector: "export-service-book-build-v1", Status: assetStatus},
	}
	delegated, err := loadDelegatedPublicationReviews(r.db.WithContext(ctx), row.ID, row.ManifestHash)
	if err != nil {
		return TextbookBookBuildExport{}, err
	}
	ledger, err := r.loadRightsLedger(ctx, row.ID, row.ManifestHash, rights)
	if err != nil {
		return TextbookBookBuildExport{}, err
	}
	preflight := sharedtextbook.EvaluatePublicationPreflight(sharedtextbook.PublicationPreflightInput{BuildID: row.ID.String(), ManifestHash: row.ManifestHash, DelegatedReviews: delegated, RequiredRightsSubjects: requiredSubjects, RightsItems: rights, RightsAmendments: ledger.Amendments, HumanReviews: reviews, AutomatedChecks: checks})
	return TextbookBookBuildExport{BuildID: row.ID, ManifestHash: row.ManifestHash, ManifestJSON: append(json.RawMessage(nil), row.ManifestJSON...), Book: manifest.Book, RequiredRightsSubjects: requiredSubjects, RightsItems: ledger.EffectiveItems, RightsLedger: &ledger, HumanReviews: reviews, DelegatedReviews: delegated, AutomatedChecks: checks, Preflight: preflight}, nil
}

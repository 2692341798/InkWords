package textbook

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// CreateBookBuild freezes approved content and its source locators in one
// transaction. Repeated requests reuse the same immutable input hash.
func (r *GormRepository) CreateBookBuild(ctx context.Context, workspaceID uuid.UUID, input CreateBookBuildInput) (*BookBuildRow, error) {
	notices, noticeErr := sharedtextbook.FreezePublicationNotices(input.Notices)
	if noticeErr != nil {
		return nil, ErrInvalidState
	}
	var build *BookBuildRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project Project
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND workspace_id = ?", input.ProjectID, workspaceID).First(&project).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if project.ApprovedBookContractRevisionID == nil || project.ApprovedStyleSheetRevisionID == nil {
			return ErrInvalidState
		}
		var revisions []bookBuildChapterSource
		if err := tx.Table("textbook_chapters AS chapter").
			Select("chapter.id AS chapter_id, revision.id AS revision_id, projection.id AS projection_revision_id, chapter.sort_order, chapter.title, revision.markdown, revision.document_json, revision.content_hash, revision.evidence_pack_hash, revision.quality_report_json, revision.book_contract_revision_id, revision.style_sheet_revision_id").
			Joins("JOIN chapter_revisions AS revision ON revision.id = chapter.approved_revision_id AND revision.chapter_id = chapter.id").
			Joins("LEFT JOIN chapter_revisions AS projection ON projection.id = revision.parent_revision_id AND projection.chapter_id = chapter.id AND projection.kind = ?", sharedtextbook.RevisionKindCandidate).
			Where("chapter.project_id = ? AND chapter.deleted_at IS NULL AND revision.kind = ?", project.ID, sharedtextbook.RevisionKindApproved).
			Order("chapter.sort_order ASC, chapter.id ASC").Find(&revisions).Error; err != nil {
			return err
		}
		if len(revisions) == 0 {
			return ErrInvalidState
		}
		approvedIDs := make([]string, 0, len(revisions))
		projectionRevisionIDs := make([]uuid.UUID, 0, len(revisions)*2)
		bookRevisionByProjection := make(map[uuid.UUID]uuid.UUID, len(revisions)*2)
		chapters := make([]bookBuildChapterSource, 0, len(revisions))
		for _, revision := range revisions {
			if err := freezeChapterRunbook(&revision); err != nil {
				return err
			}
			citations, err := freezeBookCitations(tx, project.ID, revision)
			if err != nil {
				return err
			}
			revision.Citations = citations
			approvedIDs = append(approvedIDs, revision.RevisionID.String())
			projectionRevisionIDs = append(projectionRevisionIDs, revision.RevisionID)
			bookRevisionByProjection[revision.RevisionID] = revision.RevisionID
			if revision.ProjectionRevisionID != nil {
				projectionRevisionIDs = append(projectionRevisionIDs, *revision.ProjectionRevisionID)
				bookRevisionByProjection[*revision.ProjectionRevisionID] = revision.RevisionID
			}
			chapters = append(chapters, revision)
		}
		var artifacts []bookBuildArtifactSource
		if err := tx.Table("textbook_code_artifacts").Select("id, revision_id, artifact_hash, manifest_hash, manifest_json").Where("revision_id IN ?", projectionRevisionIDs).Order("revision_id ASC, id ASC").Find(&artifacts).Error; err != nil {
			return err
		}
		var assets []bookBuildAssetSource
		if err := tx.Table("textbook_manuscript_assets").Select("id, revision_id, kind, content_hash, stable_ref, alt_text").Where("revision_id IN ?", projectionRevisionIDs).Order("revision_id ASC, id ASC").Find(&assets).Error; err != nil {
			return err
		}
		for index := range assets {
			assets[index].BookRevisionID = bookRevisionByProjection[assets[index].RevisionID]
		}
		toolVersions := map[string]string{"canonical_ast": sharedtextbook.CanonicalBookASTFormat, "package_schema": "inkwords.textbook-package.v1"}
		toolVersions["video_runbooks"] = sharedtextbook.BookVideoRunbooksFormat
		if err := validateBookNoticeSubjects(notices, chapters, artifacts); err != nil {
			return err
		}
		if len(notices) > 0 {
			toolVersions["canonical_ast"] = sharedtextbook.CanonicalBookASTWithNoticesFormat
			toolVersions["publication_notices"] = sharedtextbook.PublicationNoticeFormat
		}
		builtAt := time.Now().UTC()
		verification, err := freezeBookVerification(tx, project, artifacts, builtAt)
		if err != nil {
			return err
		}
		toolVersions["verification_snapshot"] = sharedtextbook.BookVerificationSnapshotFormat
		// Receipt content participates in idempotency; wall-clock passage alone
		// must not create a duplicate build or renew an old observation.
		verificationInput := verification
		verificationInput.CapturedAt = time.Time{}
		quality, err := freezeBookQuality(chapters, project.ApprovedBookContractRevisionID.String(), project.ApprovedStyleSheetRevisionID.String(), builtAt)
		if err != nil {
			return err
		}
		toolVersions["quality_snapshot"] = sharedtextbook.BookQualitySnapshotFormat
		qualityInput := quality
		qualityInput.CapturedAt = time.Time{}
		freezeInput := bookBuildFreezeInput{
			Notices:      notices,
			Quality:      qualityInput,
			Verification: verificationInput,
			Format:       bookBuildInputFormat, ProjectID: project.ID.String(), Title: project.Title,
			BookContractRevisionID: project.ApprovedBookContractRevisionID.String(), StyleSheetRevisionID: project.ApprovedStyleSheetRevisionID.String(),
			Chapters: chapters, Artifacts: artifacts, Assets: assets, ToolVersions: toolVersions,
		}
		inputHash := digestJSON(freezeInput)
		var existing BookBuildRow
		err = tx.Where("project_id = ? AND input_hash = ?", project.ID, inputHash).First(&existing).Error
		if err == nil {
			build = &existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		bookAST, err := newFrozenBookBuildAST(project.Title, builtAt, chapters, assets)
		if err != nil {
			return err
		}
		if len(notices) > 0 {
			bookAST.Format = sharedtextbook.CanonicalBookASTWithNoticesFormat
			bookAST.Notices = notices
		}
		if err := bookAST.Validate(); err != nil {
			return ErrInvalidState
		}
		manifest := struct {
			Quality                sharedtextbook.BookQualitySnapshot      `json:"quality_snapshot"`
			Verification           sharedtextbook.BookVerificationSnapshot `json:"verification_snapshot"`
			Format                 string                                  `json:"format"`
			InputHash              string                                  `json:"input_hash"`
			ProjectID              string                                  `json:"project_id"`
			BookContractRevisionID string                                  `json:"book_contract_revision_id"`
			StyleSheetRevisionID   string                                  `json:"style_sheet_revision_id"`
			Chapters               []bookBuildChapterSource                `json:"chapters"`
			Book                   sharedtextbook.CanonicalBookAST         `json:"book"`
			Artifacts              any                                     `json:"code_artifacts"`
			Assets                 any                                     `json:"assets"`
			ToolVersions           map[string]string                       `json:"tool_versions"`
		}{Quality: quality, Verification: verification, Format: "inkwords.book-build.v1", InputHash: inputHash, ProjectID: project.ID.String(), BookContractRevisionID: project.ApprovedBookContractRevisionID.String(), StyleSheetRevisionID: project.ApprovedStyleSheetRevisionID.String(), Chapters: chapters, Book: bookAST, Artifacts: artifacts, Assets: assets, ToolVersions: toolVersions}
		manifestJSON, err := marshalStructuredJSON(manifest)
		if err != nil {
			return err
		}
		manifestHash := digestJSON(manifest)
		err = tx.Where("project_id = ? AND manifest_hash = ?", project.ID, manifestHash).First(&existing).Error
		if err == nil {
			build = &existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		approvedJSON, err := marshalStructuredJSON(approvedIDs)
		if err != nil {
			return err
		}
		toolVersionsJSON, err := marshalStructuredJSON(manifest.ToolVersions)
		if err != nil {
			return err
		}
		blockersJSON, err := marshalStructuredJSON([]string{"尚未记录完整人工审校与整书排版校对；不能标记为出版候选。"})
		if err != nil {
			return err
		}
		build = &BookBuildRow{ProjectID: project.ID, BookContractRevisionID: *project.ApprovedBookContractRevisionID, StyleSheetRevisionID: *project.ApprovedStyleSheetRevisionID, ApprovedRevisionIDs: datatypes.JSON(approvedJSON), ToolVersionsJSON: datatypes.JSON(toolVersionsJSON), ManifestJSON: datatypes.JSON(manifestJSON), InputHash: inputHash, ManifestHash: manifestHash, Status: sharedtextbook.BookBuildReadyForReview, BlockersJSON: datatypes.JSON(blockersJSON)}
		return tx.Create(build).Error
	})
	return build, err
}

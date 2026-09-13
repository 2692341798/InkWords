package textbook

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

const bookBuildInputFormat = "inkwords.book-build-input.v4"

// bookBuildChapterSource contains the immutable fields read while a project
// row is locked. It is deliberately separate from ChapterRevision so a build
// cannot accidentally follow a later current revision.
type bookBuildChapterSource struct {
	VideoRunbook           *sharedtextbook.VideoRunbookProjection `json:"video_runbook" gorm:"-"`
	VideoRunbookHash       string                                 `json:"video_runbook_hash" gorm:"-"`
	QualityReportJSON      json.RawMessage                        `gorm:"column:quality_report_json" json:"-"`
	BookContractRevisionID uuid.UUID                              `gorm:"column:book_contract_revision_id" json:"-"`
	StyleSheetRevisionID   uuid.UUID                              `gorm:"column:style_sheet_revision_id" json:"-"`
	DocumentJSON           []byte                                 `gorm:"column:document_json" json:"-"`
	Citations              []sharedtextbook.CanonicalBookCitation `json:"citations,omitempty"`
	ChapterID              uuid.UUID                              `json:"chapter_id"`
	RevisionID             uuid.UUID                              `json:"revision_id"`
	ProjectionRevisionID   *uuid.UUID                             `gorm:"column:projection_revision_id" json:"-"`
	SortOrder              int                                    `json:"sort_order"`
	Title                  string                                 `json:"title"`
	Markdown               string                                 `json:"markdown"`
	ContentHash            string                                 `json:"content_hash"`
	EvidencePackHash       string                                 `json:"evidence_pack_hash"`
}

type bookBuildAssetSource struct {
	ID             uuid.UUID                `json:"id"`
	RevisionID     uuid.UUID                `json:"revision_id"`
	BookRevisionID uuid.UUID                `gorm:"-" json:"-"`
	Kind           sharedtextbook.AssetKind `json:"kind"`
	StableRef      string                   `json:"stable_ref"`
	ContentHash    string                   `json:"content_hash"`
	AltText        string                   `json:"alt_text"`
}

type bookBuildArtifactSource struct {
	ManifestJSON json.RawMessage `gorm:"column:manifest_json" json:"-"`
	ID           uuid.UUID       `json:"id"`
	RevisionID   uuid.UUID       `json:"revision_id"`
	ArtifactHash string          `json:"artifact_hash"`
	ManifestHash string          `json:"manifest_hash"`
}

type bookBuildFreezeInput struct {
	Notices                []sharedtextbook.PublicationNotice      `json:"publication_notices,omitempty"`
	Quality                sharedtextbook.BookQualitySnapshot      `json:"quality_snapshot"`
	Verification           sharedtextbook.BookVerificationSnapshot `json:"verification_snapshot"`
	Format                 string                                  `json:"format"`
	ProjectID              string                                  `json:"project_id"`
	Title                  string                                  `json:"title"`
	BookContractRevisionID string                                  `json:"book_contract_revision_id"`
	StyleSheetRevisionID   string                                  `json:"style_sheet_revision_id"`
	Chapters               []bookBuildChapterSource                `json:"chapters"`
	Artifacts              []bookBuildArtifactSource               `json:"code_artifacts"`
	Assets                 []bookBuildAssetSource                  `json:"assets"`
	ToolVersions           map[string]string                       `json:"tool_versions"`
}

// newFrozenBookBuildAST makes the manuscript payload that is persisted inside
// a BookBuild manifest. Renderers must use this value, never mutable chapter
// pointers or a fresh database ordering query.
func newFrozenBookBuildAST(title string, builtAt time.Time, chapters []bookBuildChapterSource, assets []bookBuildAssetSource) (sharedtextbook.CanonicalBookAST, error) {
	assetsByRevision := make(map[uuid.UUID][]sharedtextbook.CanonicalBookAsset, len(chapters))
	for _, asset := range assets {
		bookRevisionID := asset.BookRevisionID
		if bookRevisionID == uuid.Nil {
			bookRevisionID = asset.RevisionID
		}
		assetsByRevision[bookRevisionID] = append(assetsByRevision[bookRevisionID], sharedtextbook.CanonicalBookAsset{
			ID:          asset.ID.String(),
			StableRef:   asset.StableRef,
			ContentHash: asset.ContentHash,
			AltText:     asset.AltText,
		})
	}
	canonicalChapters := make([]sharedtextbook.CanonicalBookChapter, 0, len(chapters))
	for _, chapter := range chapters {
		contentHash := strings.TrimSpace(chapter.ContentHash)
		if !strings.HasPrefix(contentHash, "sha256:") {
			contentHash = "sha256:" + contentHash
		}
		canonicalChapters = append(canonicalChapters, sharedtextbook.CanonicalBookChapter{
			Citations:   chapter.Citations,
			ID:          chapter.ChapterID.String(),
			Order:       chapter.SortOrder,
			Title:       chapter.Title,
			Markdown:    chapter.Markdown,
			ContentHash: contentHash,
			Assets:      assetsByRevision[chapter.RevisionID],
		})
	}
	book, err := sharedtextbook.NewCanonicalBookAST(title, builtAt, canonicalChapters)
	if err != nil {
		return sharedtextbook.CanonicalBookAST{}, fmt.Errorf("validate frozen book build AST: %w", err)
	}
	return book, nil
}

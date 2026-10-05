package textbook

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	shared "inkwords-backend/shared/kernel/textbook"
)

// Runs inside the existing real-PostgreSQL acceptance fixture. All fixture
// changes below roll back before the surrounding repository test continues.
func testFrozenBookCitations(t *testing.T, database *gorm.DB, workspaceID uuid.UUID, project *Project, approved *ChapterRevision, chunkID string) {
	t.Helper()
	tx := database.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	hash := "sha256:" + strings.Repeat("e", 64)
	require.NoError(t, tx.Model(&ParsedChunk{}).Where("id = ?", chunkID).Update("text_hash", hash).Error)
	document := datatypes.JSON(`{"evidence_aliases":{"route":"evidence-` + chunkID + `"}}`)
	markdown := "引用 [evidence:route]。代码 `[evidence:literal]`。"
	require.NoError(t, tx.Model(&ChapterRevision{}).Where("id = ?", approved.ID).Updates(map[string]any{"markdown": markdown, "document_json": document}).Error)
	repo := NewGormRepository(tx)
	build, err := repo.CreateBookBuild(context.Background(), workspaceID, CreateBookBuildInput{ProjectID: project.ID})
	require.NoError(t, err)
	var manifest struct {
		Book shared.CanonicalBookAST `json:"book"`
	}
	require.NoError(t, json.Unmarshal(build.ManifestJSON, &manifest))
	require.NoError(t, manifest.Book.Validate())
	require.Equal(t, markdown, manifest.Book.Chapters[0].Markdown)
	require.Len(t, manifest.Book.Chapters[0].Citations, 1)
	citation := manifest.Book.Chapters[0].Citations[0]
	require.Equal(t, hash, citation.Evidence.ContentHash)
	require.Equal(t, strings.Repeat("a", 40), citation.Snapshot.ResolvedVersion)

	chapter := bookBuildChapterSource{Markdown: markdown, DocumentJSON: document}
	_, err = freezeBookCitations(tx, uuid.New(), chapter)
	require.ErrorIs(t, err, ErrInvalidState, "foreign project chunks must not resolve")
	chapter.DocumentJSON = datatypes.JSON(`{"evidence_aliases":{"route":"evidence-missing"}}`)
	_, err = freezeBookCitations(tx, project.ID, chapter)
	require.ErrorIs(t, err, ErrInvalidState)
	chapter.DocumentJSON = document
	// A chapter may cite only official supporting material. This must not
	// weaken generation's separate requirement for a primary evidence pack.
	require.NoError(t, tx.Model(&Source{}).Where("id = ?", project.PrimarySourceID).Updates(map[string]any{"role": shared.SourceRoleOfficial, "official_confirmed": true}).Error)
	refs, err := freezeBookCitations(tx, project.ID, chapter)
	require.NoError(t, err)
	require.Equal(t, shared.SourceRoleOfficial, refs[0].Snapshot.Role)
	var retained BookBuildRow
	require.NoError(t, tx.First(&retained, "id = ?", build.ID).Error)
	require.JSONEq(t, string(build.ManifestJSON), string(retained.ManifestJSON), "source table changes cannot rewrite a frozen build")

	var plan []struct {
		Line string `gorm:"column:QUERY PLAN"`
	}
	require.NoError(t, tx.Raw(`EXPLAIN SELECT source_chunks.id FROM source_chunks JOIN source_documents ON source_documents.id = source_chunks.document_id JOIN source_snapshots ON source_snapshots.id = source_documents.snapshot_id JOIN textbook_sources ON textbook_sources.id = source_snapshots.source_id WHERE textbook_sources.project_id = ? AND textbook_sources.deleted_at IS NULL AND source_chunks.id IN ?`, project.ID, []string{chunkID}).Scan(&plan).Error)
	for _, line := range plan {
		t.Log("citation ownership query:", line.Line)
	}
}

package textbook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSourceEvidenceRejectsUnboundedAndEmptyIdentifiers(t *testing.T) {
	service := NewService(nil)
	for _, ids := range [][]string{make([]string, 201), {""}, {" "}, {strings.Repeat("x", 201)}} {
		_, err := service.ListSourceEvidence(context.Background(), uuid.New(), uuid.New(), ids...)
		require.ErrorIs(t, err, ErrInvalidState)
	}
}

// Runs against the import test's migrated PostgreSQL database, after its source
// receipts have been verified, to exercise the same joins as the real workspace.
func assertSourceEvidenceResolution(t *testing.T, database *gorm.DB, service *Service, workspaceID uuid.UUID, project *Project, firstDocumentID, targetID string) {
	t.Helper()
	ctx := context.Background()
	for ordinal := 2; ordinal <= 202; ordinal++ {
		chunk := ParsedChunk{ID: fmt.Sprintf("older-chunk-%03d", ordinal), DocumentID: firstDocumentID, Ordinal: ordinal, HeadingPath: []byte(`[]`), Locator: []byte(`{"path":"older.go"}`), TextHash: sourceImportTestDigest([]byte("PRIVATE SOURCE BODY")), SearchText: "PRIVATE SOURCE BODY"}
		require.NoError(t, database.Create(&chunk).Error)
	}
	initial, err := service.ListSourceEvidence(ctx, workspaceID, project.ID)
	require.NoError(t, err)
	require.Len(t, initial, 200)
	for _, row := range initial {
		require.NotEqual(t, targetID, row.ID)
	}
	rows, err := service.ListSourceEvidence(ctx, workspaceID, project.ID, targetID, targetID, "absent", "' OR 1=1 --")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, targetID, rows[0].ID)
	var locator map[string]any
	require.NoError(t, json.Unmarshal(rows[0].Locator, &locator))
	require.Equal(t, "(*RouterGroup).GET", locator["symbol"])
	encoded, err := json.Marshal(rows)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "search_text")
	require.NotContains(t, string(encoded), "PRIVATE SOURCE BODY")

	other := *project
	other.ID, other.PrimarySourceID = uuid.New(), nil
	require.NoError(t, database.Create(&other).Error)
	rows, err = service.ListSourceEvidence(ctx, workspaceID, other.ID, targetID)
	require.NoError(t, err)
	require.Empty(t, rows)
	_, err = service.ListSourceEvidence(ctx, uuid.New(), project.ID, targetID)
	require.ErrorIs(t, err, ErrNotFound)
	var plan []string
	require.NoError(t, database.Raw(`EXPLAIN (FORMAT TEXT) SELECT source_chunks.id, source_chunks.locator FROM source_chunks
JOIN source_documents ON source_documents.id = source_chunks.document_id
JOIN source_snapshots ON source_snapshots.id = source_documents.snapshot_id
JOIN textbook_sources ON textbook_sources.id = source_snapshots.source_id
WHERE textbook_sources.project_id = ? AND textbook_sources.deleted_at IS NULL AND source_chunks.id IN (?)
ORDER BY source_documents.created_at, source_chunks.ordinal, source_chunks.id LIMIT 200`, project.ID, targetID).Scan(&plan).Error)
	t.Logf("saved evidence lookup query plan: %v", plan)
	require.NoError(t, database.Model(&Source{}).Where("id = ?", *project.PrimarySourceID).Update("deleted_at", gorm.Expr("CURRENT_TIMESTAMP")).Error)
	rows, err = service.ListSourceEvidence(ctx, workspaceID, project.ID, targetID)
	require.NoError(t, err)
	require.Empty(t, rows)
}

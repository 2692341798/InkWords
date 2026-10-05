package textbook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/datatypes"
	coretask "inkwords-backend/services/core-api/domain/task"
	shared "inkwords-backend/shared/kernel/textbook"
	platformpostgres "inkwords-backend/shared/platform/postgres"
)

func TestOfficialWebRepositoryKeepsLegacySnapshotAndPersistsV2Audit(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("textbook_test"), postgrescontainer.WithUsername("inkwords"), postgrescontainer.WithPassword("inkwords-test-password"), testcontainers.WithAdditionalWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := platformpostgres.InitCore(dsn)
	require.NoError(t, err)
	workspace, err := platformpostgres.EnsureLocalWorkspace(ctx, db)
	require.NoError(t, err)
	repo := NewGormRepository(db)
	service := NewService(repo)
	project, err := service.CreateProject(ctx, CreateProjectInput{WorkspaceID: workspace.ID, Title: "Official source audit", Audience: shared.AudienceFoundation, Primary: CreateSourceInput{Kind: shared.SourceKindGitRepository, Locator: "https://github.com/gin-gonic/gin"}})
	require.NoError(t, err)
	source, err := service.AddSource(ctx, workspace.ID, project.ID, CreateSourceInput{Kind: shared.SourceKindOfficialWeb, Role: shared.SourceRoleOfficial, Locator: "https://gin-gonic.com/en/docs/", OfficialConfirmed: true})
	require.NoError(t, err)
	payload, err := repo.PrepareOfficialWebImport(ctx, workspace.ID, PrepareOfficialWebImportInput{ProjectID: project.ID, SourceID: source.ID, SnapshotID: uuid.New(), AllowedPathPrefixes: []string{"/en/docs"}})
	require.NoError(t, err)
	require.Equal(t, 2, payload.TaskVersion)
	manifestHash := "sha256:" + strings.Repeat("a", 64)
	legacy := SourceSnapshot{SourceID: source.ID, ResolvedVersion: manifestHash, ContentHash: strings.TrimPrefix(manifestHash, "sha256:"), CapturedAt: time.Now().UTC(), Status: "captured", LimitsJSON: []byte(`{"legacy":true}`)}
	require.NoError(t, db.Create(&legacy).Error)
	audit := map[string]any{"entry_url": payload.EntryURL, "status": "complete", "snapshot_input_hash": manifestHash, "total_bytes": 100,
		"policy": map[string]any{"EntryURL": payload.EntryURL, "AllowedHosts": []string{"gin-gonic.com"}, "AllowedPathPrefixes": payload.AllowedPathPrefixes, "MaxPages": 200, "MaxDepth": 4, "MaxConcurrent": 1, "MaxTotalBytes": 32 << 20, "MaxPageBytes": 1 << 20, "RequestTimeout": int64(10 * time.Second), "TotalTimeout": int64(2 * time.Minute)},
		"pages":  []any{map[string]any{"canonical_url": payload.EntryURL, "content_hash": "sha256:" + strings.Repeat("b", 64), "status_code": 200, "content_type": "text/html", "depth": 0}}, "decisions": []any{map[string]any{"status": "skipped", "reason": "cross_origin"}}}
	auditJSON, err := json.Marshal(audit)
	require.NoError(t, err)
	contentHash := shared.OfficialWebParsedContentHash(manifestHash, payload.ParserVersion)
	result := shared.OfficialWebImportTaskResult{ResultVersion: 2, ParserVersion: payload.ParserVersion, ManifestHash: manifestHash, CrawlManifest: auditJSON, TaskSubtype: payload.TaskSubtype, ProjectID: payload.ProjectID, SourceID: payload.SourceID, SnapshotID: payload.SnapshotID, InputHash: payload.InputHash, ContentHash: contentHash, ResolvedVersion: contentHash, CapturedAt: time.Now().UTC(), Documents: []shared.SourceDocument{{ID: "official-doc", SnapshotID: payload.SnapshotID, CanonicalLocator: payload.EntryURL, Title: "Gin", MediaType: "text/html", ContentHash: "sha256:document"}}, Chunks: []shared.SourceChunk{{ID: "official-chunk", DocumentID: "official-doc", Ordinal: 1, Locator: shared.EvidenceLocator{URL: payload.EntryURL}, TextHash: "sha256:chunk", SearchText: "Body without navigation"}}}
	result.Chunks[0].Locator.HeadingPath = []string{"Gin"}
	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)
	task := coretask.JobTask{TaskType: "parse", TaskSubtype: payload.TaskSubtype, Status: coretask.JobTaskStatusRunning, WorkspaceID: &workspace.ID, PayloadJSON: datatypes.JSON(payloadJSON), ResultJSON: []byte(`{}`)}
	require.NoError(t, db.Create(&task).Error)
	resultJSON, err := json.Marshal(result)
	require.NoError(t, err)
	var resultMap map[string]any
	require.NoError(t, json.Unmarshal(resultJSON, &resultMap))
	require.NoError(t, repo.PersistTextbookSourceImportResult(ctx, task.ID, resultMap))
	require.NoError(t, repo.PersistTextbookSourceImportResult(ctx, task.ID, resultMap))
	var count int64
	require.NoError(t, db.Model(&SourceSnapshot{}).Where("source_id = ?", source.ID).Count(&count).Error)
	require.Equal(t, int64(2), count, "replay deduplicates v2 while preserving v1")
	var stored SourceSnapshot
	require.NoError(t, db.First(&stored, "id = ?", payload.SnapshotID).Error)
	require.Contains(t, string(stored.LimitsJSON), "cross_origin")
	require.Contains(t, string(stored.LimitsJSON), payload.ParserVersion)
	require.NoError(t, db.First(&stored, "id = ?", stored.ID).Error)
	var old SourceSnapshot
	require.NoError(t, db.First(&old, "id = ?", legacy.ID).Error)
	require.JSONEq(t, `{"legacy":true}`, string(old.LimitsJSON))
	resultMap["parser_version"] = "future"
	require.Error(t, repo.PersistTextbookSourceImportResult(ctx, task.ID, resultMap))
	var older []ParsedChunk
	for ordinal := 2; ordinal <= 502; ordinal++ {
		older = append(older, ParsedChunk{ID: fmt.Sprintf("older-%03d", ordinal), DocumentID: "official-doc", Ordinal: ordinal, HeadingPath: []byte(`[]`), Locator: []byte(`{"url":"https://gin-gonic.com/en/docs/","heading_path":["Gin"]}`), TextHash: "sha256:older", SearchText: "Unrelated older document"})
	}
	require.NoError(t, db.CreateInBatches(older, 100).Error)
	target := ParsedChunk{ID: "late-albums", DocumentID: "official-doc", Ordinal: 503, HeadingPath: []byte(`[]`), Locator: []byte(`{"url":"https://gin-gonic.com/en/docs/","heading_path":["Albums"]}`), TextHash: "sha256:albums", SearchText: "GET albums retrieves the album collection"}
	require.NoError(t, db.Create(&target).Error)
	plan, err := repo.RetrieveSourceEvidence(ctx, workspace.ID, RetrieveSourceInput{ProjectID: project.ID, Query: "albums", Limit: 8})
	require.NoError(t, err)
	require.Len(t, plan.Selected, 1)
	require.Equal(t, target.ID, plan.Selected[0].ChunkID)
	_, err = repo.RetrieveSourceEvidence(ctx, workspace.ID, RetrieveSourceInput{ProjectID: project.ID, Query: "'%_unmatched", Limit: 8})
	require.Error(t, err, "query punctuation must remain literal")
}

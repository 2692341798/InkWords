package textbook

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	coretask "inkwords-backend/services/core-api/domain/task"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	platformpostgres "inkwords-backend/shared/platform/postgres"
)

func TestTextbookSourceImportPersistsOnlyValidatedWorkerEvidence(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("source_import_test"), postgrescontainer.WithUsername("inkwords"), postgrescontainer.WithPassword("inkwords-test-password"), testcontainers.WithAdditionalWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	database, err := platformpostgres.InitCore(dsn)
	require.NoError(t, err)
	workspace, err := platformpostgres.EnsureLocalWorkspace(ctx, database)
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&coretask.JobTask{}))

	repository := NewGormRepository(database)
	service := NewService(repository)
	commit := "73726dc606796a025971fe451f0aa6f1b9b847f6"
	project, err := service.CreateProject(ctx, CreateProjectInput{WorkspaceID: workspace.ID, Title: "Gin 源码教材", Audience: sharedtextbook.AudienceFoundation, Primary: CreateSourceInput{Kind: sharedtextbook.SourceKindGitRepository, Locator: "https://github.com/gin-gonic/gin"}})
	require.NoError(t, err)
	sourceID := *project.PrimarySourceID
	content := []byte("package gin\n\nfunc (group *RouterGroup) GET(path string) {}\n")
	contentHash := sourceImportTestDigest(content)
	payload, err := service.PrepareSourceImport(ctx, workspace.ID, PrepareSourceImportInput{ProjectID: project.ID, SourceID: sourceID, SnapshotID: uuid.New(), Filename: "routergroup.go", ContentHash: contentHash, ResolvedVersion: commit, ByteSize: int64(len(content))})
	require.NoError(t, err)
	payload.ArtifactToken = contentHash
	require.Equal(t, sharedtextbook.SourceImportTaskVersion, payload.TaskVersion)
	// Seed the previous parser's immutable snapshot before replaying these
	// exact bytes through the new declaration-aware parser identity.
	payload.TaskVersion = 2
	payload.InputHash = sourceImportTestDigest([]byte("project_id=" + payload.ProjectID + "\nsource_id=" + payload.SourceID + "\nfilename=" + payload.Filename + "\ncontent_hash=" + payload.ContentHash + "\nresolved_version=" + payload.ResolvedVersion))
	require.NoError(t, payload.Validate())
	rawPayload, err := json.Marshal(payload)
	require.NoError(t, err)
	task := coretask.JobTask{TaskType: "parse", TaskSubtype: sharedtextbook.TextbookSourceImportTaskSubtype, Status: coretask.JobTaskStatusSucceeded, WorkspaceID: &workspace.ID, PayloadJSON: rawPayload, ResultJSON: []byte(`{}`)}
	require.NoError(t, database.Create(&task).Error)

	document := sharedtextbook.SourceDocument{ID: "source-import-document", SnapshotID: payload.SnapshotID, CanonicalLocator: payload.Locator, Title: "routergroup.go", MediaType: "text/x-go", ContentHash: "sha256:document", ArtifactPath: "routergroup.go"}
	chunk := sharedtextbook.SourceChunk{ID: "source-import-chunk", DocumentID: document.ID, Ordinal: 1, Locator: sharedtextbook.EvidenceLocator{Path: "routergroup.go", StartLine: 1, EndLine: 3}, CodeLanguage: "go", TextHash: "sha256:chunk", SearchText: "func (group *RouterGroup) GET(path string) {}"}
	result := sharedtextbook.SourceImportTaskResult{ResultVersion: 1, TaskSubtype: sharedtextbook.TextbookSourceImportTaskSubtype, ProjectID: payload.ProjectID, SourceID: payload.SourceID, SnapshotID: payload.SnapshotID, InputHash: payload.InputHash, Documents: []sharedtextbook.SourceDocument{document}, Chunks: []sharedtextbook.SourceChunk{chunk}}
	malformed := result
	malformed.SourceID = uuid.New().String()
	badResult, err := json.Marshal(malformed)
	require.NoError(t, err)
	var badMap map[string]any
	require.NoError(t, json.Unmarshal(badResult, &badMap))
	require.Error(t, repository.PersistTextbookSourceImportResult(ctx, task.ID, badMap))
	var snapshotCount int64
	require.NoError(t, database.Model(&SourceSnapshot{}).Count(&snapshotCount).Error)
	require.Zero(t, snapshotCount)

	resultJSON, err := json.Marshal(result)
	require.NoError(t, err)
	var resultMap map[string]any
	require.NoError(t, json.Unmarshal(resultJSON, &resultMap))
	require.NoError(t, repository.PersistTextbookSourceImportResult(ctx, task.ID, resultMap))
	require.NoError(t, repository.PersistTextbookSourceImportResult(ctx, task.ID, resultMap), "result reconciliation is idempotent")
	library, err := service.ListSourceLibrary(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Len(t, library, 1)
	require.Equal(t, document.ID, library[0].ID)
	require.Equal(t, int64(1), library[0].ChunkCount)
	require.NoError(t, database.Model(&SourceSnapshot{}).Count(&snapshotCount).Error)
	require.Equal(t, int64(1), snapshotCount)
	var snapshot SourceSnapshot
	require.NoError(t, database.Where("id = ?", payload.SnapshotID).First(&snapshot).Error)
	require.Equal(t, commit, snapshot.ResolvedVersion)

	// A repository commit is immutable, but it contains multiple independently
	// content-addressed files. Importing another file at the same commit must
	// add evidence instead of being mistaken for conflicting source bytes.
	secondContent := []byte("package gin\n\nfunc New() *Engine { return &Engine{} }\n")
	secondHash := sourceImportTestDigest(secondContent)
	secondPayload, err := service.PrepareSourceImport(ctx, workspace.ID, PrepareSourceImportInput{ProjectID: project.ID, SourceID: sourceID, SnapshotID: uuid.New(), Filename: "gin.go", ContentHash: secondHash, ResolvedVersion: commit, ByteSize: int64(len(secondContent))})
	require.NoError(t, err)
	secondPayload.ArtifactToken = secondHash
	secondPayload.InputHash = sharedtextbook.SourceImportInputHash(secondPayload.ProjectID, secondPayload.SourceID, secondPayload.SnapshotID, secondPayload.Filename, secondPayload.ContentHash, secondPayload.ResolvedVersion)
	secondTaskPayload, err := json.Marshal(secondPayload)
	require.NoError(t, err)
	secondTask := coretask.JobTask{TaskType: "parse", TaskSubtype: sharedtextbook.TextbookSourceImportTaskSubtype, Status: coretask.JobTaskStatusSucceeded, WorkspaceID: &workspace.ID, PayloadJSON: secondTaskPayload, ResultJSON: []byte(`{}`)}
	require.NoError(t, database.Create(&secondTask).Error)
	secondDocument := sharedtextbook.SourceDocument{ID: "source-import-second-document", SnapshotID: secondPayload.SnapshotID, CanonicalLocator: secondPayload.Locator, Title: "gin.go", MediaType: "text/x-go", ContentHash: secondHash, ArtifactPath: "gin.go"}
	secondChunk := sharedtextbook.SourceChunk{ID: "source-import-second-chunk", DocumentID: secondDocument.ID, Ordinal: 1, Locator: sharedtextbook.EvidenceLocator{Path: "gin.go", StartLine: 1, EndLine: 3}, CodeLanguage: "go", TextHash: "sha256:second-chunk", SearchText: "func New() *Engine { return &Engine{} }"}
	secondResult := sharedtextbook.SourceImportTaskResult{ResultVersion: 1, TaskSubtype: sharedtextbook.TextbookSourceImportTaskSubtype, ProjectID: secondPayload.ProjectID, SourceID: secondPayload.SourceID, SnapshotID: secondPayload.SnapshotID, InputHash: secondPayload.InputHash, Documents: []sharedtextbook.SourceDocument{secondDocument}, Chunks: []sharedtextbook.SourceChunk{secondChunk}}
	secondResultJSON, err := json.Marshal(secondResult)
	require.NoError(t, err)
	var secondResultMap map[string]any
	require.NoError(t, json.Unmarshal(secondResultJSON, &secondResultMap))
	require.NoError(t, repository.PersistTextbookSourceImportResult(ctx, secondTask.ID, secondResultMap))
	library, err = service.ListSourceLibrary(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Len(t, library, 2)
	require.NoError(t, database.Model(&SourceSnapshot{}).Count(&snapshotCount).Error)
	require.Equal(t, int64(2), snapshotCount)

	reparsedPayload := payload
	reparsedPayload.TaskVersion = sharedtextbook.SourceImportTaskVersion
	reparsedPayload.SnapshotID = uuid.NewString()
	reparsedPayload.InputHash = sharedtextbook.SourceImportInputHash(payload.ProjectID, payload.SourceID, reparsedPayload.SnapshotID, payload.Filename, payload.ContentHash, payload.ResolvedVersion)
	require.NotEqual(t, payload.InputHash, reparsedPayload.InputHash)
	reparsedPayloadJSON, err := json.Marshal(reparsedPayload)
	require.NoError(t, err)
	reparsedTask := coretask.JobTask{TaskType: "parse", TaskSubtype: sharedtextbook.TextbookSourceImportTaskSubtype, Status: coretask.JobTaskStatusSucceeded, WorkspaceID: &workspace.ID, PayloadJSON: reparsedPayloadJSON, ResultJSON: []byte(`{}`)}
	require.NoError(t, database.Create(&reparsedTask).Error)
	reparsedDocument := document
	reparsedDocument.ID, reparsedDocument.SnapshotID = "reparsed-document", reparsedPayload.SnapshotID
	reparsedChunk := chunk
	reparsedChunk.ID, reparsedChunk.DocumentID, reparsedChunk.Locator.Symbol = "reparsed-chunk", reparsedDocument.ID, "(*RouterGroup).GET"
	reparsedResult := result
	reparsedResult.SnapshotID, reparsedResult.InputHash = reparsedPayload.SnapshotID, reparsedPayload.InputHash
	reparsedResult.Documents = []sharedtextbook.SourceDocument{reparsedDocument}
	reparsedResult.Chunks = []sharedtextbook.SourceChunk{reparsedChunk}
	reparsedJSON, err := json.Marshal(reparsedResult)
	require.NoError(t, err)
	var reparsedMap map[string]any
	require.NoError(t, json.Unmarshal(reparsedJSON, &reparsedMap))
	require.NoError(t, repository.PersistTextbookSourceImportResult(ctx, reparsedTask.ID, reparsedMap))
	require.NoError(t, repository.PersistTextbookSourceImportResult(ctx, reparsedTask.ID, reparsedMap))
	library, err = service.ListSourceLibrary(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Len(t, library, 3, "new parser identity must not silently reuse the old paragraph snapshot")
	require.NoError(t, database.Model(&SourceSnapshot{}).Count(&snapshotCount).Error)
	require.Equal(t, int64(3), snapshotCount)
	var persisted struct{ Locator []byte }
	require.NoError(t, database.Table("source_chunks").Select("locator").Where("id = ?", reparsedChunk.ID).Take(&persisted).Error)
	var locator sharedtextbook.EvidenceLocator
	require.NoError(t, json.Unmarshal(persisted.Locator, &locator))
	require.Equal(t, "(*RouterGroup).GET", locator.Symbol)
	require.NoError(t, database.Where("id = ?", payload.SnapshotID).First(&snapshot).Error)
	require.Equal(t, strings.TrimPrefix(contentHash, "sha256:"), snapshot.ContentHash)
	var plan []string
	require.NoError(t, database.Raw("EXPLAIN (FORMAT TEXT) SELECT id FROM source_snapshots WHERE id = ? AND source_id = ?", reparsedPayload.SnapshotID, sourceID).Scan(&plan).Error)
	t.Logf("snapshot reconciliation query plan: %v", plan)
	assertSourceEvidenceResolution(t, database, service, workspace.ID, project, document.ID, reparsedChunk.ID)
}

func sourceImportTestDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

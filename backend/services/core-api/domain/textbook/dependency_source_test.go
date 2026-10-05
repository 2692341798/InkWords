package textbook

import (
	"context"
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

func TestDependencySelectionUsesLiveMembershipAndFrozenCandidateEvidence(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	c, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("dependency_test"), postgrescontainer.WithUsername("inkwords"), postgrescontainer.WithPassword("inkwords-test-password"), testcontainers.WithAdditionalWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Terminate(ctx)) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := platformpostgres.InitCore(dsn)
	require.NoError(t, err)
	w, err := platformpostgres.EnsureLocalWorkspace(ctx, db)
	require.NoError(t, err)
	repo := NewGormRepository(db)
	p, err := repo.CreateProject(ctx, CreateProjectInput{WorkspaceID: w.ID, Title: "Dependency selection test", Audience: sharedtextbook.AudienceFoundation, Primary: CreateSourceInput{Kind: sharedtextbook.SourceKindGitRepository, Role: sharedtextbook.SourceRolePrimary, Locator: "https://github.com/gin-gonic/gin"}})
	require.NoError(t, err)
	chapter, err := repo.CreateChapter(ctx, w.ID, CreateChapterInput{ProjectID: p.ID, Title: "Selected chapter", SortOrder: 1, ChapterProfile: sharedtextbook.ChapterProfileHandsOn})
	require.NoError(t, err)
	snapshot := SourceSnapshot{SourceID: *p.PrimarySourceID, ResolvedVersion: strings.Repeat("a", 40), ContentHash: strings.Repeat("b", 64), CapturedAt: time.Now().UTC(), Status: "captured", LimitsJSON: []byte(`{}`)}
	require.NoError(t, db.Create(&snapshot).Error)
	actual, err := repo.GetDependencySourceSnapshot(ctx, w.ID, p.ID, chapter.ID, snapshot.ID)
	require.NoError(t, err)
	selection := sharedtextbook.TeachingDependencySelection{WorkspaceID: w.ID.String(), ProjectID: p.ID.String(), ChapterID: chapter.ID.String(), Snapshot: actual, Module: "github.com/gin-gonic/gin", Version: "v1.12.0", Toolchain: "go1.26.8", DependencyManifestHash: "sha256:" + strings.Repeat("c", 64)}
	pack := sharedtextbook.GenerationEvidencePack{PrimarySnapshot: actual, Evidence: []sharedtextbook.EvidenceRef{{ID: "reference", SnapshotID: actual.ID, DocumentID: "document", Locator: sharedtextbook.EvidenceLocator{Path: "gin.go", StartLine: 1, EndLine: 1}, ContentHash: "sha256:" + strings.Repeat("d", 64), Confidence: sharedtextbook.EvidenceConfidenceDocumented, SourceRole: sharedtextbook.SourceRolePrimary}}, Excerpts: map[string]string{"reference": "package gin"}}
	require.NoError(t, pack.Validate())
	raw, err := json.Marshal(sharedtextbook.SampleGenerationTaskPayload{ProjectID: p.ID.String(), ChapterID: chapter.ID.String(), EvidencePack: pack})
	require.NoError(t, err)
	task := coretask.JobTask{TaskType: "generation", TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, WorkspaceID: &w.ID, TextbookChapterID: &chapter.ID, Status: coretask.JobTaskStatusSucceeded, PayloadJSON: raw, ResultJSON: []byte(`{}`)}
	require.NoError(t, db.Create(&task).Error)
	revision := ChapterRevision{ChapterID: chapter.ID, GenerationTaskID: &task.ID, EvidencePackHash: sharedtextbook.GenerationEvidencePackHash(pack)}
	require.NoError(t, repo.ValidateDependencyCandidateSource(ctx, revision, selection))
	// Registration repeats source/evidence checks inside its transaction, so a
	// caller cannot bypass the application preflight with a forged selection.
	revision.ID, revision.RevisionNumber, revision.Kind, revision.CreatedBy = uuid.New(), 1, sharedtextbook.RevisionKindCandidate, RevisionCreatorGeneration
	revision.Markdown, revision.ContentHash, revision.DocumentJSON = "test-only candidate", strings.Repeat("e", 64), []byte(`{}`)
	blueprint := BlueprintRevision{ID: uuid.New(), ProjectID: p.ID, RevisionNumber: 1, DocumentJSON: []byte(`{}`), ContentHash: strings.Repeat("e", 64), Status: StatusApproved}
	require.NoError(t, db.Create(&blueprint).Error)
	revision.BlueprintRevisionID = &blueprint.ID
	book := BookContractRevision{ID: uuid.New(), ProjectID: p.ID, RevisionNumber: 1, DocumentJSON: []byte(`{}`), ContentHash: strings.Repeat("2", 64), Status: StatusApproved}
	style := StyleSheetRevision{ID: uuid.New(), ProjectID: p.ID, RevisionNumber: 1, DocumentJSON: []byte(`{}`), ContentHash: strings.Repeat("3", 64), Status: StatusApproved}
	require.NoError(t, db.Create(&book).Error)
	require.NoError(t, db.Create(&style).Error)
	revision.BookContractRevisionID, revision.StyleSheetRevisionID = &book.ID, &style.ID
	revision.PromptHash, revision.ProviderName, revision.ModelName = "sha256:"+strings.Repeat("4", 64), "fixture", "no-provider-call"
	revision.ProviderUsageJSON, revision.QualityReportJSON = []byte(`{"known":false}`), []byte(`{"test_fixture":true}`)
	require.NoError(t, db.Create(&revision).Error)
	manifest := sharedtextbook.TeachingArtifactManifest{Format: sharedtextbook.TeachingArtifactManifestFormat, ArtifactID: uuid.NewString(), RevisionID: revision.ID.String(), ArtifactHash: "sha256:" + strings.Repeat("1", 64), BookContractHash: "sha256:" + strings.Repeat("2", 64), StyleSheetHash: "sha256:" + strings.Repeat("3", 64), Language: "go", ToolchainVersion: selection.Toolchain, DependencyManifestHash: selection.DependencyManifestHash, DependencySelection: &selection, Commands: []sharedtextbook.VerificationCommand{{Kind: "go_test"}}}
	register := func() error {
		mh, err := sharedtextbook.TeachingArtifactManifestHash(manifest)
		require.NoError(t, err)
		raw, err := json.Marshal(manifest)
		require.NoError(t, err)
		_, err = repo.RegisterGeneratedCodeArtifact(ctx, w.ID, RegisterGeneratedCodeArtifactInput{RevisionID: revision.ID, ManifestJSON: raw, Artifact: sharedtextbook.CodeArtifact{ID: manifest.ArtifactID, RevisionID: revision.ID.String(), Kind: sharedtextbook.CodeArtifactTeachingImplementation, Language: "go", Entrypoint: "main.go", ArtifactHash: manifest.ArtifactHash, ManifestHash: mh, Status: sharedtextbook.ArtifactStatusUnverified, Limitations: []string{"test-only persistence fixture"}}})
		return err
	}
	selection.ProjectID = uuid.NewString()
	require.ErrorIs(t, register(), ErrInvalidState)
	selection.ProjectID = p.ID.String()
	require.NoError(t, register())
	require.NoError(t, register(), "identical registered selection is idempotent")
	wrong := selection
	wrong.Snapshot.ContentHash = "sha256:" + strings.Repeat("f", 64)
	require.ErrorIs(t, repo.ValidateDependencyCandidateSource(ctx, revision, wrong), ErrInvalidState)
	revision.EvidencePackHash = "sha256:" + strings.Repeat("f", 64)
	require.ErrorIs(t, repo.ValidateDependencyCandidateSource(ctx, revision, selection), ErrInvalidState)
	for _, ids := range [][4]uuid.UUID{{uuid.New(), p.ID, chapter.ID, snapshot.ID}, {w.ID, uuid.New(), chapter.ID, snapshot.ID}, {w.ID, p.ID, uuid.New(), snapshot.ID}, {w.ID, p.ID, chapter.ID, uuid.New()}} {
		_, err := repo.GetDependencySourceSnapshot(ctx, ids[0], ids[1], ids[2], ids[3])
		require.ErrorIs(t, err, ErrNotFound)
	}
	for _, target := range []any{chapter, p, &Source{ID: *p.PrimarySourceID}} {
		tx := db.Begin()
		require.NoError(t, tx.Delete(target).Error)
		_, err := NewGormRepository(tx).GetDependencySourceSnapshot(ctx, w.ID, p.ID, chapter.ID, snapshot.ID)
		require.ErrorIs(t, err, ErrNotFound)
		require.NoError(t, tx.Rollback().Error)
	}
}

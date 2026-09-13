package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// This asset check deliberately complements, rather than replaces, the PostgreSQL integration
// tests required when the goose runner and Testcontainers dependency are introduced.
func TestLocalWorkspaceMigrationIsEmbeddedAndProtectsExistingDataOnDown(t *testing.T) {
	contents, err := Files.ReadFile("00001_local_workspace.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}

	migration := string(contents)
	for _, expected := range []string{
		"-- +goose Up",
		"CREATE TABLE local_workspaces",
		"CREATE TABLE local_workspace_legacy_owner",
		"CHECK (installation_key = 'local-default')",
		"REFERENCES users (id) ON DELETE RESTRICT",
		"-- +goose Down",
		"-- +goose StatementBegin",
		"refusing to roll back local workspace migration after data exists",
		"-- +goose StatementEnd",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("migration is missing required safety contract %q", expected)
		}
	}
}

func TestTextbookTaskWorkspaceIdentityMigrationIsEmbeddedAndFailClosed(t *testing.T) {
	contents, err := Files.ReadFile("00019_textbook_task_workspace_identity.sql")
	require.NoError(t, err)
	migration := string(contents)
	for _, expected := range []string{
		"ALTER TABLE job_tasks",
		"ADD COLUMN workspace_id UUID",
		"REFERENCES local_workspaces (id) ON DELETE RESTRICT",
		"textbook_sample_generate",
		"ck_job_tasks_textbook_workspace",
		"ux_job_tasks_workspace_type_idempotency",
		"refusing to roll back textbook task workspace identity",
	} {
		require.Contains(t, migration, expected)
	}
}

func TestTextbookTaskOptionalLegacyOwnerMigrationPreservesLegacyGuards(t *testing.T) {
	contents, err := Files.ReadFile("00020_textbook_task_optional_legacy_owner.sql")
	require.NoError(t, err)
	migration := string(contents)
	for _, expected := range []string{
		"ALTER COLUMN requested_by DROP NOT NULL",
		"ck_job_tasks_legacy_owner_or_textbook_workspace",
		"textbook_sample_generate",
		"workspace_id IS NOT NULL",
		"refusing to restore required legacy task owners",
		"ALTER COLUMN requested_by SET NOT NULL",
	} {
		require.Contains(t, migration, expected)
	}
}

func TestTextbookCoreMigrationIsEmbeddedWithSourceAndRevisionGuards(t *testing.T) {
	contents, err := Files.ReadFile("00002_textbook_core.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}

	migration := string(contents)
	for _, expected := range []string{
		"-- InkWords migration role: core",
		"CREATE TABLE textbook_projects",
		"CREATE TABLE textbook_sources",
		"CREATE TABLE chapter_revisions",
		"CREATE TABLE chapter_locks",
		"ux_textbook_sources_one_primary_per_project",
		"ux_textbook_chapters_project_sort",
		"idx_textbook_chapters_project_status_sort",
		"role <> 'official_supporting' OR official_confirmed",
		"refusing to roll back textbook core migration after data exists",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("migration is missing required textbook contract %q", expected)
		}
	}
}

func TestSourceDocumentMigrationPreservesChunkProvenance(t *testing.T) {
	contents, err := Files.ReadFile("00003_source_documents.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	migration := string(contents)
	for _, expected := range []string{
		"CREATE TABLE source_documents",
		"CREATE TABLE source_chunks",
		"REFERENCES source_snapshots (id) ON DELETE RESTRICT",
		"UNIQUE (document_id, ordinal)",
		"start_byte >= 0",
		"refusing to roll back source document migration after data exists",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("source document migration is missing required contract %q", expected)
		}
	}
}

func TestChapterRevisionProvenanceMigrationRequiresGeneratedCandidateLineage(t *testing.T) {
	contents, err := Files.ReadFile("00004_chapter_revision_provenance.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	migration := string(contents)
	for _, expected := range []string{
		"ADD COLUMN parent_revision_id",
		"ADD COLUMN book_contract_revision_id",
		"ADD COLUMN style_sheet_revision_id",
		"ADD COLUMN evidence_pack_hash",
		"ADD CONSTRAINT ck_generated_candidate_provenance",
		"refusing to roll back chapter revision provenance migration after data exists",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("chapter revision provenance migration is missing required contract %q", expected)
		}
	}
}

func TestGenerationContractApprovalMigrationPinsProjectApprovalPointers(t *testing.T) {
	contents, err := Files.ReadFile("00005_generation_contract_approval.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	migration := string(contents)
	for _, expected := range []string{
		"ADD COLUMN approved_book_contract_revision_id",
		"ADD COLUMN approved_style_sheet_revision_id",
		"idx_textbook_projects_generation_contracts",
		"refusing to roll back generation contract approval migration after data exists",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("generation contract approval migration is missing required contract %q", expected)
		}
	}
}

func TestChapterBlueprintProvenanceMigrationRequiresCurrentBlueprint(t *testing.T) {
	contents, err := Files.ReadFile("00006_chapter_blueprint_provenance.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	migration := string(contents)
	for _, expected := range []string{
		"ADD COLUMN blueprint_revision_id",
		"ck_generated_candidate_blueprint_provenance",
		"idx_chapter_revisions_blueprint_provenance",
		"refusing to roll back chapter blueprint provenance migration after data exists",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("blueprint provenance migration is missing required contract %q", expected)
		}
	}
}

func TestFirstCandidateProvenanceMigrationAllowsOnlyInitialCandidateWithoutParent(t *testing.T) {
	contents, err := Files.ReadFile("00007_first_candidate_provenance.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	migration := string(contents)
	for _, expected := range []string{
		"DROP CONSTRAINT ck_generated_candidate_provenance",
		"parent_revision_id IS NOT NULL OR revision_number = 1",
		"refusing to roll back first candidate provenance migration after initial generated candidates exist",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("first candidate provenance migration is missing required contract %q", expected)
		}
	}
}

func TestGenerationTaskIdempotencyMigrationPinsOneCandidateToOneTask(t *testing.T) {
	contents, err := Files.ReadFile("00008_chapter_generation_task_idempotency.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	migration := string(contents)
	for _, expected := range []string{
		"ADD COLUMN generation_task_id UUID",
		"CREATE UNIQUE INDEX ux_chapter_revisions_generation_task",
		"refusing to roll back generation task idempotency migration after task-linked candidates exist",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("generation task idempotency migration is missing required contract %q", expected)
		}
	}
}

func TestSourceRetrievalMigrationPreservesExplainableSelection(t *testing.T) {
	contents, err := Files.ReadFile("00009_source_retrieval_runs.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	migration := string(contents)
	for _, expected := range []string{
		"CREATE TABLE source_retrieval_runs",
		"candidates_json JSONB NOT NULL",
		"selected_json JSONB NOT NULL",
		"ux_source_retrieval_runs_project_input_hash",
		"refusing to roll back source retrieval migration after data exists",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("source retrieval migration is missing required contract %q", expected)
		}
	}
}

func TestTextbookRuntimeEvidenceMigrationRequiresImmutableExecutionProvenance(t *testing.T) {
	contents, err := Files.ReadFile("00010_textbook_runtime_evidence.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	migration := string(contents)
	for _, expected := range []string{
		"CREATE TABLE textbook_code_artifacts",
		"CREATE TABLE textbook_runtime_evidence",
		"CREATE TABLE textbook_manuscript_assets",
		"manifest_json JSONB NOT NULL",
		"code_artifact_hash LIKE 'sha256:%'",
		"status <> 'verified' OR",
		"raw_evidence_ref IS NOT NULL",
		"stable_ref TEXT NOT NULL UNIQUE",
		"refusing to roll back textbook runtime evidence after data exists",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("runtime evidence migration is missing required contract %q", expected)
		}
	}
}

func TestTextbookBookBuildMigrationPinsManifestAndApprovedRevisions(t *testing.T) {
	contents, err := Files.ReadFile("00011_textbook_book_builds.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	migration := string(contents)
	for _, expected := range []string{
		"CREATE TABLE textbook_book_builds",
		"approved_revision_ids JSONB NOT NULL",
		"manifest_json JSONB NOT NULL",
		"manifest_hash TEXT NOT NULL CHECK (manifest_hash LIKE 'sha256:%')",
		"status IN ('draft', 'ready_for_review', 'publication_candidate', 'blocked')",
		"refusing to roll back book builds after data exists",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("book build migration is missing required contract %q", expected)
		}
	}
}

func TestCandidateReviewMigrationPreservesHumanRejectionEvidence(t *testing.T) {
	contents, err := Files.ReadFile("00016_textbook_candidate_reviews.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	migration := string(contents)
	for _, expected := range []string{
		"CREATE TABLE textbook_candidate_reviews",
		"candidate_revision_id UUID NOT NULL REFERENCES chapter_revisions",
		"reviewer_workspace_id UUID NOT NULL REFERENCES local_workspaces",
		"decision = 'rejected'",
		"char_length(trim(reason)) BETWEEN 8 AND 2000",
		"UNIQUE (candidate_revision_id)",
		"refusing to roll back candidate reviews after data exists",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("candidate review migration is missing required contract %q", expected)
		}
	}
}

func TestCandidateApprovalReviewMigrationRequiresStructuredHumanEvidence(t *testing.T) {
	contents, err := Files.ReadFile("00026_candidate_approval_review.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	migration := string(contents)
	for _, expected := range []string{
		"decision IN ('approved', 'rejected')",
		"human_review_json JSONB NOT NULL",
		"refusing to roll back candidate approval reviews after evidence exists",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("candidate approval migration is missing required contract %q", expected)
		}
	}
}

func TestTextbookTaskChapterIdentityMigrationBackfillsAndIndexesRecoveryRelation(t *testing.T) {
	contents, err := Files.ReadFile("00034_textbook_task_chapter_identity.sql")
	require.NoError(t, err)
	migration := string(contents)
	for _, expected := range []string{
		"ADD COLUMN textbook_chapter_id UUID REFERENCES textbook_chapters",
		"payload_json ->> 'chapter_id'",
		"job_tasks_textbook_sample_chapter_check",
		"VALIDATE CONSTRAINT job_tasks_textbook_sample_chapter_check",
		"idx_job_tasks_textbook_chapter_created",
		"workspace_id, textbook_chapter_id, created_at DESC, id DESC",
	} {
		require.Contains(t, migration, expected)
	}
}

func TestEditorialEvidenceMigrationPinsHumanAndRightsRecordsToOneBuild(t *testing.T) {
	contents, err := Files.ReadFile("00017_textbook_editorial_evidence.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	migration := string(contents)
	for _, expected := range []string{
		"CREATE TABLE textbook_rights_items",
		"build_id UUID NOT NULL REFERENCES textbook_book_builds",
		"subject_ref TEXT NOT NULL",
		"UNIQUE (build_id, subject_ref)",
		"CREATE TABLE textbook_publication_reviews",
		"automated BOOLEAN NOT NULL DEFAULT FALSE CHECK (automated = FALSE)",
		"UNIQUE (build_id, stage)",
		"refusing to roll back textbook editorial evidence after data exists",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("editorial evidence migration is missing required contract %q", expected)
		}
	}
}

func TestBookBuildInputIdentityMigrationSeparatesStableInputFromTimedManifest(t *testing.T) {
	contents, err := Files.ReadFile("00018_book_build_input_identity.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	migration := string(contents)
	for _, expected := range []string{
		"ADD COLUMN input_hash TEXT",
		"COALESCE(NULLIF(manifest_json ->> 'input_hash', ''), manifest_hash)",
		"ALTER COLUMN input_hash SET NOT NULL",
		"CREATE UNIQUE INDEX ux_textbook_book_builds_project_input",
		"ON textbook_book_builds (project_id, input_hash)",
		"refusing to roll back book build input identity after builds exist",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("book build input identity migration is missing required contract %q", expected)
		}
	}
}

func TestCoreMigrationProviderRejectsNilDatabase(t *testing.T) {
	_, err := NewCoreProvider(nil)
	require.ErrorContains(t, err, "database is nil")
}

func TestRoleScopedMigrationAssetsKeepReviewSchemaOutOfCore(t *testing.T) {
	coreAssets, err := assetsForRole("core")
	require.NoError(t, err)
	reviewAssets, err := assetsForRole("review")
	require.NoError(t, err)

	_, err = fs.ReadFile(coreAssets, "00001_mastery_review.sql")
	require.Error(t, err)
	_, err = fs.ReadFile(coreAssets, "00015_mastery_objective_identity.sql")
	require.Error(t, err)
	contents, err := fs.ReadFile(reviewAssets, "00001_mastery_review.sql")
	require.NoError(t, err)
	require.Contains(t, string(contents), "CREATE TABLE mastery_objectives")
	contents, err = fs.ReadFile(reviewAssets, "00015_mastery_objective_identity.sql")
	require.NoError(t, err)
	require.Contains(t, string(contents), "uq_mastery_objectives_workspace_chapter_active")
	_, err = fs.ReadFile(reviewAssets, "00002_textbook_core.sql")
	require.Error(t, err)
}

func TestReviewSessionWorkspaceMigrationIsAdditiveAndFailClosed(t *testing.T) {
	contents, err := Files.ReadFile("00021_review_session_workspace.sql")
	require.NoError(t, err)
	migration := string(contents)
	for _, expected := range []string{
		"-- InkWords migration role: review",
		"ADD COLUMN workspace_id UUID",
		"ALTER COLUMN user_id DROP NOT NULL",
		"ck_review_sessions_workspace_or_legacy_owner",
		"idx_review_sessions_workspace_note_created",
		"refusing to roll back review workspace identity",
	} {
		require.Contains(t, migration, expected)
	}
}

func TestBlogWorkspaceMigrationPreservesLegacyOwnerAndFailsClosedOnDown(t *testing.T) {
	contents, err := Files.ReadFile("00022_blog_workspace.sql")
	require.NoError(t, err)
	migration := string(contents)
	for _, expected := range []string{
		"ADD COLUMN workspace_id UUID",
		"local_workspace_legacy_owner",
		"ALTER COLUMN workspace_id SET NOT NULL",
		"ALTER COLUMN user_id DROP NOT NULL",
		"idx_blogs_workspace_parent_chapter",
		"refusing to roll back blog workspace identity",
	} {
		require.Contains(t, migration, expected)
	}
}

func TestTaskWorkspaceMigrationPreservesLegacyOwnerAndFailsClosedOnDown(t *testing.T) {
	contents, err := Files.ReadFile("00023_task_workspace.sql")
	require.NoError(t, err)
	migration := string(contents)
	for _, expected := range []string{
		"local_workspace_legacy_owner",
		"ALTER COLUMN workspace_id SET NOT NULL",
		"DROP CONSTRAINT ck_job_tasks_legacy_owner_or_textbook_workspace",
		"requested_by only as historical audit evidence",
		"refusing to roll back task workspace identity",
	} {
		require.Contains(t, migration, expected)
	}
}

func TestLegacyIdentityCleanupMigrationsAreEmbeddedAndIrreversible(t *testing.T) {
	coreContents, err := Files.ReadFile("00024_remove_legacy_identity.sql")
	require.NoError(t, err)
	for _, expected := range []string{
		"-- InkWords migration role: core",
		"tasks lack workspace ownership",
		"blogs lack workspace ownership",
		"OAuth token rows exist",
		"ProjectCourse rows exist",
		"user prompt setting rows exist",
		"DROP COLUMN IF EXISTS user_id",
		"DROP COLUMN IF EXISTS requested_by",
		"DROP TABLE local_workspace_legacy_owner",
		"DROP TABLE users",
		"restore the verified local backup instead",
	} {
		require.Contains(t, string(coreContents), expected)
	}

	reviewContents, err := Files.ReadFile("00025_review_remove_legacy_identity.sql")
	require.NoError(t, err)
	for _, expected := range []string{
		"-- InkWords migration role: review",
		"sessions lack workspace ownership",
		"ALTER COLUMN workspace_id SET NOT NULL",
		"DROP COLUMN IF EXISTS user_id",
		"restore the verified local backup instead",
	} {
		require.Contains(t, string(reviewContents), expected)
	}
}

func TestReviewMigrationsCreateOneLegacyObjectivePerWorkspaceNote(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(
		ctx,
		"postgres:14-alpine",
		postgrescontainer.WithDatabase("inkwords_review_migration_test"),
		postgrescontainer.WithUsername("inkwords"),
		postgrescontainer.WithPassword("inkwords-test-password"),
		testcontainers.WithAdditionalWaitStrategy(
			wait.ForSQL("5432/tcp", "pgx", migrationPostgresURL("inkwords_review_migration_test")).WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })

	database := openMigrationTestDatabase(t, ctx, container)
	defer closeMigrationTestDatabase(t, database)
	_, err = database.ExecContext(ctx, `
CREATE TABLE review_sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    note_path TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);
INSERT INTO review_sessions (id, user_id, note_path)
VALUES ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'wiki/concepts/router.md')`)
	require.NoError(t, err)
	provider, err := NewReviewProvider(database)
	require.NoError(t, err)
	results, err := provider.UpTo(ctx, 21)
	require.NoError(t, err)
	_, err = UpReview(ctx, database)
	require.ErrorContains(t, err, "sessions lack workspace ownership")
	_, err = database.ExecContext(ctx, `
UPDATE review_sessions
SET workspace_id = '22222222-2222-2222-2222-222222222222'
WHERE id = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'`)
	require.NoError(t, err)
	cleanupResults, err := UpReview(ctx, database)
	require.NoError(t, err)
	results = append(results, cleanupResults...)
	require.False(t, postgresColumnExists(t, ctx, database, "review_sessions", "user_id"))
	require.True(t, postgresColumnNotNull(t, ctx, database, "review_sessions", "workspace_id"))
	reviewAssets, err := assetsForRole("review")
	require.NoError(t, err)
	entries, err := fs.ReadDir(reviewAssets, ".")
	require.NoError(t, err)
	require.Len(t, results, len(entries))

	_, err = database.ExecContext(ctx, `
INSERT INTO mastery_objectives (id, workspace_id, chapter_id, title, behavior, required_skills, rubric, key_points, evidence_refs)
VALUES ('11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 'legacy-note:router', '路由', '解释路由', '["explain","retain"]', '["说明概念"]', '["路由"]', '["legacy-note:wiki/concepts/router.md"]')`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
INSERT INTO mastery_objectives (id, workspace_id, chapter_id, title, behavior, required_skills, rubric, key_points, evidence_refs)
VALUES ('33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', 'legacy-note:router', '路由副本', '解释路由', '["explain","retain"]', '["说明概念"]', '["路由"]', '["legacy-note:wiki/concepts/router.md"]')`)
	require.Error(t, err, "one local workspace may only migrate one stable legacy note identity")

	_, err = database.ExecContext(ctx, `
INSERT INTO review_sessions (id, workspace_id, note_path)
VALUES ('cccccccc-cccc-cccc-cccc-cccccccccccc', '22222222-2222-2222-2222-222222222222', 'wiki/concepts/gin.md')`)
	require.NoError(t, err, "new workspace-owned sessions must not require a legacy owner")
	_, err = database.ExecContext(ctx, `
INSERT INTO review_sessions (id, note_path)
VALUES ('dddddddd-dddd-dddd-dddd-dddddddddddd', 'wiki/concepts/ownerless.md')`)
	require.Error(t, err, "sessions without either identity must fail closed")

	_, err = database.ExecContext(ctx, `INSERT INTO mastery_attempts
(id, objective_id, skill, correct, independent, hint_count, took_millis, confidence, error_kinds, attempted_at, answer)
VALUES ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', '11111111-1111-1111-1111-111111111111', 'explain', false, false, 0, 1000, 3, '[]', CURRENT_TIMESTAMP, $1)`, "我的作答：按方法和路径选择处理函数。")
	require.NoError(t, err)
	var answer string
	err = database.QueryRowContext(ctx, `SELECT answer FROM mastery_attempts WHERE id = $1`, "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee").Scan(&answer)
	require.NoError(t, err)
	require.Equal(t, "我的作答：按方法和路径选择处理函数。", answer)
	_, err = database.ExecContext(ctx, `UPDATE mastery_attempts SET answer = repeat('中', 20001) WHERE id = $1`, "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	require.Error(t, err, "answer size is enforced by PostgreSQL as well as the application")
	// v33, v32, v31 and v30 are empty in this legacy-answer fixture.
	_, err = provider.Down(ctx)
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.NoError(t, err)
	// v29 may be removed only without any session evidence.
	_, err = provider.Down(ctx)
	require.NoError(t, err)
	// v28 may be removed only while no frozen task evidence exists; then the
	// older answer-preservation guard must still protect v27.
	_, err = provider.Down(ctx)
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.ErrorContains(t, err, "refusing to roll back mastery answers")
}

// TestCoreMigrationsUsePostgreSQLSemantics instead of SQLite because UUID foreign keys,
// advisory locks, migration tracking and the protected Down path are PostgreSQL contracts.
func TestCoreMigrationsUsePostgreSQLSemantics(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(
		ctx,
		"postgres:14-alpine",
		postgrescontainer.WithDatabase("inkwords_migration_test"),
		postgrescontainer.WithUsername("inkwords"),
		postgrescontainer.WithPassword("inkwords-test-password"),
		testcontainers.WithAdditionalWaitStrategy(
			wait.ForSQL("5432/tcp", "pgx", migrationPostgresURL("inkwords_migration_test")).WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })

	database := openMigrationTestDatabase(t, ctx, container)
	defer closeMigrationTestDatabase(t, database)

	_, err = database.ExecContext(ctx, "CREATE TABLE users (id UUID PRIMARY KEY)")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
CREATE TABLE blogs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id),
    parent_id UUID,
    chapter_sort INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
)`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
CREATE TABLE job_tasks (
    id UUID PRIMARY KEY,
    task_type TEXT NOT NULL,
    task_subtype TEXT NOT NULL,
    requested_by UUID NOT NULL,
    idempotency_key TEXT
)`)
	require.NoError(t, err)

	provider, err := NewCoreProvider(database)
	require.NoError(t, err)
	preWorkspaceResults, err := provider.UpTo(ctx, 21)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "INSERT INTO users (id) VALUES ('11111111-1111-1111-1111-111111111111')")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
INSERT INTO local_workspaces (id, installation_key, display_name)
VALUES ('22222222-2222-2222-2222-222222222222', 'local-default', '迁移测试工作区')`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
INSERT INTO local_workspace_legacy_owner (workspace_id, legacy_user_id)
VALUES ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111')`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
INSERT INTO blogs (id, user_id, chapter_sort)
VALUES ('77777777-7777-7777-7777-777777777778', '11111111-1111-1111-1111-111111111111', 0)`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
INSERT INTO job_tasks (id, task_type, task_subtype, requested_by)
VALUES ('12121212-1212-1212-1212-121212121212', 'generation', 'generate_single', '11111111-1111-1111-1111-111111111111')`)
	require.NoError(t, err)

	workspaceResults, err := provider.UpTo(ctx, 23)
	require.NoError(t, err)
	results := append(preWorkspaceResults, workspaceResults...)
	require.Len(t, results, 20, "every reversible core migration through v23 must run exactly once")
	require.True(t, postgresRelationExists(t, ctx, database, "local_workspaces"))
	require.True(t, postgresRelationExists(t, ctx, database, "local_workspace_legacy_owner"))
	require.True(t, postgresRelationExists(t, ctx, database, "textbook_projects"))
	require.True(t, postgresRelationExists(t, ctx, database, "chapter_revisions"))
	require.True(t, postgresRelationExists(t, ctx, database, "chapter_locks"))
	require.True(t, postgresRelationExists(t, ctx, database, "source_documents"))
	require.True(t, postgresRelationExists(t, ctx, database, "source_chunks"))
	require.True(t, postgresRelationExists(t, ctx, database, "source_retrieval_runs"))
	require.True(t, postgresRelationExists(t, ctx, database, "textbook_code_artifacts"))
	require.True(t, postgresRelationExists(t, ctx, database, "textbook_runtime_evidence"))
	require.True(t, postgresRelationExists(t, ctx, database, "textbook_manuscript_assets"))
	require.True(t, postgresRelationExists(t, ctx, database, "textbook_candidate_reviews"))
	require.True(t, postgresRelationExists(t, ctx, database, "textbook_rights_items"))
	require.True(t, postgresRelationExists(t, ctx, database, "textbook_publication_reviews"))
	require.True(t, postgresRelationExists(t, ctx, database, CoreVersionTable))
	var migratedWorkspaceID, preservedUserID string
	require.NoError(t, database.QueryRowContext(ctx, `
SELECT workspace_id::text, user_id::text
FROM blogs
WHERE id = '77777777-7777-7777-7777-777777777778'`).Scan(&migratedWorkspaceID, &preservedUserID))
	require.Equal(t, "22222222-2222-2222-2222-222222222222", migratedWorkspaceID)
	require.Equal(t, "11111111-1111-1111-1111-111111111111", preservedUserID)
	var taskWorkspaceID, preservedRequestedBy string
	require.NoError(t, database.QueryRowContext(ctx, `
SELECT workspace_id::text, requested_by::text
FROM job_tasks
WHERE id = '12121212-1212-1212-1212-121212121212'`).Scan(&taskWorkspaceID, &preservedRequestedBy))
	require.Equal(t, "22222222-2222-2222-2222-222222222222", taskWorkspaceID)
	require.Equal(t, "11111111-1111-1111-1111-111111111111", preservedRequestedBy)
	require.Contains(t, postgresExplain(t, ctx, database, `
SELECT id FROM textbook_book_builds
WHERE project_id = '33333333-3333-3333-3333-333333333333'
  AND input_hash = 'sha256:stable-freeze-input'`), "ux_textbook_book_builds_project_input")
	require.Contains(t, postgresExplain(t, ctx, database, `
SELECT id FROM job_tasks
WHERE workspace_id = '22222222-2222-2222-2222-222222222222'
  AND task_type = 'generation'
  AND idempotency_key = 'textbook-sample:stable-input'`), "ux_job_tasks_workspace_type_idempotency")

	results, err = provider.UpTo(ctx, 23)
	require.NoError(t, err)
	require.Empty(t, results, "running Up twice must not reapply a migration")

	_, err = database.ExecContext(ctx, `
INSERT INTO job_tasks (id, task_type, task_subtype, workspace_id)
VALUES ('23232323-2323-2323-2323-232323232323', 'generation', 'textbook_sample_generate', '22222222-2222-2222-2222-222222222222')`)
	require.NoError(t, err, "new textbook tasks may be owned only by the local workspace")
	_, err = database.ExecContext(ctx, `
INSERT INTO job_tasks (id, task_type, task_subtype, workspace_id)
VALUES ('24242424-2424-2424-2424-242424242424', 'generation', 'generate_single', '22222222-2222-2222-2222-222222222222')`)
	require.NoError(t, err, "new compatibility tasks may be owned only by the local workspace")
	_, err = database.ExecContext(ctx, "DELETE FROM job_tasks WHERE id IN ('23232323-2323-2323-2323-232323232323', '24242424-2424-2424-2424-242424242424')")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
INSERT INTO textbook_projects (id, workspace_id, title, audience_level)
VALUES ('33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', '迁移测试教材', 'foundation')`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
INSERT INTO textbook_sources (id, project_id, kind, role, locator)
VALUES ('44444444-4444-4444-4444-444444444444', '33333333-3333-3333-3333-333333333333', 'git_repository', 'primary', 'https://github.com/gin-gonic/gin')`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
INSERT INTO source_snapshots (id, source_id, resolved_version, content_hash, status)
VALUES ('45454545-4545-4545-4545-454545454545', '44444444-4444-4444-4444-444444444444', '0123456789012345678901234567890123456789', repeat('a', 64), 'captured')`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
INSERT INTO source_documents (id, snapshot_id, canonical_locator, title, media_type, content_hash)
VALUES ('document-readme', '45454545-4545-4545-4545-454545454545', 'README.md', 'README', 'text/markdown', 'sha256:readme')`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
INSERT INTO source_chunks (id, document_id, ordinal, locator, text_hash, search_text)
VALUES ('chunk-readme-1', 'document-readme', 1, '{"path":"README.md","start_line":1,"end_line":1}', 'sha256:chunk', 'Gin documentation')`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
INSERT INTO textbook_sources (id, project_id, kind, role, locator)
VALUES ('55555555-5555-5555-5555-555555555555', '33333333-3333-3333-3333-333333333333', 'git_repository', 'primary', 'https://github.com/gin-gonic/gin-fork')`)
	require.Error(t, err, "a project may have only one active primary source")
	_, err = database.ExecContext(ctx, `
INSERT INTO textbook_sources (id, project_id, kind, role, locator, official_confirmed)
VALUES ('66666666-6666-6666-6666-666666666666', '33333333-3333-3333-3333-333333333333', 'official_web', 'official_supporting', 'https://go.dev', FALSE)`)
	require.Error(t, err, "supporting sources require official confirmation")
	_, err = database.ExecContext(ctx, `
INSERT INTO textbook_chapters (id, project_id, sort_order, title, chapter_profile)
VALUES ('77777777-7777-7777-7777-777777777777', '33333333-3333-3333-3333-333333333333', 1, '第一章', 'concept')`)
	require.NoError(t, err)
	require.Contains(t, postgresExplain(t, ctx, database, `
SELECT id FROM textbook_chapters
WHERE project_id = '33333333-3333-3333-3333-333333333333'
  AND status = 'draft'
  AND deleted_at IS NULL
ORDER BY sort_order`), "idx_textbook_chapters_project_status_sort")

	_, err = database.ExecContext(ctx, `
INSERT INTO source_retrieval_runs (id, project_id, query, candidates_json, selected_json, input_hash)
VALUES ('88888888-8888-8888-8888-888888888888', '33333333-3333-3333-3333-333333333333', 'Gin route', '[]', '[]', 'sha256:retrieval')`)
	require.NoError(t, err)
	for _, name := range []string{"task-workspace-identity", "blog-workspace-identity", "textbook-task-optional-legacy-owner"} {
		_, err = DownCore(ctx, database)
		require.NoErrorf(t, err, "empty %s migration is reversibly removable", name)
	}
	_, err = database.ExecContext(ctx, `
UPDATE job_tasks
SET workspace_id = NULL
WHERE id = '12121212-1212-1212-1212-121212121212'`)
	require.NoError(t, err, "restore the pre-v19 task shape before testing its protected rollback")
	_, err = DownCore(ctx, database)
	require.NoError(t, err, "empty textbook-task-workspace-identity migration is reversibly removable")
	for _, name := range []string{"book-build-input-identity", "editorial-evidence", "candidate-reviews", "visual-evidence-purpose", "book-build-tool-versions", "book-builds", "runtime-evidence"} {
		_, err = DownCore(ctx, database)
		require.NoErrorf(t, err, "empty %s migration is reversibly removable", name)
	}
	_, err = DownCore(ctx, database)
	require.ErrorContains(t, err, "refusing to roll back source retrieval migration after data exists")
	_, err = database.ExecContext(ctx, "DELETE FROM source_retrieval_runs")
	require.NoError(t, err)
	_, err = DownCore(ctx, database)
	require.NoError(t, err, "an empty source retrieval migration is reversibly removable")
	_, err = DownCore(ctx, database)
	require.NoError(t, err, "an empty generation-task idempotency migration is reversibly removable")
	_, err = DownCore(ctx, database)
	require.NoError(t, err, "an empty first-candidate provenance migration is reversibly removable")
	_, err = DownCore(ctx, database)
	require.NoError(t, err, "an empty blueprint provenance migration is reversibly removable")
	_, err = DownCore(ctx, database)
	require.NoError(t, err, "an empty generation-contract migration is reversibly removable")
	_, err = DownCore(ctx, database)
	require.NoError(t, err, "an empty revision provenance migration is reversibly removable")
	_, err = DownCore(ctx, database)
	require.ErrorContains(t, err, "refusing to roll back source document migration after data exists")

	_, err = database.ExecContext(ctx, "DELETE FROM source_chunks")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM source_documents")
	require.NoError(t, err)
	_, err = DownCore(ctx, database)
	require.NoError(t, err)
	require.False(t, postgresRelationExists(t, ctx, database, "source_documents"))
	_, err = DownCore(ctx, database)
	require.ErrorContains(t, err, "refusing to roll back textbook core migration after data exists")
	_, err = database.ExecContext(ctx, "DELETE FROM textbook_chapters")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM source_snapshots")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM textbook_sources")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM textbook_projects")
	require.NoError(t, err)
	_, err = DownCore(ctx, database)
	require.NoError(t, err)
	require.False(t, postgresRelationExists(t, ctx, database, "textbook_projects"))

	_, err = DownCore(ctx, database)
	require.ErrorContains(t, err, "refusing to roll back local workspace migration after data exists")
	_, err = database.ExecContext(ctx, "DELETE FROM local_workspace_legacy_owner")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM local_workspaces")
	require.NoError(t, err)
	_, err = DownCore(ctx, database)
	require.NoError(t, err)
	require.False(t, postgresRelationExists(t, ctx, database, "local_workspaces"))
}

func TestCoreLegacyIdentityCleanupFailsClosedThenPreservesWorkspaceData(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(
		ctx,
		"postgres:14-alpine",
		postgrescontainer.WithDatabase("inkwords_cleanup_test"),
		postgrescontainer.WithUsername("inkwords"),
		postgrescontainer.WithPassword("inkwords-test-password"),
		testcontainers.WithAdditionalWaitStrategy(
			wait.ForSQL("5432/tcp", "pgx", migrationPostgresURL("inkwords_cleanup_test")).WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })

	database := openMigrationTestDatabase(t, ctx, container)
	defer closeMigrationTestDatabase(t, database)
	_, err = database.ExecContext(ctx, `
CREATE TABLE users (id UUID PRIMARY KEY);
CREATE TABLE blogs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id),
    parent_id UUID,
    chapter_sort INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);
CREATE TABLE job_tasks (
    id UUID PRIMARY KEY,
    task_type TEXT NOT NULL,
    task_subtype TEXT NOT NULL,
    requested_by UUID NOT NULL,
    idempotency_key TEXT
);
CREATE TABLE o_auth_tokens (id UUID PRIMARY KEY);
CREATE TABLE project_courses (id UUID PRIMARY KEY);
CREATE TABLE user_prompt_settings (user_id UUID PRIMARY KEY);
`)
	require.NoError(t, err)

	provider, err := NewCoreProvider(database)
	require.NoError(t, err)
	_, err = provider.UpTo(ctx, 21)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `
INSERT INTO users (id) VALUES ('11111111-1111-1111-1111-111111111111');
INSERT INTO local_workspaces (id, installation_key, display_name)
VALUES ('22222222-2222-2222-2222-222222222222', 'local-default', '清理测试工作区');
INSERT INTO local_workspace_legacy_owner (workspace_id, legacy_user_id)
VALUES ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111');
INSERT INTO blogs (id, user_id, chapter_sort)
VALUES ('33333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 0);
INSERT INTO job_tasks (id, task_type, task_subtype, requested_by)
VALUES ('44444444-4444-4444-4444-444444444444', 'generation', 'generate_single', '11111111-1111-1111-1111-111111111111');
INSERT INTO o_auth_tokens (id) VALUES ('55555555-5555-5555-5555-555555555555');
`)
	require.NoError(t, err)

	// This fixture exercises migration 24's destructive cleanup boundary.
	// Later independent migrations must not change its rollback target.
	_, err = provider.UpTo(ctx, 24)
	require.ErrorContains(t, err, "OAuth token rows exist")
	require.True(t, postgresRelationExists(t, ctx, database, "users"))
	require.NoError(t, execSQL(ctx, database, "DELETE FROM o_auth_tokens"))

	results, err := provider.UpTo(ctx, 24)
	require.NoError(t, err)
	require.Len(t, results, 1, "the failed cleanup must remain pending until its precondition is resolved")
	for _, retiredTable := range []string{"users", "local_workspace_legacy_owner", "o_auth_tokens", "project_courses", "user_prompt_settings"} {
		require.False(t, postgresRelationExists(t, ctx, database, retiredTable), retiredTable)
	}
	require.False(t, postgresColumnExists(t, ctx, database, "blogs", "user_id"))
	require.False(t, postgresColumnExists(t, ctx, database, "job_tasks", "requested_by"))
	require.True(t, postgresColumnNotNull(t, ctx, database, "blogs", "workspace_id"))
	require.True(t, postgresColumnNotNull(t, ctx, database, "job_tasks", "workspace_id"))
	var blogCount, taskCount int
	require.NoError(t, database.QueryRowContext(ctx, "SELECT COUNT(*) FROM blogs").Scan(&blogCount))
	require.NoError(t, database.QueryRowContext(ctx, "SELECT COUNT(*) FROM job_tasks").Scan(&taskCount))
	require.Equal(t, 1, blogCount)
	require.Equal(t, 1, taskCount)

	_, err = DownCore(ctx, database)
	require.ErrorContains(t, err, "legacy identity cleanup is irreversible")
}

func openMigrationTestDatabase(t *testing.T, ctx context.Context, container *postgrescontainer.PostgresContainer) *sql.DB {
	t.Helper()
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	database, err := gormDB.DB()
	require.NoError(t, err)
	require.NoError(t, database.PingContext(ctx))
	return database
}

func closeMigrationTestDatabase(t *testing.T, database *sql.DB) {
	t.Helper()
	require.NoError(t, database.Close())
}

func postgresRelationExists(t *testing.T, ctx context.Context, database *sql.DB, name string) bool {
	t.Helper()
	var relation sql.NullString
	require.NoError(t, database.QueryRowContext(ctx, "SELECT to_regclass('public.' || $1)", name).Scan(&relation))
	return relation.Valid
}

func postgresColumnExists(t *testing.T, ctx context.Context, database *sql.DB, tableName, columnName string) bool {
	t.Helper()
	var exists bool
	require.NoError(t, database.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
)`, tableName, columnName).Scan(&exists))
	return exists
}

func postgresColumnNotNull(t *testing.T, ctx context.Context, database *sql.DB, tableName, columnName string) bool {
	t.Helper()
	var nullable string
	require.NoError(t, database.QueryRowContext(ctx, `
SELECT is_nullable FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
`, tableName, columnName).Scan(&nullable))
	return nullable == "NO"
}

func execSQL(ctx context.Context, database *sql.DB, statement string) error {
	_, err := database.ExecContext(ctx, statement)
	return err
}

func postgresExplain(t *testing.T, ctx context.Context, database *sql.DB, query string) string {
	t.Helper()
	_, err := database.ExecContext(ctx, "SET enable_seqscan = off")
	require.NoError(t, err)
	rows, err := database.QueryContext(ctx, "EXPLAIN (COSTS OFF) "+query)
	require.NoError(t, err)
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		lines = append(lines, line)
	}
	require.NoError(t, rows.Err())
	return strings.Join(lines, "\n")
}

func migrationPostgresURL(database string) func(string, network.Port) string {
	return func(host string, port network.Port) string {
		return fmt.Sprintf(
			"postgres://inkwords:inkwords-test-password@%s:%s/%s?sslmode=disable",
			host,
			port.Port(),
			database,
		)
	}
}

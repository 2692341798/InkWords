package export

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	platformpostgres "inkwords-backend/shared/platform/postgres"
)

func TestGormRepositoryGetApprovedTextbookChapterFollowsApprovedPointerAndWorkspace(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine",
		postgrescontainer.WithDatabase("textbook_export_test"),
		postgrescontainer.WithUsername("inkwords"),
		postgrescontainer.WithPassword("inkwords-test-password"),
		testcontainers.WithAdditionalWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	database, err := platformpostgres.InitCore(dsn)
	require.NoError(t, err)
	workspace, err := platformpostgres.EnsureLocalWorkspace(ctx, database)
	require.NoError(t, err)

	projectID, chapterID, candidateRevisionID, approvedRevisionID, artifactID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	bookContractID, styleSheetID, blueprintID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, database.Exec(
		`INSERT INTO textbook_projects (id, workspace_id, title, audience_level, status, revision_version)
		 VALUES (?, ?, ?, 'foundation', 'approved', 1)`, projectID, workspace.ID, "Gin 自学教材",
	).Error)
	require.NoError(t, database.Exec("INSERT INTO book_contract_revisions (id, project_id, revision_number, document_json, content_hash, status) VALUES (?, ?, 1, '{}'::jsonb, ?, 'approved')", bookContractID, projectID, strings.Repeat("b", 64)).Error)
	require.NoError(t, database.Exec("INSERT INTO style_sheet_revisions (id, project_id, revision_number, document_json, content_hash, status) VALUES (?, ?, 1, '{}'::jsonb, ?, 'approved')", styleSheetID, projectID, strings.Repeat("b", 64)).Error)
	require.NoError(t, database.Exec("INSERT INTO blueprint_revisions (id, project_id, revision_number, document_json, content_hash, status) VALUES (?, ?, 1, '{}'::jsonb, ?, 'approved')", blueprintID, projectID, strings.Repeat("b", 64)).Error)
	require.NoError(t, database.Exec("UPDATE textbook_projects SET approved_book_contract_revision_id = ?, approved_style_sheet_revision_id = ?, approved_blueprint_revision_id = ? WHERE id = ?", bookContractID, styleSheetID, blueprintID, projectID).Error)
	require.NoError(t, database.Exec(
		`INSERT INTO textbook_chapters (id, project_id, sort_order, title, chapter_profile, status, revision_version)
		 VALUES (?, ?, 1, ?, 'concept', 'approved', 2)`, chapterID, projectID, "请求生命周期",
	).Error)
	require.NoError(t, database.Exec(
		`INSERT INTO chapter_revisions (id, chapter_id, revision_number, kind, markdown, document_json, content_hash, created_by,
			book_contract_revision_id, style_sheet_revision_id, blueprint_revision_id, evidence_pack_hash, prompt_hash, provider_name, model_name, provider_usage_json, quality_report_json)
		 VALUES (?, ?, 1, 'candidate', ?, '{}'::jsonb, ?, 'generation', ?, ?, ?, 'sha256:evidence', 'sha256:prompt', 'fake', 'fixture', '{}'::jsonb, '{}'::jsonb)`,
		candidateRevisionID, chapterID, "# 候选稿", strings.Repeat("c", 64), bookContractID, styleSheetID, blueprintID,
	).Error)
	require.NoError(t, database.Exec(
		`INSERT INTO chapter_revisions (id, chapter_id, revision_number, kind, markdown, document_json, content_hash, created_by, parent_revision_id,
			book_contract_revision_id, style_sheet_revision_id, blueprint_revision_id, evidence_pack_hash, prompt_hash, provider_name, model_name, provider_usage_json, quality_report_json)
		 VALUES (?, ?, 2, 'approved', ?, '{}'::jsonb, ?, 'manual', ?, ?, ?, ?, 'sha256:evidence', 'sha256:prompt', 'fake', 'fixture', '{}'::jsonb, '{}'::jsonb)`,
		approvedRevisionID, chapterID, "# 批准稿\n\n这是可导出的内容。", strings.Repeat("a", 64), candidateRevisionID, bookContractID, styleSheetID, blueprintID,
	).Error)
	require.NoError(t, database.Exec(
		`INSERT INTO textbook_code_artifacts (id, revision_id, kind, language, entrypoint, manifest_json, manifest_hash, artifact_hash, limitations_json, status)
		 VALUES (?, ?, 'teaching_implementation', 'go', 'main.go', '{}'::jsonb, 'sha256:manifest', 'sha256:artifact', '["教学用途"]'::jsonb, 'unverified')`, artifactID, candidateRevisionID,
	).Error)
	require.NoError(t, database.Exec(
		"UPDATE textbook_chapters SET current_revision_id = ?, approved_revision_id = ? WHERE id = ?", approvedRevisionID, approvedRevisionID, chapterID,
	).Error)

	repo := NewGormRepository(database)
	exported, err := repo.GetApprovedTextbookChapter(ctx, workspace.ID, chapterID)
	require.NoError(t, err)
	require.Equal(t, chapterID, exported.ChapterID)
	require.Equal(t, approvedRevisionID, exported.RevisionID)
	require.Equal(t, "# 批准稿\n\n这是可导出的内容。", exported.Markdown)
	require.Len(t, exported.CodeArtifacts, 1, "an applied candidate's teaching artifact must remain in the approved chapter package")
	require.Equal(t, artifactID, exported.CodeArtifacts[0].ID)

	_, err = repo.GetApprovedTextbookChapter(ctx, uuid.New(), chapterID)
	require.ErrorIs(t, err, ErrTextbookChapterNotFound)

	unapprovedChapterID := uuid.New()
	require.NoError(t, database.Exec(
		`INSERT INTO textbook_chapters (id, project_id, sort_order, title, chapter_profile, status, revision_version)
		 VALUES (?, ?, 2, ?, 'concept', 'draft', 0)`, unapprovedChapterID, projectID, "未批准章节",
	).Error)
	_, err = repo.GetApprovedTextbookChapter(ctx, workspace.ID, unapprovedChapterID)
	require.ErrorIs(t, err, ErrTextbookChapterNotApproved)
	testExportRightsLedger(t, database, projectID, bookContractID, styleSheetID)
	testExportHumanReviewHistory(t, database, workspace.ID, projectID, bookContractID, styleSheetID)
}

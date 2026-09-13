package migrations

import (
	"context"
	"testing"
	"testing/fstest"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestHumanReviewMigrationPreservesLegacyAndRefusesEvidenceLoss(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine",
		postgrescontainer.WithDatabase("inkwords_migration_test"), postgrescontainer.WithUsername("inkwords"), postgrescontainer.WithPassword("inkwords-test-password"),
		testcontainers.WithAdditionalWaitStrategy(wait.ForSQL("5432/tcp", "pgx", migrationPostgresURL("inkwords_migration_test")).WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })
	db := openMigrationTestDatabase(t, ctx, container)
	defer closeMigrationTestDatabase(t, db)
	_, err = db.ExecContext(ctx, "CREATE TABLE textbook_projects (id UUID PRIMARY KEY); CREATE TABLE textbook_book_builds (id UUID PRIMARY KEY)")
	require.NoError(t, err)
	assets := fstest.MapFS{}
	for _, name := range []string{"00017_textbook_editorial_evidence.sql", "00037_human_publication_review_revisions.sql"} {
		contents, readErr := Files.ReadFile(name)
		require.NoError(t, readErr)
		assets[name] = &fstest.MapFile{Data: contents}
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, assets, goose.WithDisableGlobalRegistry(true))
	require.NoError(t, err)
	_, err = provider.UpTo(ctx, 17)
	require.NoError(t, err)
	const build = "11111111-1111-1111-1111-111111111111"
	const old = "22222222-2222-2222-2222-222222222222"
	_, err = db.ExecContext(ctx, "INSERT INTO textbook_book_builds VALUES ($1)", build)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO textbook_publication_reviews (id,build_id,stage,reviewer,notes,completed_at) VALUES ($1,$2,'rights','迁移测试夹具','迁移前的原始说明不可改写。','2026-09-01T00:00:00Z')", old, build)
	require.NoError(t, err)
	var before string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT row_to_json(r)::text FROM textbook_publication_reviews r WHERE id=$1", old).Scan(&before))
	_, err = provider.Up(ctx)
	require.NoError(t, err)
	var after string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT row_to_json(r)::text FROM (SELECT id,build_id,stage,reviewer,notes,automated,completed_at,created_at FROM textbook_publication_reviews WHERE id=$1) r", old).Scan(&after))
	require.JSONEq(t, before, after)
	var unassessed bool
	require.NoError(t, db.QueryRowContext(ctx, "SELECT contract_version IS NULL AND verdict IS NULL AND score IS NULL AND revision=1 FROM textbook_publication_reviews WHERE id=$1", old).Scan(&unassessed))
	require.True(t, unassessed)
	_, err = provider.Down(ctx)
	require.NoError(t, err, "legacy-only migration can roll back without losing notes")
	_, err = provider.Up(ctx)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO textbook_publication_reviews
		(id,build_id,stage,reviewer,notes,completed_at,contract_version,manifest_hash,revision,reviewer_kind,verdict,score,scope,evidence_refs,hard_failures,input_hash)
		VALUES ('33333333-3333-3333-3333-333333333333',$1,'rights','隔离测试夹具','这是隔离迁移测试的复审说明。',CURRENT_TIMESTAMP,
		'inkwords.human-publication-review.v2','sha256:'||repeat('a',64),2,'human','pass',3,'隔离迁移测试的显式复审范围。','["test:fixture"]','[]','sha256:'||repeat('b',64))`, build)
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.ErrorContains(t, err, "refusing to discard explicit human review evidence")
	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM textbook_publication_reviews").Scan(&count))
	require.Equal(t, 2, count)
}

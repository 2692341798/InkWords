package task

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	shared "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/postgres/migrations"
)

func TestVerificationAttemptMigrationAndConcurrentClaimsPostgres(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("attempt_test"), postgrescontainer.WithUsername("inkwords"), postgrescontainer.WithPassword("inkwords-test-password"), testcontainers.WithAdditionalWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()
	// This isolated database reconstructs the pre-migration task table. Only the
	// two foreign-key parents are minimal fixtures, never production manuscripts.
	require.NoError(t, db.AutoMigrate(&JobTask{}))
	require.NoError(t, db.Exec(`ALTER TABLE job_tasks DROP COLUMN verification_artifact_id, DROP COLUMN verification_attempt, DROP COLUMN previous_verification_task_id, DROP COLUMN verification_worker_token, DROP COLUMN verification_worker_released_at, DROP COLUMN verification_release_kind;
 CREATE TABLE local_workspaces(id UUID PRIMARY KEY);
 CREATE TABLE textbook_code_artifacts(id UUID PRIMARY KEY);`).Error)
	workspace, artifact := uuid.New(), uuid.New()
	require.NoError(t, db.Exec("INSERT INTO local_workspaces VALUES (?)", workspace).Error)
	oldTasks := []uuid.UUID{}
	var firstPayload shared.ArtifactVerificationRequest
	for i, status := range []string{"cancelled", "succeeded", "failed"} {
		art := artifact
		if i > 0 {
			art = uuid.New()
		}
		require.NoError(t, db.Exec("INSERT INTO textbook_code_artifacts VALUES (?)", art).Error)
		payload := shared.ArtifactVerificationRequest{ArtifactID: art.String(), RevisionID: uuid.NewString(), ArtifactHash: "sha256:" + strings.Repeat("a", 64), ManifestHash: "sha256:" + strings.Repeat("b", 64)}
		if i == 0 {
			firstPayload = payload
		}
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		id := uuid.New()
		oldTasks = append(oldTasks, id)
		require.NoError(t, db.Exec(`INSERT INTO job_tasks(id,task_type,task_subtype,status,workspace_id,payload_json,result_json,started_at,finished_at)
   VALUES (?,'verification',?,?,?,?,'{"preserved":true}','2026-09-01T00:00:00Z','2026-09-01T00:01:00Z')`, id, shared.TextbookTeachingArtifactVerifyTaskSubtype, status, workspace, datatypes.JSON(raw)).Error)
	}
	var before string
	require.NoError(t, sqlDB.QueryRowContext(ctx, "SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM job_tasks t").Scan(&before))
	contents, err := migrations.Files.ReadFile("00038_teaching_verification_attempts.sql")
	require.NoError(t, err)
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, fstest.MapFS{"00038_teaching_verification_attempts.sql": &fstest.MapFile{Data: contents}}, goose.WithDisableGlobalRegistry(true))
	require.NoError(t, err)
	_, err = provider.Up(ctx)
	require.NoError(t, err)
	var after string
	require.NoError(t, sqlDB.QueryRowContext(ctx, `SELECT jsonb_agg(to_jsonb(t)-ARRAY['verification_artifact_id','verification_attempt','previous_verification_task_id','verification_worker_token','verification_worker_released_at','verification_release_kind'] ORDER BY id)::text FROM job_tasks t`).Scan(&after))
	require.JSONEq(t, before, after)
	repo := NewGormRepository(db)
	legacy, err := repo.GetByID(ctx, oldTasks[0])
	require.NoError(t, err)
	require.Nil(t, legacy.VerificationWorkerReleasedAt)
	require.NotNil(t, legacy.VerificationWorkerToken)
	for _, id := range oldTasks[1:] {
		row, err := repo.GetByID(ctx, id)
		require.NoError(t, err)
		require.NotNil(t, row.VerificationWorkerReleasedAt)
		require.Equal(t, "legacy_terminal", row.VerificationReleaseKind)
	}
	input := CreateVerificationAttemptInput{RequestID: uuid.New(), WorkspaceID: workspace, ExpectedPreviousTaskID: &legacy.ID, Payload: firstPayload}
	_, _, err = repo.CreateVerificationAttempt(ctx, input)
	require.ErrorIs(t, err, ErrVerificationExecutionActive)
	// Synthetic operator receipt in this isolated fixture; no human evidence is created.
	require.NoError(t, db.Model(&JobTask{}).Where("id = ?", legacy.ID).Updates(map[string]any{"verification_worker_released_at": time.Now().UTC(), "verification_release_kind": "operator_observed_exit"}).Error)
	type creation struct {
		row     JobTask
		created bool
		err     error
	}
	results := make(chan creation, 8)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			row, created, err := repo.CreateVerificationAttempt(ctx, input)
			results <- creation{row, created, err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	createdCount := 0
	for result := range results {
		require.NoError(t, result.err)
		require.Equal(t, input.RequestID, result.row.ID)
		require.Equal(t, 2, result.row.VerificationAttempt)
		if result.created {
			createdCount++
		}
	}
	require.Equal(t, 1, createdCount)
	type claim struct {
		token uuid.UUID
		err   error
	}
	claims := make(chan claim, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := repo.ClaimVerificationWorker(ctx, input.RequestID)
			claims <- claim{token, err}
		}()
	}
	wg.Wait()
	close(claims)
	claimCount := 0
	var token uuid.UUID
	for result := range claims {
		if result.err == nil {
			claimCount++
			token = result.token
		} else {
			require.ErrorIs(t, result.err, ErrVerificationExecutionActive)
		}
	}
	require.Equal(t, 1, claimCount)
	require.NoError(t, repo.UpdateStatus(ctx, input.RequestID, JobTaskStatusSucceeded, ""))
	next := input
	next.RequestID = uuid.New()
	next.ExpectedPreviousTaskID = &input.RequestID
	_, _, err = repo.CreateVerificationAttempt(ctx, next)
	require.ErrorIs(t, err, ErrVerificationExecutionActive)
	require.NoError(t, repo.ReleaseVerificationWorker(ctx, input.RequestID, token))
	released, err := repo.GetByID(ctx, input.RequestID)
	require.NoError(t, err)
	require.NoError(t, repo.ReleaseVerificationWorker(ctx, input.RequestID, token))
	repeated, err := repo.GetByID(ctx, input.RequestID)
	require.NoError(t, err)
	require.Equal(t, released.VerificationWorkerReleasedAt, repeated.VerificationWorkerReleasedAt)
	third, created, err := repo.CreateVerificationAttempt(ctx, next)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, 3, third.VerificationAttempt)
	next.RequestID = uuid.New()
	_, _, err = repo.CreateVerificationAttempt(ctx, next)
	require.ErrorIs(t, err, ErrVerificationAttemptConflict)
	history, err := repo.ListVerificationAttempts(ctx, workspace, artifact)
	require.NoError(t, err)
	require.Len(t, history, 3)
	require.Equal(t, JobTaskStatusCancelled, history[2].Status)
	require.JSONEq(t, `{"preserved":true}`, string(history[2].ResultJSON))
	_, err = provider.Down(ctx)
	require.ErrorContains(t, err, "refusing to discard verification attempts")
}

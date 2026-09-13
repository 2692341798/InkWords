package bootstrap

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbook "inkwords-backend/services/core-api/domain/textbook"
	verification "inkwords-backend/services/course-runner/domain/textbookverification"
	shared "inkwords-backend/shared/kernel/textbook"
	"strings"
	"testing"
	"time"
)

func TestTextbookCompletionAndCancellationSerializeInPostgres(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("cancel_test"), postgrescontainer.WithUsername("inkwords"), postgrescontainer.WithPassword("inkwords-test-password"), testcontainers.WithAdditionalWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()
	// Isolated database only; production continues using its versioned migrations.
	require.NoError(t, db.AutoMigrate(&coretask.JobTask{}, &textbook.CodeArtifactRow{}, &textbook.RuntimeEvidenceRow{}))
	for _, cancelFirst := range []bool{true, false} {
		name := "completion first"
		if cancelFirst {
			name = "cancellation first"
		}
		t.Run(name, func(t *testing.T) {
			ctx, finish := context.WithTimeout(t.Context(), 10*time.Second)
			defer finish()
			taskID, artifactID, revisionID := uuid.New(), uuid.New(), uuid.New()
			hash := "sha256:" + strings.Repeat("a", 64)
			require.NoError(t, db.Create(&coretask.JobTask{ID: taskID, TaskType: "verification", TaskSubtype: verification.TaskSubtype, Status: coretask.JobTaskStatusRunning}).Error)
			require.NoError(t, db.Create(&textbook.CodeArtifactRow{ID: artifactID, RevisionID: revisionID, Kind: shared.CodeArtifactTeachingImplementation, Language: "go", ManifestJSON: []byte(`{}`), ManifestHash: hash, ArtifactHash: hash, LimitationsJSON: []byte(`[]`), Status: shared.ArtifactStatusUnverified}).Error)
			payload := verification.VerificationPayload{ArtifactID: artifactID.String(), RevisionID: revisionID.String(), ArtifactHash: hash, ManifestHash: hash}
			report := verification.Report{Status: shared.ArtifactStatusVerified, InputHash: hash, ArtifactHash: hash, RunnerImageDigest: hash, ToolchainVersion: "go1.26.8", Results: []verification.CommandResult{{Command: shared.VerificationCommand{Kind: "go_test"}, Status: shared.ArtifactStatusVerified, Output: "isolated fixture"}}}
			store := textbookEvidenceStore{db: db}
			result := make(chan error, 1)
			if cancelFirst {
				tx := db.WithContext(ctx).Begin()
				require.NoError(t, tx.Error)
				var row coretask.JobTask
				require.NoError(t, tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", taskID).Error)
				go func() { result <- store.PersistVerificationReport(ctx, taskID, payload, report) }()
				select {
				case err := <-result:
					t.Fatalf("completion bypassed row lock: %v", err)
				case <-time.After(150 * time.Millisecond):
				}
				require.NoError(t, coretask.NewGormRepository(tx).UpdateStatus(ctx, taskID, coretask.JobTaskStatusCancelled, ""))
				require.NoError(t, tx.Commit().Error)
				require.ErrorIs(t, <-result, verification.ErrVerificationCancelled)
			} else {
				entered, release := make(chan struct{}), make(chan struct{})
				require.NoError(t, db.Callback().Create().Before("gorm:create").Register("test:hold-evidence", func(tx *gorm.DB) {
					if tx.Statement.Table == "textbook_runtime_evidence" {
						close(entered)
						select {
						case <-release:
						case <-ctx.Done():
						}
					}
				}))
				go func() { result <- store.PersistVerificationReport(ctx, taskID, payload, report) }()
				select {
				case <-entered:
				case <-ctx.Done():
					t.Fatal("evidence transaction did not begin")
				}
				cancelDone := make(chan error, 1)
				go func() {
					cancelDone <- coretask.NewGormRepository(db).UpdateStatus(ctx, taskID, coretask.JobTaskStatusCancelled, "")
				}()
				select {
				case err := <-cancelDone:
					t.Fatalf("cancellation bypassed completion lock: %v", err)
				case <-time.After(150 * time.Millisecond):
				}
				close(release)
				require.NoError(t, <-result)
				require.NoError(t, <-cancelDone)
				require.NoError(t, db.Callback().Create().Remove("test:hold-evidence"))
			}
			var task coretask.JobTask
			require.NoError(t, db.First(&task, "id = ?", taskID).Error)
			var count int64
			require.NoError(t, db.Model(&textbook.RuntimeEvidenceRow{}).Where("code_artifact_id = ?", artifactID).Count(&count).Error)
			var artifact textbook.CodeArtifactRow
			require.NoError(t, db.First(&artifact, "id = ?", artifactID).Error)
			if cancelFirst {
				require.Equal(t, coretask.JobTaskStatusCancelled, task.Status)
				require.Zero(t, count)
				require.Equal(t, shared.ArtifactStatusUnverified, artifact.Status)
			} else {
				require.Equal(t, coretask.JobTaskStatusSucceeded, task.Status)
				require.EqualValues(t, 1, count)
				require.Equal(t, shared.ArtifactStatusVerified, artifact.Status)
			}
		})
	}
}

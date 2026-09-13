package mastery

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	platformpostgres "inkwords-backend/shared/platform/postgres"
	"inkwords-backend/shared/platform/postgres/migrations"
)

func TestLearnerFilesCommitWithOriginalAnswerAndCannotBeReplacedOnRetry(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("learner_test"), postgrescontainer.WithUsername("inkwords"), postgrescontainer.WithPassword("test-only-password"), testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := platformpostgres.InitReview(dsn)
	require.NoError(t, err)
	store, err := NewGormStore(db)
	require.NoError(t, err)
	basis := practiceBasisFixture()
	service := NewService(store).WithPracticeSource(fixedPracticeSource{basis})
	now := time.Now().UTC()
	service.now = func() time.Time { return now }
	objective, err := service.CreateObjective(ctx, basis.WorkspaceID, ObjectiveInput{ChapterID: "approved-revision:" + basis.Learning.ChapterID + ":" + basis.Learning.RevisionID})
	require.NoError(t, err)
	session, err := service.BeginPracticeSession(ctx, basis.WorkspaceID, objective.ID, Reproduce)
	require.NoError(t, err)
	now = now.Add(time.Minute)
	input := Attempt{Skill: Reproduce, PracticeTaskID: session.PracticeTaskID, PracticeContentHash: session.PracticeContentHash, PracticeSessionID: session.ID, Answer: "我先复现最小路由实现。", Correct: false, Independent: true, Confidence: 3, LearnerFiles: []sharedtextbook.LearnerCodeFile{{Path: "router_test.go", Content: "package main\r\n"}, {Path: "router.go", Content: "package main\n// learner implementation\n"}}}
	result, err := service.RecordAttempt(ctx, basis.WorkspaceID, objective.ID, input)
	require.NoError(t, err)
	records, err := store.ListAttempts(ctx, objective.ID)
	require.NoError(t, err)
	require.Len(t, records, 1)
	artifact, err := service.GetLearnerArtifact(ctx, basis.WorkspaceID, objective.ID, records[0].ID)
	require.NoError(t, err)
	require.NotNil(t, artifact)
	require.Equal(t, records[0].LearnerArtifactHash, artifact.SnapshotHash)
	require.True(t, artifact.SubmittedAt.Equal(records[0].AttemptedAt))
	require.Equal(t, input.LearnerFiles[0].Content, artifact.Files[1].Content)
	identity := sharedtextbook.LearnerRunnerIdentity{ImageDigest: sharedtextbook.PracticeExcerptHash("fixed runner fixture"), ToolchainVersion: "go1.25.4", SandboxProfileDigest: sharedtextbook.LearnerSandboxProfileDigest}
	verificationPlan, err := service.PrepareLearnerVerificationPlan(ctx, basis.WorkspaceID, objective.ID, records[0].ID, identity)
	require.NoError(t, err)
	require.Equal(t, artifact.SnapshotHash, verificationPlan.SnapshotHash)
	require.NoError(t, verificationPlan.ValidateFor(*artifact, basis.Learning))
	_, err = service.PrepareLearnerVerificationPlan(ctx, uuid.New(), objective.ID, records[0].ID, identity)
	require.Error(t, err, "planning does not bypass workspace ownership")
	_, err = service.PrepareLearnerVerificationPlan(ctx, basis.WorkspaceID, objective.ID, uuid.New(), identity)
	require.Error(t, err)
	unchanged, err := store.ListAttempts(ctx, objective.ID)
	require.NoError(t, err)
	require.Equal(t, records, unchanged, "planning must not rewrite or append learner evidence")
	now = now.Add(time.Hour)
	input.LearnerFiles[0], input.LearnerFiles[1] = input.LearnerFiles[1], input.LearnerFiles[0]
	repeated, err := service.RecordAttempt(ctx, basis.WorkspaceID, objective.ID, input)
	require.NoError(t, err)
	require.Equal(t, result, repeated)
	input.LearnerFiles[0].Content += "// another version\n"
	_, err = service.RecordAttempt(ctx, basis.WorkspaceID, objective.ID, input)
	require.ErrorIs(t, err, ErrPracticeSessionClosed)
	input.LearnerFiles = nil
	_, err = service.RecordAttempt(ctx, basis.WorkspaceID, objective.ID, input)
	require.ErrorIs(t, err, ErrPracticeSessionClosed, "retry cannot omit code from saved submission")
	_, err = service.GetLearnerArtifact(ctx, uuid.New(), objective.ID, records[0].ID)
	require.Error(t, err)
	restored, err := service.GetLearnerArtifact(ctx, basis.WorkspaceID, objective.ID, records[0].ID)
	require.NoError(t, err)
	require.Equal(t, artifact, restored)
	var count int64
	require.NoError(t, db.Table("mastery_learner_artifacts").Count(&count).Error)
	require.EqualValues(t, 1, count)
	var plan string
	require.NoError(t, db.Raw("EXPLAIN (FORMAT JSON) SELECT snapshot_json FROM mastery_learner_artifacts WHERE attempt_id = ?", records[0].ID).Row().Scan(&plan))
	require.Contains(t, plan, "mastery_learner_artifacts_pkey")
	t.Log("learner snapshot read:", plan)
	// A snapshot insert failure must roll back the new attempt, schedule and session.
	other, err := service.BeginPracticeSession(ctx, basis.WorkspaceID, objective.ID, Complete)
	require.NoError(t, err)
	input.Skill, input.PracticeSessionID, input.PracticeTaskID = Complete, other.ID, other.PracticeTaskID
	input.LearnerFiles = []sharedtextbook.LearnerCodeFile{{Path: "fail.go", Content: "package main"}}
	var before Schedule
	require.NoError(t, db.First(&before, "objective_id = ?", objective.ID).Error)
	require.NoError(t, db.Exec(`CREATE FUNCTION reject_learner_snapshot() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'fixture rejects snapshot'; END $$ LANGUAGE plpgsql; CREATE TRIGGER reject_learner_snapshot BEFORE INSERT ON mastery_learner_artifacts FOR EACH ROW EXECUTE FUNCTION reject_learner_snapshot();`).Error)
	_, err = service.RecordAttempt(ctx, basis.WorkspaceID, objective.ID, input)
	require.Error(t, err)
	current, err := store.ListAttempts(ctx, objective.ID)
	require.NoError(t, err)
	require.Len(t, current, 1)
	var after Schedule
	require.NoError(t, db.First(&after, "objective_id = ?", objective.ID).Error)
	require.Equal(t, before, after)
	open, err := service.LoadPracticeSession(ctx, basis.WorkspaceID, objective.ID, other.ID)
	require.NoError(t, err)
	require.Nil(t, open.SubmittedAt)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	provider, err := migrations.NewReviewProvider(sqlDB)
	require.NoError(t, err)
	_, err = provider.Down(ctx) // v33 has no verification runs in this snapshot fixture.
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.ErrorContains(t, err, "learner code snapshots")
	encoded, err := json.Marshal(artifact)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), `"verified"`)
}

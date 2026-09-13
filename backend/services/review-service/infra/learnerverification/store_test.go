package learnerverification

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
	app "inkwords-backend/services/review-service/app/masteryverification"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	platformpostgres "inkwords-backend/shared/platform/postgres"
	"inkwords-backend/shared/platform/postgres/migrations"
)

func TestPostgresStoreConsumesCapabilityOnceAndPreservesEvidence(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("learner_verification_test"), postgrescontainer.WithUsername("inkwords"), postgrescontainer.WithPassword("test-only-password"), testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := platformpostgres.InitReview(dsn)
	require.NoError(t, err)
	owner, objective, attempt := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO mastery_objectives(id,workspace_id,chapter_id,title,behavior,required_skills,rubric,key_points,evidence_refs) VALUES(?,?,?,'复现','复现','["reproduce"]','["正确"]','["路由"]','["evidence-1"]')`, objective, owner, "fixture-chapter").Error)
	require.NoError(t, db.Exec(`INSERT INTO mastery_attempts(id,objective_id,skill,answer,correct,independent,hint_count,took_millis,confidence,error_kinds,attempted_at) VALUES(?,?,'reproduce','代码见快照',false,true,0,1000,3,'[]',CURRENT_TIMESTAMP)`, attempt, objective).Error)
	input := verificationInputFixture(t, owner, objective, attempt)
	preview := app.Preview{Capability: sharedtextbook.LearnerVerificationCapability{Format: sharedtextbook.LearnerVerificationCapabilityFormat, Accepted: true, Available: true, Profile: sharedtextbook.LearnerGoTestProfile, Runner: &input.Plan.Runner}, InputHash: input.Plan.InputHash, SnapshotHash: input.Plan.SnapshotHash, FilesHash: input.Plan.FilesHash, ExecutionHash: input.Plan.ExecutionFilesHash, DerivedFiles: input.Plan.DerivedFiles, RequiresExplicit: true}
	job := app.Job{ID: uuid.New(), WorkspaceID: owner, ObjectiveID: objective, AttemptID: attempt, RequestID: uuid.New(), Status: "queued", Preview: preview, Input: input, CreatedAt: time.Now().UTC()}
	store := NewStore(db)
	saved, created, err := store.Create(ctx, job, digest('f'))
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, job.ID, saved.ID)
	reference := sharedtextbook.LearnerVerificationReference{Format: sharedtextbook.LearnerVerificationReferenceFormat, RunID: job.ID.String(), WorkspaceID: owner.String(), ObjectiveID: objective.String(), AttemptID: attempt.String(), InputHash: input.Plan.InputHash, ClaimToken: strings.Repeat("A", 43)}
	_, err = store.Resolve(ctx, reference, digest('e'))
	require.ErrorIs(t, err, app.ErrNotFound)
	resolved, err := store.Resolve(ctx, reference, digest('f'))
	require.NoError(t, err)
	require.Equal(t, input, resolved)
	_, err = store.Resolve(ctx, reference, digest('f'))
	require.ErrorIs(t, err, app.ErrNotFound, "the claim token is consumed by the first resolve")
	now, zero := time.Now().UTC(), 0
	report := sharedtextbook.LearnerVerificationReport{Format: sharedtextbook.LearnerVerificationReportFormat, RunID: job.ID.String(), InputHash: input.Plan.InputHash, SnapshotHash: input.Plan.SnapshotHash, ExecutionTreeHash: digest('d'), Runner: input.Plan.Runner, Profile: input.Plan.Profile, Policy: input.Plan.Policy, Status: sharedtextbook.LearnerVerificationPassed, ExecutionStarted: true, StartedAt: &now, CompletedAt: now, ExitCode: &zero}
	require.ErrorIs(t, store.Finish(ctx, job.ID, report, "failed", ""), app.ErrConflict)
	require.NoError(t, store.Finish(ctx, job.ID, report, "passed", ""))
	current, err := store.Read(ctx, owner, objective, job.ID)
	require.NoError(t, err)
	require.Equal(t, "passed", current.Status)
	require.Equal(t, &report, current.Report)
	verificationService := app.NewService(nil, nil, store)
	assessmentInput, assessmentReport, err := verificationService.LatestAssessmentEvidence(ctx, owner, objective, attempt)
	require.NoError(t, err)
	require.Equal(t, input, *assessmentInput)
	require.Equal(t, report, *assessmentReport)
	encoded, err := json.Marshal(current)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), reference.ClaimToken)
	require.NotContains(t, string(encoded), "package learner")
	var plan string
	require.NoError(t, db.Raw("EXPLAIN (FORMAT JSON) SELECT id FROM mastery_learner_verification_runs WHERE workspace_id=? AND objective_id=? AND attempt_id=? ORDER BY created_at DESC,id DESC LIMIT 1", owner, objective, attempt).Row().Scan(&plan))
	require.Contains(t, plan, "idx_mastery_learner_verification_attempt")
	sqlDB, err := db.DB()
	require.NoError(t, err)
	provider, err := migrations.NewReviewProvider(sqlDB)
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.ErrorContains(t, err, "learner verification evidence")
}

func verificationInputFixture(t *testing.T, owner, objective, attempt uuid.UUID) sharedtextbook.LearnerVerificationInput {
	t.Helper()
	modes := []sharedtextbook.LearningTaskMode{sharedtextbook.LearningTaskExplain, sharedtextbook.LearningTaskComplete, sharedtextbook.LearningTaskReproduce, sharedtextbook.LearningTaskTransfer, sharedtextbook.LearningTaskDiagnose, sharedtextbook.LearningTaskRetain}
	set := sharedtextbook.PracticeSet{Version: sharedtextbook.PracticeSetVersion}
	for _, mode := range modes {
		task := sharedtextbook.PracticeTask{ID: string(mode), Mode: mode, Prompt: "题目 " + string(mode), Variation: "变式", ExpectedAnswer: "答案", EvidenceIDs: []string{"evidence-1"}, Hints: []sharedtextbook.PracticeHint{{Level: 1, Text: "提示一"}, {Level: 2, Text: "提示二"}, {Level: 3, Text: "提示三"}}, Rubric: sharedtextbook.PracticeRubricDimensions(mode)}
		for index := range task.Rubric {
			task.Rubric[index].Description = "核对 " + task.Rubric[index].ID
		}
		if mode == sharedtextbook.LearningTaskRetain {
			task.MinDelayHours = 24
		}
		set.Tasks = append(set.Tasks, task)
	}
	arc := sharedtextbook.DefaultLearningArc()
	for index := range arc.Stages {
		arc.Stages[index].Objective = "复现"
		arc.Stages[index].SuccessEvidence = []string{"evidence-1"}
		arc.Stages[index].RecoveryRoute = "回看"
	}
	chapter, revision := uuid.NewString(), uuid.NewString()
	projection := sharedtextbook.LearningProjection{Format: "inkwords.learning-projection.v2", RevisionID: revision, ChapterID: chapter, ContentHash: digest('a'), LearningArc: arc, PracticeSet: &set, EvidenceIDs: []string{"evidence-1"}, Objectives: []sharedtextbook.LearningObjective{{ID: "objective", ChapterID: chapter, Text: "复现", RequiredModes: []sharedtextbook.LearningTaskMode{sharedtextbook.LearningTaskReproduce}}}}
	artifact := sharedtextbook.LearnerArtifact{Format: sharedtextbook.LearnerArtifactFormat, WorkspaceID: owner.String(), ObjectiveID: objective.String(), AttemptID: attempt.String(), SessionID: uuid.NewString(), RevisionID: revision, TaskID: string(sharedtextbook.LearningTaskReproduce), PracticeContentHash: projection.ContentHash, Skill: sharedtextbook.LearningTaskReproduce, SubmittedAt: time.Now().UTC(), Files: []sharedtextbook.LearnerCodeFile{{Path: "main.go", Content: "package learner\n"}}}
	var err error
	artifact.SnapshotHash, err = sharedtextbook.LearnerArtifactHash(artifact)
	require.NoError(t, err)
	runner := sharedtextbook.LearnerRunnerIdentity{ImageDigest: digest('b'), ToolchainVersion: "go1.25.4", SandboxProfileDigest: sharedtextbook.LearnerSandboxProfileDigest}
	plan, err := sharedtextbook.NewLearnerVerificationPlan(artifact, projection, runner)
	require.NoError(t, err)
	return sharedtextbook.LearnerVerificationInput{Format: sharedtextbook.LearnerVerificationInputFormat, Plan: plan, Artifact: artifact, Task: set.Tasks[2]}
}

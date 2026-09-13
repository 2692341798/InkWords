package textbook

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func learnerVerificationFixture(t *testing.T) (LearnerArtifact, LearningProjection, LearnerRunnerIdentity) {
	t.Helper()
	set := practiceSetFixture()
	revision, chapter := uuid.NewString(), uuid.NewString()
	arc := DefaultLearningArc()
	for i := range arc.Stages {
		arc.Stages[i].Objective = "解释路由"
		arc.Stages[i].SuccessEvidence = []string{"evidence-1"}
		arc.Stages[i].RecoveryRoute = "回到方法与路径"
	}
	projection := LearningProjection{Format: "inkwords.learning-projection.v2", RevisionID: revision, ChapterID: chapter, ContentHash: PracticeExcerptHash("approved practice"), LearningArc: arc, PracticeSet: &set, EvidenceIDs: []string{"evidence-1"}, Objectives: []LearningObjective{{ID: "objective-1", ChapterID: chapter, Text: "复现路由", RequiredModes: []LearningTaskMode{LearningTaskReproduce}}}}
	artifact := LearnerArtifact{Format: LearnerArtifactFormat, WorkspaceID: uuid.NewString(), ObjectiveID: uuid.NewString(), AttemptID: uuid.NewString(), SessionID: uuid.NewString(), RevisionID: revision, TaskID: set.Tasks[2].ID, PracticeContentHash: projection.ContentHash, Skill: LearningTaskReproduce, SubmittedAt: time.Unix(100, 0).UTC(), Files: []LearnerCodeFile{{Path: "router_test.go", Content: "package main\r\n// 未执行的学习者文件\r\n"}, {Path: "router.go", Content: "package main\n"}}}
	var err error
	artifact.SnapshotHash, err = LearnerArtifactHash(artifact)
	require.NoError(t, err)
	return artifact, projection, LearnerRunnerIdentity{ImageDigest: "sha256:" + strings.Repeat("a", 64), ToolchainVersion: "go1.25.4", SandboxProfileDigest: LearnerSandboxProfileDigest}
}

func TestLearnerVerificationPlanBindsSavedFilesTaskAndRunnerWithoutExecuting(t *testing.T) {
	artifact, projection, runner := learnerVerificationFixture(t)
	plan, err := NewLearnerVerificationPlan(artifact, projection, runner)
	require.NoError(t, err)
	require.NoError(t, plan.ValidateFor(artifact, projection))
	require.Equal(t, artifact.SnapshotHash, plan.SnapshotHash)
	require.Equal(t, LearnerGoTestProfile, plan.Profile)
	require.NotEmpty(t, plan.InputHash)
	// Selection order is not a different source set; contents and environment are.
	artifact.Files[0], artifact.Files[1] = artifact.Files[1], artifact.Files[0]
	repeat, err := NewLearnerVerificationPlan(artifact, projection, runner)
	require.NoError(t, err)
	require.Equal(t, plan, repeat)
	runner.ToolchainVersion = "go1.25.5"
	changed, err := NewLearnerVerificationPlan(artifact, projection, runner)
	require.NoError(t, err)
	require.NotEqual(t, plan.InputHash, changed.InputHash)
	changed.Runner = plan.Runner
	require.Error(t, changed.ValidateFor(artifact, projection), "editing runtime fields invalidates the frozen plan")
}

func TestLearnerVerificationPlanRejectsRebindingAndMisleadingManifests(t *testing.T) {
	artifact, projection, runner := learnerVerificationFixture(t)
	plan, err := NewLearnerVerificationPlan(artifact, projection, runner)
	require.NoError(t, err)
	for _, alter := range []func(*LearnerVerificationPlan){
		func(p *LearnerVerificationPlan) { p.Format = TeachingArtifactManifestFormat },
		func(p *LearnerVerificationPlan) { p.Profile = "arbitrary-shell" },
		func(p *LearnerVerificationPlan) { p.AttemptID = uuid.NewString() },
		func(p *LearnerVerificationPlan) { p.FilesHash = PracticeExcerptHash("reference implementation") },
		func(p *LearnerVerificationPlan) { p.TaskHash = PracticeExcerptHash("different rubric") },
		func(p *LearnerVerificationPlan) { p.Policy.Command = []string{"sh", "-c", "go test"} },
	} {
		changed := plan
		alter(&changed)
		require.Error(t, changed.ValidateFor(artifact, projection))
	}
	projection.PracticeSet.Tasks[2].Variation += "新需求"
	require.Error(t, plan.ValidateFor(artifact, projection), "the exact task is frozen, even if a bad adapter reuses its content hash")
	artifact.Files[0].Content += "// changed"
	_, err = NewLearnerVerificationPlan(artifact, projection, runner)
	require.Error(t, err)
}

func TestLearnerVerificationPlanNeedsApprovedBoundPracticeAndKnownRunner(t *testing.T) {
	for _, alter := range []func(*LearnerArtifact, *LearningProjection, *LearnerRunnerIdentity){
		func(a *LearnerArtifact, p *LearningProjection, r *LearnerRunnerIdentity) {
			r.ImageDigest = "local:latest"
		},
		func(a *LearnerArtifact, p *LearningProjection, r *LearnerRunnerIdentity) {
			r.ToolchainVersion = "go1.25.4; curl"
		},
		func(a *LearnerArtifact, p *LearningProjection, r *LearnerRunnerIdentity) {
			r.SandboxProfileDigest = "sha256:" + strings.Repeat("b", 64)
		},
		func(a *LearnerArtifact, p *LearningProjection, r *LearnerRunnerIdentity) {
			p.RevisionID = uuid.NewString()
		},
		func(a *LearnerArtifact, p *LearningProjection, r *LearnerRunnerIdentity) {
			p.ContentHash = PracticeExcerptHash("changed")
		},
		func(a *LearnerArtifact, p *LearningProjection, r *LearnerRunnerIdentity) {
			p.Format = "inkwords.learning-projection.v1"
			p.PracticeSet = nil
		},
		func(a *LearnerArtifact, p *LearningProjection, r *LearnerRunnerIdentity) {
			a.TaskID = "nonexistent"
			a.SnapshotHash, _ = LearnerArtifactHash(*a)
		},
	} {
		artifact, projection, runner := learnerVerificationFixture(t)
		alter(&artifact, &projection, &runner)
		_, err := NewLearnerVerificationPlan(artifact, projection, runner)
		require.Error(t, err)
	}
}

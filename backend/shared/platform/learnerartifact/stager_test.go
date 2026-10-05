package learnerartifact

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestStageBuildsAndRemovesOnlyTheBoundEphemeralTree(t *testing.T) {
	artifact, plan := learnerStageFixture(t)
	workspace, err := Stage(context.Background(), artifact, plan)
	require.NoError(t, err)
	require.NotEmpty(t, workspace.TreeHash)
	require.FileExists(t, filepath.Join(workspace.RootDir, "main.go"))
	derived, err := os.ReadFile(filepath.Join(workspace.RootDir, "go.mod"))
	require.NoError(t, err)
	require.Equal(t, plan.DerivedFiles[0].Content, string(derived))
	require.NoError(t, workspace.Cleanup())
	require.NoDirExists(t, workspace.RootDir)
}

func TestStageRejectsChangedDerivedOrSourceBytes(t *testing.T) {
	artifact, plan := learnerStageFixture(t)
	plan.DerivedFiles[0].Content += "replace example.com/evil => /host\n"
	_, err := Stage(context.Background(), artifact, plan)
	require.ErrorContains(t, err, "execution files mismatch")

	artifact, plan = learnerStageFixture(t)
	artifact.Files[0].Content += "// changed"
	_, err = Stage(context.Background(), artifact, plan)
	require.ErrorContains(t, err, "frozen snapshot mismatch")
}

func learnerStageFixture(t *testing.T) (sharedtextbook.LearnerArtifact, sharedtextbook.LearnerVerificationPlan) {
	t.Helper()
	artifact := sharedtextbook.LearnerArtifact{Format: sharedtextbook.LearnerArtifactFormat, WorkspaceID: "11111111-1111-1111-1111-111111111111", ObjectiveID: "22222222-2222-2222-2222-222222222222", AttemptID: "33333333-3333-3333-3333-333333333333", SessionID: "44444444-4444-4444-4444-444444444444", RevisionID: "55555555-5555-5555-5555-555555555555", TaskID: "task", PracticeContentHash: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Skill: sharedtextbook.LearningTaskReproduce, SubmittedAt: time.Unix(100, 0).UTC(), Files: []sharedtextbook.LearnerCodeFile{{Path: "main.go", Content: "package learner\n"}}}
	var err error
	artifact.SnapshotHash, err = sharedtextbook.LearnerArtifactHash(artifact)
	require.NoError(t, err)
	derived := []sharedtextbook.LearnerCodeFile{{Path: "go.mod", Content: "module inkwords.local/learner\n\ngo 1.25\n"}}
	executionHash, err := sharedtextbook.LearnerFilesHash(append(append([]sharedtextbook.LearnerCodeFile(nil), artifact.Files...), derived...))
	require.NoError(t, err)
	return artifact, sharedtextbook.LearnerVerificationPlan{SnapshotHash: artifact.SnapshotHash, DerivedFiles: derived, ExecutionFilesHash: executionHash}
}

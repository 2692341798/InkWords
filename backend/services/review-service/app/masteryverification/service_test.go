package masteryverification

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type fakeCapabilities struct {
	value sharedtextbook.LearnerVerificationCapability
}

func (source fakeCapabilities) Capability(context.Context) (sharedtextbook.LearnerVerificationCapability, error) {
	return source.value, nil
}
func (source fakeCapabilities) Execute(context.Context, sharedtextbook.LearnerVerificationReference) (sharedtextbook.LearnerVerificationReport, error) {
	return sharedtextbook.LearnerVerificationReport{}, nil
}

type countingInputs struct{ calls int }

func (source *countingInputs) PrepareLearnerVerificationInput(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, sharedtextbook.LearnerRunnerIdentity) (sharedtextbook.LearnerVerificationInput, error) {
	source.calls++
	return sharedtextbook.LearnerVerificationInput{Plan: sharedtextbook.LearnerVerificationPlan{InputHash: digest('b'), SnapshotHash: digest('c'), FilesHash: digest('d'), ExecutionFilesHash: digest('e'), DerivedFiles: []sharedtextbook.LearnerCodeFile{{Path: "go.mod", Content: "module inkwords.local/learner\n"}}}}, nil
}

func TestPreviewDoesNotResolveOrStageFilesWhenSandboxIsUnavailable(t *testing.T) {
	inputs := &countingInputs{}
	capability := sharedtextbook.LearnerVerificationCapability{Format: sharedtextbook.LearnerVerificationCapabilityFormat, Accepted: true, Profile: sharedtextbook.LearnerGoTestProfile, Reason: "namespace preflight failed"}
	preview, err := NewService(fakeCapabilities{value: capability}, inputs).Preview(context.Background(), uuid.New(), uuid.New(), uuid.New())
	require.NoError(t, err)
	require.False(t, preview.Capability.Available)
	require.True(t, preview.RequiresExplicit)
	require.Zero(t, inputs.calls)
	require.Empty(t, preview.InputHash)
}

func TestPreviewBindsDerivedFilesOnlyAfterTrustedCapability(t *testing.T) {
	inputs := &countingInputs{}
	runner := &sharedtextbook.LearnerRunnerIdentity{ImageDigest: digest('a'), ToolchainVersion: "go1.25.4", SandboxProfileDigest: sharedtextbook.LearnerSandboxProfileDigest}
	capability := sharedtextbook.LearnerVerificationCapability{Format: sharedtextbook.LearnerVerificationCapabilityFormat, Accepted: true, Available: true, Profile: sharedtextbook.LearnerGoTestProfile, Runner: runner}
	preview, err := NewService(fakeCapabilities{value: capability}, inputs).Preview(context.Background(), uuid.New(), uuid.New(), uuid.New())
	require.NoError(t, err)
	require.Equal(t, 1, inputs.calls)
	require.NotEmpty(t, preview.InputHash)
	require.Equal(t, "go.mod", preview.DerivedFiles[0].Path)
}

func digest(character byte) string { return "sha256:" + strings.Repeat(string(character), 64) }

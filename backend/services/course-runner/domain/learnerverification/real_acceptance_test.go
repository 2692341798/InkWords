package learnerverification

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	verification "inkwords-backend/services/course-runner/domain/verification"
	shared "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/learnerartifact"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type realAcceptanceStager struct{}

func (realAcceptanceStager) Stage(ctx context.Context, a shared.LearnerArtifact, p shared.LearnerVerificationPlan) (Workspace, error) {
	w, err := learnerartifact.Stage(ctx, a, p)
	return Workspace{RootDir: w.RootDir, TreeHash: w.TreeHash, Cleanup: w.Cleanup}, err
}

// TestRealLearnerRuntimeAcceptance uses synthetic operator-owned records, the
// actual stager, and the unchanged fixed Bubblewrap executor. No HTTP resolver,
// production learning store, provider, or host execution fallback is present.
func TestRealLearnerRuntimeAcceptance(t *testing.T) {
	if os.Getenv("INKWORDS_REAL_LEARNER_RUNTIME") != "approved" {
		t.Skip("real learner sandbox acceptance requires opt-in")
	}
	require.Equal(t, "linux", runtime.GOOS)
	dir := os.Getenv("INKWORDS_RUNTIME_ASSESSMENT_DIR")
	require.True(t, filepath.IsAbs(dir))
	executor := verification.BubblewrapExecutor{Binary: "/usr/bin/bwrap"}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	require.NoError(t, executor.Preflight(ctx))
	require.NoError(t, executor.PreflightLearnerGoTest(ctx))
	for _, name := range []string{"correct-method-selection", "wrong-method-selection"} {
		var input shared.LearnerVerificationInput
		var ref shared.LearnerVerificationReference
		for suffix, value := range map[string]any{"input": &input, "reference": &ref} {
			data, err := os.ReadFile(filepath.Join(dir, name+"-"+suffix+".json"))
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(data, value))
		}
		require.Equal(t, runtime.Version(), input.Plan.Runner.ToolchainVersion)
		require.Equal(t, os.Getenv("INKWORDS_EXPECTED_RUNNER_IMAGE"), input.Plan.Runner.ImageDigest)
		report := (Runner{Resolver: fakeResolver{input: input}, Stager: realAcceptanceStager{}, Executor: &executor, Identity: input.Plan.Runner}).Verify(ctx, ref)
		data, err := json.MarshalIndent(report, "", "  ")
		require.NoError(t, err)
		file, err := os.OpenFile(filepath.Join(dir, name+"-report.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		require.NoError(t, err)
		_, err = file.Write(data)
		require.NoError(t, err)
		require.NoError(t, file.Close())
		require.NoError(t, report.ValidateFor(ref, input.Plan))
		require.True(t, report.ExecutionStarted, report.Reason)
		expected := shared.LearnerVerificationPassed
		if name == "wrong-method-selection" {
			expected = shared.LearnerVerificationFailed
		}
		require.Equal(t, expected, report.Status, report.Reason)
		t.Logf("case=%s run_id=%s status=%s exit_code=%d", name, report.RunID, report.Status, *report.ExitCode)
	}
}

package verification

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

type fakeExecutor struct{ commands []string }

func (f *fakeExecutor) Execute(_ context.Context, _ string, command string, _ time.Duration, env []string) (int, string, error) {
	f.commands = append(f.commands, command)
	if env != nil {
		return 1, "environment leaked", nil
	}
	return 0, "ok", nil
}

func validManifest() sharedkernel.LabManifest {
	return sharedkernel.LabManifest{Language: "go", ToolchainVersion: "1.25", AllowedCommands: []string{"test", "test -run TestCheckpoint"}, ResourceLimits: map[string]string{"timeout_seconds": "30"}, Starter: []sharedkernel.LabFile{{Path: "main.go", Content: "package main"}}, Checkpoints: []sharedkernel.LabCheckpoint{{ID: "checkpoint-01", Verified: true}}, Solution: []sharedkernel.LabFile{{Path: "main.go", Content: "package main"}}, Tests: []sharedkernel.LabFile{{Path: "main_test.go", Content: "package main"}}}
}

func TestRunnerUsesAllowlistedCommandsWithCleanEnvironment(t *testing.T) {
	executor := &fakeExecutor{}
	report := (Runner{Executor: executor, Timeout: time.Second}).Verify(context.Background(), RunRequest{Manifest: validManifest(), RootDir: "/tmp/course-runner"})
	require.True(t, report.Passed)
	require.Equal(t, []string{"test", "test -run TestCheckpoint"}, executor.commands)
}

func TestRunnerRejectsUnsafeCommandAndMissingExecutor(t *testing.T) {
	manifest := validManifest()
	manifest.AllowedCommands = []string{"test; curl evil"}
	report := (Runner{Executor: &fakeExecutor{}}).Verify(context.Background(), RunRequest{Manifest: manifest, RootDir: "/tmp/course-runner"})
	require.Contains(t, report.Error, "unsafe lab command")
	report = (Runner{}).Verify(context.Background(), RunRequest{Manifest: validManifest(), RootDir: "/tmp/course-runner"})
	require.Equal(t, ErrRunnerNotConfigured.Error(), report.Error)
}

func TestRunnerRejectsHostRoot(t *testing.T) {
	report := (Runner{Executor: &fakeExecutor{}}).Verify(context.Background(), RunRequest{Manifest: validManifest(), RootDir: "/"})
	require.Equal(t, ErrUnsafeRoot.Error(), report.Error)
}

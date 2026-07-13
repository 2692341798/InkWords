package projectcourse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildLabManifestCreatesOrderedHashBoundCheckpoints(t *testing.T) {
	manifest, err := BuildLabManifest("go", "1.25", []LabPatch{{Path: "main.go", Content: "package main"}}, [][]LabPatch{{{Path: "main.go", Content: "package main\nfunc main() {}"}}, {{Path: "main.go", Content: "package main\nfunc Run() {}"}}}, []LabPatch{{Path: "main_test.go", Content: "package main"}})
	require.NoError(t, err)
	require.NoError(t, manifest.Validate())
	require.Equal(t, "checkpoint-01", manifest.Checkpoints[1].PreviousID)
	require.NotEmpty(t, manifest.Checkpoints[0].Files[0].Hash)
}

func TestLabManifestRejectsUnsafeCommandsAndBrokenDependencies(t *testing.T) {
	manifest, err := BuildLabManifest("go", "1.25", []LabPatch{{Path: "main.go", Content: "package main"}}, [][]LabPatch{{{Path: "main.go", Content: "x"}}}, []LabPatch{{Path: "main_test.go", Content: "x"}})
	require.NoError(t, err)
	manifest.AllowedCommands = []string{"test && curl evil"}
	require.Error(t, manifest.Validate())
	manifest.AllowedCommands = []string{"test"}
	manifest.Checkpoints[0].PreviousID = "missing"
	require.Error(t, manifest.Validate())
}

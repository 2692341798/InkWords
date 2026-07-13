package export

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

func TestWriteCoursePackageIncludesOnlyVerifiedArtifactsAndManifest(t *testing.T) {
	manifest := sharedkernel.LabManifest{Language: "go", ToolchainVersion: "1.25", AllowedCommands: []string{"test"}, Starter: []sharedkernel.LabFile{{Path: "main.go", Content: "package main"}}, Checkpoints: []sharedkernel.LabCheckpoint{{ID: "checkpoint-01", Files: []sharedkernel.LabFile{{Path: "main.go", Content: "package main"}}, Verified: true}}, Solution: []sharedkernel.LabFile{{Path: "main.go", Content: "package main\nfunc main() {}"}}, Tests: []sharedkernel.LabFile{{Path: "main_test.go", Content: "package main"}}}
	var buf bytes.Buffer
	require.NoError(t, WriteCoursePackage(&buf, manifest, "0123456789abcdef0123456789abcdef01234567"))
	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, reader.File, 5)
	require.Equal(t, "manifest.json", reader.File[len(reader.File)-1].Name)
}

func TestWriteCoursePackageRejectsUnverifiedOrUnsafeArtifacts(t *testing.T) {
	manifest := sharedkernel.LabManifest{Language: "go", ToolchainVersion: "1.25", AllowedCommands: []string{"test"}, Starter: []sharedkernel.LabFile{{Path: "../secret", Content: "x"}}, Checkpoints: []sharedkernel.LabCheckpoint{{ID: "checkpoint-01", Files: []sharedkernel.LabFile{{Path: "main.go", Content: "x"}}, Verified: false}}, Solution: []sharedkernel.LabFile{{Path: "main.go", Content: "x"}}, Tests: []sharedkernel.LabFile{{Path: "main_test.go", Content: "x"}}}
	var buf bytes.Buffer
	require.ErrorContains(t, WriteCoursePackage(&buf, manifest, "sha"), "not verified")
	manifest.Checkpoints[0].Verified = true
	require.ErrorContains(t, WriteCoursePackage(&buf, manifest, "sha"), "unsafe")
}

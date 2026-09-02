package export

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

func TestWriteCoursePackageIncludesOnlyVerifiedArtifactsAndManifest(t *testing.T) {
	manifest := sharedkernel.LabManifest{Language: "go", ToolchainVersion: "1.25", AllowedCommands: []string{"test"}, ResourceLimits: map[string]string{"timeout_seconds": "30"}, Starter: []sharedkernel.LabFile{{Path: "main.go", Content: "package main"}}, Checkpoints: []sharedkernel.LabCheckpoint{{ID: "checkpoint-01", Files: []sharedkernel.LabFile{{Path: "main.go", Content: "package main"}}, Verified: true}}, Solution: []sharedkernel.LabFile{{Path: "main.go", Content: "package main\nfunc main() {}"}}, Tests: []sharedkernel.LabFile{{Path: "main_test.go", Content: "package main"}}}
	var buf bytes.Buffer
	require.NoError(t, WriteCoursePackage(&buf, manifest, "0123456789abcdef0123456789abcdef01234567"))
	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, reader.File, 7)
	require.Equal(t, "manifest.json", reader.File[len(reader.File)-1].Name)
	require.Equal(t, "README.md", reader.File[0].Name)
}

func TestWriteCoursePackageRejectsUnverifiedOrUnsafeArtifacts(t *testing.T) {
	manifest := sharedkernel.LabManifest{Language: "go", ToolchainVersion: "1.25", AllowedCommands: []string{"test"}, ResourceLimits: map[string]string{"timeout_seconds": "30"}, Starter: []sharedkernel.LabFile{{Path: "../secret", Content: "x"}}, Checkpoints: []sharedkernel.LabCheckpoint{{ID: "checkpoint-01", Files: []sharedkernel.LabFile{{Path: "main.go", Content: "x"}}, Verified: false}}, Solution: []sharedkernel.LabFile{{Path: "main.go", Content: "x"}}, Tests: []sharedkernel.LabFile{{Path: "main_test.go", Content: "x"}}}
	var buf bytes.Buffer
	require.ErrorContains(t, WriteCoursePackage(&buf, manifest, "sha"), "unsafe lab file path")
	manifest.Starter[0].Path = "main.go"
	require.ErrorContains(t, WriteCoursePackage(&buf, manifest, "sha"), "not verified")
	manifest.Checkpoints[0].Verified = true
	manifest.Starter[0].Path = "../secret"
	require.ErrorContains(t, WriteCoursePackage(&buf, manifest, "sha"), "unsafe lab file path")
}

func TestWriteCoursePackageBundlesMultipleChapterArtifacts(t *testing.T) {
	manifest := sharedkernel.LabManifest{Language: "go", ToolchainVersion: "1.25", AllowedCommands: []string{"test"}, ResourceLimits: map[string]string{"timeout_seconds": "30"}, Starter: []sharedkernel.LabFile{{Path: "main.go", Content: "package main"}}, Checkpoints: []sharedkernel.LabCheckpoint{{ID: "checkpoint-01", Files: []sharedkernel.LabFile{{Path: "main.go", Content: "package main"}}, Verified: true}}, Solution: []sharedkernel.LabFile{{Path: "main.go", Content: "package main"}}, Tests: []sharedkernel.LabFile{{Path: "main_test.go", Content: "package main"}}}
	var buf bytes.Buffer
	require.NoError(t, WriteCoursePackageWithMetadata(&buf, CoursePackageInput{CourseID: "course-1", BlueprintVersion: 2, CommitSHA: "0123456789abcdef0123456789abcdef01234567", Coverage: map[string]any{"files": []string{"main.go"}}, Verification: VerificationSummary{Passed: true}, Artifacts: []CourseArtifact{{ChapterID: "chapter-a", Title: "A", Manifest: manifest}, {ChapterID: "chapter-b", Title: "B", Manifest: manifest}}}))
	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	names := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		names = append(names, file.Name)
	}
	require.Contains(t, names, "chapters/chapter-a/starter/main.go")
	require.Contains(t, names, "chapters/chapter-b/tests/main_test.go")
	require.Equal(t, "manifest.json", names[len(names)-1])
}

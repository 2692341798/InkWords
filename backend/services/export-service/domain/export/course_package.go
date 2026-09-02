package export

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

type VerificationSummary struct {
	Passed       bool   `json:"passed"`
	CheckpointID string `json:"checkpoint_id,omitempty"`
	Error        string `json:"error,omitempty"`
}

type CoursePackageInput struct {
	CourseID         string                   `json:"course_id,omitempty"`
	BlueprintVersion int                      `json:"blueprint_version,omitempty"`
	RepositoryURL    string                   `json:"repository_url,omitempty"`
	CommitSHA        string                   `json:"commit_sha"`
	Manifest         sharedkernel.LabManifest `json:"manifest,omitempty"`
	Artifacts        []CourseArtifact         `json:"artifacts,omitempty"`
	Readme           string                   `json:"readme,omitempty"`
	Coverage         any                      `json:"coverage,omitempty"`
	Verification     VerificationSummary      `json:"verification"`
}

type CourseArtifact struct {
	ChapterID string                   `json:"chapter_id"`
	Title     string                   `json:"title"`
	Manifest  sharedkernel.LabManifest `json:"manifest"`
}

type packageEntry struct {
	name    string
	content string
}

type CoursePackagePayload struct {
	Package CoursePackageInput `json:"package"`
}

type FileCoursePackageBuilder struct {
	TempDir string
}

func NewFileCoursePackageBuilder(tempDir string) FileCoursePackageBuilder {
	return FileCoursePackageBuilder{TempDir: tempDir}
}

func (b FileCoursePackageBuilder) BuildCoursePackage(ctx context.Context, payload CoursePackagePayload) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	dir := strings.TrimSpace(b.TempDir)
	if dir == "" {
		dir = os.TempDir()
	}
	file, err := os.CreateTemp(dir, "inkwords-course-*.zip")
	if err != nil {
		return "", "", err
	}
	path := file.Name()
	if err := WriteCoursePackageWithMetadata(file, payload.Package); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", "", err
	}
	filename := "project-course.zip"
	if strings.TrimSpace(payload.Package.CourseID) != "" {
		filename = "project-course-" + safeFilenamePart(payload.Package.CourseID) + ".zip"
	}
	return path, filename, nil
}

func safeFilenamePart(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			builder.WriteRune(char)
		}
	}
	if builder.Len() == 0 {
		return "course"
	}
	return builder.String()
}

// WriteCoursePackage writes only verified course artifacts and includes the manifest last.
// It never runs commands or reads files outside the supplied manifest.
func WriteCoursePackage(w io.Writer, manifest sharedkernel.LabManifest, commitSHA string) error {
	return WriteCoursePackageWithMetadata(w, CoursePackageInput{CommitSHA: commitSHA, Manifest: manifest, Readme: "# Project Course\n\nThis package contains the verified course artifacts.\n", Coverage: map[string]any{}, Verification: VerificationSummary{Passed: true}})
}

// WriteCoursePackageWithMetadata creates a deterministic, self-describing
// package. It only serializes already materialized artifacts; it never runs a
// test or reads a path outside the supplied manifest.
//
//nolint:gocyclo // The branching mirrors the manifest's ordered validation and archive sections.
func WriteCoursePackageWithMetadata(w io.Writer, input CoursePackageInput) error {
	artifacts := input.Artifacts
	bundle := len(artifacts) > 0
	if !bundle {
		artifacts = []CourseArtifact{{ChapterID: "course", Title: "课程实验", Manifest: input.Manifest}}
	}
	for _, artifact := range artifacts {
		if err := artifact.Manifest.Validate(); err != nil {
			return fmt.Errorf("chapter %q manifest: %w", artifact.ChapterID, err)
		}
	}
	if strings.TrimSpace(input.CommitSHA) == "" {
		return fmt.Errorf("course package commit SHA is required")
	}
	if !input.Verification.Passed {
		return fmt.Errorf("course package verification has not passed")
	}
	for _, artifact := range artifacts {
		for _, checkpoint := range artifact.Manifest.Checkpoints {
			if !checkpoint.Verified {
				return fmt.Errorf("chapter %q checkpoint %q is not verified", artifact.ChapterID, checkpoint.ID)
			}
		}
	}
	if strings.TrimSpace(input.Readme) == "" {
		input.Readme = "# Project Course\n\nThis package contains the verified course artifacts.\n"
	}
	entries := make([]packageEntry, 0)
	addFiles := func(prefix string, files []sharedkernel.LabFile) error {
		for _, file := range files {
			entry, err := safeZipPath(prefix, file.Path)
			if err != nil {
				return err
			}
			entries = append(entries, packageEntry{name: entry, content: file.Content})
		}
		return nil
	}
	entries = append(entries, packageEntry{name: "README.md", content: input.Readme})
	for _, artifact := range artifacts {
		prefix := ""
		if bundle {
			prefix = path.Join("chapters", artifact.ChapterID)
		}
		if err := addFiles(path.Join(prefix, "starter"), artifact.Manifest.Starter); err != nil {
			return err
		}
		for _, checkpoint := range artifact.Manifest.Checkpoints {
			if err := addFiles(path.Join(prefix, "checkpoints", checkpoint.ID), checkpoint.Files); err != nil {
				return err
			}
		}
		for _, exercise := range artifact.Manifest.Exercises {
			for _, hint := range exercise.Hints {
				entry, err := safeZipPath(path.Join(prefix, "hints", exercise.ExerciseID), fmt.Sprintf("level-%d.md", hint.Level))
				if err != nil {
					return err
				}
				entries = append(entries, packageEntry{name: entry, content: hint.Content})
			}
		}
		if err := addFiles(path.Join(prefix, "solution"), artifact.Manifest.Solution); err != nil {
			return err
		}
		if err := addFiles(path.Join(prefix, "tests"), artifact.Manifest.Tests); err != nil {
			return err
		}
	}
	coverageJSON, err := json.Marshal(input.Coverage)
	if err != nil {
		return err
	}
	entries = append(entries, packageEntry{name: "coverage.json", content: string(coverageJSON)})
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	hashes := make(map[string]string, len(entries))
	for _, entry := range entries {
		sum := sha256.Sum256([]byte(entry.content))
		hashes[entry.name] = "sha256:" + fmt.Sprintf("%x", sum[:])
	}
	archive := zip.NewWriter(w)
	defer func() { _ = archive.Close() }()
	for _, entry := range entries {
		writer, err := archive.Create(entry.name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(writer, entry.content); err != nil {
			return err
		}
	}
	meta, err := json.Marshal(struct {
		CourseID         string                    `json:"course_id,omitempty"`
		BlueprintVersion int                       `json:"blueprint_version,omitempty"`
		RepositoryURL    string                    `json:"repository_url,omitempty"`
		CommitSHA        string                    `json:"commit_sha"`
		Toolchain        string                    `json:"toolchain"`
		Verification     VerificationSummary       `json:"verification"`
		FileHashes       map[string]string         `json:"file_hashes"`
		Manifest         *sharedkernel.LabManifest `json:"manifest,omitempty"`
		Artifacts        []CourseArtifact          `json:"artifacts,omitempty"`
	}{CourseID: input.CourseID, BlueprintVersion: input.BlueprintVersion, RepositoryURL: input.RepositoryURL, CommitSHA: input.CommitSHA, Toolchain: artifacts[0].Manifest.ToolchainVersion, Verification: input.Verification, FileHashes: hashes, Manifest: singleManifest(artifacts, bundle), Artifacts: bundleArtifacts(artifacts, bundle)})
	if err != nil {
		return err
	}
	writer, err := archive.Create("manifest.json")
	if err != nil {
		return err
	}
	if _, err = writer.Write(meta); err != nil {
		return err
	}
	return archive.Close()
}

func safeZipPath(prefix, filePath string) (string, error) {
	clean := path.Clean(strings.ReplaceAll(filePath, "\\", "/"))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("unsafe course artifact path %q", filePath)
	}
	return path.Join(prefix, clean), nil
}

func singleManifest(artifacts []CourseArtifact, bundle bool) *sharedkernel.LabManifest {
	if bundle {
		return nil
	}
	return &artifacts[0].Manifest
}

func bundleArtifacts(artifacts []CourseArtifact, bundle bool) []CourseArtifact {
	if !bundle {
		return nil
	}
	return artifacts
}

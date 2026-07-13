package export

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
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
	CourseID         string
	BlueprintVersion int
	RepositoryURL    string
	CommitSHA        string
	Manifest         sharedkernel.LabManifest
	Readme           string
	Coverage         any
	Verification     VerificationSummary
}

type packageEntry struct {
	name    string
	content string
}

// WriteCoursePackage writes only verified course artifacts and includes the manifest last.
// It never runs commands or reads files outside the supplied manifest.
func WriteCoursePackage(w io.Writer, manifest sharedkernel.LabManifest, commitSHA string) error {
	return WriteCoursePackageWithMetadata(w, CoursePackageInput{CommitSHA: commitSHA, Manifest: manifest, Readme: "# Project Course\n\nThis package contains the verified course artifacts.\n", Coverage: map[string]any{}, Verification: VerificationSummary{Passed: true}})
}

// WriteCoursePackageWithMetadata creates a deterministic, self-describing
// package. It only serializes already materialized artifacts; it never runs a
// test or reads a path outside the supplied manifest.
func WriteCoursePackageWithMetadata(w io.Writer, input CoursePackageInput) error {
	if err := input.Manifest.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(input.CommitSHA) == "" {
		return fmt.Errorf("course package commit SHA is required")
	}
	if !input.Verification.Passed {
		return fmt.Errorf("course package verification has not passed")
	}
	for _, checkpoint := range input.Manifest.Checkpoints {
		if !checkpoint.Verified {
			return fmt.Errorf("checkpoint %q is not verified", checkpoint.ID)
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
	if err := addFiles("starter", input.Manifest.Starter); err != nil {
		return err
	}
	for _, checkpoint := range input.Manifest.Checkpoints {
		if err := addFiles(path.Join("checkpoints", checkpoint.ID), checkpoint.Files); err != nil {
			return err
		}
	}
	for _, exercise := range input.Manifest.Exercises {
		for _, hint := range exercise.Hints {
			entry, err := safeZipPath(path.Join("hints", exercise.ExerciseID), fmt.Sprintf("level-%d.md", hint.Level))
			if err != nil {
				return err
			}
			entries = append(entries, packageEntry{name: entry, content: hint.Content})
		}
	}
	if err := addFiles("solution", input.Manifest.Solution); err != nil {
		return err
	}
	if err := addFiles("tests", input.Manifest.Tests); err != nil {
		return err
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
	defer archive.Close()
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
		CourseID         string                   `json:"course_id,omitempty"`
		BlueprintVersion int                      `json:"blueprint_version,omitempty"`
		RepositoryURL    string                   `json:"repository_url,omitempty"`
		CommitSHA        string                   `json:"commit_sha"`
		Toolchain        string                   `json:"toolchain"`
		Verification     VerificationSummary      `json:"verification"`
		FileHashes       map[string]string        `json:"file_hashes"`
		Manifest         sharedkernel.LabManifest `json:"manifest"`
	}{CourseID: input.CourseID, BlueprintVersion: input.BlueprintVersion, RepositoryURL: input.RepositoryURL, CommitSHA: input.CommitSHA, Toolchain: input.Manifest.ToolchainVersion, Verification: input.Verification, FileHashes: hashes, Manifest: input.Manifest})
	if err != nil {
		return err
	}
	writer, err := archive.Create("manifest.json")
	if err != nil {
		return err
	}
	_, err = writer.Write(meta)
	return err
}

func safeZipPath(prefix, filePath string) (string, error) {
	clean := path.Clean(strings.ReplaceAll(filePath, "\\", "/"))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("unsafe course artifact path %q", filePath)
	}
	return path.Join(prefix, clean), nil
}

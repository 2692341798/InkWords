package export

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

// WriteCoursePackage writes only verified course artifacts and includes the manifest last.
// It never runs commands or reads files outside the supplied manifest.
func WriteCoursePackage(w io.Writer, manifest sharedkernel.LabManifest, commitSHA string) error {
	if err := manifest.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(commitSHA) == "" {
		return fmt.Errorf("course package commit SHA is required")
	}
	for _, checkpoint := range manifest.Checkpoints {
		if !checkpoint.Verified {
			return fmt.Errorf("checkpoint %q is not verified", checkpoint.ID)
		}
	}
	archive := zip.NewWriter(w)
	defer archive.Close()
	writeFiles := func(prefix string, files []sharedkernel.LabFile) error {
		for _, file := range files {
			entry, err := safeZipPath(prefix, file.Path)
			if err != nil {
				return err
			}
			writer, err := archive.Create(entry)
			if err != nil {
				return err
			}
			if _, err := io.WriteString(writer, file.Content); err != nil {
				return err
			}
		}
		return nil
	}
	if err := writeFiles("starter", manifest.Starter); err != nil {
		return err
	}
	for _, checkpoint := range manifest.Checkpoints {
		if err := writeFiles(path.Join("checkpoints", checkpoint.ID), checkpoint.Files); err != nil {
			return err
		}
	}
	if err := writeFiles("solution", manifest.Solution); err != nil {
		return err
	}
	if err := writeFiles("tests", manifest.Tests); err != nil {
		return err
	}
	meta, err := json.Marshal(struct {
		CommitSHA string                   `json:"commit_sha"`
		Manifest  sharedkernel.LabManifest `json:"manifest"`
	}{CommitSHA: commitSHA, Manifest: manifest})
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

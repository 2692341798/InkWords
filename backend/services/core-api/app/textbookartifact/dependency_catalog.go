package textbookartifact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// DependencyCatalogEntry contains operator-only paths, never accepted from a browser.
type DependencyCatalogEntry struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	SelectionPath string `json:"selection_path"`
	SelectionHash string `json:"selection_hash"`
	Root          string `json:"root"`
	ManifestPath  string `json:"manifest_path"`
}

// DependencyOption exposes verified provenance without exposing filesystem configuration.
type DependencyOption struct {
	ID            string                                     `json:"id"`
	Title         string                                     `json:"title"`
	SelectionHash string                                     `json:"selection_hash"`
	Selection     sharedtextbook.TeachingDependencySelection `json:"selection"`
}

// LocalDependencyCatalog reads only the explicit operator configuration on each request.
// An unset path means no prepared options, not an automatic dependency download.
type LocalDependencyCatalog struct {
	path     string
	resolver dependencySourceResolver
}

// NewLocalDependencyCatalog configures a local immutable-package catalog.
func NewLocalDependencyCatalog(path string, resolver dependencySourceResolver) *LocalDependencyCatalog {
	return &LocalDependencyCatalog{path: path, resolver: resolver}
}

func (c *LocalDependencyCatalog) entries() ([]DependencyCatalogEntry, error) {
	if c == nil || c.path == "" {
		return []DependencyCatalogEntry{}, nil
	}
	data, err := readSelectionFile(c.path, 64<<10)
	if err != nil {
		return nil, err
	}
	var document struct {
		Contract string                   `json:"contract"`
		Entries  []DependencyCatalogEntry `json:"entries"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&document) != nil || decoder.Decode(new(any)) != io.EOF || document.Contract != "inkwords.dependency-catalog.v1" || len(document.Entries) > 16 {
		return nil, fmt.Errorf("invalid dependency catalog")
	}
	seen := map[string]bool{}
	for _, entry := range document.Entries {
		if entry.ID == "" || len(entry.ID) > 128 || strings.TrimSpace(entry.ID) != entry.ID || seen[entry.ID] || strings.TrimSpace(entry.Title) == "" || len([]rune(entry.Title)) > 128 || !filepath.IsAbs(entry.SelectionPath) || !filepath.IsAbs(entry.Root) || !filepath.IsAbs(entry.ManifestPath) {
			return nil, fmt.Errorf("invalid dependency catalog entry")
		}
		seen[entry.ID] = true
	}
	return document.Entries, nil
}

// Options verifies bytes and current source membership, returning only this chapter's selections.
func (c *LocalDependencyCatalog) Options(ctx context.Context, workspaceID, chapterID uuid.UUID) ([]DependencyOption, error) {
	if workspaceID == uuid.Nil || chapterID == uuid.Nil {
		return nil, textbookdomain.ErrInvalidState
	}
	entries, err := c.entries()
	if err != nil {
		return nil, err
	}
	options := []DependencyOption{}
	for _, entry := range entries {
		selection, err := readDependencySelection(entry.SelectionPath, entry.SelectionHash)
		if err != nil {
			return nil, err
		}
		if selection.WorkspaceID != workspaceID.String() || selection.ChapterID != chapterID.String() {
			continue
		}
		selected, err := ReadSelectedOfflineGoBundle(ctx, c.resolver, entry.SelectionPath, entry.SelectionHash, entry.Root, entry.ManifestPath)
		if err != nil {
			return nil, err
		}
		options = append(options, DependencyOption{entry.ID, entry.Title, entry.SelectionHash, selected.Selection()})
	}
	return options, nil
}

// Resolve rereads the selected entry and package; stale or unrelated choices fail closed.
func (c *LocalDependencyCatalog) Resolve(ctx context.Context, workspaceID, chapterID uuid.UUID, id string) (*SelectedOfflineGoBundle, string, error) {
	entries, err := c.entries()
	if err != nil {
		return nil, "", err
	}
	for _, entry := range entries {
		if entry.ID != id {
			continue
		}
		selection, err := readDependencySelection(entry.SelectionPath, entry.SelectionHash)
		if err != nil {
			return nil, "", err
		}
		if workspaceID == uuid.Nil || chapterID == uuid.Nil || selection.WorkspaceID != workspaceID.String() || selection.ChapterID != chapterID.String() {
			return nil, "", textbookdomain.ErrNotFound
		}
		selected, err := ReadSelectedOfflineGoBundle(ctx, c.resolver, entry.SelectionPath, entry.SelectionHash, entry.Root, entry.ManifestPath)
		return selected, entry.SelectionHash, err
	}
	return nil, "", textbookdomain.ErrNotFound
}

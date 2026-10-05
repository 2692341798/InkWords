package textbook

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var selectedGoToolchain = regexp.MustCompile(`^go[0-9]+\.[0-9]+\.[0-9]+$`)
var selectedModuleVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

// TeachingDependencySelection records an explicit operator choice for one
// chapter. It contains provenance and bounded observation templates, never
// local paths, arbitrary URLs or shell commands.
type TeachingDependencySelection struct {
	WorkspaceID            string               `json:"workspace_id"`
	ProjectID              string               `json:"project_id"`
	ChapterID              string               `json:"chapter_id"`
	Snapshot               SourceSnapshot       `json:"snapshot"`
	Module                 string               `json:"module"`
	Version                string               `json:"version"`
	Toolchain              string               `json:"toolchain"`
	DependencyManifestHash string               `json:"dependency_manifest_hash"`
	BrowserObservation     *VerificationCommand `json:"browser_observation,omitempty"`
}

// Validate restricts V1 selection to a primary Git repository whose canonical
// HTTPS locator matches the module. Vanity paths and submodules need a separate
// reviewed mapping contract; an inferred mapping must not grant authorization.
func (s TeachingDependencySelection) Validate() error {
	if command := s.BrowserObservation; command != nil {
		if command.Kind != "browser_page" || command.Validate() != nil {
			return fmt.Errorf("dependency selection browser observation is invalid")
		}
	}
	for _, id := range []string{s.WorkspaceID, s.ProjectID, s.ChapterID, s.Snapshot.ID, s.Snapshot.SourceID} {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil || parsed.String() != id {
			return fmt.Errorf("dependency selection requires canonical identities")
		}
	}
	if err := s.Snapshot.Validate(); err != nil {
		return err
	}
	if s.Snapshot.Kind != SourceKindGitRepository || s.Snapshot.Role != SourceRolePrimary || !isFullSHA256Digest(s.DependencyManifestHash) || !selectedGoToolchain.MatchString(s.Toolchain) || !selectedModuleVersion.MatchString(s.Version) {
		return fmt.Errorf("dependency selection requires a pinned primary Git source and exact toolchain")
	}
	u, err := url.Parse(s.Snapshot.Locator)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || u.Port() != "" || u.Host+strings.TrimSuffix(u.Path, ".git") != s.Module {
		return fmt.Errorf("dependency module does not match the canonical source locator")
	}
	return nil
}

// Clone detaches the optional browser expectation before exposing a selection.
func (s TeachingDependencySelection) Clone() TeachingDependencySelection {
	if s.BrowserObservation != nil {
		command := *s.BrowserObservation
		s.BrowserObservation = &command
	}
	return s
}

// MatchesSnapshot compares frozen source identity without depending on time zone formatting.
func (s TeachingDependencySelection) MatchesSnapshot(actual SourceSnapshot) bool {
	want := s.Snapshot
	return actual.ID == want.ID && actual.SourceID == want.SourceID && actual.Kind == want.Kind && actual.Role == want.Role && actual.Locator == want.Locator && actual.ResolvedVersion == want.ResolvedVersion && actual.ContentHash == want.ContentHash && actual.CapturedAt.Equal(want.CapturedAt)
}

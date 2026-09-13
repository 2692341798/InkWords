package textbook

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const TeachingArtifactManifestFormat = "inkwords.teaching-artifact.v1"

const TextbookTeachingArtifactVerifyTaskSubtype = "textbook_teaching_artifact_verify"

// ArtifactVerificationRequest identifies an immutable generated teaching tree.
// Worker code must reload the manifest and bytes from trusted storage instead
// of treating this message as an executable command or path.
type ArtifactVerificationRequest struct {
	ArtifactID   string `json:"artifact_id"`
	RevisionID   string `json:"revision_id"`
	ArtifactHash string `json:"artifact_hash"`
	ManifestHash string `json:"manifest_hash"`
}

func (request ArtifactVerificationRequest) Validate() error {
	if uuid.Validate(request.ArtifactID) != nil || uuid.Validate(request.RevisionID) != nil || !isFullSHA256Digest(request.ArtifactHash) || !isFullSHA256Digest(request.ManifestHash) {
		return fmt.Errorf("textbook artifact verification request is incomplete")
	}
	return nil
}

// VerificationCommand is a template, never caller-provided shell syntax.
type VerificationCommand struct {
	Kind         string `json:"kind"`
	TestPattern  string `json:"test_pattern,omitempty"`
	BrowserPath  string `json:"browser_path,omitempty"`
	ExpectedText string `json:"expected_text,omitempty"`
}

func (command VerificationCommand) Validate() error {
	switch command.Kind {
	case "go_test":
		if command.BrowserPath != "" || command.ExpectedText != "" {
			return fmt.Errorf("Go test command cannot declare browser fields")
		}
		for _, char := range command.TestPattern {
			if !isAllowedVerificationPatternRune(char) {
				return fmt.Errorf("unsafe Go test pattern")
			}
		}
		return nil
	case "browser_page":
		if command.TestPattern != "" {
			return fmt.Errorf("browser-page command cannot declare a Go test pattern")
		}
		if !isSafeLocalBrowserPath(command.BrowserPath) {
			return fmt.Errorf("browser-page command requires a safe local browser path")
		}
		if !isSafeBrowserExpectedText(command.ExpectedText) {
			return fmt.Errorf("browser-page command requires bounded expected text")
		}
		return nil
	default:
		return fmt.Errorf("unsupported textbook command template %q", command.Kind)
	}
}

// ExecutableForm is internal runner input, not a general shell command API.
func (command VerificationCommand) ExecutableForm() string {
	if command.TestPattern == "" {
		return "test"
	}
	return "test -run " + command.TestPattern
}

// TeachingArtifactManifest is created with generated teaching code and later
// persisted alongside its content-addressed tree.
type TeachingArtifactManifest struct {
	Format                 string                       `json:"format"`
	ArtifactID             string                       `json:"artifact_id"`
	RevisionID             string                       `json:"revision_id"`
	ArtifactHash           string                       `json:"artifact_hash"`
	BookContractHash       string                       `json:"book_contract_hash"`
	StyleSheetHash         string                       `json:"style_sheet_hash"`
	Language               string                       `json:"language"`
	ToolchainVersion       string                       `json:"toolchain_version"`
	DependencyManifestHash string                       `json:"dependency_manifest_hash,omitempty"`
	DependencySelection    *TeachingDependencySelection `json:"dependency_selection,omitempty"`
	Commands               []VerificationCommand        `json:"commands"`
}

func (manifest TeachingArtifactManifest) Validate() error {
	if selection := manifest.DependencySelection; selection != nil {
		if err := selection.Validate(); err != nil {
			return err
		}
		if selection.DependencyManifestHash != manifest.DependencyManifestHash || selection.Toolchain != manifest.ToolchainVersion {
			return fmt.Errorf("teaching dependency selection does not match the execution manifest")
		}
		browserCount := 0
		for _, command := range manifest.Commands {
			if command.Kind == "browser_page" {
				browserCount++
				if selection.BrowserObservation == nil || command != *selection.BrowserObservation {
					return fmt.Errorf("teaching browser command differs from operator selection")
				}
			}
		}
		if (selection.BrowserObservation != nil && browserCount != 1) || browserCount > 1 {
			return fmt.Errorf("selected browser observation is missing or duplicated")
		}
	}
	if manifest.DependencyManifestHash != "" && !isFullSHA256Digest(manifest.DependencyManifestHash) {
		return fmt.Errorf("teaching dependency inventory hash is invalid")
	}
	if manifest.Format != TeachingArtifactManifestFormat || uuid.Validate(manifest.ArtifactID) != nil || uuid.Validate(manifest.RevisionID) != nil || !isFullSHA256Digest(manifest.ArtifactHash) || !isFullSHA256Digest(manifest.BookContractHash) || !isFullSHA256Digest(manifest.StyleSheetHash) || manifest.Language != "go" || strings.TrimSpace(manifest.ToolchainVersion) == "" || len(manifest.Commands) == 0 {
		return fmt.Errorf("teaching artifact manifest is incomplete")
	}
	for _, command := range manifest.Commands {
		if err := command.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func TeachingArtifactManifestHash(manifest TeachingArtifactManifest) (string, error) {
	if err := manifest.Validate(); err != nil {
		return "", err
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("marshal artifact manifest: %w", err)
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// VerificationInputHash contains all values whose change makes a previous run
// stale: code bytes, command template, runner image, and toolchain version.
func TeachingArtifactVerificationInputHash(manifest TeachingArtifactManifest, runnerImageDigest string) (string, error) {
	if err := manifest.Validate(); err != nil {
		return "", err
	}
	if !isFullSHA256Digest(runnerImageDigest) {
		return "", fmt.Errorf("runner image digest is required")
	}
	payload, err := json.Marshal(struct {
		TeachingArtifactManifest
		RunnerImageDigest string `json:"runner_image_digest"`
	}{TeachingArtifactManifest: manifest, RunnerImageDigest: runnerImageDigest})
	if err != nil {
		return "", fmt.Errorf("marshal verification input: %w", err)
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func isFullSHA256Digest(value string) bool {
	digest := strings.TrimPrefix(strings.TrimSpace(value), "sha256:")
	if !strings.HasPrefix(strings.TrimSpace(value), "sha256:") || len(digest) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(digest)
	return err == nil
}

func isAllowedVerificationPatternRune(char rune) bool {
	return char == '_' || char == '-' || char == '.' || char == '*' || char == '+' || char == '^' || char == '$' || char == '(' || char == ')' ||
		char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9'
}

func isSafeLocalBrowserPath(value string) bool {
	if len(value) == 0 || len(value) > 256 || !strings.HasPrefix(value, "/") || strings.ContainsAny(value, "?#\\") || strings.Contains(value, "//") || strings.Contains(value, "..") {
		return false
	}
	for _, char := range value {
		if char != '/' && char != '-' && char != '_' && char != '.' && (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}

func isSafeBrowserExpectedText(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 256 {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

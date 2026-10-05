// Package textbookverification verifies only generated textbook code artifacts.
// It intentionally does not share the retired course payload contract.
package textbookverification

import (
	"time"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

const ArtifactManifestFormat = sharedtextbook.TeachingArtifactManifestFormat

type CommandTemplate = sharedtextbook.VerificationCommand
type ArtifactManifest = sharedtextbook.TeachingArtifactManifest

func ManifestHash(manifest ArtifactManifest) (string, error) {
	return sharedtextbook.TeachingArtifactManifestHash(manifest)
}

func VerificationInputHash(manifest ArtifactManifest, runnerImageDigest string) (string, error) {
	return sharedtextbook.TeachingArtifactVerificationInputHash(manifest, runnerImageDigest)
}

type CommandResult struct {
	Command  CommandTemplate               `json:"command"`
	Status   sharedtextbook.ArtifactStatus `json:"status"`
	ExitCode int                           `json:"exit_code,omitempty"`
	Output   string                        `json:"output,omitempty"`
	Browser  *BrowserPageCapture           `json:"browser,omitempty"`
	Reason   string                        `json:"reason,omitempty"`
	Duration time.Duration                 `json:"duration"`
}

// BrowserPageCapture is raw automation output. It intentionally contains no
// explanatory prose: the shared RuntimeEvidence contract validates the saved
// observation bundle before any result can be presented as verified.
type BrowserPageCapture struct {
	URL               string `json:"url"`
	FinalURL          string `json:"final_url"`
	BrowserName       string `json:"browser_name"`
	BrowserVersion    string `json:"browser_version"`
	PlaywrightVersion string `json:"playwright_version"`
	ScreenshotRef     string `json:"screenshot_ref"`
	// ScreenshotContent is staged into the trusted visual-asset store before
	// persistence. Raw pixels never travel through task JSON or an HTTP API.
	ScreenshotContent []byte                      `json:"-"`
	DOMAssertions     []BrowserDOMAssertion       `json:"dom_assertions"`
	Console           []BrowserConsoleObservation `json:"console"`
	Network           []BrowserNetworkObservation `json:"network"`
}

type BrowserDOMAssertion struct {
	Locator   string `json:"locator"`
	Assertion string `json:"assertion"`
	Expected  string `json:"expected"`
}

type BrowserConsoleObservation struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type BrowserNetworkObservation struct {
	URL          string `json:"url"`
	ResourceType string `json:"resource_type"`
	Status       int    `json:"status"`
}

type Report struct {
	Status            sharedtextbook.ArtifactStatus `json:"status"`
	InputHash         string                        `json:"input_hash,omitempty"`
	ArtifactHash      string                        `json:"artifact_hash,omitempty"`
	RunnerImageDigest string                        `json:"runner_image_digest,omitempty"`
	ToolchainVersion  string                        `json:"toolchain_version,omitempty"`
	Results           []CommandResult               `json:"results"`
	Reason            string                        `json:"reason,omitempty"`
}

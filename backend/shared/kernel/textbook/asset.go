package textbook

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type RuntimeEvidenceKind string

const (
	RuntimeEvidenceTerminalOutput RuntimeEvidenceKind = "terminal_output"
	RuntimeEvidenceBrowserPage    RuntimeEvidenceKind = "browser_page"
	RuntimeEvidenceIDECapture     RuntimeEvidenceKind = "ide_capture"
	RuntimeEvidenceImage          RuntimeEvidenceKind = "image"
)

func (value RuntimeEvidenceKind) Validate() error {
	return validateEnum(value, []RuntimeEvidenceKind{RuntimeEvidenceTerminalOutput, RuntimeEvidenceBrowserPage, RuntimeEvidenceIDECapture, RuntimeEvidenceImage}, "runtime evidence kind")
}

// RuntimeEvidence records a concrete observation of an InkWords teaching
// artifact. A source excerpt, IDE instruction, or inferred explanation cannot
// be presented as a verified execution result through this contract.
type RuntimeEvidence struct {
	ID                  string              `json:"id"`
	RevisionID          string              `json:"revision_id"`
	CodeArtifactID      string              `json:"code_artifact_id"`
	CodeArtifactHash    string              `json:"code_artifact_hash"`
	InputHash           string              `json:"input_hash"`
	Kind                RuntimeEvidenceKind `json:"kind"`
	Status              ArtifactStatus      `json:"status"`
	CommandManifestHash string              `json:"command_manifest_hash,omitempty"`
	RunnerImageDigest   string              `json:"runner_image_digest,omitempty"`
	ToolchainVersion    string              `json:"toolchain_version,omitempty"`
	ToolName            string              `json:"tool_name,omitempty"`
	ToolVersion         string              `json:"tool_version,omitempty"`
	SamplingConditions  []string            `json:"sampling_conditions,omitempty"`
	StructuredOutput    string              `json:"structured_output,omitempty"`
	RawEvidenceRef      string              `json:"raw_evidence_ref,omitempty"`
	OutputTruncated     bool                `json:"output_truncated"`
	CapturedAt          time.Time           `json:"captured_at,omitempty"`
	ExpiresAt           *time.Time          `json:"expires_at,omitempty"`
	StaleReason         string              `json:"stale_reason,omitempty"`
}

func (evidence RuntimeEvidence) Validate() error {
	if strings.TrimSpace(evidence.ID) == "" || strings.TrimSpace(evidence.RevisionID) == "" || strings.TrimSpace(evidence.CodeArtifactID) == "" || !isSHA256Digest(evidence.CodeArtifactHash) || !isSHA256Digest(evidence.InputHash) {
		return fmt.Errorf("runtime evidence identity, artifact hash, and input hash are required")
	}
	if err := evidence.Kind.Validate(); err != nil {
		return err
	}
	if err := evidence.Status.Validate(); err != nil {
		return err
	}
	if evidence.Status == ArtifactStatusVerified {
		if strings.TrimSpace(evidence.CommandManifestHash) == "" || strings.TrimSpace(evidence.RunnerImageDigest) == "" || strings.TrimSpace(evidence.ToolchainVersion) == "" || strings.TrimSpace(evidence.ToolName) == "" || strings.TrimSpace(evidence.ToolVersion) == "" || len(evidence.SamplingConditions) == 0 || strings.TrimSpace(evidence.RawEvidenceRef) == "" || evidence.CapturedAt.IsZero() {
			return fmt.Errorf("verified runtime evidence requires tool, sampling, execution provenance, and raw evidence")
		}
		output, err := ParseRuntimeObservationOutput(evidence.StructuredOutput)
		if err != nil {
			return fmt.Errorf("verified runtime evidence requires structured observations: %w", err)
		}
		if evidence.Kind == RuntimeEvidenceBrowserPage {
			if err := validateBrowserPageObservationBundle(output); err != nil {
				return fmt.Errorf("verified browser runtime evidence requires a captured Playwright bundle: %w", err)
			}
			if err := validateBrowserToolVersion(output, evidence); err != nil {
				return fmt.Errorf("verified browser runtime evidence requires captured tool versions: %w", err)
			}
		}
	}
	if evidence.Status == ArtifactStatusBlocked && strings.TrimSpace(evidence.StaleReason) == "" {
		return fmt.Errorf("blocked runtime evidence requires its reason")
	}
	if evidence.ExpiresAt != nil && !evidence.CapturedAt.IsZero() && evidence.ExpiresAt.Before(evidence.CapturedAt) {
		return fmt.Errorf("runtime evidence cannot expire before it is captured")
	}
	return nil
}

// browserPageObservationBundle is intentionally a narrow wire contract. A
// browser-page result is only a direct observation when it proves which local
// page was loaded and retains the four artifacts required for review: an image,
// DOM assertions, console collection, and a network summary.
type browserPageObservationBundle struct {
	BrowserPages []browserPageObservation `json:"browser_pages"`
}

type browserPageObservation struct {
	URL               string                      `json:"url"`
	FinalURL          string                      `json:"final_url"`
	BrowserName       string                      `json:"browser_name"`
	BrowserVersion    string                      `json:"browser_version"`
	PlaywrightVersion string                      `json:"playwright_version"`
	ScreenshotRef     string                      `json:"screenshot_ref"`
	DOMAssertions     []browserDOMAssertion       `json:"dom_assertions"`
	Console           []browserConsoleObservation `json:"console"`
	Network           []browserNetworkObservation `json:"network"`
}

type browserDOMAssertion struct {
	Locator   string `json:"locator"`
	Assertion string `json:"assertion"`
	Expected  string `json:"expected"`
}

type browserConsoleObservation struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type browserNetworkObservation struct {
	URL          string `json:"url"`
	ResourceType string `json:"resource_type"`
	Status       int    `json:"status"`
}

func validateBrowserPageObservationBundle(output RuntimeObservationOutput) error {
	var bundle browserPageObservationBundle
	if err := json.Unmarshal(output.Observations, &bundle); err != nil || len(bundle.BrowserPages) == 0 {
		return fmt.Errorf("browser_pages observation is required")
	}
	for _, page := range bundle.BrowserPages {
		if strings.TrimSpace(page.BrowserName) == "" || strings.TrimSpace(page.BrowserVersion) == "" || strings.TrimSpace(page.PlaywrightVersion) == "" {
			return fmt.Errorf("browser and Playwright versions are required")
		}
		pageURL, err := localTeachingPageURL(page.URL)
		if err != nil {
			return fmt.Errorf("browser page URL: %w", err)
		}
		finalURL, err := localTeachingPageURL(page.FinalURL)
		if err != nil || pageURL.Scheme != finalURL.Scheme || pageURL.Host != finalURL.Host {
			return fmt.Errorf("browser final URL must remain on the local teaching origin")
		}
		if strings.TrimSpace(page.ScreenshotRef) == "" {
			return fmt.Errorf("browser screenshot reference is required")
		}
		if len(page.DOMAssertions) == 0 {
			return fmt.Errorf("browser DOM assertions are required")
		}
		for _, assertion := range page.DOMAssertions {
			if strings.TrimSpace(assertion.Locator) == "" || strings.TrimSpace(assertion.Assertion) == "" || strings.TrimSpace(assertion.Expected) == "" {
				return fmt.Errorf("browser DOM assertion requires locator, assertion, and expected value")
			}
		}
		// An explicit empty array proves collection ran without console events;
		// omitting it would make a clean console indistinguishable from no probe.
		if page.Console == nil {
			return fmt.Errorf("browser console summary is required")
		}
		for _, entry := range page.Console {
			if strings.TrimSpace(entry.Type) == "" || strings.TrimSpace(entry.Text) == "" {
				return fmt.Errorf("browser console entries require type and text")
			}
		}
		if len(page.Network) == 0 {
			return fmt.Errorf("browser network summary is required")
		}
		for _, request := range page.Network {
			if _, err := localTeachingPageURL(request.URL); err != nil || strings.TrimSpace(request.ResourceType) == "" || request.Status < 100 || request.Status > 599 {
				return fmt.Errorf("browser network summary must contain local URL, resource type, and HTTP status")
			}
		}
	}
	return nil
}

func validateBrowserToolVersion(output RuntimeObservationOutput, evidence RuntimeEvidence) error {
	var bundle browserPageObservationBundle
	if err := json.Unmarshal(output.Observations, &bundle); err != nil || len(bundle.BrowserPages) == 0 {
		return fmt.Errorf("browser page version observation is required")
	}
	if !strings.Contains(evidence.ToolName, "Playwright") || !strings.HasSuffix(evidence.ToolVersion, "runner-image:"+evidence.RunnerImageDigest) {
		return fmt.Errorf("tool name and runner image provenance are inconsistent")
	}
	for _, page := range bundle.BrowserPages {
		if !strings.Contains(evidence.ToolVersion, "Playwright "+page.PlaywrightVersion) || !strings.Contains(evidence.ToolVersion, page.BrowserName+" "+page.BrowserVersion) {
			return fmt.Errorf("tool version does not match the captured browser observation")
		}
	}
	return nil
}

func localTeachingPageURL(value string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" || parsed.Path == "" || (parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1") {
		return nil, fmt.Errorf("must use an explicit local HTTP teaching-page URL")
	}
	return parsed, nil
}

// IsCurrent reports whether evidence may be presented as current for the
// revision. Missing or elapsed expiry never degrades to a passing result.
func (evidence RuntimeEvidence) IsCurrent(now time.Time) bool {
	return evidence.Status == ArtifactStatusVerified && evidence.ExpiresAt != nil && now.Before(*evidence.ExpiresAt)
}

// IsCurrentFor also requires the frozen verification input fingerprint. Callers
// derive it from the artifact, manuscript contracts, command template, runner
// image, and toolchain; any one of those inputs changing makes old evidence
// unavailable instead of silently reusing a passing result.
func (evidence RuntimeEvidence) IsCurrentFor(inputHash string, now time.Time) bool {
	return isSHA256Digest(inputHash) && evidence.InputHash == inputHash && evidence.IsCurrent(now)
}

// TeachingArtifactEvidenceStaleReason derives currentness from the immutable
// manifest and the currently approved writing contracts. It deliberately does
// not mutate the observation: a past run remains useful provenance even after
// a later contract, code tree, or runner input makes it unsuitable as a
// statement about the current teaching artifact.
func TeachingArtifactEvidenceStaleReason(manifest TeachingArtifactManifest, currentBookContractHash, currentStyleSheetHash string, evidence RuntimeEvidence, now time.Time) string {
	if err := manifest.Validate(); err != nil {
		return "教学工件清单无效，不能将此运行记录作为当前证据。"
	}
	if !isFullSHA256Digest(currentBookContractHash) || !isFullSHA256Digest(currentStyleSheetHash) {
		return "当前教学合同不完整，不能将此运行记录作为当前证据。"
	}
	if evidence.CodeArtifactHash != manifest.ArtifactHash {
		return "代码工件哈希已变化，需重新验证。"
	}
	if manifest.BookContractHash != currentBookContractHash || manifest.StyleSheetHash != currentStyleSheetHash {
		return "BookContract 或 StyleSheet 已变化，需重新验证并重新审阅相关资产。"
	}
	if evidence.ToolchainVersion != manifest.ToolchainVersion || !runtimeToolVersionMatchesRunner(evidence) {
		return "验证工具链或运行器版本与教学工件清单不一致，需重新验证。"
	}
	expectedInputHash, err := TeachingArtifactVerificationInputHash(manifest, evidence.RunnerImageDigest)
	if err != nil || !evidence.IsCurrentFor(expectedInputHash, now) {
		if evidence.ExpiresAt == nil || !now.Before(*evidence.ExpiresAt) {
			return "运行证据已过期，需重新验证后才能作为当前事实使用。"
		}
		return "验证输入已变化，需重新验证。"
	}
	return ""
}

// HasCurrentTeachingArtifactEvidence requires at least one complete observation
// for the exact frozen artifact and teaching contracts. A mutable artifact status
// alone is never publication evidence.
func HasCurrentTeachingArtifactEvidence(manifest TeachingArtifactManifest, bookContractHash, styleSheetHash string, evidence []RuntimeEvidence, now time.Time) bool {
	for _, item := range evidence {
		if item.CodeArtifactID != manifest.ArtifactID || item.Validate() != nil {
			continue
		}
		if TeachingArtifactEvidenceStaleReason(manifest, bookContractHash, styleSheetHash, item, now) == "" {
			return true
		}
	}
	return false
}

func runtimeToolVersionMatchesRunner(evidence RuntimeEvidence) bool {
	expected := "runner-image:" + evidence.RunnerImageDigest
	if evidence.Kind == RuntimeEvidenceBrowserPage {
		return strings.HasSuffix(evidence.ToolVersion, expected)
	}
	return evidence.ToolVersion == expected
}

type AssetKind string

const (
	AssetKindScreenshot AssetKind = "screenshot"
	AssetKindDiagram    AssetKind = "diagram"
	AssetKindRecording  AssetKind = "recording"
)

func (value AssetKind) Validate() error {
	return validateEnum(value, []AssetKind{AssetKindScreenshot, AssetKindDiagram, AssetKindRecording}, "asset kind")
}

// VisualEvidencePurpose states why a screenshot is necessary. Structured
// command output remains the default evidence format; a capture is reserved
// for information whose visual arrangement is material to the claim.
type VisualEvidencePurpose string

const (
	VisualEvidencePurposeLayout      VisualEvidencePurpose = "layout"
	VisualEvidencePurposeMemoryMap   VisualEvidencePurpose = "memory_map"
	VisualEvidencePurposeCallStack   VisualEvidencePurpose = "call_stack"
	VisualEvidencePurposeNetworkFlow VisualEvidencePurpose = "network_flow"
	VisualEvidencePurposeRenderedUI  VisualEvidencePurpose = "rendered_ui"
	// Legacy is read-only compatibility for assets created before a visual
	// purpose was required. New uploads must never use it.
	VisualEvidencePurposeLegacy VisualEvidencePurpose = "legacy_unclassified"
)

func (value VisualEvidencePurpose) Validate() error {
	return validateEnum(value, []VisualEvidencePurpose{VisualEvidencePurposeLayout, VisualEvidencePurposeMemoryMap, VisualEvidencePurposeCallStack, VisualEvidencePurposeNetworkFlow, VisualEvidencePurposeRenderedUI, VisualEvidencePurposeLegacy}, "visual evidence purpose")
}

func (value VisualEvidencePurpose) ValidateForNewCapture() error {
	if value == VisualEvidencePurposeLegacy {
		return fmt.Errorf("new visual evidence cannot be legacy-unclassified")
	}
	return value.Validate()
}

// ManuscriptAsset is a revision-bound visual or recording asset. StableRef is
// the only reference a manuscript may use, so replacing a file cannot silently
// change already-approved teaching material.
type ManuscriptAsset struct {
	ID               string                `json:"id"`
	RevisionID       string                `json:"revision_id"`
	EvidenceID       string                `json:"evidence_id"`
	StableRef        string                `json:"stable_ref"`
	Kind             AssetKind             `json:"kind"`
	ContentHash      string                `json:"content_hash"`
	AltText          string                `json:"alt_text"`
	Source           string                `json:"source"`
	GenerationMethod string                `json:"generation_method"`
	VisualPurpose    VisualEvidencePurpose `json:"visual_purpose,omitempty"`
	RightsStatus     RightsStatus          `json:"rights_status"`
	Status           ArtifactStatus        `json:"status"`
}

func (asset ManuscriptAsset) Validate() error {
	if strings.TrimSpace(asset.ID) == "" || strings.TrimSpace(asset.RevisionID) == "" || strings.TrimSpace(asset.EvidenceID) == "" || strings.TrimSpace(asset.StableRef) == "" || !isSHA256Digest(asset.ContentHash) || strings.TrimSpace(asset.AltText) == "" || strings.TrimSpace(asset.Source) == "" || strings.TrimSpace(asset.GenerationMethod) == "" {
		return fmt.Errorf("manuscript asset identity, provenance, alt text, and content hash are required")
	}
	if err := asset.Kind.Validate(); err != nil {
		return err
	}
	if asset.Kind == AssetKindScreenshot {
		if err := asset.VisualPurpose.Validate(); err != nil {
			return err
		}
	}
	if err := asset.RightsStatus.Validate(); err != nil {
		return err
	}
	return asset.Status.Validate()
}

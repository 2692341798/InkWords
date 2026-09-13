package textbook

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRuntimeEvidenceAndManuscriptAssetsKeepObservationProvenance(t *testing.T) {
	capturedAt := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	expiresAt := capturedAt.Add(24 * time.Hour)
	structuredOutput, err := MarshalRuntimeObservationOutput(map[string]any{"exit_code": 0}, nil)
	require.NoError(t, err)
	evidence := RuntimeEvidence{ID: "evidence-1", RevisionID: "revision-1", CodeArtifactID: "artifact-1", CodeArtifactHash: "sha256:" + strings.Repeat("a", 64), InputHash: "sha256:verification-input", Kind: RuntimeEvidenceTerminalOutput, Status: ArtifactStatusVerified, CommandManifestHash: "sha256:commands", RunnerImageDigest: "sha256:runner", ToolchainVersion: "go1.25.4", ToolName: "bubblewrap", ToolVersion: "runner-image:sha256:runner", SamplingConditions: []string{"network disabled"}, StructuredOutput: string(structuredOutput), RawEvidenceRef: "runtime-evidence:evidence-1:stdout", CapturedAt: capturedAt, ExpiresAt: &expiresAt}
	require.NoError(t, evidence.Validate())
	require.True(t, evidence.IsCurrent(capturedAt.Add(time.Hour)))
	require.False(t, evidence.IsCurrent(expiresAt))
	require.True(t, evidence.IsCurrentFor(evidence.InputHash, capturedAt.Add(time.Hour)))
	require.False(t, evidence.IsCurrentFor("sha256:changed-contract", capturedAt.Add(time.Hour)))

	evidence.ToolVersion = ""
	require.ErrorContains(t, evidence.Validate(), "tool")
	evidence.ToolVersion = "runner-image:sha256:runner"
	evidence.SamplingConditions = nil
	require.ErrorContains(t, evidence.Validate(), "sampling")
	evidence.SamplingConditions = []string{"network disabled"}
	evidence.StructuredOutput = `{"exit_code":0}`
	require.ErrorContains(t, evidence.Validate(), "structured observations")
	evidence.StructuredOutput = string(structuredOutput)

	asset := ManuscriptAsset{ID: "asset-1", RevisionID: "revision-1", EvidenceID: evidence.ID, StableRef: "asset:revision-1:runtime-terminal", Kind: AssetKindScreenshot, ContentHash: "sha256:" + strings.Repeat("b", 64), AltText: "终端显示测试通过与退出码 0", Source: evidence.RawEvidenceRef, GenerationMethod: "manual_capture", VisualPurpose: VisualEvidencePurposeCallStack, RightsStatus: RightsStatusReady, Status: ArtifactStatusUnverified}
	require.NoError(t, asset.Validate())
	asset.VisualPurpose = VisualEvidencePurposeLegacy
	require.Error(t, asset.VisualPurpose.ValidateForNewCapture())

	evidence.RawEvidenceRef = ""
	require.ErrorContains(t, evidence.Validate(), "raw evidence")
}

func TestVerifiedBrowserRuntimeEvidenceRequiresCapturedPlaywrightBundle(t *testing.T) {
	capturedAt := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	expiresAt := capturedAt.Add(24 * time.Hour)
	structuredOutput, err := MarshalRuntimeObservationOutput(map[string]any{
		"browser_pages": []map[string]any{{
			"url":                "http://127.0.0.1:4173/route",
			"final_url":          "http://127.0.0.1:4173/route",
			"browser_name":       "Chromium",
			"browser_version":    "151.0.7922.34",
			"playwright_version": "1.62.1",
			"screenshot_ref":     "visual-asset:sha256:route-page",
			"dom_assertions":     []map[string]string{{"locator": "main h1", "assertion": "has_text", "expected": "路由已注册"}},
			"console":            []map[string]string{},
			"network":            []map[string]any{{"url": "http://127.0.0.1:4173/route", "resource_type": "document", "status": 200}},
		}},
	}, nil)
	require.NoError(t, err)
	evidence := RuntimeEvidence{
		ID: "browser-evidence", RevisionID: "revision-1", CodeArtifactID: "artifact-1",
		CodeArtifactHash: "sha256:" + strings.Repeat("a", 64), InputHash: "sha256:verification-input",
		Kind: RuntimeEvidenceBrowserPage, Status: ArtifactStatusVerified,
		CommandManifestHash: "sha256:commands", RunnerImageDigest: "sha256:runner", ToolchainVersion: "go1.25.4",
		ToolName: "Playwright/Chromium", ToolVersion: "Playwright 1.62.1; Chromium 151.0.7922.34; runner-image:sha256:runner", SamplingConditions: []string{"network disabled outside the local teaching page"},
		StructuredOutput: string(structuredOutput), RawEvidenceRef: "runtime-evidence:browser-evidence:playwright", CapturedAt: capturedAt, ExpiresAt: &expiresAt,
	}

	require.NoError(t, evidence.Validate())

	var missingVersionOutput RuntimeObservationOutput
	require.NoError(t, json.Unmarshal([]byte(structuredOutput), &missingVersionOutput))
	var missingVersionBundle map[string]any
	require.NoError(t, json.Unmarshal(missingVersionOutput.Observations, &missingVersionBundle))
	pages := missingVersionBundle["browser_pages"].([]any)
	delete(pages[0].(map[string]any), "browser_version")
	missingVersion, err := MarshalRuntimeObservationOutput(missingVersionBundle, nil)
	require.NoError(t, err)
	evidence.StructuredOutput = string(missingVersion)
	require.ErrorContains(t, evidence.Validate(), "browser and Playwright versions")
	evidence.StructuredOutput = string(structuredOutput)

	missingBundle, err := MarshalRuntimeObservationOutput(map[string]any{
		"browser_pages": []map[string]any{{"url": "http://127.0.0.1:4173/route", "final_url": "http://127.0.0.1:4173/route", "browser_name": "Chromium", "browser_version": "151.0.7922.34", "playwright_version": "1.62.1"}},
	}, nil)
	require.NoError(t, err)
	evidence.StructuredOutput = string(missingBundle)
	require.ErrorContains(t, evidence.Validate(), "screenshot")
}

func TestTeachingArtifactEvidenceStaleReasonDetectsContractAndRunnerChanges(t *testing.T) {
	now := time.Now().UTC()
	bookHash, styleHash, artifactHash, runnerHash := testDigest('b'), testDigest('c'), testDigest('a'), testDigest('d')
	manifest := TeachingArtifactManifest{Format: TeachingArtifactManifestFormat, ArtifactID: uuid.NewString(), RevisionID: uuid.NewString(), ArtifactHash: artifactHash, BookContractHash: bookHash, StyleSheetHash: styleHash, Language: "go", ToolchainVersion: "go1.25.4", Commands: []VerificationCommand{{Kind: "go_test"}}}
	inputHash, err := TeachingArtifactVerificationInputHash(manifest, runnerHash)
	require.NoError(t, err)
	expiresAt := now.Add(time.Hour)
	structuredOutput, err := MarshalRuntimeObservationOutput(map[string]any{"exit_code": 0}, nil)
	require.NoError(t, err)
	evidence := RuntimeEvidence{ID: "evidence-1", RevisionID: manifest.RevisionID, CodeArtifactID: manifest.ArtifactID, CodeArtifactHash: artifactHash, InputHash: inputHash, Kind: RuntimeEvidenceTerminalOutput, Status: ArtifactStatusVerified, CommandManifestHash: testDigest('m'), RunnerImageDigest: runnerHash, ToolchainVersion: manifest.ToolchainVersion, ToolName: "bubblewrap", ToolVersion: "runner-image:" + runnerHash, SamplingConditions: []string{"network disabled"}, StructuredOutput: string(structuredOutput), RawEvidenceRef: "database:evidence", CapturedAt: now, ExpiresAt: &expiresAt}
	require.Empty(t, TeachingArtifactEvidenceStaleReason(manifest, bookHash, styleHash, evidence, now))
	require.True(t, HasCurrentTeachingArtifactEvidence(manifest, bookHash, styleHash, []RuntimeEvidence{evidence}, now))
	require.False(t, HasCurrentTeachingArtifactEvidence(manifest, bookHash, styleHash, nil, now), "artifact status without an observation is not evidence")
	require.Contains(t, TeachingArtifactEvidenceStaleReason(manifest, testDigest('c'), styleHash, evidence, now), "BookContract")
	evidence.ToolVersion = "runner-image:" + testDigest('e')
	require.Contains(t, TeachingArtifactEvidenceStaleReason(manifest, bookHash, styleHash, evidence, now), "工具链")
	require.False(t, HasCurrentTeachingArtifactEvidence(manifest, bookHash, styleHash, []RuntimeEvidence{evidence}, now))
}

func testDigest(character byte) string { return "sha256:" + strings.Repeat(string(character), 64) }

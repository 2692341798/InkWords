package bootstrap

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	textbookverification "inkwords-backend/services/course-runner/domain/textbookverification"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
	"inkwords-backend/shared/platform/visualasset"
)

func TestTextbookArtifactResolverReloadsStoredManifestAndContentAddressedTree(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&textbookdomain.CodeArtifactRow{}, &textbookdomain.RuntimeEvidenceRow{}))
	store := teachingartifact.NewStore(t.TempDir())
	staged, err := store.Stage(context.Background(), []teachingartifact.File{{Path: "go.mod", Content: []byte("module example.com/demo\n\ngo 1.25.4\n")}, {Path: "router.go", Content: []byte("package demo\n")}})
	require.NoError(t, err)
	artifactID, revisionID := uuid.New(), uuid.New()
	manifest := sharedtextbook.TeachingArtifactManifest{Format: sharedtextbook.TeachingArtifactManifestFormat, ArtifactID: artifactID.String(), RevisionID: revisionID.String(), ArtifactHash: staged.ArtifactHash, BookContractHash: "sha256:" + strings.Repeat("c", 64), StyleSheetHash: "sha256:" + strings.Repeat("d", 64), Language: "go", ToolchainVersion: "go1.25.4", Commands: []sharedtextbook.VerificationCommand{{Kind: "go_test"}}}
	manifestHash, err := sharedtextbook.TeachingArtifactManifestHash(manifest)
	require.NoError(t, err)
	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, database.Create(&textbookdomain.CodeArtifactRow{ID: artifactID, RevisionID: revisionID, Kind: sharedtextbook.CodeArtifactTeachingImplementation, Language: "go", ManifestJSON: manifestJSON, ManifestHash: manifestHash, ArtifactHash: staged.ArtifactHash, LimitationsJSON: []byte(`[]`), Status: sharedtextbook.ArtifactStatusUnverified}).Error)

	resolver := textbookArtifactResolver{db: database, store: store}
	request, err := resolver.Resolve(context.Background(), textbookverification.VerificationPayload{ArtifactID: artifactID.String(), RevisionID: revisionID.String(), ArtifactHash: staged.ArtifactHash, ManifestHash: manifestHash})
	require.NoError(t, err)
	require.Equal(t, manifest, request.Manifest)
	require.NotEmpty(t, request.RootDir)

	_, err = resolver.Resolve(context.Background(), textbookverification.VerificationPayload{ArtifactID: artifactID.String(), RevisionID: revisionID.String(), ArtifactHash: "sha256:" + strings.Repeat("0", 64), ManifestHash: manifestHash})
	require.Error(t, err, "message hash must not select a different tree")
}

func TestTextbookEvidenceStoreRecordsVerifiedEvidenceWithExpiry(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&textbookdomain.ChapterRevision{}, &textbookdomain.CodeArtifactRow{}, &textbookdomain.RuntimeEvidenceRow{}, &textbookdomain.ManuscriptAssetRow{}))
	require.NoError(t, database.AutoMigrate(&coretask.JobTask{}))
	taskID := uuid.New()
	require.NoError(t, database.Create(&coretask.JobTask{ID: taskID, TaskType: "verification", TaskSubtype: textbookverification.TaskSubtype, Status: coretask.JobTaskStatusRunning}).Error)
	artifactID, revisionID := uuid.New(), uuid.New()
	require.NoError(t, database.Create(&textbookdomain.ChapterRevision{ID: revisionID, ChapterID: uuid.New(), RevisionNumber: 1, Kind: sharedtextbook.RevisionKindCandidate, Markdown: "# test", DocumentJSON: []byte(`{}`), ContentHash: strings.Repeat("a", 64), CreatedBy: "system"}).Error)
	hash := "sha256:" + strings.Repeat("a", 64)
	manifestHash := "sha256:" + strings.Repeat("b", 64)
	require.NoError(t, database.Create(&textbookdomain.CodeArtifactRow{ID: artifactID, RevisionID: revisionID, Kind: sharedtextbook.CodeArtifactTeachingImplementation, Language: "go", ManifestJSON: []byte(`{}`), ManifestHash: manifestHash, ArtifactHash: hash, LimitationsJSON: []byte(`[]`), Status: sharedtextbook.ArtifactStatusUnverified}).Error)
	visualStore := visualasset.NewStore(t.TempDir())
	store := textbookEvidenceStore{db: database, visual: visualStore}
	require.NoError(t, store.PersistVerificationReport(context.Background(), taskID, textbookverification.VerificationPayload{ArtifactID: artifactID.String(), RevisionID: revisionID.String(), ArtifactHash: hash, ManifestHash: manifestHash}, textbookverification.Report{Status: sharedtextbook.ArtifactStatusVerified, InputHash: "sha256:" + strings.Repeat("c", 64), RunnerImageDigest: "sha256:" + strings.Repeat("d", 64), ToolchainVersion: "go1.25.4", Results: []textbookverification.CommandResult{
		{Command: sharedtextbook.VerificationCommand{Kind: "go_test"}, Status: sharedtextbook.ArtifactStatusVerified, ExitCode: 0},
		{Command: sharedtextbook.VerificationCommand{Kind: "browser_page", BrowserPath: "/route", ExpectedText: "路由已注册"}, Status: sharedtextbook.ArtifactStatusVerified, Browser: &textbookverification.BrowserPageCapture{
			URL: "http://127.0.0.1:38080/route", FinalURL: "http://127.0.0.1:38080/route", ScreenshotContent: []byte("png-bytes"),
			BrowserName: "Chromium", BrowserVersion: "151.0.7922.34", PlaywrightVersion: "1.62.1",
			DOMAssertions: []textbookverification.BrowserDOMAssertion{{Locator: "main", Assertion: "has_text", Expected: "路由已注册"}},
			Console:       []textbookverification.BrowserConsoleObservation{},
			Network:       []textbookverification.BrowserNetworkObservation{{URL: "http://127.0.0.1:38080/route", ResourceType: "document", Status: 200}},
		}},
	}}))
	var evidenceRows []textbookdomain.RuntimeEvidenceRow
	require.NoError(t, database.Order("kind ASC").Find(&evidenceRows).Error)
	require.Len(t, evidenceRows, 2)
	var evidence textbookdomain.RuntimeEvidenceRow
	for _, row := range evidenceRows {
		require.Equal(t, sharedtextbook.ArtifactStatusVerified, row.Status)
		require.NotNil(t, row.ExpiresAt)
		require.True(t, row.ExpiresAt.After(time.Now().UTC()))
		if row.Kind == sharedtextbook.RuntimeEvidenceBrowserPage {
			evidence = row
		}
	}
	require.Equal(t, sharedtextbook.RuntimeEvidenceBrowserPage, evidence.Kind)
	require.Equal(t, "Playwright/Chromium", evidence.ToolName)
	require.Contains(t, evidence.ToolVersion, "Playwright 1.62.1")
	require.Contains(t, evidence.ToolVersion, "Chromium 151.0.7922.34")
	var output sharedtextbook.RuntimeObservationOutput
	require.NoError(t, json.Unmarshal([]byte(evidence.StructuredOutput), &output))
	require.NoError(t, output.Validate())
	var browserObservation struct {
		BrowserPages []struct {
			ScreenshotRef string `json:"screenshot_ref"`
		} `json:"browser_pages"`
	}
	require.NoError(t, json.Unmarshal(output.Observations, &browserObservation))
	require.Len(t, browserObservation.BrowserPages, 1)
	require.True(t, strings.HasPrefix(browserObservation.BrowserPages[0].ScreenshotRef, "visual-asset:sha256:"))
	_, mediaType, err := visualStore.Resolve(strings.TrimPrefix(browserObservation.BrowserPages[0].ScreenshotRef, "visual-asset:"))
	require.NoError(t, err)
	require.Equal(t, "image/png", mediaType)
	var screenshotAsset textbookdomain.ManuscriptAssetRow
	require.NoError(t, database.First(&screenshotAsset, "evidence_id = ?", evidence.ID).Error)
	require.Equal(t, sharedtextbook.AssetKindScreenshot, screenshotAsset.Kind)
	require.Equal(t, sharedtextbook.RightsStatusPending, screenshotAsset.RightsStatus)
	require.Equal(t, sharedtextbook.ArtifactStatusVerified, screenshotAsset.Status)
	require.NoError(t, (&sharedtextbook.RuntimeEvidence{ID: evidence.ID.String(), RevisionID: evidence.RevisionID.String(), CodeArtifactID: evidence.CodeArtifactID.String(), CodeArtifactHash: evidence.CodeArtifactHash, InputHash: evidence.InputHash, Kind: evidence.Kind, Status: evidence.Status, CommandManifestHash: evidence.CommandManifestHash, RunnerImageDigest: evidence.RunnerImageDigest, ToolchainVersion: evidence.ToolchainVersion, ToolName: evidence.ToolName, ToolVersion: evidence.ToolVersion, SamplingConditions: []string{"network disabled", "generated content-addressed artifact only", "read-only system directories", "temporary workspace"}, StructuredOutput: evidence.StructuredOutput, RawEvidenceRef: evidence.RawEvidenceRef, CapturedAt: *evidence.CapturedAt, ExpiresAt: evidence.ExpiresAt}).Validate())
	require.Empty(t, output.Interpretations, "a passing command is an observation, not a generated mechanism explanation")
	var artifact textbookdomain.CodeArtifactRow
	require.NoError(t, database.First(&artifact, "id = ?", artifactID).Error)
	require.Equal(t, sharedtextbook.ArtifactStatusVerified, artifact.Status)
}

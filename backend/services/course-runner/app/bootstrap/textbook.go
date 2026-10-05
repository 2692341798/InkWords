package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	textbookverification "inkwords-backend/services/course-runner/domain/textbookverification"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
	"inkwords-backend/shared/platform/visualasset"
)

type textbookArtifactResolver struct {
	db    *gorm.DB
	store *teachingartifact.Store
}

func (resolver textbookArtifactResolver) Resolve(ctx context.Context, payload textbookverification.VerificationPayload) (textbookverification.RunRequest, error) {
	if resolver.db == nil || resolver.store == nil || payload.Validate() != nil {
		return textbookverification.RunRequest{}, fmt.Errorf("invalid textbook verification request")
	}
	artifactID, err := uuid.Parse(payload.ArtifactID)
	if err != nil {
		return textbookverification.RunRequest{}, fmt.Errorf("parse textbook artifact id: %w", err)
	}
	revisionID, err := uuid.Parse(payload.RevisionID)
	if err != nil {
		return textbookverification.RunRequest{}, fmt.Errorf("parse textbook revision id: %w", err)
	}
	var artifact textbookdomain.CodeArtifactRow
	if err := resolver.db.WithContext(ctx).Where("id = ? AND revision_id = ? AND artifact_hash = ? AND manifest_hash = ?", artifactID, revisionID, payload.ArtifactHash, payload.ManifestHash).First(&artifact).Error; err != nil {
		return textbookverification.RunRequest{}, fmt.Errorf("load immutable textbook artifact: %w", err)
	}
	if artifact.Kind != sharedtextbook.CodeArtifactTeachingImplementation || artifact.Language != "go" {
		return textbookverification.RunRequest{}, fmt.Errorf("textbook artifact is not an executable generated Go teaching implementation")
	}
	var manifest sharedtextbook.TeachingArtifactManifest
	if err := json.Unmarshal(artifact.ManifestJSON, &manifest); err != nil || manifest.Validate() != nil {
		return textbookverification.RunRequest{}, fmt.Errorf("decode persisted teaching artifact manifest")
	}
	manifestHash, err := sharedtextbook.TeachingArtifactManifestHash(manifest)
	if err != nil || manifestHash != payload.ManifestHash || manifest.ArtifactID != payload.ArtifactID || manifest.RevisionID != payload.RevisionID || manifest.ArtifactHash != payload.ArtifactHash {
		return textbookverification.RunRequest{}, fmt.Errorf("persisted teaching manifest does not match verification request")
	}
	root, err := resolver.store.ResolveWithGoDependencies(payload.ArtifactHash, manifest.DependencyManifestHash)
	if err != nil {
		return textbookverification.RunRequest{}, fmt.Errorf("resolve immutable teaching artifact: %w", err)
	}
	return textbookverification.RunRequest{Manifest: manifest, RootDir: root}, nil
}

type visualEvidenceStager interface {
	Stage(context.Context, string, io.Reader) (visualasset.Artifact, error)
}

type textbookEvidenceStore struct {
	db     *gorm.DB
	visual visualEvidenceStager
}

func (store textbookEvidenceStore) PersistVerificationReport(ctx context.Context, taskID uuid.UUID, payload textbookverification.VerificationPayload, report textbookverification.Report) error {
	if store.db == nil || taskID == uuid.Nil || payload.Validate() != nil {
		return fmt.Errorf("invalid textbook verification evidence")
	}
	artifactID, _ := uuid.Parse(payload.ArtifactID)
	revisionID, _ := uuid.Parse(payload.RevisionID)
	now := time.Now().UTC()
	results := report.Results
	if len(results) == 0 {
		// Keep an unavailable runner observable even though no command began.
		results = []textbookverification.CommandResult{{Command: sharedtextbook.VerificationCommand{Kind: "go_test"}, Status: report.Status, Reason: report.Reason}}
	}
	evidenceRows := make([]textbookdomain.RuntimeEvidenceRow, 0, len(results))
	for index := range results {
		if err := store.stageBrowserScreenshot(ctx, &results[index]); err != nil {
			return err
		}
		result := results[index]
		evidence, err := textbookEvidenceForCommandResult(taskID, revisionID, artifactID, payload, report, result, index, now)
		if err != nil {
			return err
		}
		evidenceRows = append(evidenceRows, evidence)
	}
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task struct {
			Status      string
			TaskSubtype string
		}
		if err := tx.Table("job_tasks").Clauses(clause.Locking{Strength: "UPDATE"}).Select("status, task_subtype").Where("id = ?", taskID).Take(&task).Error; err != nil {
			return fmt.Errorf("lock verification completion: %w", err)
		}
		if task.Status == "cancelled" {
			return textbookverification.ErrVerificationCancelled
		}
		if task.Status == "succeeded" || task.Status == "failed" {
			return textbookverification.ErrVerificationTaskTerminal
		}
		if task.Status != "running" || task.TaskSubtype != textbookverification.TaskSubtype {
			return fmt.Errorf("verification completion requires its running task")
		}
		for _, evidence := range evidenceRows {
			if err := tx.Create(&evidence).Error; err != nil {
				return fmt.Errorf("persist textbook runtime evidence: %w", err)
			}
			if err := persistVerifiedBrowserScreenshotAsset(tx, evidence); err != nil {
				return err
			}
		}
		if err := tx.Model(&textbookdomain.CodeArtifactRow{}).Where("id = ? AND artifact_hash = ?", artifactID, payload.ArtifactHash).Update("status", report.Status).Error; err != nil {
			return fmt.Errorf("update textbook artifact status: %w", err)
		}
		body, err := json.Marshal(report)
		if err != nil {
			return err
		}
		// Evidence and the terminal task become visible together. A concurrent
		// cancellation either wins this row first or observes completed work.
		return tx.Table("job_tasks").Where("id = ?", taskID).Updates(map[string]any{
			"status": "succeeded", "result_json": datatypes.JSON(body), "finished_at": now, "updated_at": now,
		}).Error
	})
}

func persistVerifiedBrowserScreenshotAsset(tx *gorm.DB, evidence textbookdomain.RuntimeEvidenceRow) error {
	if evidence.Kind != sharedtextbook.RuntimeEvidenceBrowserPage || evidence.Status != sharedtextbook.ArtifactStatusVerified || !strings.HasPrefix(evidence.StructuredOutput, "{") {
		return nil
	}
	var output struct {
		Observations struct {
			BrowserPages []struct {
				ScreenshotRef string `json:"screenshot_ref"`
			} `json:"browser_pages"`
		} `json:"observations"`
	}
	if err := json.Unmarshal([]byte(evidence.StructuredOutput), &output); err != nil || len(output.Observations.BrowserPages) != 1 {
		return fmt.Errorf("decode verified browser screenshot reference")
	}
	contentHash := strings.TrimPrefix(output.Observations.BrowserPages[0].ScreenshotRef, "visual-asset:")
	if !strings.HasPrefix(output.Observations.BrowserPages[0].ScreenshotRef, "visual-asset:sha256:") {
		return fmt.Errorf("verified browser screenshot is not content addressed")
	}
	var revision textbookdomain.ChapterRevision
	if err := tx.Select("id", "chapter_id").Where("id = ?", evidence.RevisionID).First(&revision).Error; err != nil {
		return fmt.Errorf("load browser evidence revision: %w", err)
	}
	assetID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("inkwords:playwright-browser:"+evidence.ID.String()+":"+contentHash))
	asset := textbookdomain.ManuscriptAssetRow{
		ID: assetID, RevisionID: evidence.RevisionID, EvidenceID: evidence.ID,
		StableRef: "inkwords-asset:" + evidence.RevisionID.String() + ":browser:" + evidence.ID.String(),
		Kind:      sharedtextbook.AssetKindScreenshot, ContentHash: contentHash,
		AltText:          "受控本地教学页面的 Playwright 运行截图",
		Source:           "Bubblewrap 内 Playwright 受控 loopback 教学页",
		GenerationMethod: "isolated_playwright_probe",
		VisualPurpose:    sharedtextbook.VisualEvidencePurposeRenderedUI,
		RightsStatus:     sharedtextbook.RightsStatusPending,
		Status:           sharedtextbook.ArtifactStatusVerified,
	}
	if err := tx.Create(&asset).Error; err != nil {
		return fmt.Errorf("persist browser screenshot asset: %w", err)
	}
	return nil
}

func (store textbookEvidenceStore) stageBrowserScreenshot(ctx context.Context, result *textbookverification.CommandResult) error {
	if result == nil || result.Command.Kind != "browser_page" || result.Browser == nil || len(result.Browser.ScreenshotContent) == 0 {
		return nil
	}
	if store.visual == nil {
		return fmt.Errorf("browser screenshot storage is not configured")
	}
	artifact, err := store.visual.Stage(ctx, "image/png", bytes.NewReader(result.Browser.ScreenshotContent))
	if err != nil {
		return fmt.Errorf("stage browser screenshot: %w", err)
	}
	result.Browser.ScreenshotContent = nil
	result.Browser.ScreenshotRef = "visual-asset:" + artifact.Token
	return nil
}

func textbookEvidenceForCommandResult(taskID, revisionID, artifactID uuid.UUID, payload textbookverification.VerificationPayload, report textbookverification.Report, result textbookverification.CommandResult, index int, capturedAt time.Time) (textbookdomain.RuntimeEvidenceRow, error) {
	status := result.Status
	if status == "" {
		status = report.Status
	}
	if err := status.Validate(); err != nil {
		return textbookdomain.RuntimeEvidenceRow{}, fmt.Errorf("invalid textbook command evidence status: %w", err)
	}
	kind := sharedtextbook.RuntimeEvidenceTerminalOutput
	toolName := "bubblewrap"
	samplingConditions := []string{"network disabled", "generated content-addressed artifact only", "read-only system directories", "temporary workspace", "sandbox profile:" + sharedtextbook.LearnerSandboxProfileDigest}
	var observations any = map[string]any{
		"status":    status,
		"reason":    commandEvidenceReason(result, report),
		"command":   result.Command,
		"exit_code": result.ExitCode,
		"output":    result.Output,
	}
	if result.Command.Kind == "browser_page" {
		kind = sharedtextbook.RuntimeEvidenceBrowserPage
		toolName = "Playwright/Chromium"
		samplingConditions = append(samplingConditions, "browser requests restricted to the local teaching page")
		if result.Browser == nil {
			observations = map[string]any{"browser_pages": []textbookverification.BrowserPageCapture{}}
		} else {
			observations = map[string]any{"browser_pages": []textbookverification.BrowserPageCapture{*result.Browser}}
		}
	}
	toolVersion := "runner-image:" + report.RunnerImageDigest
	if kind == sharedtextbook.RuntimeEvidenceBrowserPage && result.Browser != nil {
		toolVersion = fmt.Sprintf("Playwright %s; %s %s; runner-image:%s", result.Browser.PlaywrightVersion, result.Browser.BrowserName, result.Browser.BrowserVersion, report.RunnerImageDigest)
	}
	structured, err := sharedtextbook.MarshalRuntimeObservationOutput(observations, nil)
	if err != nil {
		return textbookdomain.RuntimeEvidenceRow{}, fmt.Errorf("marshal textbook command evidence output: %w", err)
	}
	samplingConditionsJSON, err := json.Marshal(samplingConditions)
	if err != nil {
		return textbookdomain.RuntimeEvidenceRow{}, fmt.Errorf("marshal textbook evidence sampling conditions: %w", err)
	}
	evidence := textbookdomain.RuntimeEvidenceRow{
		RevisionID:             revisionID,
		CodeArtifactID:         artifactID,
		CodeArtifactHash:       payload.ArtifactHash,
		InputHash:              reportInputHash(payload, report),
		Kind:                   kind,
		Status:                 status,
		CommandManifestHash:    payload.ManifestHash,
		RunnerImageDigest:      report.RunnerImageDigest,
		ToolchainVersion:       report.ToolchainVersion,
		ToolName:               toolName,
		ToolVersion:            toolVersion,
		SamplingConditionsJSON: datatypes.JSON(samplingConditionsJSON),
		StructuredOutput:       string(structured),
		RawEvidenceRef:         "database:textbook_runtime_evidence:task:" + taskID.String() + ":command:" + fmt.Sprint(index),
		OutputTruncated:        strings.Contains(result.Output, "[output truncated]"),
		CapturedAt:             &capturedAt,
	}
	if status == sharedtextbook.ArtifactStatusVerified {
		expires := capturedAt.Add(30 * 24 * time.Hour)
		evidence.ExpiresAt = &expires
	} else {
		evidence.StaleReason = commandEvidenceReason(result, report)
		if evidence.StaleReason == "" {
			evidence.StaleReason = "隔离执行器未产生可验证的运行结果。"
		}
	}
	return evidence, nil
}

func commandEvidenceReason(result textbookverification.CommandResult, report textbookverification.Report) string {
	if reason := strings.TrimSpace(result.Reason); reason != "" {
		return reason
	}
	return strings.TrimSpace(report.Reason)
}

func reportInputHash(payload textbookverification.VerificationPayload, report textbookverification.Report) string {
	if strings.HasPrefix(report.InputHash, "sha256:") {
		return report.InputHash
	}
	// An unavailable sandbox has no image digest to bind. Record a deterministic
	// negative-observation fingerprint rather than fabricate a runnable image.
	sum := sha256.Sum256([]byte(payload.ArtifactHash + "\x00" + payload.ManifestHash + "\x00" + report.Reason))
	return "sha256:" + hex.EncodeToString(sum[:])
}

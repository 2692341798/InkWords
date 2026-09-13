package textbook

import (
	"fmt"
	"strings"
)

// DemonstrationToolRecommendation records a tool choice for a human-operated
// recording. It is advice, not a capability declaration: InkWords never drives
// the selected IDE or browser from this contract.
type DemonstrationToolRecommendation struct {
	Primary               string `json:"primary"`
	Alternative           string `json:"alternative"`
	Reason                string `json:"reason"`
	ManualCaptureRequired bool   `json:"manual_capture_required"`
}

func (recommendation DemonstrationToolRecommendation) Validate() error {
	if strings.TrimSpace(recommendation.Primary) == "" || strings.TrimSpace(recommendation.Alternative) == "" || strings.TrimSpace(recommendation.Reason) == "" {
		return fmt.Errorf("demonstration tool recommendation is incomplete")
	}
	return nil
}

// VideoRunbookStep is one recordable action for a second-screen video script.
// Every field is intentional: a step must be executable, narratable, and
// recoverable without turning a reader's local project into a test fixture.
type VideoRunbookStep struct {
	Tool             string `json:"tool"`
	ToolVersion      string `json:"tool_version"`
	StartState       string `json:"start_state"`
	Action           string `json:"action"`
	ShortcutOrMenu   string `json:"shortcut_or_menu"`
	Input            string `json:"input"`
	ExpectedView     string `json:"expected_view"`
	Narration        string `json:"narration"`
	CapturePoint     string `json:"capture_point"`
	Recovery         string `json:"recovery"`
	CompletionSignal string `json:"completion_signal"`
}

func (step VideoRunbookStep) Validate() error {
	if strings.TrimSpace(step.Tool) == "" || strings.TrimSpace(step.ToolVersion) == "" || strings.TrimSpace(step.StartState) == "" || strings.TrimSpace(step.Action) == "" || strings.TrimSpace(step.ShortcutOrMenu) == "" || strings.TrimSpace(step.Input) == "" || strings.TrimSpace(step.ExpectedView) == "" || strings.TrimSpace(step.Narration) == "" || strings.TrimSpace(step.CapturePoint) == "" || strings.TrimSpace(step.Recovery) == "" || strings.TrimSpace(step.CompletionSignal) == "" {
		return fmt.Errorf("video runbook step is incomplete")
	}
	return nil
}

// VideoRunbookProjection is an immutable, revision-bound filming guide. Its
// capture checklist deliberately remains pending until a human records the
// evidence with the stated tool and sampling conditions.
type VideoRunbookProjection struct {
	Format               string                          `json:"format"`
	Stack                string                          `json:"stack"`
	ObservationGoal      string                          `json:"observation_goal"`
	Recommendation       DemonstrationToolRecommendation `json:"recommendation"`
	Steps                []VideoRunbookStep              `json:"steps"`
	CaptureChecklist     []string                        `json:"capture_checklist"`
	ManualCapturePending bool                            `json:"manual_capture_pending"`
	VerificationStatus   ArtifactStatus                  `json:"verification_status"`
}

func (projection VideoRunbookProjection) Validate() error {
	if projection.Format != "inkwords.video-runbook.v1" || strings.TrimSpace(projection.Stack) == "" || strings.TrimSpace(projection.ObservationGoal) == "" || len(projection.Steps) == 0 || len(projection.CaptureChecklist) == 0 {
		return fmt.Errorf("video runbook projection is incomplete")
	}
	if err := projection.Recommendation.Validate(); err != nil {
		return err
	}
	if err := projection.VerificationStatus.Validate(); err != nil {
		return err
	}
	for _, step := range projection.Steps {
		if err := step.Validate(); err != nil {
			return err
		}
	}
	for _, item := range projection.CaptureChecklist {
		if strings.TrimSpace(item) == "" {
			return fmt.Errorf("video runbook capture checklist contains an empty item")
		}
	}
	return nil
}

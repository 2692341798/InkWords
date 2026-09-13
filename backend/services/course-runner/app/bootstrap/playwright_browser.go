package bootstrap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	textbookverification "inkwords-backend/services/course-runner/domain/textbookverification"
	verification "inkwords-backend/services/course-runner/domain/verification"
	"inkwords-backend/shared/platform/visualasset"
)

// playwrightBrowserExecutor delegates only to the Bubblewrap-owned probe. It
// cannot accept a remote URL or execute a task-provided JavaScript snippet.
type playwrightBrowserExecutor struct {
	sandbox verification.BubblewrapExecutor
	config  verification.BrowserProbeConfig
}

func (executor playwrightBrowserExecutor) Capture(ctx context.Context, rootDir string, command textbookverification.CommandTemplate, _ time.Duration) (textbookverification.BrowserPageCapture, error) {
	if command.Kind != "browser_page" || command.Validate() != nil {
		return textbookverification.BrowserPageCapture{}, fmt.Errorf("invalid browser-page command")
	}
	exitCode, output, err := executor.sandbox.ExecuteBrowserProbe(ctx, rootDir, command.BrowserPath, command.ExpectedText, executor.config)
	if err != nil {
		return textbookverification.BrowserPageCapture{}, fmt.Errorf("run isolated Playwright probe: %w", err)
	}
	if exitCode != 0 {
		return textbookverification.BrowserPageCapture{}, fmt.Errorf("isolated Playwright probe exited with code %d: %s", exitCode, boundedProbeError(output))
	}
	return decodePlaywrightBrowserCapture(output)
}

func decodePlaywrightBrowserCapture(output string) (textbookverification.BrowserPageCapture, error) {
	var raw struct {
		URL               string                                           `json:"url"`
		FinalURL          string                                           `json:"final_url"`
		BrowserName       string                                           `json:"browser_name"`
		BrowserVersion    string                                           `json:"browser_version"`
		PlaywrightVersion string                                           `json:"playwright_version"`
		ScreenshotBase64  string                                           `json:"screenshot_base64"`
		DOMAssertions     []textbookverification.BrowserDOMAssertion       `json:"dom_assertions"`
		Console           []textbookverification.BrowserConsoleObservation `json:"console"`
		Network           []textbookverification.BrowserNetworkObservation `json:"network"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &raw); err != nil {
		return textbookverification.BrowserPageCapture{}, fmt.Errorf("decode isolated Playwright observation: %w", err)
	}
	screenshot, err := base64.StdEncoding.DecodeString(raw.ScreenshotBase64)
	if err != nil || len(screenshot) == 0 || len(screenshot) > int(visualasset.MaxImageBytes) {
		return textbookverification.BrowserPageCapture{}, fmt.Errorf("isolated Playwright screenshot is invalid")
	}
	if !boundedRuntimeVersion(raw.BrowserName, 32) || !boundedRuntimeVersion(raw.BrowserVersion, 64) || !boundedRuntimeVersion(raw.PlaywrightVersion, 32) {
		return textbookverification.BrowserPageCapture{}, fmt.Errorf("isolated Playwright browser and tool version are required")
	}
	return textbookverification.BrowserPageCapture{
		URL: raw.URL, FinalURL: raw.FinalURL, BrowserName: raw.BrowserName, BrowserVersion: raw.BrowserVersion, PlaywrightVersion: raw.PlaywrightVersion, ScreenshotContent: screenshot,
		DOMAssertions: raw.DOMAssertions, Console: raw.Console, Network: raw.Network,
	}, nil
}

func boundedRuntimeVersion(value string, limit int) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > limit {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func boundedProbeError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 1024 {
		return value[:1024] + " [truncated]"
	}
	return value
}

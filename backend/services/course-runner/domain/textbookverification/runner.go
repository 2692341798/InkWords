package textbookverification

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
)

var (
	ErrRunnerNotConfigured = errors.New("textbook isolated executor is not configured")
	bearerCredential       = regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)[^\s,;]+`)
	namedCredential        = regexp.MustCompile(`(?i)\b((?:api[_-]?key|access[_-]?token|refresh[_-]?token|password|secret)\s*[:=]\s*)[^\s,;]+`)
)

type CommandExecutor interface {
	Execute(ctx context.Context, rootDir, command string, timeout time.Duration, env []string) (exitCode int, output string, err error)
}

// BrowserPageExecutor is a deliberately narrow adapter for a reviewed browser
// runtime. It receives the immutable artifact tree and a validated local-page
// command, never a caller-provided URL, script, environment, or shell string.
type BrowserPageExecutor interface {
	Capture(ctx context.Context, rootDir string, command CommandTemplate, timeout time.Duration) (BrowserPageCapture, error)
}

type RunRequest struct {
	Manifest ArtifactManifest
	RootDir  string
}

type Runner struct {
	Executor          CommandExecutor
	BrowserExecutor   BrowserPageExecutor
	Timeout           time.Duration
	RunnerImageDigest string
	// ToolchainVersion comes from the operator's runner image, never the manuscript manifest.
	ToolchainVersion string
}

// Verify fails closed: every problem is represented as unverified evidence so
// no caller can upgrade an unavailable sandbox into a passing textbook claim.
func (runner Runner) Verify(ctx context.Context, request RunRequest) (report Report) {
	defer func() { report.ToolchainVersion = runner.ToolchainVersion }()
	if err := request.Manifest.Validate(); err != nil {
		return unverifiedReport(request.Manifest, runner.RunnerImageDigest, err.Error())
	}
	for _, command := range request.Manifest.Commands {
		if command.Kind == "go_test" && runner.Executor == nil {
			return unverifiedCommandReport(request.Manifest, runner.RunnerImageDigest, command, ErrRunnerNotConfigured.Error())
		}
	}
	inputHash, err := VerificationInputHash(request.Manifest, runner.RunnerImageDigest)
	if err != nil {
		return unverifiedReport(request.Manifest, runner.RunnerImageDigest, err.Error())
	}
	if request.Manifest.DependencyManifestHash != "" && runner.ToolchainVersion != request.Manifest.ToolchainVersion {
		return unverifiedReport(request.Manifest, runner.RunnerImageDigest, "dependency profile requires the exact runner toolchain")
	}
	root, cleanup, err := teachingartifact.VerifiedManifestSnapshot(ctx, request.RootDir, request.Manifest)
	if err != nil {
		return unverifiedReport(request.Manifest, runner.RunnerImageDigest, err.Error())
	}
	defer cleanup()
	actualHash := request.Manifest.ArtifactHash
	timeout := runner.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	report = Report{Status: sharedtextbook.ArtifactStatusUnverified, InputHash: inputHash, ArtifactHash: actualHash, RunnerImageDigest: runner.RunnerImageDigest, Results: make([]CommandResult, 0, len(request.Manifest.Commands))}
	for _, command := range request.Manifest.Commands {
		started := time.Now()
		execCtx, cancel := context.WithTimeout(ctx, timeout)
		if command.Kind == "browser_page" {
			if runner.BrowserExecutor == nil {
				cancel()
				report.Reason = ErrRunnerNotConfigured.Error() + ": browser-page executor"
				report.Results = append(report.Results, CommandResult{Command: command, Status: sharedtextbook.ArtifactStatusUnverified, Reason: report.Reason, Duration: time.Since(started)})
				return report
			}
			capture, executeErr := runner.BrowserExecutor.Capture(execCtx, root, command, timeout)
			cancel()
			if executeErr != nil {
				report.Reason = fmt.Sprintf("browser-page command failed: %v", executeErr)
				report.Results = append(report.Results, CommandResult{Command: command, Status: sharedtextbook.ArtifactStatusUnverified, Browser: &capture, Reason: report.Reason, Duration: time.Since(started)})
				return report
			}
			report.Results = append(report.Results, CommandResult{Command: command, Status: sharedtextbook.ArtifactStatusVerified, Browser: &capture, Duration: time.Since(started)})
			continue
		}
		if runner.Executor == nil {
			cancel()
			report.Reason = ErrRunnerNotConfigured.Error()
			report.Results = append(report.Results, CommandResult{Command: command, Status: sharedtextbook.ArtifactStatusUnverified, Reason: report.Reason, Duration: time.Since(started)})
			return report
		}
		exitCode, output, executeErr := runner.Executor.Execute(execCtx, root, command.ExecutableForm(), timeout, nil)
		cancel()
		result := CommandResult{Command: command, Status: sharedtextbook.ArtifactStatusVerified, ExitCode: exitCode, Output: redactTerminalOutput(output), Duration: time.Since(started)}
		if executeErr != nil || exitCode != 0 {
			report.Reason = commandFailure(command, exitCode, executeErr)
			result.Status = sharedtextbook.ArtifactStatusUnverified
			result.Reason = report.Reason
			report.Results = append(report.Results, result)
			return report
		}
		report.Results = append(report.Results, result)
	}
	report.Status = sharedtextbook.ArtifactStatusVerified
	return report
}

// redactTerminalOutput protects the durable evidence record from common
// environment-style credentials emitted by generated code or a toolchain.
// It deliberately preserves surrounding diagnostics so a learner can still
// understand a failed command without receiving the credential value.
func redactTerminalOutput(output string) string {
	output = bearerCredential.ReplaceAllString(output, "${1}[REDACTED]")
	return namedCredential.ReplaceAllString(output, "${1}[REDACTED]")
}

func unverifiedReport(manifest ArtifactManifest, runnerImageDigest, reason string) Report {
	return Report{Status: sharedtextbook.ArtifactStatusUnverified, ArtifactHash: manifest.ArtifactHash, RunnerImageDigest: runnerImageDigest, Reason: reason}
}

func unverifiedCommandReport(manifest ArtifactManifest, runnerImageDigest string, command CommandTemplate, reason string) Report {
	report := unverifiedReport(manifest, runnerImageDigest, reason)
	report.Results = []CommandResult{{Command: command, Status: sharedtextbook.ArtifactStatusUnverified, Reason: reason}}
	return report
}

// ArtifactTreeHash delegates to the same content-addressed tree algorithm used
// when core-api stages generated teaching artifacts.
func ArtifactTreeHash(root string) (string, error) {
	return teachingartifact.TreeHash(root)
}

func commandFailure(command CommandTemplate, exitCode int, err error) string {
	if err != nil {
		return fmt.Sprintf("command template %q failed: %v", command.Kind, err)
	}
	return fmt.Sprintf("command template %q exited with code %d", command.Kind, exitCode)
}

package verification

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrRunnerNotConfigured = errors.New("isolated command executor is not configured")
	ErrUnsafeCommand       = errors.New("command is not in the lab allowlist")
	ErrUnsafeRoot          = errors.New("lab root directory is invalid")
)

type CommandExecutor interface {
	Execute(ctx context.Context, rootDir, command string, timeout time.Duration, env []string) (exitCode int, output string, err error)
}

type Runner struct {
	Executor CommandExecutor
	Timeout  time.Duration
}

func (r Runner) Verify(ctx context.Context, request RunRequest) Report {
	if r.Executor == nil {
		return Report{Error: ErrRunnerNotConfigured.Error()}
	}
	if err := validateRoot(request.RootDir); err != nil {
		return Report{Error: err.Error()}
	}
	if err := request.Manifest.Validate(); err != nil {
		return Report{Error: err.Error()}
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	results := make([]CommandResult, 0, len(request.Manifest.AllowedCommands))
	for _, command := range request.Manifest.AllowedCommands {
		if err := validateCommand(command); err != nil {
			return Report{Results: results, Error: err.Error()}
		}
		started := time.Now()
		exitCode, output, err := r.Executor.Execute(ctx, request.RootDir, command, timeout, nil)
		result := CommandResult{Command: command, ExitCode: exitCode, Output: output, Duration: time.Since(started)}
		results = append(results, result)
		if err != nil || exitCode != 0 {
			return Report{Passed: false, Results: results, Error: commandFailure(command, exitCode, err)}
		}
	}
	return Report{Passed: true, Results: results}
}

func validateRoot(root string) error {
	if strings.TrimSpace(root) == "" || !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) {
		return ErrUnsafeRoot
	}
	return nil
}

func validateCommand(command string) error {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" || strings.ContainsAny(trimmed, ";|&`$\n\r") {
		return ErrUnsafeCommand
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || parts[0] != "test" {
		return ErrUnsafeCommand
	}
	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "-") && part != "-run" {
			return ErrUnsafeCommand
		}
	}
	return nil
}

func commandFailure(command string, exitCode int, err error) string {
	if err != nil {
		return fmt.Sprintf("command %q failed: %v", command, err)
	}
	return fmt.Sprintf("command %q exited with code %d", command, exitCode)
}

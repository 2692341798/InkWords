package verification

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BubblewrapExecutor runs generated Go labs in Linux namespaces. It never
// mounts the host Docker socket and never receives a target repository path.
type BubblewrapExecutor struct {
	Binary         string
	MaxOutputBytes int
}

func (e BubblewrapExecutor) Execute(ctx context.Context, rootDir, command string, _ time.Duration, _ []string) (int, string, error) {
	if strings.TrimSpace(e.Binary) == "" {
		return -1, "", errors.New("bubblewrap binary is not configured")
	}
	if err := validateRoot(rootDir); err != nil {
		return -1, "", err
	}
	if err := validateCommand(command); err != nil {
		return -1, "", err
	}
	workspace, err := os.MkdirTemp("", "inkwords-course-lab-")
	if err != nil {
		return -1, "", fmt.Errorf("create temporary lab workspace: %w", err)
	}
	defer func() { _ = os.RemoveAll(workspace) }()
	if err := copyArtifactTree(rootDir, workspace); err != nil {
		return -1, "", err
	}
	// Binary is operator configuration, while command and all subprocess arguments are allowlisted above.
	cmd := exec.CommandContext(ctx, e.Binary, buildBubblewrapArgs(workspace, command)...) //nolint:gosec
	output := limitedBuffer{limit: e.MaxOutputBytes}
	if output.limit <= 0 {
		output.limit = 1 << 20
	}
	cmd.Stdout = &output
	cmd.Stderr = &output
	err = cmd.Run()
	if ctx.Err() != nil {
		return -1, output.String(), ctx.Err()
	}
	if err == nil {
		return 0, output.String(), nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), output.String(), nil
	}
	return -1, output.String(), err
}

func buildBubblewrapArgs(workspace, command string) []string {
	args := []string{"--unshare-all", "--die-with-parent", "--new-session", "--rlimit-as", "268435456:268435456", "--rlimit-nproc", "64:64", "--rlimit-fsize", "10485760:10485760", "--rlimit-cpu", "30:30", "--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp", "--bind", workspace, "/workspace", "--chdir", "/workspace", "--clearenv", "--setenv", "HOME", "/tmp", "--setenv", "PATH", "/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin", "--setenv", "GOCACHE", "/tmp/go-cache", "--setenv", "GOMODCACHE", "/tmp/gomod-cache", "--setenv", "GOPROXY", "off", "--setenv", "GOSUMDB", "off"}
	for _, dir := range []string{"/usr", "/bin", "/lib", "/lib64", "/etc"} {
		if _, err := os.Stat(dir); err == nil {
			args = append(args, "--ro-bind", dir, dir)
		}
	}
	args = append(args, "--", "go", "test", "./...")
	parts := strings.Fields(command)
	if len(parts) == 3 && parts[1] == "-run" {
		args = append(args, "-run", parts[2])
	}
	return args
}

type limitedBuffer struct {
	bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = b.Buffer.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	return b.Buffer.Write(p)
}

func (b *limitedBuffer) String() string {
	value := b.Buffer.String()
	if b.truncated {
		return value + "\n[output truncated]"
	}
	return value
}

func copyArtifactTree(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in course artifact is not allowed: %s", relative)
		}
		target := filepath.Join(destination, relative)
		if info.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported course artifact file: %s", relative)
		}
		// path is supplied by filepath.Walk under the validated artifact root.
		input, err := os.Open(path) //nolint:gosec
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			_ = input.Close()
			return err
		}
		// target is derived from filepath.Rel and remains inside the fresh temporary workspace.
		output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) //nolint:gosec
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		inputCloseErr := input.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		return inputCloseErr
	})
}

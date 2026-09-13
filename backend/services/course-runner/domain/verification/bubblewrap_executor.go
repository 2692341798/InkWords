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
	"runtime"
	"strconv"
	"strings"
	"time"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
)

var (
	ErrUnsafeCommand = errors.New("command is not in the lab allowlist")
	ErrUnsafeRoot    = errors.New("lab root directory is invalid")
)

// BubblewrapExecutor runs generated Go labs in Linux namespaces. It never
// mounts the host Docker socket and never receives a target repository path.
type BubblewrapExecutor struct {
	Binary         string
	MaxOutputBytes int
}

// BrowserProbeConfig identifies operator-owned runtime files. These paths are
// never derived from a manuscript or task payload and are mounted read-only in
// the same Bubblewrap namespace as the generated teaching page.
type BrowserProbeConfig struct {
	NodeBinary       string
	ProbeScript      string
	ModuleDirectory  string
	BrowserDirectory string
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
	if len(parts) == 1 {
		return nil
	}
	if len(parts) != 3 || parts[1] != "-run" || strings.TrimSpace(parts[2]) == "" {
		return ErrUnsafeCommand
	}
	for _, char := range parts[2] {
		if !isAllowedTestPatternRune(char) {
			return ErrUnsafeCommand
		}
	}
	return nil
}

func isAllowedTestPatternRune(char rune) bool {
	return char == '_' || char == '-' || char == '.' || char == '*' || char == '+' || char == '^' || char == '$' || char == '(' || char == ')' ||
		char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9'
}

// Preflight proves that this host can create the namespaces required by the
// executor before a queue consumer is enabled. It intentionally runs no
// teaching artifact and never relaxes isolation when a desktop container
// runtime rejects user namespaces.
func (e BubblewrapExecutor) Preflight(ctx context.Context) error {
	if strings.TrimSpace(e.Binary) == "" {
		return errors.New("bubblewrap binary is not configured")
	}
	if runtime.GOOS != "linux" {
		return errors.New("bubblewrap textbook verification requires Linux")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	preflightCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	command, cleanup, err := sandboxCommand(preflightCtx, e.Binary, buildBubblewrapPreflightArgs())
	if err != nil {
		return fmt.Errorf("prepare bubblewrap isolation preflight: %w", err)
	}
	defer cleanup()
	output := limitedBuffer{limit: 4096}
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		if preflightCtx.Err() != nil {
			return fmt.Errorf("bubblewrap isolation preflight: %w", preflightCtx.Err())
		}
		message := strings.TrimSpace(output.String())
		if message == "" {
			return fmt.Errorf("bubblewrap isolation preflight: %w", err)
		}
		return fmt.Errorf("bubblewrap isolation preflight: %s", message)
	}
	return nil
}

// PreflightLearnerGoTest proves the complete accepted profile with a fixed,
// operator-owned Go module. It exercises compilation, resource limits, the
// read-only source mount, the private cache and both seccomp layers without
// reading or executing any learner submission.
func (e BubblewrapExecutor) PreflightLearnerGoTest(ctx context.Context) error {
	if strings.TrimSpace(e.Binary) == "" {
		return errors.New("bubblewrap binary is not configured")
	}
	if runtime.GOOS != "linux" {
		return errors.New("bubblewrap learner verification requires Linux")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	root, err := os.MkdirTemp("", "inkwords-learner-preflight-")
	if err != nil {
		return fmt.Errorf("create fixed learner preflight: %w", err)
	}
	defer func() { _ = os.RemoveAll(root) }()
	for name, content := range map[string]string{
		"go.mod": "module inkwords.local/preflight\n\ngo 1.26\n",
		"add.go": "package preflight\n\nfunc add(left, right int) int { return left + right }\n",
		"add_test.go": `package preflight

import (
	"errors"
	"net"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestFixedSandbox(t *testing.T) {
	const rlimitNPROC = 6
	if add(2, 3) != 5 {
		t.Fatal("unexpected sum")
	}
	if os.Getuid() != 65534 || os.Getgid() != 65534 {
		t.Fatalf("sandbox identity = %d:%d", os.Getuid(), os.Getgid())
	}
	if err := os.WriteFile("mutation", []byte("blocked"), 0600); !errors.Is(err, syscall.EROFS) {
		t.Fatalf("source workspace must be read-only, got %v", err)
	}
	if _, err := os.Stat("/app/course-runner"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("host application path must be hidden, got %v", err)
	}
	connection, err := net.DialTimeout("tcp", "1.1.1.1:53", 250*time.Millisecond)
	if err == nil {
		connection.Close()
		t.Fatal("external network unexpectedly reachable")
	}
	limits := map[int]uint64{
		syscall.RLIMIT_DATA:  402653184,
		rlimitNPROC:          64,
		syscall.RLIMIT_FSIZE: 67108864,
		syscall.RLIMIT_CPU:   30,
	}
	for resource, expected := range limits {
		var limit syscall.Rlimit
		if err := syscall.Getrlimit(resource, &limit); err != nil || limit.Cur != expected || limit.Max != expected {
			t.Fatalf("resource %d limit = %d/%d, expected %d: %v", resource, limit.Cur, limit.Max, expected, err)
		}
	}
	_, _, errno := syscall.RawSyscall(syscall.SYS_UNSHARE, uintptr(syscall.CLONE_NEWUSER), 0, 0)
	if errno != syscall.EPERM {
		t.Fatalf("nested namespace must be rejected with EPERM, got %v", errno)
	}
}
`,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			return fmt.Errorf("write fixed learner preflight: %w", err)
		}
	}
	exitCode, output, err := e.ExecuteLearnerGoTest(ctx, root, time.Duration(sharedtextbook.DefaultLearnerGoTestPolicy().TimeoutMillis)*time.Millisecond)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		message := strings.TrimSpace(output)
		if message == "" {
			message = fmt.Sprintf("exit code %d", exitCode)
		}
		return errors.New(message)
	}
	return nil
}

// PreflightTeachingGoTest exercises the exact generated-artifact command path
// with fixed operator-owned source before its consumer is enabled.
func (e BubblewrapExecutor) PreflightTeachingGoTest(ctx context.Context) error {
	root, err := os.MkdirTemp("", "inkwords-teaching-preflight-")
	if err != nil {
		return fmt.Errorf("create fixed teaching preflight: %w", err)
	}
	defer func() { _ = os.RemoveAll(root) }()
	for name, content := range map[string]string{
		"go.mod":      "module inkwords.local/teaching-preflight\n\ngo 1.26\n",
		"add.go":      "package preflight\n\nfunc add(left, right int) int { return left + right }\n",
		"add_test.go": "package preflight\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) { if add(2, 3) != 5 { t.Fatal(\"unexpected sum\") } }\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			return fmt.Errorf("write fixed teaching preflight: %w", err)
		}
	}
	exitCode, output, err := e.Execute(ctx, root, "test", 30*time.Second, nil)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		message := strings.TrimSpace(output)
		if message == "" {
			message = fmt.Sprintf("exit code %d", exitCode)
		}
		return errors.New(message)
	}
	return nil
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
	// bwrap changes into an unprivileged namespace user. The copied artifact is
	// read-only in that namespace, but its directory must remain traversable by
	// the verifier user; it never exposes the original host artifact tree.
	if err := os.Chmod(workspace, 0o755); err != nil {
		return -1, "", fmt.Errorf("prepare temporary lab workspace permissions: %w", err)
	}
	if err := teachingartifact.CopyBoundedTree(ctx, rootDir, workspace); err != nil {
		return -1, "", err
	}
	// Binary is operator configuration, while command and all subprocess arguments are allowlisted above.
	cmd, cleanup, err := sandboxCommand(ctx, e.Binary, buildBubblewrapArgs(workspace, command))
	if err != nil {
		return -1, "", fmt.Errorf("prepare lab sandbox: %w", err)
	}
	defer cleanup()
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

// ExecuteLearnerGoTest accepts no command or environment from the caller. It
// runs the separately approved learner profile with module/network access off,
// a read-only source tree, fresh caches and no test-result cache reuse.
func (e BubblewrapExecutor) ExecuteLearnerGoTest(ctx context.Context, rootDir string, _ time.Duration) (int, string, error) {
	if strings.TrimSpace(e.Binary) == "" {
		return -1, "", errors.New("bubblewrap binary is not configured")
	}
	if err := validateRoot(rootDir); err != nil {
		return -1, "", err
	}
	workspace, err := os.MkdirTemp("", "inkwords-learner-lab-")
	if err != nil {
		return -1, "", fmt.Errorf("create temporary learner workspace: %w", err)
	}
	defer func() { _ = os.RemoveAll(workspace) }()
	if err := os.Chmod(workspace, 0o755); err != nil {
		return -1, "", fmt.Errorf("prepare temporary learner workspace permissions: %w", err)
	}
	if err := teachingartifact.CopyBoundedTree(ctx, rootDir, workspace); err != nil {
		return -1, "", err
	}
	cmd, cleanup, err := sandboxCommand(ctx, e.Binary, buildBubblewrapLearnerArgs(workspace))
	if err != nil {
		return -1, "", fmt.Errorf("prepare learner sandbox: %w", err)
	}
	defer cleanup()
	output := limitedBuffer{limit: e.MaxOutputBytes}
	if output.limit <= 0 {
		output.limit = 1 << 20
	}
	cmd.Stdout, cmd.Stderr = &output, &output
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

// ExecuteBrowserProbe starts the fixed Playwright probe inside the same
// network-isolated namespace as a temporary copy of the generated teaching
// artifact. The only accepted browser address is the probe's hard-coded
// loopback server; no user URL, shell fragment, environment, or host path is
// accepted here.
func (e BubblewrapExecutor) ExecuteBrowserProbe(ctx context.Context, rootDir, browserPath, expectedText string, config BrowserProbeConfig) (int, string, error) {
	if strings.TrimSpace(e.Binary) == "" {
		return -1, "", errors.New("bubblewrap binary is not configured")
	}
	if err := validateRoot(rootDir); err != nil {
		return -1, "", err
	}
	if err := config.Validate(); err != nil {
		return -1, "", err
	}
	if !safeBrowserProbePath(browserPath) || strings.TrimSpace(expectedText) == "" || len(expectedText) > 256 {
		return -1, "", errors.New("invalid browser probe command")
	}
	workspace, err := os.MkdirTemp("", "inkwords-browser-lab-")
	if err != nil {
		return -1, "", fmt.Errorf("create temporary browser workspace: %w", err)
	}
	defer func() { _ = os.RemoveAll(workspace) }()
	if err := os.Chmod(workspace, 0o755); err != nil {
		return -1, "", fmt.Errorf("prepare temporary browser workspace permissions: %w", err)
	}
	if err := teachingartifact.CopyBoundedTree(ctx, rootDir, workspace); err != nil {
		return -1, "", err
	}
	cmd, cleanup, err := sandboxCommand(ctx, e.Binary, buildBubblewrapBrowserArgs(workspace, browserPath, expectedText, config))
	if err != nil {
		return -1, "", fmt.Errorf("prepare browser sandbox: %w", err)
	}
	defer cleanup()
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

// PreflightBrowser exercises the operator-owned probe and a fixed local page.
// Teaching verification must remain disabled when Chromium cannot retain its
// own process sandbox inside the Bubblewrap boundary.
func (e BubblewrapExecutor) PreflightBrowser(ctx context.Context, config BrowserProbeConfig) error {
	root, err := os.MkdirTemp("", "inkwords-browser-preflight-")
	if err != nil {
		return fmt.Errorf("create fixed browser preflight: %w", err)
	}
	defer func() { _ = os.RemoveAll(root) }()
	for name, content := range map[string]string{
		"go.mod": "module inkwords.local/browser-preflight\n\ngo 1.26\n",
		"main.go": `package main

import "net/http"

func main() {
	http.HandleFunc("/route-registration", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = writer.Write([]byte("<!doctype html><main><h1>路由已注册</h1></main>"))
	})
	_ = http.ListenAndServe("127.0.0.1:38080", nil)
}
`,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			return fmt.Errorf("write fixed browser preflight: %w", err)
		}
	}
	exitCode, output, err := e.ExecuteBrowserProbe(ctx, root, "/route-registration", "路由已注册", config)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		message := strings.TrimSpace(output)
		if len(message) > 2048 {
			message = message[:2048] + " [truncated]"
		}
		if message == "" {
			message = fmt.Sprintf("exit code %d", exitCode)
		}
		return errors.New(message)
	}
	return nil
}

func buildBubblewrapArgs(workspace, command string) []string {
	// The executor copies a content-addressed artifact into a fresh directory
	// before entering the namespace. Mount that copy read-only: Go's compiler
	// cache and all other scratch state belong under the isolated /tmp tmpfs,
	// so a generated test cannot mutate the artifact it is proving.
	args := []string{"--unshare-all", "--die-with-parent", "--new-session", "--uid", "65534", "--gid", "65534", "--dir", "/proc", "--dev", "/dev", "--tmpfs", "/tmp", "--ro-bind", workspace, "/workspace", "--chdir", "/workspace", "--clearenv"}
	for _, item := range goSandboxEnvironment() {
		name, value, _ := strings.Cut(item, "=")
		args = append(args, "--setenv", name, value)
	}
	for _, dir := range []string{"/usr", "/bin", "/lib", "/lib64", "/etc"} {
		if _, err := os.Stat(dir); err == nil {
			args = append(args, "--ro-bind", dir, dir)
		}
	}
	goCommand := []string{"go", "test", "./..."}
	parts := strings.Fields(command)
	if len(parts) == 3 && parts[1] == "-run" {
		goCommand = append(goCommand, "-run", parts[2])
	}
	return append(args, append([]string{"--"}, limitedCommand(402653184, 64, 67108864, 30, goCommand)...)...)
}

func buildBubblewrapLearnerArgs(workspace string) []string {
	policy := sharedtextbook.DefaultLearnerGoTestPolicy()
	args := []string{"--unshare-all", "--die-with-parent", "--new-session", "--uid", strconv.Itoa(policy.UID), "--gid", strconv.Itoa(policy.GID), "--dir", "/proc", "--dev", "/dev", "--tmpfs", "/tmp", "--ro-bind", workspace, "/workspace", "--chdir", "/workspace", "--clearenv"}
	for _, item := range policy.Environment {
		name, value, _ := strings.Cut(item, "=")
		args = append(args, "--setenv", name, value)
	}
	for _, dir := range []string{"/usr", "/bin", "/lib", "/lib64", "/etc"} {
		if _, err := os.Stat(dir); err == nil {
			args = append(args, "--ro-bind", dir, dir)
		}
	}
	return append(args, append([]string{"--"}, limitedCommand(policy.MemoryBytes, policy.PIDs, policy.FileBytes, policy.CPUSeconds, policy.Command)...)...)
}

func buildBubblewrapPreflightArgs() []string {
	// Keep this narrower than a lab invocation: the only executable is the
	// image's fixed /bin/true. The namespace, non-root identity, read-only
	// system mounts and private tmpfs match the properties a real lab needs.
	args := []string{"--unshare-all", "--die-with-parent", "--new-session", "--uid", "65534", "--gid", "65534", "--dir", "/proc", "--dev", "/dev", "--tmpfs", "/tmp", "--chdir", "/tmp", "--clearenv", "--setenv", "PATH", "/usr/local/bin:/usr/bin:/bin"}
	for _, dir := range []string{"/usr", "/bin", "/lib", "/lib64", "/etc"} {
		if _, err := os.Stat(dir); err == nil {
			args = append(args, "--ro-bind", dir, dir)
		}
	}
	return append(args, "--", "/bin/true")
}

func buildBubblewrapBrowserArgs(workspace, browserPath, expectedText string, config BrowserProbeConfig) []string {
	args := []string{"--unshare-all", "--die-with-parent", "--new-session", "--uid", "65534", "--gid", "65534", "--dir", "/proc", "--dev", "/dev", "--tmpfs", "/tmp", "--ro-bind", workspace, "/workspace", "--chdir", "/workspace", "--dir", "/runner", "--ro-bind", config.NodeBinary, "/runner/node", "--ro-bind", config.ProbeScript, "/runner/probe.mjs", "--ro-bind", config.ModuleDirectory, "/runner/node_modules", "--ro-bind", config.BrowserDirectory, "/ms-playwright", "--clearenv"}
	for _, item := range goSandboxEnvironment() {
		name, value, _ := strings.Cut(item, "=")
		args = append(args, "--setenv", name, value)
	}
	args = append(args, "--setenv", "PLAYWRIGHT_BROWSERS_PATH", "/ms-playwright")
	for _, dir := range []string{"/usr", "/bin", "/lib", "/lib64", "/etc"} {
		if _, err := os.Stat(dir); err == nil {
			args = append(args, "--ro-bind", dir, dir)
		}
	}
	command := []string{"/runner/node", "/runner/probe.mjs", "--base-url", "http://127.0.0.1:38080", "--path", browserPath, "--expected-text", expectedText}
	return append(args, append([]string{"--"}, limitedCommand(536870912, 96, 67108864, 30, command)...)...)
}

func goSandboxEnvironment() []string {
	return []string{"CGO_ENABLED=0", "GOCACHE=/tmp/go-cache", "GOFLAGS=-p=1", "GOMAXPROCS=1", "GOMODCACHE=/tmp/gomod-cache", "GOPATH=/tmp/go-path", "GOPROXY=off", "GOROOT=/usr/local/go", "GOSUMDB=off", "GOTELEMETRY=off", "GO_TELEMETRY_CHILD=2", "GOTOOLCHAIN=local", "GOWORK=off", "HOME=/tmp", "PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"}
}

func limitedCommand(memoryBytes int64, pids int, fileBytes int64, cpuSeconds int, command []string) []string {
	args := []string{"/usr/bin/prlimit", "--data=" + strconv.FormatInt(memoryBytes, 10), "--nproc=" + strconv.Itoa(pids), "--fsize=" + strconv.FormatInt(fileBytes, 10), "--cpu=" + strconv.Itoa(cpuSeconds), "--"}
	return append(args, command...)
}

func sandboxCommand(ctx context.Context, binary string, args []string) (*exec.Cmd, func(), error) {
	filter, err := sandboxSeccompBPF(runtime.GOARCH)
	if err != nil {
		return nil, func() {}, err
	}
	file, err := os.CreateTemp("", "inkwords-sandbox-seccomp-*.bpf")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}
	if err := file.Chmod(0o600); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if _, err := file.Write(filter); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	secured := append([]string{"--seccomp", "3"}, args...)
	command := exec.CommandContext(ctx, binary, secured...) //nolint:gosec // Binary is fixed operator configuration; arguments are built above.
	command.ExtraFiles = []*os.File{file}
	return command, cleanup, nil
}

// Validate rejects any non-absolute or unavailable operator runtime paths
// before a generated artifact is ever copied into a browser namespace.
func (config BrowserProbeConfig) Validate() error {
	for _, value := range []struct {
		name string
		path string
	}{
		{"node binary", config.NodeBinary},
		{"browser probe script", config.ProbeScript},
		{"browser module directory", config.ModuleDirectory},
		{"browser directory", config.BrowserDirectory},
	} {
		if !filepath.IsAbs(value.path) || strings.TrimSpace(value.path) == "" {
			return fmt.Errorf("%s must be an absolute operator path", value.name)
		}
		if _, err := os.Stat(value.path); err != nil {
			return fmt.Errorf("%s is unavailable: %w", value.name, err)
		}
	}
	return nil
}

func safeBrowserProbePath(value string) bool {
	if len(value) == 0 || len(value) > 256 || !strings.HasPrefix(value, "/") || strings.ContainsAny(value, "?#\\") || strings.Contains(value, "//") || strings.Contains(value, "..") {
		return false
	}
	for _, char := range value {
		if char != '/' && char != '-' && char != '_' && char != '.' && (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
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
	return teachingartifact.CopyBoundedTree(context.Background(), source, destination)
}

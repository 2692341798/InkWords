package verification

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestBubblewrapArgsDisableNetworkAndExposeOnlyWorkspace(t *testing.T) {
	args := buildBubblewrapArgs("/tmp/lab", "test -run TestCheckpoint")
	joined := strings.Join(args, " ")
	require.Contains(t, joined, "--unshare-all")
	require.Contains(t, joined, "--uid 65534 --gid 65534")
	require.Contains(t, joined, "--tmpfs /tmp")
	require.Contains(t, joined, "--ro-bind /tmp/lab /workspace")
	require.Contains(t, joined, "--ro-bind /usr /usr")
	require.Contains(t, joined, "--dir /proc")
	require.NotContains(t, joined, "--proc /proc")
	require.Contains(t, joined, "/usr/bin/prlimit --data=402653184 --nproc=64 --fsize=67108864 --cpu=30 -- go test ./...")
	require.NotContains(t, joined, "--share-net")
	require.Contains(t, joined, "--setenv GOCACHE /tmp/go-cache")
	require.Contains(t, joined, "go test ./... -run TestCheckpoint")
}

func TestBubblewrapLearnerArgsAreFixedOfflineAndDoNotReuseTestCache(t *testing.T) {
	args := buildBubblewrapLearnerArgs("/tmp/learner")
	joined := strings.Join(args, " ")
	policy := sharedtextbook.DefaultLearnerGoTestPolicy()
	require.Contains(t, joined, "--unshare-all")
	require.Contains(t, joined, "--uid 65534 --gid 65534")
	require.Contains(t, joined, "--ro-bind /tmp/learner /workspace")
	require.Contains(t, joined, "--setenv GOPROXY off")
	require.Contains(t, joined, "--setenv GOSUMDB off")
	require.Contains(t, joined, "--setenv GOTOOLCHAIN local")
	require.Contains(t, joined, "--setenv GOWORK off")
	require.Contains(t, joined, "--setenv CGO_ENABLED 0")
	require.Contains(t, joined, "--setenv GOROOT /usr/local/go")
	require.Contains(t, joined, "--setenv GO_TELEMETRY_CHILD 2")
	require.Contains(t, joined, "--setenv GOMAXPROCS 1")
	require.Contains(t, joined, "--setenv GOFLAGS -p=1")
	require.Contains(t, joined, "/usr/bin/prlimit --data=402653184 --nproc=64 --fsize=67108864 --cpu=30 -- go test -count=1 -mod=readonly -trimpath ./...")
	require.NotContains(t, joined, "--share-net")
	separator := slices.Index(args, "--")
	require.Greater(t, separator, 0)
	require.Equal(t, policy.Command, args[len(args)-len(policy.Command):])
	for _, item := range policy.Environment {
		name, value, ok := strings.Cut(item, "=")
		require.True(t, ok)
		require.Contains(t, joined, "--setenv "+name+" "+value)
	}
	require.Contains(t, joined, fmt.Sprintf("--uid %d --gid %d", policy.UID, policy.GID))
	require.Contains(t, joined, fmt.Sprintf("--data=%d", policy.MemoryBytes))
	require.Contains(t, joined, fmt.Sprintf("--nproc=%d", policy.PIDs))
	require.Contains(t, joined, fmt.Sprintf("--fsize=%d", policy.FileBytes))
	require.Contains(t, joined, fmt.Sprintf("--cpu=%d", policy.CPUSeconds))
}

func TestBubblewrapPreflightArgsExerciseIsolationWithoutATeachingArtifact(t *testing.T) {
	args := buildBubblewrapPreflightArgs()
	joined := strings.Join(args, " ")
	require.Contains(t, joined, "--unshare-all")
	require.Contains(t, joined, "--uid 65534 --gid 65534")
	require.Contains(t, joined, "--tmpfs /tmp")
	require.Contains(t, joined, "--dir /proc")
	require.NotContains(t, joined, "--proc /proc")
	require.Contains(t, joined, "--clearenv")
	require.Contains(t, joined, "-- /bin/true")
	require.NotContains(t, joined, "/workspace")
	require.NotContains(t, joined, "go test")
}

func TestSandboxSeccompProgramSupportsContainerArchitectures(t *testing.T) {
	for _, architecture := range []string{"arm64", "amd64"} {
		program, err := sandboxSeccompBPF(architecture)
		require.NoError(t, err)
		require.NotEmpty(t, program)
		require.Zero(t, len(program)%8, "classic BPF instructions are eight bytes")
	}
	_, err := sandboxSeccompBPF("unsupported")
	require.ErrorContains(t, err, "unsupported")
}

func TestReviewedOuterSeccompProfileDigestMatchesContract(t *testing.T) {
	profile, err := os.ReadFile("../../seccomp/bubblewrap-outer.json")
	require.NoError(t, err)
	require.Equal(t, sharedtextbook.LearnerSandboxProfileDigest, fmt.Sprintf("sha256:%x", sha256.Sum256(profile)))
}

func TestBubblewrapBrowserArgsKeepProbeAndGeneratedPageInsideOneNamespace(t *testing.T) {
	args := buildBubblewrapBrowserArgs("/tmp/lab", "/route-registration", "路由已注册", BrowserProbeConfig{
		NodeBinary: "/usr/bin/node", ProbeScript: "/app/probe.mjs", ModuleDirectory: "/app/node_modules", BrowserDirectory: "/ms-playwright",
	})
	joined := strings.Join(args, " ")
	require.Contains(t, joined, "--unshare-all")
	require.Contains(t, joined, "--ro-bind /tmp/lab /workspace")
	require.Contains(t, joined, "--ro-bind /app/probe.mjs /runner/probe.mjs")
	require.Contains(t, joined, "--ro-bind /app/node_modules /runner/node_modules")
	require.Contains(t, joined, "--ro-bind /ms-playwright /ms-playwright")
	require.Contains(t, joined, "--setenv PLAYWRIGHT_BROWSERS_PATH /ms-playwright")
	require.Contains(t, joined, "--setenv GOROOT /usr/local/go")
	require.Contains(t, joined, "--setenv GOTELEMETRY off")
	require.Contains(t, joined, "--setenv GO_TELEMETRY_CHILD 2")
	require.Contains(t, joined, "--setenv GOPROXY off")
	require.Contains(t, joined, "--base-url http://127.0.0.1:38080 --path /route-registration --expected-text 路由已注册")
	require.NotContains(t, joined, "--share-net")
}

func TestPlaywrightProbeRequiresChromiumSandbox(t *testing.T) {
	script, err := os.ReadFile("../../playwright-probe/probe.mjs")
	require.NoError(t, err)
	require.Contains(t, string(script), "chromiumSandbox: true")
	require.NotContains(t, string(script), "--no-sandbox")
	require.Contains(t, string(script), "Date.now() + 20_000")
	require.Contains(t, string(script), "chromium_sandbox_unavailable")
	require.Contains(t, string(script), "/proc/self/uid_map")
}

func TestBubblewrapPreflightRejectsMissingBinaryBeforeExecutingAnything(t *testing.T) {
	err := (BubblewrapExecutor{}).Preflight(context.Background())
	require.ErrorContains(t, err, "binary is not configured")
}

func TestCopyArtifactTreeMakesOnlyTheTemporaryCopyReadableByRunnerUser(t *testing.T) {
	source, destination := t.TempDir(), t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(source, "nested"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(source, "nested", "main.go"), []byte("package example\n"), 0o600))

	require.NoError(t, copyArtifactTree(source, destination))
	directory, err := os.Stat(filepath.Join(destination, "nested"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o755), directory.Mode().Perm())
	file, err := os.Stat(filepath.Join(destination, "nested", "main.go"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), file.Mode().Perm())
}

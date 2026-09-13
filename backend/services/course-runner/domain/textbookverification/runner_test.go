package textbookverification

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type fakeExecutor struct {
	commands []string
	exitCode int
	err      error
	output   string
}

func TestReportToolchainComesFromRunnerNotManifest(t *testing.T) {
	manifest := validManifest(digest('a'))
	for _, actual := range []string{"go1.26.8", ""} {
		report := (Runner{ToolchainVersion: actual}).Verify(context.Background(), RunRequest{Manifest: manifest})
		require.Equal(t, actual, report.ToolchainVersion)
		require.Equal(t, sharedtextbook.ArtifactStatusUnverified, report.Status)
	}
}

type fakeBrowserExecutor struct {
	commands []CommandTemplate
	err      error
}

func (executor *fakeBrowserExecutor) Capture(_ context.Context, _ string, command CommandTemplate, _ time.Duration) (BrowserPageCapture, error) {
	executor.commands = append(executor.commands, command)
	return BrowserPageCapture{
		URL: command.BrowserPath, FinalURL: command.BrowserPath, ScreenshotRef: "fixture:browser-page",
		DOMAssertions: []BrowserDOMAssertion{{Locator: "main", Assertion: "has_text", Expected: command.ExpectedText}},
		Console:       []BrowserConsoleObservation{},
		Network:       []BrowserNetworkObservation{{URL: command.BrowserPath, ResourceType: "document", Status: 200}},
	}, executor.err
}

func (executor *fakeExecutor) Execute(_ context.Context, _ string, command string, _ time.Duration, environment []string) (int, string, error) {
	executor.commands = append(executor.commands, command)
	if environment != nil {
		return 1, "environment leaked", nil
	}
	if executor.output != "" {
		return executor.exitCode, executor.output, executor.err
	}
	return executor.exitCode, "{" + `"exit_code":0` + "}", executor.err
}

func TestRunnerVerifiesOnlyMatchingGeneratedArtifact(t *testing.T) {
	root := teachingArtifact(t)
	artifactHash, err := ArtifactTreeHash(root)
	require.NoError(t, err)
	manifest := validManifest(artifactHash)
	manifestHash, err := ManifestHash(manifest)
	require.NoError(t, err)
	manifest.Commands = append(manifest.Commands, CommandTemplate{Kind: "go_test", TestPattern: "TestOther"})
	changedManifestHash, err := ManifestHash(manifest)
	require.NoError(t, err)
	require.NotEqual(t, manifestHash, changedManifestHash)
	manifest.Commands = manifest.Commands[:2]
	executor := &fakeExecutor{}
	report := (Runner{Executor: executor, RunnerImageDigest: digest('b'), ToolchainVersion: "go1.26.8"}).Verify(context.Background(), RunRequest{Manifest: manifest, RootDir: root})
	require.Equal(t, "verified", string(report.Status))
	require.Equal(t, "go1.26.8", report.ToolchainVersion)
	require.Len(t, report.Results, 2)
	require.Equal(t, []string{"test", "test -run TestGreeting"}, executor.commands)
	require.NotEmpty(t, report.InputHash)
}

func TestRunnerFailsClosedForMissingExecutorHashMismatchAndCommandFailure(t *testing.T) {
	root := teachingArtifact(t)
	artifactHash, err := ArtifactTreeHash(root)
	require.NoError(t, err)
	manifest := validManifest(artifactHash)

	report := (Runner{}).Verify(context.Background(), RunRequest{Manifest: manifest, RootDir: root})
	require.Equal(t, "unverified", string(report.Status))
	require.Contains(t, report.Reason, "not configured")

	manifest.ArtifactHash = digest('c')
	executor := &fakeExecutor{}
	report = (Runner{Executor: executor, RunnerImageDigest: digest('b')}).Verify(context.Background(), RunRequest{Manifest: manifest, RootDir: root})
	require.Equal(t, "unverified", string(report.Status))
	require.Contains(t, report.Reason, "does not match")
	require.Empty(t, executor.commands)

	manifest.ArtifactHash = artifactHash
	executor = &fakeExecutor{exitCode: 1, err: errors.New("sandbox denied")}
	report = (Runner{Executor: executor, RunnerImageDigest: digest('b')}).Verify(context.Background(), RunRequest{Manifest: manifest, RootDir: root})
	require.Equal(t, "unverified", string(report.Status))
	require.Len(t, report.Results, 1)
	require.Contains(t, report.Reason, "sandbox denied")
}

func TestArtifactTreeHashRejectsSymlinksAndManifestRejectsShell(t *testing.T) {
	root := teachingArtifact(t)
	require.NoError(t, os.Symlink("main.go", filepath.Join(root, "linked.go")))
	_, err := ArtifactTreeHash(root)
	require.ErrorContains(t, err, "symlink")

	manifest := validManifest(digest('a'))
	manifest.Commands = []CommandTemplate{{Kind: "go_test", TestPattern: "TestA;curl"}}
	require.ErrorContains(t, manifest.Validate(), "unsafe")
}

func TestRunnerRedactsCredentialLikeTerminalOutput(t *testing.T) {
	root := teachingArtifact(t)
	artifactHash, err := ArtifactTreeHash(root)
	require.NoError(t, err)
	manifest := validManifest(artifactHash)
	executor := &fakeExecutor{output: "API_KEY=example-value\nAuthorization: Bearer example-token\ncompile failed"}

	report := (Runner{Executor: executor, RunnerImageDigest: digest('b')}).Verify(context.Background(), RunRequest{Manifest: manifest, RootDir: root})
	require.Len(t, report.Results, 2)
	require.NotContains(t, report.Results[0].Output, "example-value")
	require.NotContains(t, report.Results[0].Output, "example-token")
	require.Contains(t, report.Results[0].Output, "[REDACTED]")
	require.Contains(t, report.Results[0].Output, "compile failed")
}

func TestRunnerCapturesBrowserPageAsASeparateVerifiedCommand(t *testing.T) {
	root := teachingArtifact(t)
	artifactHash, err := ArtifactTreeHash(root)
	require.NoError(t, err)
	manifest := validManifest(artifactHash)
	manifest.Commands = append(manifest.Commands, CommandTemplate{Kind: "browser_page", BrowserPath: "/route", ExpectedText: "路由已注册"})
	browser := &fakeBrowserExecutor{}

	report := (Runner{Executor: &fakeExecutor{}, BrowserExecutor: browser, RunnerImageDigest: digest('b')}).Verify(context.Background(), RunRequest{Manifest: manifest, RootDir: root})

	require.Equal(t, sharedtextbook.ArtifactStatusVerified, report.Status)
	require.Len(t, report.Results, 3)
	require.Equal(t, sharedtextbook.ArtifactStatusVerified, report.Results[2].Status)
	require.NotNil(t, report.Results[2].Browser)
	require.Equal(t, []CommandTemplate{{Kind: "browser_page", BrowserPath: "/route", ExpectedText: "路由已注册"}}, browser.commands)
}

func TestRunnerRecordsAnUnavailableBrowserExecutorAsBrowserEvidence(t *testing.T) {
	root := teachingArtifact(t)
	artifactHash, err := ArtifactTreeHash(root)
	require.NoError(t, err)
	manifest := validManifest(artifactHash)
	manifest.Commands = []CommandTemplate{{Kind: "browser_page", BrowserPath: "/route", ExpectedText: "路由已注册"}}

	report := (Runner{RunnerImageDigest: digest('b')}).Verify(context.Background(), RunRequest{Manifest: manifest, RootDir: root})

	require.Equal(t, sharedtextbook.ArtifactStatusUnverified, report.Status)
	require.Contains(t, report.Reason, "browser-page executor")
	require.Len(t, report.Results, 1)
	require.Equal(t, "browser_page", report.Results[0].Command.Kind)
	require.Equal(t, sharedtextbook.ArtifactStatusUnverified, report.Results[0].Status)
}

func TestRunnerPreservesVerifiedGoEvidenceBeforeUnavailableBrowser(t *testing.T) {
	root := teachingArtifact(t)
	artifactHash, err := ArtifactTreeHash(root)
	require.NoError(t, err)
	manifest := validManifest(artifactHash)
	manifest.Commands = append(manifest.Commands, CommandTemplate{Kind: "browser_page", BrowserPath: "/route", ExpectedText: "路由已注册"})

	report := (Runner{Executor: &fakeExecutor{}, RunnerImageDigest: digest('b')}).Verify(context.Background(), RunRequest{Manifest: manifest, RootDir: root})

	require.Equal(t, sharedtextbook.ArtifactStatusUnverified, report.Status)
	require.Len(t, report.Results, 3)
	require.Equal(t, sharedtextbook.ArtifactStatusVerified, report.Results[0].Status)
	require.Equal(t, sharedtextbook.ArtifactStatusVerified, report.Results[1].Status)
	require.Equal(t, sharedtextbook.ArtifactStatusUnverified, report.Results[2].Status)
	require.Equal(t, "browser_page", report.Results[2].Command.Kind)
}

func teachingArtifact(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/teaching\n\ngo 1.25\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package teaching\n\nfunc Greeting() string { return \"hello\" }\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "main_test.go"), []byte("package teaching\n\nimport \"testing\"\n\nfunc TestGreeting(t *testing.T) { if Greeting() != \"hello\" { t.Fatal(\"unexpected\") } }\n"), 0o600))
	return root
}

func validManifest(artifactHash string) ArtifactManifest {
	return ArtifactManifest{Format: ArtifactManifestFormat, ArtifactID: "11111111-1111-1111-1111-111111111111", RevisionID: "22222222-2222-2222-2222-222222222222", ArtifactHash: artifactHash, BookContractHash: digest('c'), StyleSheetHash: digest('d'), Language: "go", ToolchainVersion: "go1.25.4", Commands: []CommandTemplate{{Kind: "go_test"}, {Kind: "go_test", TestPattern: "TestGreeting"}}}
}

func digest(character byte) string {
	return "sha256:" + string([]byte{
		character, character, character, character, character, character, character, character,
		character, character, character, character, character, character, character, character,
		character, character, character, character, character, character, character, character,
		character, character, character, character, character, character, character, character,
		character, character, character, character, character, character, character, character,
		character, character, character, character, character, character, character, character,
		character, character, character, character, character, character, character, character,
		character, character, character, character, character, character, character, character,
	})
}

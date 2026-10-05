package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyCleanupPreflightIsReadOnlyAndFailsClosed(t *testing.T) {
	contents, err := os.ReadFile("check_legacy_cleanup_ready.sh")
	if err != nil {
		t.Fatalf("read cleanup preflight: %v", err)
	}
	script := string(contents)
	for _, required := range []string{
		"set -Eeuo pipefail",
		"backup directory must be an absolute path",
		"backup directory must stay outside the repository",
		"backup directory does not exist",
		"missing regular backup file",
		"inkwords-local-backup.v2",
		"core restore check did not pass",
		"review restore check did not pass",
		"core dump SHA-256 does not match the manifest",
		"review dump SHA-256 does not match the manifest",
		"pg_restore --list",
		"core dump is not a readable PostgreSQL archive",
		"review dump is not a readable PostgreSQL archive",
		"writer services must be stopped before cleanup preflight",
		"core cleanup-critical counts changed after the verified backup",
		"review cleanup-critical counts changed after the verified backup",
		"tasks without workspace ownership remain",
		"blogs without workspace ownership remain",
		"OAuth rows remain",
		"ProjectCourse rows remain",
		"user prompt setting rows remain",
		"core schema must be exactly migration 23 before cleanup",
		"review schema must be exactly migration 21 before cleanup",
		"review sessions without workspace ownership remain",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("cleanup preflight is missing safety contract %q", required)
		}
	}
	for _, forbidden := range []string{"DROP TABLE", "DROP COLUMN", "DELETE FROM", "docker compose down", "docker compose pause"} {
		if strings.Contains(strings.ToUpper(script), strings.ToUpper(forbidden)) {
			t.Fatalf("cleanup preflight must stay read-only; found %q", forbidden)
		}
	}

	if output, err := exec.Command("bash", "-n", "check_legacy_cleanup_ready.sh").CombinedOutput(); err != nil {
		t.Fatalf("bash syntax check failed: %v\n%s", err, output)
	}
	if output, err := exec.Command("bash", "check_legacy_cleanup_ready.sh", "relative/path").CombinedOutput(); err == nil || !strings.Contains(string(output), "must be an absolute path") {
		t.Fatalf("relative backup must fail before docker access: err=%v output=%s", err, output)
	}
	if output, err := exec.Command("bash", "check_legacy_cleanup_ready.sh", "/").CombinedOutput(); err == nil || !strings.Contains(string(output), "filesystem root") {
		t.Fatalf("filesystem root must fail before docker access: err=%v output=%s", err, output)
	}
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	if output, err := exec.Command("bash", "check_legacy_cleanup_ready.sh", repositoryRoot).CombinedOutput(); err == nil || !strings.Contains(string(output), "must stay outside the repository") {
		t.Fatalf("repository backup must fail before docker access: err=%v output=%s", err, output)
	}
	emptyBackup := t.TempDir()
	if output, err := exec.Command("bash", "check_legacy_cleanup_ready.sh", emptyBackup).CombinedOutput(); err == nil || !strings.Contains(string(output), "missing regular backup file: manifest.txt") {
		t.Fatalf("incomplete backup must fail before docker access: err=%v output=%s", err, output)
	}
}

func TestLegacyCleanupPreflightRejectsRunningWriters(t *testing.T) {
	backup := writeCleanupPreflightFixture(t)
	fakeBin := writeFakeDocker(t)
	command := exec.Command("bash", "check_legacy_cleanup_ready.sh", backup)
	command.Env = append(os.Environ(),
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_RUNNING_SERVICES=db\\ncore-api\\n",
	)
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "writer services must be stopped before cleanup preflight: core-api") {
		t.Fatalf("running writer must fail before archive/database checks: err=%v output=%s", err, output)
	}
}

func TestLegacyCleanupPreflightAcceptsMatchingReadOnlyFixture(t *testing.T) {
	backup := writeCleanupPreflightFixture(t)
	fakeBin := writeFakeDocker(t)
	command := exec.Command("bash", "check_legacy_cleanup_ready.sh", backup)
	command.Env = append(os.Environ(),
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_RUNNING_SERVICES=db\\n",
	)
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "verified backup and cleanup-critical counts match") {
		t.Fatalf("matching read-only fixture must be ready: err=%v output=%s", err, output)
	}
}

func writeCleanupPreflightFixture(t *testing.T) string {
	t.Helper()
	backup := t.TempDir()
	coreDump := []byte("core archive fixture")
	reviewDump := []byte("review archive fixture")
	coreSHA := fmt.Sprintf("%x", sha256.Sum256(coreDump))
	reviewSHA := fmt.Sprintf("%x", sha256.Sum256(reviewDump))
	files := map[string][]byte{
		"core.dump":                  coreDump,
		"review.dump":                reviewDump,
		"core-source-counts.txt":     []byte("26|1|59|100|200|300|23\n"),
		"core-restored-counts.txt":   []byte("26|1|59|100|200|300|23\n"),
		"review-source-counts.txt":   []byte("0|4|12|-|400|500|21\n"),
		"review-restored-counts.txt": []byte("0|4|12|-|400|500|21\n"),
		"images.txt":                 []byte("fixture image list\n"),
		"manifest.txt":               []byte(fmt.Sprintf("format=inkwords-local-backup.v2\ncore_file=core.dump\ncore_sha256=%s\nreview_file=review.dump\nreview_sha256=%s\nimages_file=images.txt\ncore_restore_check=passed\nreview_restore_check=passed\n", coreSHA, reviewSHA)),
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(backup, name), contents, 0o600); err != nil {
			t.Fatalf("write backup fixture %s: %v", name, err)
		}
	}
	return backup
}

func writeFakeDocker(t *testing.T) string {
	t.Helper()
	fakeBin := t.TempDir()
	fakeDocker := `#!/usr/bin/env bash
set -euo pipefail
args=$*
case "$args" in
  "compose version") exit 0 ;;
  *" config") exit 0 ;;
  *" ps --status running --services") printf '%b' "${FAKE_RUNNING_SERVICES:?}" ;;
  *" pg_restore --list") exit 0 ;;
  *"local_workspaces"*) printf '1|0|0|0|0|0|23\n' ;;
  *"review_sessions WHERE workspace_id IS NULL"*) printf '0|21\n' ;;
  *" -d inkwords_review_db -At"*) printf '0|4|12|-|400|500|21\n' ;;
  *" -At"*) printf '26|1|59|100|200|300|23\n' ;;
  *) printf 'unexpected fake docker call: %s\n' "$args" >&2; exit 97 ;;
esac
`
	path := filepath.Join(fakeBin, "docker")
	if err := os.WriteFile(path, []byte(fakeDocker), 0o755); err != nil {
		t.Fatalf("write fake docker: %v", err)
	}
	return fakeBin
}

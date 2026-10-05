package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupLocalScriptIsSyntacticallyValidAndFailsClosed(t *testing.T) {
	contents, err := os.ReadFile("backup_local.sh")
	if err != nil {
		t.Fatalf("read backup script: %v", err)
	}
	script := string(contents)
	for _, required := range []string{
		"set -Eeuo pipefail",
		"backup directory must be an absolute path",
		"backup directory must stay outside the repository",
		"backup directory must be empty",
		"umask 077",
		"pg_dump",
		"pg_restore",
		"--exit-on-error",
		"core database dump is empty",
		"review database dump is empty",
		"images_file=images.txt",
		"sql/backup_core_snapshot.sql",
		"sql/backup_review_snapshot.sql",
		"core_restore_check=passed",
		"review_restore_check=passed",
		"format=inkwords-local-backup.v2",
		"trap cleanup EXIT",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("backup script is missing safety contract %q", required)
		}
	}
	coreSQL, err := os.ReadFile(filepath.Join("sql", "backup_core_snapshot.sql"))
	if err != nil {
		t.Fatalf("read core backup snapshot SQL: %v", err)
	}
	reviewSQL, err := os.ReadFile(filepath.Join("sql", "backup_review_snapshot.sql"))
	if err != nil {
		t.Fatalf("read review backup snapshot SQL: %v", err)
	}
	for _, required := range []string{"COUNT(*)::text FROM job_tasks", "COUNT(*)::text FROM textbook_projects", "COUNT(*)::text FROM blogs", "EXTRACT(EPOCH FROM MAX(updated_at))", "inkwords_core_schema_migrations"} {
		if !strings.Contains(string(coreSQL), required) {
			t.Fatalf("core snapshot SQL is missing %q", required)
		}
	}
	for _, required := range []string{"COUNT(*)::text FROM mastery_objectives", "COUNT(*)::text FROM review_sessions", "COUNT(*)::text FROM review_turns", "EXTRACT(EPOCH FROM MAX(updated_at))", "inkwords_review_schema_migrations"} {
		if !strings.Contains(string(reviewSQL), required) {
			t.Fatalf("review snapshot SQL is missing %q", required)
		}
	}

	if output, err := exec.Command("bash", "-n", "backup_local.sh").CombinedOutput(); err != nil {
		t.Fatalf("bash syntax check failed: %v\n%s", err, output)
	}
	if output, err := exec.Command("bash", "backup_local.sh", "relative/path").CombinedOutput(); err == nil || !strings.Contains(string(output), "must be an absolute path") {
		t.Fatalf("relative target must fail before docker access: err=%v output=%s", err, output)
	}
	if output, err := exec.Command("bash", "backup_local.sh", "/").CombinedOutput(); err == nil || !strings.Contains(string(output), "filesystem root") {
		t.Fatalf("filesystem root target must fail before docker access: err=%v output=%s", err, output)
	}
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	if output, err := exec.Command("bash", "backup_local.sh", repositoryRoot).CombinedOutput(); err == nil || !strings.Contains(string(output), "must stay outside the repository") {
		t.Fatalf("repository target must fail before docker access: err=%v output=%s", err, output)
	}
}

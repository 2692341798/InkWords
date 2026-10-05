package services_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTextbookPDFRuntimeProfileOnlyPermitsChromiumSandboxSetup prevents an
// accidental broad seccomp exception from being shipped as a PDF fix.
func TestTextbookPDFRuntimeProfileOnlyPermitsChromiumSandboxSetup(t *testing.T) {
	readProfile := func(path string) map[string]any {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var profile map[string]any
		require.NoError(t, json.Unmarshal(data, &profile))
		return profile
	}
	profile := readProfile(filepath.Join("export-service", "seccomp", "chromium-outer.json"))
	reference := readProfile(filepath.Join("course-runner", "seccomp", "bubblewrap-outer.json"))
	rules := profile["syscalls"].([]any)
	referenceRules := reference["syscalls"].([]any)
	last := rules[len(rules)-1].(map[string]any)
	require.Equal(t, "SCMP_ACT_ALLOW", last["action"])
	require.ElementsMatch(t, []any{"clone", "setns", "unshare", "chroot"}, last["names"])
	profile["syscalls"] = rules[:len(rules)-1]
	reference["syscalls"] = referenceRules[:len(referenceRules)-1]
	require.Equal(t, reference, profile, "preserve every upstream Moby rule; do not copy the Bubblewrap mount exception")

	compose, err := os.ReadFile(filepath.Join("..", "..", "docker-compose.textbook-pdf.yml"))
	require.NoError(t, err)
	for _, control := range []string{
		"TEXTBOOK_CHROMIUM_BIN: /usr/lib/chromium/chromium", "read_only: true",
		"user: \"10001:10001\"", "cap_drop:\n      - ALL", "no-new-privileges:true",
		"seccomp=./backend/services/export-service/seccomp/chromium-outer.json",
		"pids_limit: 192", "mem_limit: 1g", "cpus: 1.5", "size=256m",
	} {
		require.Contains(t, string(compose), control)
	}
	for _, forbidden := range []string{"privileged:", "cap_add:", "unconfined", "docker.sock", "--no-sandbox", "--disable-setuid-sandbox", "network_mode: host", "pid: host", "ipc: host"} {
		require.NotContains(t, string(compose), forbidden)
	}
}

func TestTextbookExporterUsesTheTeachingArtifactReadGroup(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docker-compose.yml"))
	require.NoError(t, err)
	text := string(data)
	start := strings.Index(text, "\n  export-service:")
	require.Positive(t, start)
	end := strings.Index(text[start:], "\n  course-runner:")
	require.Positive(t, end)
	service := text[start : start+end]
	require.Contains(t, service, "group_add:\n      - \"20001\"")
	require.Contains(t, service, "teaching-artifacts:/app/teaching-artifacts:ro")
}

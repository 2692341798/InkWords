package textbook

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func frozenVerificationFixture(t *testing.T) BookVerificationSnapshot {
	t.Helper()
	hash := "sha256:" + strings.Repeat("a", 64)
	now := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	m := TeachingArtifactManifest{Format: TeachingArtifactManifestFormat, ArtifactID: uuid.NewString(), RevisionID: uuid.NewString(), ArtifactHash: hash, BookContractHash: hash, StyleSheetHash: hash, Language: "go", ToolchainVersion: "go1.26.8", Commands: []VerificationCommand{{Kind: "go_test"}}}
	mh, err := TeachingArtifactManifestHash(m)
	require.NoError(t, err)
	ih, err := TeachingArtifactVerificationInputHash(m, hash)
	require.NoError(t, err)
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	output, err := MarshalRuntimeObservationOutput(map[string]any{"command": m.Commands[0], "exit_code": 0, "status": "verified", "output": "ok teaching"}, nil)
	require.NoError(t, err)
	expires := now.Add(time.Hour)
	e := RuntimeEvidence{ID: uuid.NewString(), RevisionID: m.RevisionID, CodeArtifactID: m.ArtifactID, CodeArtifactHash: hash, InputHash: ih, Kind: RuntimeEvidenceTerminalOutput, Status: ArtifactStatusVerified, CommandManifestHash: mh, RunnerImageDigest: hash, ToolchainVersion: m.ToolchainVersion, ToolName: "bubblewrap", ToolVersion: "runner-image:" + hash, SamplingConditions: []string{"network disabled"}, StructuredOutput: string(output), RawEvidenceRef: "database:run:1", CapturedAt: now.Add(-time.Minute), ExpiresAt: &expires}
	return BookVerificationSnapshot{Format: BookVerificationSnapshotFormat, CapturedAt: now, BookContractHash: hash, StyleSheetHash: hash, Artifacts: []FrozenBookVerificationArtifact{{ID: m.ArtifactID, RevisionID: m.RevisionID, ArtifactHash: hash, ManifestHash: mh, Manifest: raw, Evidence: []RuntimeEvidence{e}}}}
}

func TestBookVerificationSnapshotKeepsHistoricalFactsButExpires(t *testing.T) {
	s := frozenVerificationFixture(t)
	require.NoError(t, s.Validate())
	require.True(t, s.Assess(s.CapturedAt).Passed)
	before, _ := json.Marshal(s)
	expired := s.Assess(s.CapturedAt.Add(2 * time.Hour))
	require.False(t, expired.Passed)
	require.NotEmpty(t, expired.Artifacts[0].Reasons)
	after, _ := json.Marshal(s)
	require.Equal(t, string(before), string(after))
	for _, mutate := range []func(*BookVerificationSnapshot){
		func(s *BookVerificationSnapshot) { s.Artifacts[0].Evidence = nil },
		func(s *BookVerificationSnapshot) { s.Artifacts[0].Evidence[0].RevisionID = uuid.NewString() },
		func(s *BookVerificationSnapshot) {
			s.Artifacts[0].Evidence[0].InputHash = "sha256:" + strings.Repeat("b", 64)
		},
		func(s *BookVerificationSnapshot) {
			s.Artifacts[0].Evidence[0].CommandManifestHash = "sha256:" + strings.Repeat("b", 64)
		},
		func(s *BookVerificationSnapshot) { s.Artifacts[0].Evidence[0].Status = ArtifactStatusBlocked },
		func(s *BookVerificationSnapshot) { s.Artifacts[0].Evidence[0].Kind = RuntimeEvidenceImage },
		func(s *BookVerificationSnapshot) { s.Artifacts[0].Evidence[0].ExpiresAt = nil },
		func(s *BookVerificationSnapshot) {
			s.Artifacts[0].Evidence[0].CapturedAt = s.CapturedAt.Add(time.Second)
		},
		func(s *BookVerificationSnapshot) {
			s.Artifacts[0].Evidence[0].StructuredOutput = `{"format":"inkwords.runtime-observation.v1","observations":{"command":{"kind":"go_test","test_pattern":"Other"},"exit_code":0,"status":"verified"}}`
		},
		func(s *BookVerificationSnapshot) { s.BookContractHash = "sha256:" + strings.Repeat("c", 64) },
	} {
		changed := frozenVerificationFixture(t)
		mutate(&changed)
		require.False(t, changed.Assess(changed.CapturedAt).Passed)
	}
}

func TestBookVerificationSnapshotReaderRejectsOmittedAndForeignArtifacts(t *testing.T) {
	s := frozenVerificationFixture(t)
	raw, err := json.Marshal(map[string]any{"format": "inkwords.book-build.v1", "code_artifacts": s.Artifacts, "verification_snapshot": s})
	require.NoError(t, err)
	got, err := ReadBookVerificationSnapshot(raw)
	require.NoError(t, err)
	require.NotNil(t, got)
	legacy, err := ReadBookVerificationSnapshot([]byte(`{"format":"inkwords.book-build.v1"}`))
	require.NoError(t, err)
	require.Nil(t, legacy)
	for _, raw := range []string{`{"verification_snapshot":null}`, `{"verification_snapshot":{"format":"future"}}`, `{"verification_snapshot":{}}`} {
		_, err := ReadBookVerificationSnapshot([]byte(raw))
		require.Error(t, err)
	}
	raw, err = json.Marshal(map[string]any{"code_artifacts": []any{}, "verification_snapshot": s})
	require.NoError(t, err)
	_, err = ReadBookVerificationSnapshot(raw)
	require.Error(t, err)
}

func TestBookVerificationSnapshotRequiresEveryCommandAndBrowserObservation(t *testing.T) {
	s := frozenVerificationFixture(t)
	a := &s.Artifacts[0]
	var m TeachingArtifactManifest
	require.NoError(t, json.Unmarshal(a.Manifest, &m))
	m.Commands = append(m.Commands, VerificationCommand{Kind: "browser_page", BrowserPath: "/route", ExpectedText: "路由已注册"})
	var err error
	a.Manifest, err = json.Marshal(m)
	require.NoError(t, err)
	a.ManifestHash, err = TeachingArtifactManifestHash(m)
	require.NoError(t, err)
	a.Evidence[0].CommandManifestHash = a.ManifestHash
	a.Evidence[0].InputHash, err = TeachingArtifactVerificationInputHash(m, a.Evidence[0].RunnerImageDigest)
	require.NoError(t, err)
	require.False(t, s.Assess(s.CapturedAt).Passed, "Go success cannot cover a browser command")
	e := a.Evidence[0]
	e.ID = uuid.NewString()
	e.Kind = RuntimeEvidenceBrowserPage
	e.ToolName = "Playwright/Chromium"
	e.ToolVersion = "Playwright 1.62.1; Chromium 151; runner-image:" + e.RunnerImageDigest
	output, err := MarshalRuntimeObservationOutput(map[string]any{"browser_pages": []map[string]any{{"url": "http://127.0.0.1:4173/route", "final_url": "http://127.0.0.1:4173/route", "browser_name": "Chromium", "browser_version": "151", "playwright_version": "1.62.1", "screenshot_ref": "visual-asset:sha256:page", "dom_assertions": []map[string]string{{"locator": "text=路由已注册", "assertion": "has_text", "expected": "路由已注册"}}, "console": []any{}, "network": []map[string]any{{"url": "http://127.0.0.1:4173/route", "resource_type": "document", "status": 200}}}}}, nil)
	require.NoError(t, err)
	e.StructuredOutput = string(output)
	require.NoError(t, e.Validate())
	a.Evidence = append(a.Evidence, e)
	require.True(t, s.Assess(s.CapturedAt).Passed)
	a.Evidence[1].StructuredOutput = strings.ReplaceAll(e.StructuredOutput, "/route", "/other")
	require.False(t, s.Assess(s.CapturedAt).Passed)
}

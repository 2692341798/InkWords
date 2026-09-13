package export

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	shared "inkwords-backend/shared/kernel/textbook"
	"strings"
	"testing"
	"time"
)

func TestFrozenBookVerificationSummaryCarriesReceiptsAndLegacyAbsence(t *testing.T) {
	now := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	hash := "sha256:" + strings.Repeat("a", 64)
	a := shared.FrozenBookVerificationArtifact{ID: uuid.NewString(), RevisionID: uuid.NewString(), ArtifactHash: hash, ManifestHash: hash, Manifest: json.RawMessage(`{}`), Evidence: []shared.RuntimeEvidence{}}
	s := shared.BookVerificationSnapshot{Format: shared.BookVerificationSnapshotFormat, CapturedAt: now, BookContractHash: hash, StyleSheetHash: hash, Artifacts: []shared.FrozenBookVerificationArtifact{a}}
	raw, err := json.Marshal(map[string]any{"code_artifacts": s.Artifacts, "verification_snapshot": s})
	require.NoError(t, err)
	build := TextbookBookBuildExport{BuildID: uuid.New(), ManifestHash: hash, ManifestJSON: raw}
	out, err := frozenBookVerificationSummary(build, now.Add(time.Minute))
	require.NoError(t, err)
	var report struct {
		Snapshot shared.BookVerificationSnapshot   `json:"snapshot"`
		AtFreeze shared.BookVerificationAssessment `json:"at_freeze"`
		AtExport shared.BookVerificationAssessment `json:"at_export"`
	}
	require.NoError(t, json.Unmarshal(out, &report))
	require.Equal(t, s, report.Snapshot)
	require.False(t, report.AtFreeze.Passed)
	require.False(t, report.AtExport.Passed)
	require.Contains(t, string(out), build.BuildID.String())
	require.Contains(t, string(out), hash)
	build.ManifestJSON = json.RawMessage(`{"format":"inkwords.book-build.v1"}`)
	out, err = frozenBookVerificationSummary(build, now)
	require.NoError(t, err)
	require.Contains(t, string(out), "unavailable")
	build.ManifestJSON = json.RawMessage(`{"verification_snapshot":null}`)
	_, err = frozenBookVerificationSummary(build, now)
	require.Error(t, err)
}

package textbook

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	shared "inkwords-backend/shared/kernel/textbook"
	"testing"
)

// Run against the existing isolated PostgreSQL fixture; production revisions
// are never edited to manufacture this mutation test.
func testBookBuildFreezesQualityReports(t *testing.T, db *gorm.DB, workspace, project uuid.UUID, build *BookBuildRow) {
	t.Helper()
	snapshot, err := shared.ReadBookQualitySnapshot(json.RawMessage(build.ManifestJSON))
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.True(t, snapshot.Assess().Passed)
	require.Len(t, snapshot.Chapters, 1)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })
	raw := `{"contract_version":"` + shared.SampleQualityContractVersion + `","passed":false,"failures":[{"code":"injected_fixture_failure"}],"manual_review_required":true}`
	require.NoError(t, tx.Table("chapter_revisions").Where("id = ?", snapshot.Chapters[0].RevisionID).Update("quality_report_json", raw).Error)
	checks, err := publicationAutomatedChecks(tx, *build)
	require.NoError(t, err)
	require.Equal(t, shared.QualityStatusPass, checks[2].Status, "frozen quality cannot follow a later report mutation")
	repository := NewGormRepository(tx)
	fresh, err := repository.CreateBookBuild(context.Background(), workspace, CreateBookBuildInput{ProjectID: project})
	require.NoError(t, err)
	require.NotEqual(t, build.ID, fresh.ID, "changed quality reports participate in input identity")
	failed, err := shared.ReadBookQualitySnapshot(json.RawMessage(fresh.ManifestJSON))
	require.NoError(t, err)
	require.False(t, failed.Assess().Passed)
	require.JSONEq(t, raw, string(failed.Chapters[0].Report))
	checks, err = publicationAutomatedChecks(tx, *fresh)
	require.NoError(t, err)
	require.Equal(t, shared.QualityStatusHardFail, checks[2].Status)
	var retained BookBuildRow
	require.NoError(t, tx.First(&retained, "id = ?", build.ID).Error)
	require.JSONEq(t, string(build.ManifestJSON), string(retained.ManifestJSON))
	require.NoError(t, tx.Rollback().Error)
}

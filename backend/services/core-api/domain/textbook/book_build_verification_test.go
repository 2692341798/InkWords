package textbook

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	shared "inkwords-backend/shared/kernel/textbook"
	"testing"
	"time"
)

func testBookBuildFreezesRuntimeEvidence(t *testing.T, db *gorm.DB, service *Service, workspace, project uuid.UUID, old *BookBuildRow, receipt RuntimeEvidenceRow) *BookBuildRow {
	t.Helper()
	ctx := context.Background()
	prior, err := service.GetEditorialWorkspace(ctx, workspace, old.ID)
	require.NoError(t, err)
	require.False(t, prior.Preflight.Passed, "later receipts cannot repair a frozen build")
	fresh, err := service.CreateBookBuild(ctx, workspace, project)
	require.NoError(t, err)
	require.NotEqual(t, old.ID, fresh.ID)
	snapshot, err := shared.ReadBookVerificationSnapshot(json.RawMessage(fresh.ManifestJSON))
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Len(t, snapshot.Artifacts, 1)
	require.Len(t, snapshot.Artifacts[0].Evidence, 1)
	require.Equal(t, receipt.ID.String(), snapshot.Artifacts[0].Evidence[0].ID)
	require.True(t, snapshot.Assess(time.Now()).Passed)
	require.False(t, snapshot.Assess(time.Now().Add(2*time.Hour)).Passed)
	retry, err := service.CreateBookBuild(ctx, workspace, project)
	require.NoError(t, err)
	require.Equal(t, fresh.ID, retry.ID)
	current, err := service.GetEditorialWorkspace(ctx, workspace, fresh.ID)
	require.NoError(t, err)
	require.Empty(t, current.HumanReviews)
	require.Empty(t, current.RightsItems)
	require.Empty(t, current.DelegatedReviews)
	require.Equal(t, shared.QualityStatusPass, current.AutomatedChecks[3].Status)
	// The isolated test fixture explicitly performs new reviews for the new
	// build; production never transfers approvals between manifests.
	for _, item := range prior.RightsItems {
		_, err = service.AddRightsItem(ctx, workspace, AddRightsItemInput{BuildID: fresh.ID, SubjectRef: item.SubjectRef, WorkType: item.WorkType, RightsBasis: item.RightsBasis, AllowedUse: item.AllowedUse, Attribution: item.Attribution, PublicationStatus: item.PublicationStatus})
		require.NoError(t, err)
	}
	for _, stage := range shared.RequiredPublicationReviewStages() {
		_, err = service.CompletePublicationReview(ctx, workspace, humanReviewFixture(fresh, stage))
		require.NoError(t, err)
	}
	tx := db.Begin()
	require.NoError(t, tx.Error)
	require.NoError(t, tx.Model(&RuntimeEvidenceRow{}).Where("id = ?", receipt.ID).Update("structured_output", "changed after freeze").Error)
	var retained BookBuildRow
	require.NoError(t, tx.First(&retained, "id = ?", fresh.ID).Error)
	require.JSONEq(t, string(fresh.ManifestJSON), string(retained.ManifestJSON))
	checks, err := publicationAutomatedChecks(tx, retained)
	require.NoError(t, err)
	require.Equal(t, shared.QualityStatusPass, checks[3].Status)
	var plan []struct {
		Line string `gorm:"column:QUERY PLAN"`
	}
	require.NoError(t, tx.Raw("EXPLAIN SELECT * FROM textbook_runtime_evidence WHERE code_artifact_id IN ? ORDER BY code_artifact_id ASC, captured_at ASC, id ASC", []uuid.UUID{receipt.CodeArtifactID}).Scan(&plan).Error)
	for _, line := range plan {
		t.Log("freeze verification query:", line.Line)
	}
	require.NoError(t, tx.Rollback().Error)
	var retainedOld BookBuildRow
	require.NoError(t, db.First(&retainedOld, "id = ?", old.ID).Error)
	require.JSONEq(t, string(old.ManifestJSON), string(retainedOld.ManifestJSON))
	return fresh
}

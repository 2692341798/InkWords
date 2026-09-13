package textbook

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"inkwords-backend/shared/kernel/httpx"
	shared "inkwords-backend/shared/kernel/textbook"
)

func humanReviewFixture(build *BookBuildRow, stage shared.PublicationReviewStage) CompletePublicationReviewInput {
	score := 3
	return CompletePublicationReviewInput{ID: uuid.New(), BuildID: build.ID, ManifestHash: build.ManifestHash, Stage: stage, Reviewer: "隔离数据库测试夹具（非真人证据）", Verdict: "pass", Score: &score, Scope: "仅验证隔离测试数据库中的版本与出版门禁。", Notes: "这是隔离测试记录，不进入实际教材的真人审校记录。", EvidenceRefs: []string{"test:isolated-review-fixture"}, HardFailures: []string{}}
}

func testHumanPublicationRevisions(t *testing.T, db *gorm.DB, workspaceID uuid.UUID, build *BookBuildRow) {
	t.Helper()
	rollback := errors.New("rollback isolated human review fixture")
	err := db.Transaction(func(tx *gorm.DB) error {
		ctx := context.Background()
		legacyID := uuid.New()
		at := time.Now().UTC().Add(-time.Minute)
		require.NoError(t, tx.Exec("INSERT INTO textbook_publication_reviews (id, build_id, stage, reviewer, notes, automated, completed_at) VALUES (?, ?, ?, ?, ?, FALSE, ?)", legacyID, build.ID, shared.PublicationReviewRights, "历史测试夹具", "历史测试说明没有显式结论或评分。", at).Error)
		var before PublicationReviewRow
		require.NoError(t, tx.First(&before, "id = ?", legacyID).Error)
		service := NewService(NewGormRepository(tx))
		input := humanReviewFixture(build, shared.PublicationReviewRights)
		_, err := service.CompletePublicationReview(ctx, workspaceID, input)
		require.ErrorIs(t, err, ErrVersionConflict, "legacy record is stage revision one")
		input.ExpectedRevision = 1
		review, err := service.CompletePublicationReview(ctx, workspaceID, input)
		require.NoError(t, err)
		require.Equal(t, 2, review.Revision)
		retry, err := service.CompletePublicationReview(ctx, workspaceID, input)
		require.NoError(t, err)
		require.True(t, review.CompletedAt.Equal(retry.CompletedAt))
		review.CompletedAt = review.CompletedAt.UTC()
		retry.CompletedAt = retry.CompletedAt.UTC()
		require.Equal(t, review, retry)
		changed := input
		changed.Notes += "禁止改写已有记录。"
		_, err = service.CompletePublicationReview(ctx, workspaceID, changed)
		require.ErrorIs(t, err, ErrVersionConflict)
		changed = input
		changed.ID = uuid.New()
		_, err = service.CompletePublicationReview(ctx, workspaceID, changed)
		require.ErrorIs(t, err, ErrVersionConflict)
		changed.ExpectedRevision = 2
		changed.ManifestHash = "sha256:" + strings.Repeat("f", 64)
		_, err = service.CompletePublicationReview(ctx, workspaceID, changed)
		require.ErrorIs(t, err, ErrVersionConflict)
		changed.ManifestHash = input.ManifestHash
		changed.Verdict = "needs_revision"
		changed.HardFailures = []string{"测试中新证据有未解决的署名问题"}
		_, err = service.CompletePublicationReview(ctx, workspaceID, changed)
		require.NoError(t, err)
		_, err = service.CompletePublicationReview(ctx, uuid.New(), changed)
		require.ErrorIs(t, err, ErrNotFound)
		editorial, err := service.GetEditorialWorkspace(ctx, workspaceID, build.ID)
		require.NoError(t, err)
		require.Len(t, editorial.HumanReviews, 3)
		require.Contains(t, strings.Join(editorial.Preflight.Blockers, " "), changed.HardFailures[0])
		var after PublicationReviewRow
		require.NoError(t, tx.First(&after, "id = ?", legacyID).Error)
		require.Equal(t, before, after, "legacy record must remain byte-for-byte equivalent at the model boundary")
		var plan []struct {
			QueryPlan string `gorm:"column:QUERY PLAN"`
		}
		require.NoError(t, tx.Exec("SET LOCAL enable_seqscan = off").Error)
		require.NoError(t, tx.Raw("EXPLAIN SELECT revision FROM textbook_publication_reviews WHERE build_id = ? AND stage = ? ORDER BY revision DESC LIMIT 1", build.ID, input.Stage).Scan(&plan).Error)
		raw, err := json.Marshal(plan)
		require.NoError(t, err)
		require.Contains(t, string(raw), "human_review_build_stage_revision")
		t.Logf("human review latest-revision index eligibility (seqscan disabled): %s", raw)
		return rollback
	})
	require.ErrorIs(t, err, rollback)
}

func TestHumanReviewHandlerRejectsLegacyWritesAndSpoofedMetadata(t *testing.T) {
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return uuid.New(), nil }))
	router.POST("/book-builds/:buildID/reviews", NewHandler(NewService(nil)).CompletePublicationReview)
	for _, body := range []string{
		`{"stage":"rights","reviewer":"名字","notes":"只有说明不能当作通过"}`,
		`{"reviewer_kind":"delegated_ai"}`,
		`{"completed_at":"2026-09-10T00:00:00Z"}`,
		`{"build_id":"other"}`,
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/book-builds/"+uuid.NewString()+"/reviews", strings.NewReader(body)))
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	}
}

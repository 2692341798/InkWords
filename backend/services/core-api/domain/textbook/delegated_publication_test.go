package textbook

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"inkwords-backend/shared/kernel/httpx"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func testDelegatedPublicationReview(t *testing.T, db *gorm.DB, workspaceID uuid.UUID, build *BookBuildRow) {
	t.Helper()
	rollback := errors.New("rollback isolated review acceptance fixture")
	err := db.Transaction(func(tx *gorm.DB) error {
		service := NewService(NewGormRepository(tx))
		input := RecordDelegatedPublicationReviewInput{ID: uuid.New(), BuildID: build.ID, ManifestHash: build.ManifestHash, Stage: sharedtextbook.PublicationReviewTechnical, Reviewer: "Codex / delegated AI", DelegationNote: "用户明确委托 AI 进行审阅并代为决策。", Verdict: "pass", Score: 3, Scope: "固定教学实现及原始来源范围核对。", Notes: "这是用户委托的 AI 审阅，不是实际真人同行审校。", EvidenceRefs: []string{"fixture:bounded-technical-review"}, HardFailures: []string{}}
		ctx := context.Background()
		review, err := service.RecordDelegatedPublicationReview(ctx, workspaceID, input)
		require.NoError(t, err)
		require.Equal(t, "delegated_ai", review.ReviewerKind)
		retry, err := service.RecordDelegatedPublicationReview(ctx, workspaceID, input)
		require.NoError(t, err)
		require.Equal(t, review, retry)
		_, err = service.RecordDelegatedPublicationReview(ctx, uuid.New(), input)
		require.ErrorIs(t, err, ErrNotFound)
		changed := input
		changed.Notes += "试图修改已有结论。"
		_, err = service.RecordDelegatedPublicationReview(ctx, workspaceID, changed)
		require.ErrorIs(t, err, ErrVersionConflict)
		changed = input
		changed.ID = uuid.New()
		_, err = service.RecordDelegatedPublicationReview(ctx, workspaceID, changed)
		require.ErrorIs(t, err, ErrVersionConflict, "stale stage version must not append")
		changed.ExpectedRevision = 1
		changed.ManifestHash = "sha256:" + strings.Repeat("f", 64)
		_, err = service.RecordDelegatedPublicationReview(ctx, workspaceID, changed)
		require.ErrorIs(t, err, ErrVersionConflict)
		changed.ManifestHash = input.ManifestHash
		changed.Verdict, changed.Score = "needs_revision", 2
		changed.HardFailures = []string{"测试证据尚未覆盖冻结工件。"}
		_, err = service.RecordDelegatedPublicationReview(ctx, workspaceID, changed)
		require.NoError(t, err)
		editorial, err := service.GetEditorialWorkspace(ctx, workspaceID, build.ID)
		require.NoError(t, err)
		require.Empty(t, editorial.HumanReviews)
		require.Len(t, editorial.DelegatedReviews, 2)
		require.Equal(t, "pass", editorial.DelegatedReviews[0].Verdict, "old verdict remains immutable")
		require.Contains(t, strings.Join(editorial.Preflight.Blockers, "\n"), changed.HardFailures[0])
		_, err = service.PromoteBookBuild(ctx, workspaceID, build.ID)
		require.ErrorIs(t, err, ErrInvalidState, "delegation cannot waive failed code, missing rights or unresolved findings")
		var plan []struct {
			QueryPlan string `gorm:"column:QUERY PLAN"`
		}
		require.NoError(t, tx.Exec("SET LOCAL enable_seqscan = off").Error)
		require.NoError(t, tx.Raw("EXPLAIN SELECT * FROM textbook_delegated_publication_reviews WHERE build_id = ? ORDER BY stage, revision", build.ID).Scan(&plan).Error)
		raw, err := json.Marshal(plan)
		require.NoError(t, err)
		require.Contains(t, string(raw), "Index Scan")
		t.Logf("delegated review history EXPLAIN: %s", raw)
		return rollback
	})
	require.ErrorIs(t, err, rollback)
}

func TestDelegatedPublicationReviewRejectsMissingDelegationAndSpoofedActor(t *testing.T) {
	service := NewService(nil)
	_, err := service.RecordDelegatedPublicationReview(context.Background(), uuid.New(), RecordDelegatedPublicationReviewInput{ID: uuid.New(), BuildID: uuid.New()})
	require.ErrorIs(t, err, ErrInvalidState)
	handler := &Handler{service: service}
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return uuid.New(), nil }))
	router.POST("/book-builds/:buildID/delegated-reviews", handler.RecordDelegatedPublicationReview)
	for _, body := range []string{`{"reviewer_kind":"human"}`, `{"completed_at":"2026-09-10T00:00:00Z"}`, `{"build_id":"other"}`} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/book-builds/"+uuid.NewString()+"/delegated-reviews", strings.NewReader(body)))
		require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	}
}

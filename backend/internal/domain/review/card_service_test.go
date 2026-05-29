package review

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"inkwords-backend/internal/model"
)

func TestService_GetTodayCard_PrefersUnreviewed(t *testing.T) {
	t.Parallel()

	svc := newTestReviewServiceWithNotes([]ReviewNote{
		{NotePath: "wiki/concepts/a.md", Title: "A", SourceTitle: "系列 A"},
		{NotePath: "wiki/concepts/b.md", Title: "B", SourceTitle: "系列 B"},
	})
	svc.repo = &stubReviewRepository{
		recentSessions: []model.ReviewSession{
			{NotePath: "wiki/concepts/a.md", Status: model.ReviewStatusCompleted},
		},
	}

	resp, err := svc.GetTodayCard(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Equal(t, "wiki/concepts/b.md", resp.NotePath)
	require.Equal(t, "这是你最近导入但还没复习过的一篇内容。", resp.ReviewReason)
	require.ElementsMatch(t, []string{model.ReviewModeLightRecall, model.ReviewModeDetailedQA}, resp.AvailableModes)
}

func TestService_PickRandomCard_AvoidsRecentReviewedNotes(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 27, 9, 0, 0, 0, time.UTC)
	svc := newTestReviewServiceWithNotes([]ReviewNote{
		{NotePath: "wiki/concepts/a.md", Title: "A"},
		{NotePath: "wiki/concepts/b.md", Title: "B"},
	})
	svc.now = func() time.Time { return now }
	svc.repo = &stubReviewRepository{
		recentSessions: []model.ReviewSession{
			{
				NotePath:    "wiki/concepts/a.md",
				Status:      model.ReviewStatusCompleted,
				CompletedAt: timePtr(now.Add(-24 * time.Hour)),
			},
		},
	}

	resp, err := svc.PickRandomCard(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Equal(t, "wiki/concepts/b.md", resp.NotePath)
	require.Equal(t, "这是你主动开始的一次随机漫游复习。", resp.ReviewReason)
}

func TestService_ListNotes_UsesKeywordFilter(t *testing.T) {
	t.Parallel()

	svc := newTestReviewServiceWithNotes([]ReviewNote{
		{NotePath: "wiki/concepts/并发控制与速率限制.md", Title: "并发控制与速率限制"},
		{NotePath: "wiki/concepts/前端状态管理.md", Title: "前端状态管理"},
	})

	resp, err := svc.ListNotes(context.Background(), uuid.New(), ListNotesQuery{Query: "并发"})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	require.Equal(t, "并发控制与速率限制", resp.Items[0].Title)
}

func TestService_ListNotes_UsesSeriesFilterAndPagination(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 27, 9, 0, 0, 0, time.UTC)
	svc := newTestReviewServiceWithNotes([]ReviewNote{
		{NotePath: "wiki/concepts/并发控制与速率限制.md", Title: "并发控制与速率限制", SourceTitle: "后端系列", PreferredMode: model.ReviewModeDetailedQA},
		{NotePath: "wiki/concepts/前端状态管理.md", Title: "前端状态管理", SourceTitle: "前端系列", PreferredMode: model.ReviewModeLightRecall},
		{NotePath: "wiki/concepts/并发原语.md", Title: "并发原语", SourceTitle: "后端系列", PreferredMode: model.ReviewModeLightRecall},
	})
	svc.repo = &stubReviewRepository{
		recentSessions: []model.ReviewSession{
			{
				NotePath:    "wiki/concepts/并发原语.md",
				Status:      model.ReviewStatusCompleted,
				CompletedAt: timePtr(now.Add(-48 * time.Hour)),
			},
		},
	}

	resp, err := svc.ListNotes(context.Background(), uuid.New(), ListNotesQuery{
		SeriesTitle: "后端",
		Page:        2,
		PageSize:    1,
	})
	require.NoError(t, err)
	require.Equal(t, 2, resp.Total)
	require.Equal(t, 2, resp.Page)
	require.Equal(t, 1, resp.PageSize)
	require.Len(t, resp.Items, 1)
	require.Equal(t, "并发原语", resp.Items[0].Title)
	require.Equal(t, model.ReviewModeLightRecall, resp.Items[0].PreferredMode)
	require.NotNil(t, resp.Items[0].LastReviewedAt)
}


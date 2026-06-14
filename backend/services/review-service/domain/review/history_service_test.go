package review

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestService_GetHistory_ReturnsNewestSessions(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 27, 9, 0, 0, 0, time.UTC)
	svc := newTestReviewServiceWithNotes(nil)
	svc.repo = &stubReviewRepository{
		recentSessions: []ReviewSession{
			{
				ID:           uuid.New(),
				NotePath:     "wiki/concepts/history-a.md",
				NoteTitle:    "历史 A",
				SourceTitle:  "后端系列",
				Mode:         ReviewModeLightRecall,
				Status:       ReviewStatusCompleted,
				FinalSummary: "主线已经比较清楚",
				CompletedAt:  timePtr(now.Add(-2 * time.Hour)),
			},
			{
				ID:           uuid.New(),
				NotePath:     "wiki/concepts/history-b.md",
				NoteTitle:    "历史 B",
				SourceTitle:  "前端系列",
				Mode:         ReviewModeDetailedQA,
				Status:       ReviewStatusInProgress,
				FinalSummary: "",
				UpdatedAt:    now.Add(-1 * time.Hour),
			},
		},
	}

	resp, err := svc.GetHistory(context.Background(), uuid.New(), 1)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	require.Equal(t, "历史 A", resp.Items[0].Title)
	require.Equal(t, ReviewStatusCompleted, resp.Items[0].Status)
	require.Equal(t, "主线已经比较清楚", resp.Items[0].Summary)
	require.Equal(t, 1, resp.Limit)
}

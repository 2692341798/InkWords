package review

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPicker_PickToday_PrefersUnreviewed(t *testing.T) {
	t.Parallel()

	notes := []ReviewNote{
		{NotePath: "wiki/concepts/a.md", Title: "A"},
		{NotePath: "wiki/concepts/b.md", Title: "B"},
	}
	stats := map[string]ReviewItemState{
		"wiki/concepts/a.md": {CompletedCount: 2},
	}

	got := PickToday(notes, stats, time.Now())
	require.Equal(t, "wiki/concepts/b.md", got.NotePath)
}

func TestPicker_PickToday_FallsBackToOldestReviewed(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 27, 9, 0, 0, 0, time.UTC)
	notes := []ReviewNote{
		{NotePath: "wiki/concepts/a.md", Title: "A"},
		{NotePath: "wiki/concepts/b.md", Title: "B"},
	}
	stats := map[string]ReviewItemState{
		"wiki/concepts/a.md": {
			CompletedCount:  1,
			LastReviewedAt: now.Add(-24 * time.Hour),
		},
		"wiki/concepts/b.md": {
			CompletedCount:  3,
			LastReviewedAt: now.Add(-72 * time.Hour),
		},
	}

	got := PickToday(notes, stats, now)
	require.Equal(t, "wiki/concepts/b.md", got.NotePath)
}

func TestPicker_PickRandom_AvoidsRecentItems(t *testing.T) {
	t.Parallel()

	notes := []ReviewNote{
		{NotePath: "wiki/concepts/a.md", Title: "A"},
		{NotePath: "wiki/concepts/b.md", Title: "B"},
	}

	got := PickRandom(notes, map[string]bool{
		"wiki/concepts/a.md": true,
	})
	require.Equal(t, "wiki/concepts/b.md", got.NotePath)
}

func TestPicker_PickRandom_ReturnsDifferentEligibleItemsAcrossCalls(t *testing.T) {
	t.Parallel()

	notes := []ReviewNote{
		{NotePath: "wiki/concepts/a.md", Title: "A"},
		{NotePath: "wiki/concepts/b.md", Title: "B"},
		{NotePath: "wiki/concepts/c.md", Title: "C"},
		{NotePath: "wiki/concepts/d.md", Title: "D"},
	}
	recent := map[string]bool{
		"wiki/concepts/a.md": true,
	}

	picked := make(map[string]struct{})
	for range 128 {
		got := PickRandom(notes, recent)
		require.NotEqual(t, "wiki/concepts/a.md", got.NotePath)
		picked[got.NotePath] = struct{}{}
	}

	require.Greater(t, len(picked), 1, "expected random picker to choose more than one eligible note across repeated calls")
}

func TestPicker_PickRandom_UsesFullPoolWhenAllItemsAreRecent(t *testing.T) {
	t.Parallel()

	notes := []ReviewNote{
		{NotePath: "wiki/concepts/a.md", Title: "A"},
		{NotePath: "wiki/concepts/b.md", Title: "B"},
		{NotePath: "wiki/concepts/c.md", Title: "C"},
	}

	picked := make(map[string]struct{})
	for range 128 {
		got := PickRandom(notes, map[string]bool{
		"wiki/concepts/a.md": true,
		"wiki/concepts/b.md": true,
			"wiki/concepts/c.md": true,
		})
		picked[got.NotePath] = struct{}{}
	}

	require.Greater(t, len(picked), 1, "expected random picker to use the full pool when every note is recent")
}

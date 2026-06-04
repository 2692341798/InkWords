package review

import (
	cryptorand "crypto/rand"
	"math/big"
	"sort"
	"time"
)

// ReviewItemState 表示某篇笔记最近的复习状态摘要。
type ReviewItemState struct {
	CompletedCount int
	LastReviewedAt time.Time
}

// PickToday 从候选池中选出今日推荐题卡。
func PickToday(notes []ReviewNote, stats map[string]ReviewItemState, _ time.Time) ReviewNote {
	if len(notes) == 0 {
		return ReviewNote{}
	}

	for _, note := range notes {
		if stats[note.NotePath].CompletedCount == 0 {
			return note
		}
	}

	ordered := append([]ReviewNote(nil), notes...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := stats[ordered[i].NotePath]
		right := stats[ordered[j].NotePath]
		if left.LastReviewedAt.Equal(right.LastReviewedAt) {
			return ordered[i].NotePath < ordered[j].NotePath
		}
		return left.LastReviewedAt.Before(right.LastReviewedAt)
	})

	return ordered[0]
}

// PickRandom 从候选池中选出一篇尽量避开最近复习记录的题卡。
func PickRandom(notes []ReviewNote, recent map[string]bool) ReviewNote {
	if len(notes) == 0 {
		return ReviewNote{}
	}

	// Why: the manual random entry should rotate across the whole eligible pool
	// instead of always biasing toward the first non-recent note in source order.
	eligible := make([]ReviewNote, 0, len(notes))
	for _, note := range notes {
		if !recent[note.NotePath] {
			eligible = append(eligible, note)
		}
	}

	if len(eligible) == 0 {
		eligible = notes
	}

	return eligible[randomIndex(len(eligible))]
}

func randomIndex(length int) int {
	if length <= 1 {
		return 0
	}

	index, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(length)))
	if err != nil {
		return int(time.Now().UnixNano() % int64(length))
	}
	return int(index.Int64())
}

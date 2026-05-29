package review

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// GetHistory 返回最近复习记录的摘要列表。
func (s *Service) GetHistory(ctx context.Context, userID uuid.UUID, limit int) (ReviewHistoryResponse, error) {
	if limit <= 0 {
		limit = 5
	}

	sessions, err := s.repo.GetRecentSessions(ctx, userID, limit)
	if err != nil {
		return ReviewHistoryResponse{}, err
	}

	items := make([]ReviewHistoryItem, 0, len(sessions))
	for _, session := range sessions {
		item := ReviewHistoryItem{
			SessionID:   session.ID,
			NotePath:    session.NotePath,
			Title:       session.NoteTitle,
			SourceTitle: session.SourceTitle,
			Mode:        session.Mode,
			Status:      session.Status,
			Summary:     firstNonEmpty(strings.TrimSpace(session.FinalSummary), "这次复习还没有生成总结。"),
		}
		if reviewedAt := resolveReviewedAt(session); !reviewedAt.IsZero() {
			copyValue := reviewedAt
			item.ReviewedAt = &copyValue
		}
		items = append(items, item)
	}

	return ReviewHistoryResponse{
		Items: items,
		Limit: limit,
	}, nil
}

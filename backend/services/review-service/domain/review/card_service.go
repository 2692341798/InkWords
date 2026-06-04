package review

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"inkwords-backend/internal/model"
)

// GetTodayCard 返回今日推荐题卡。
func (s *Service) GetTodayCard(ctx context.Context, userID uuid.UUID) (ReviewCardResponse, error) {
	notes, err := s.noteSource.ListEligibleNotes(ctx)
	if err != nil {
		return ReviewCardResponse{}, err
	}
	if len(notes) == 0 {
		return ReviewCardResponse{}, errNoEligibleReviewNotes
	}

	stats, err := s.loadItemState(ctx, userID)
	if err != nil {
		return ReviewCardResponse{}, err
	}

	picked := PickToday(notes, stats, s.now())
	reason := "这篇内容你已经有一段时间没回顾了。"
	if stats[picked.NotePath].CompletedCount == 0 {
		reason = "这是你最近导入但还没复习过的一篇内容。"
	}

	return toReviewCardResponse(picked, reason), nil
}

// PickRandomCard 返回手动随机抽取的题卡。
func (s *Service) PickRandomCard(ctx context.Context, userID uuid.UUID) (ReviewCardResponse, error) {
	notes, err := s.noteSource.ListEligibleNotes(ctx)
	if err != nil {
		return ReviewCardResponse{}, err
	}
	if len(notes) == 0 {
		return ReviewCardResponse{}, errNoEligibleReviewNotes
	}

	recent, err := s.loadRecentItems(ctx, userID)
	if err != nil {
		return ReviewCardResponse{}, err
	}

	return toReviewCardResponse(PickRandom(notes, recent), "这是你主动开始的一次随机漫游复习。"), nil
}

// ListNotes 返回手动选择文章复习入口所需的候选列表。
func (s *Service) ListNotes(ctx context.Context, userID uuid.UUID, query ListNotesQuery) (ListNotesResponse, error) {
	notes, err := s.noteSource.ListEligibleNotes(ctx)
	if err != nil {
		return ListNotesResponse{}, err
	}

	stats, err := s.loadItemState(ctx, userID)
	if err != nil {
		return ListNotesResponse{}, err
	}

	filtered := make([]ListNotesItem, 0, len(notes))
	for _, note := range notes {
		if !matchesListNotesQuery(note, query) {
			continue
		}

		item := ListNotesItem{
			NotePath:      note.NotePath,
			Title:         note.Title,
			SourceTitle:   note.SourceTitle,
			PreferredMode: firstNonEmpty(note.PreferredMode, model.ReviewModeLightRecall),
		}
		if lastReviewedAt := stats[note.NotePath].LastReviewedAt; !lastReviewedAt.IsZero() {
			copyValue := lastReviewedAt
			item.LastReviewedAt = &copyValue
		}

		filtered = append(filtered, item)
	}

	page, pageSize := normalizeListNotesPagination(query.Page, query.PageSize)
	start := (page - 1) * pageSize
	if start >= len(filtered) {
		return ListNotesResponse{
			Items:    []ListNotesItem{},
			Total:    len(filtered),
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	return ListNotesResponse{
		Items:    filtered[start:end],
		Total:    len(filtered),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) loadItemState(ctx context.Context, userID uuid.UUID) (map[string]ReviewItemState, error) {
	sessions, err := s.repo.GetRecentSessions(ctx, userID, 200)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]ReviewItemState, len(sessions))
	for _, session := range sessions {
		if strings.TrimSpace(session.NotePath) == "" {
			continue
		}

		state := stats[session.NotePath]
		if session.Status == model.ReviewStatusCompleted {
			state.CompletedCount++
		}
		reviewedAt := resolveReviewedAt(session)
		if state.LastReviewedAt.IsZero() || reviewedAt.After(state.LastReviewedAt) {
			state.LastReviewedAt = reviewedAt
		}
		stats[session.NotePath] = state
	}

	return stats, nil
}

func (s *Service) loadRecentItems(ctx context.Context, userID uuid.UUID) (map[string]bool, error) {
	sessions, err := s.repo.GetRecentSessions(ctx, userID, 50)
	if err != nil {
		return nil, err
	}

	now := s.now()
	recent := make(map[string]bool, len(sessions))
	for _, session := range sessions {
		reviewedAt := resolveReviewedAt(session)
		if strings.TrimSpace(session.NotePath) == "" || reviewedAt.IsZero() {
			continue
		}
		if now.Sub(reviewedAt) <= recentReviewWindow {
			recent[session.NotePath] = true
		}
	}

	return recent, nil
}

func toReviewCardResponse(note ReviewNote, reason string) ReviewCardResponse {
	availableModes := []string{model.ReviewModeLightRecall, model.ReviewModeDetailedQA}
	if note.PreferredMode == model.ReviewModeDetailedQA {
		availableModes = []string{model.ReviewModeDetailedQA, model.ReviewModeLightRecall}
	}

	return ReviewCardResponse{
		NotePath:         note.NotePath,
		Title:            note.Title,
		SourceTitle:      note.SourceTitle,
		ReviewReason:     reason,
		EstimatedMinutes: defaultReviewCardEstimatedMinutes,
		AvailableModes:   availableModes,
	}
}

func matchesListNotesQuery(note ReviewNote, query ListNotesQuery) bool {
	keyword := strings.TrimSpace(query.Query)
	if keyword != "" {
		target := strings.ToLower(note.Title + " " + note.SourceTitle)
		if !strings.Contains(target, strings.ToLower(keyword)) {
			return false
		}
	}

	seriesTitle := strings.TrimSpace(query.SeriesTitle)
	if seriesTitle != "" && !strings.Contains(strings.ToLower(note.SourceTitle), strings.ToLower(seriesTitle)) {
		return false
	}

	return true
}

func normalizeListNotesPagination(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultListNotesPage
	}
	if pageSize <= 0 {
		pageSize = defaultListNotesPageSize
	}
	if pageSize > maxListNotesPageSize {
		pageSize = maxListNotesPageSize
	}
	return page, pageSize
}

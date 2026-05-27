package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"inkwords-backend/internal/model"
)

const recentReviewWindow = 72 * time.Hour

var (
	errNoEligibleReviewNotes = errors.New("暂无可复习的知识卡")
	errInvalidReviewMode     = errors.New("不支持的复习模式")
	errInvalidReviewEntry    = errors.New("不支持的复习入口")
	errReviewNoteNotFound    = errors.New("指定的复习笔记不存在")
	errReviewSessionNotFound = errors.New("复习会话不存在")
	errReviewSessionDenied   = errors.New("无权访问该复习会话")
	errReviewSessionClosed   = errors.New("复习会话已结束")
	errReviewHintExhausted   = errors.New("提示次数已用尽")
	errEmptyReviewAnswer     = errors.New("回答内容不能为空")
)

// NoteSource 定义 review 领域所需的笔记读取能力。
type NoteSource interface {
	ListEligibleNotes(ctx context.Context) ([]ReviewNote, error)
}

// Service 提供 today、pick、notes 三类入口能力。
type Service struct {
	repo       Repository
	noteSource NoteSource
	now        func() time.Time
}

// NewService 创建 review 领域服务。
func NewService(repo Repository, noteSource NoteSource) *Service {
	return &Service{
		repo:       repo,
		noteSource: noteSource,
		now:        time.Now,
	}
}

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

// CreateSession 创建一条新的复习会话，并生成训练快照与开场提示。
func (s *Service) CreateSession(ctx context.Context, userID uuid.UUID, req CreateSessionRequest) (ReviewSessionResponse, error) {
	if !isSupportedReviewMode(req.Mode) {
		return ReviewSessionResponse{}, errInvalidReviewMode
	}
	if !isSupportedReviewEntryType(req.EntryType) {
		return ReviewSessionResponse{}, errInvalidReviewEntry
	}

	note, err := s.findNoteByPath(ctx, req.NotePath)
	if err != nil {
		return ReviewSessionResponse{}, err
	}

	summarySnapshot, keyPoints := buildSessionSnapshot(note)
	opening := openingPrompt(req.Mode)
	hints := initialHints(req.Mode, keyPoints)
	now := s.now()

	session := model.ReviewSession{
		ID:                uuid.New(),
		UserID:            userID,
		NotePath:          note.NotePath,
		NoteTitle:         note.Title,
		SourceTitle:       note.SourceTitle,
		EntryType:         req.EntryType,
		Mode:              req.Mode,
		Status:            model.ReviewStatusCreated,
		EstimatedMinutes:  defaultReviewCardEstimatedMinutes,
		SummarySnapshot:   summarySnapshot,
		KeyPointsSnapshot: mustMarshalJSON(keyPoints),
		MetadataSnapshot:  mustMarshalJSON(map[string]any{"preferred_mode": note.PreferredMode}),
		MaxHintCount:      2,
		TurnCount:         1,
		StartedAt:         now,
	}

	if err := s.repo.CreateSession(ctx, &session); err != nil {
		return ReviewSessionResponse{}, fmt.Errorf("创建复习会话失败: %w", err)
	}

	openingTurn := model.ReviewTurn{
		SessionID: session.ID,
		TurnIndex: 1,
		Role:      model.ReviewTurnRoleSystem,
		TurnType:  model.ReviewTurnTypeOpening,
		Content:   opening,
	}
	if err := s.repo.AppendTurn(ctx, &openingTurn); err != nil {
		return ReviewSessionResponse{}, fmt.Errorf("写入开场提示失败: %w", err)
	}

	return ReviewSessionResponse{
		SessionID:     session.ID,
		Status:        session.Status,
		Mode:          session.Mode,
		Title:         session.NoteTitle,
		OpeningPrompt: opening,
		InitialHints:  hints,
		NextQuestion:  nextQuestionForSession(session, []model.ReviewTurn{openingTurn}),
		TurnIndex:     openingTurn.TurnIndex,
		Turns:         []ReviewTurnResponse{toTurnResponse(openingTurn)},
	}, nil
}

// GetSession 返回一次复习会话的当前状态与历史轮次。
func (s *Service) GetSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (ReviewSessionResponse, error) {
	session, turns, err := s.loadOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return ReviewSessionResponse{}, err
	}

	return buildSessionResponse(session, turns), nil
}

// Respond 处理用户的一轮回答，并根据模式推进问题或结束会话。
func (s *Service) Respond(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, req RespondRequest) (RespondResponse, error) {
	answer := strings.TrimSpace(req.Answer)
	if answer == "" {
		return RespondResponse{}, errEmptyReviewAnswer
	}

	session, turns, err := s.loadOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return RespondResponse{}, err
	}
	if isClosedStatus(session.Status) {
		return RespondResponse{}, errReviewSessionClosed
	}

	answerTurn := model.ReviewTurn{
		SessionID: session.ID,
		TurnIndex: nextTurnIndex(turns),
		Role:      model.ReviewTurnRoleUser,
		TurnType:  model.ReviewTurnTypeAnswer,
		Content:   answer,
	}
	if err := s.repo.AppendTurn(ctx, &answerTurn); err != nil {
		return RespondResponse{}, fmt.Errorf("写入回答失败: %w", err)
	}

	updatedTurns := append(append([]model.ReviewTurn(nil), turns...), answerTurn)
	session.Status = model.ReviewStatusInProgress
	session.TurnCount = answerTurn.TurnIndex

	if session.Mode == model.ReviewModeDetailedQA {
		answerCount := countUserAnswers(updatedTurns)
		if answerCount >= maxDetailedQARounds {
			feedback := buildFinalFeedback(answer)
			if err := s.completeSession(ctx, &session, updatedTurns, feedback); err != nil {
				return RespondResponse{}, err
			}
			return RespondResponse{
				SessionID:     session.ID,
				SessionStatus: session.Status,
				TurnIndex:     session.TurnCount,
				Completed:     true,
				FinalFeedback: feedback,
			}, nil
		}

		nextQuestion := nextDetailedQuestion(answerCount)
		stageFeedback := buildStageFeedback(session.Mode, answer)
		questionTurn := model.ReviewTurn{
			SessionID: session.ID,
			TurnIndex: answerTurn.TurnIndex + 1,
			Role:      model.ReviewTurnRoleSystem,
			TurnType:  model.ReviewTurnTypeQuestion,
			Content:   nextQuestion,
		}
		if err := s.repo.AppendTurn(ctx, &questionTurn); err != nil {
			return RespondResponse{}, fmt.Errorf("写入下一轮问题失败: %w", err)
		}

		session.TurnCount = questionTurn.TurnIndex
		if err := s.repo.UpdateSession(ctx, &session); err != nil {
			return RespondResponse{}, fmt.Errorf("更新复习会话失败: %w", err)
		}

		return RespondResponse{
			SessionID:     session.ID,
			SessionStatus: session.Status,
			TurnIndex:     session.TurnCount,
			StageFeedback: stageFeedback,
			NextQuestion:  nextQuestion,
			Completed:     false,
		}, nil
	}

	stageFeedback := buildStageFeedback(session.Mode, answer)
	feedbackTurn := model.ReviewTurn{
		SessionID: session.ID,
		TurnIndex: answerTurn.TurnIndex + 1,
		Role:      model.ReviewTurnRoleSystem,
		TurnType:  model.ReviewTurnTypeFeedback,
		Content:   stageFeedback,
	}
	if err := s.repo.AppendTurn(ctx, &feedbackTurn); err != nil {
		return RespondResponse{}, fmt.Errorf("写入阶段反馈失败: %w", err)
	}

	session.TurnCount = feedbackTurn.TurnIndex
	if err := s.repo.UpdateSession(ctx, &session); err != nil {
		return RespondResponse{}, fmt.Errorf("更新复习会话失败: %w", err)
	}

	return RespondResponse{
		SessionID:     session.ID,
		SessionStatus: session.Status,
		TurnIndex:     session.TurnCount,
		StageFeedback: stageFeedback,
		Completed:     false,
	}, nil
}

// RequestHint 根据当前会话状态返回一条更具体的提示。
func (s *Service) RequestHint(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (HintResponse, error) {
	session, turns, err := s.loadOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return HintResponse{}, err
	}
	if isClosedStatus(session.Status) {
		return HintResponse{}, errReviewSessionClosed
	}
	if session.HintUsedCount >= session.MaxHintCount {
		return HintResponse{}, errReviewHintExhausted
	}

	keyPoints := decodeStringSlice(session.KeyPointsSnapshot)
	hintText := buildHintText(session, keyPoints)
	hintTurn := model.ReviewTurn{
		SessionID: session.ID,
		TurnIndex: nextTurnIndex(turns),
		Role:      model.ReviewTurnRoleSystem,
		TurnType:  model.ReviewTurnTypeHint,
		Content:   hintText,
	}
	if err := s.repo.AppendTurn(ctx, &hintTurn); err != nil {
		return HintResponse{}, fmt.Errorf("写入提示失败: %w", err)
	}

	session.HintUsedCount++
	session.TurnCount = hintTurn.TurnIndex
	if err := s.repo.UpdateSession(ctx, &session); err != nil {
		return HintResponse{}, fmt.Errorf("更新复习会话失败: %w", err)
	}

	return HintResponse{
		SessionID:          session.ID,
		HintText:           hintText,
		RemainingHintCount: session.MaxHintCount - session.HintUsedCount,
	}, nil
}

// Finish 显式结束复习训练，并返回最终反馈。
func (s *Service) Finish(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (FinishResponse, error) {
	session, turns, err := s.loadOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return FinishResponse{}, err
	}
	if isClosedStatus(session.Status) {
		return FinishResponse{
			SessionID:     session.ID,
			SessionStatus: session.Status,
			FinalFeedback: FinalFeedback{
				Summary:   session.FinalSummary,
				Strengths: decodeStringSlice(session.Strengths),
				Gaps:      decodeStringSlice(session.Gaps),
				NextFocus: decodeStringSlice(session.NextFocus),
			},
		}, nil
	}

	feedback := buildFinalFeedback(lastUserAnswer(turns))
	if err := s.completeSession(ctx, &session, turns, feedback); err != nil {
		return FinishResponse{}, err
	}

	return FinishResponse{
		SessionID:     session.ID,
		SessionStatus: session.Status,
		FinalFeedback: feedback,
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

func resolveReviewedAt(session model.ReviewSession) time.Time {
	if session.CompletedAt != nil && !session.CompletedAt.IsZero() {
		return *session.CompletedAt
	}
	if !session.UpdatedAt.IsZero() {
		return session.UpdatedAt
	}
	if !session.StartedAt.IsZero() {
		return session.StartedAt
	}
	return session.CreatedAt
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

func (s *Service) findNoteByPath(ctx context.Context, notePath string) (ReviewNote, error) {
	notes, err := s.noteSource.ListEligibleNotes(ctx)
	if err != nil {
		return ReviewNote{}, err
	}
	for _, note := range notes {
		if note.NotePath == strings.TrimSpace(notePath) {
			return note, nil
		}
	}
	return ReviewNote{}, errReviewNoteNotFound
}

func (s *Service) loadOwnedSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (model.ReviewSession, []model.ReviewTurn, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return model.ReviewSession{}, nil, fmt.Errorf("查询复习会话失败: %w", err)
	}
	if session.ID == uuid.Nil {
		return model.ReviewSession{}, nil, errReviewSessionNotFound
	}
	if session.UserID != userID {
		return model.ReviewSession{}, nil, errReviewSessionDenied
	}

	turns, err := s.repo.ListTurns(ctx, session.ID)
	if err != nil {
		return model.ReviewSession{}, nil, fmt.Errorf("查询复习轮次失败: %w", err)
	}
	return session, turns, nil
}

func (s *Service) completeSession(ctx context.Context, session *model.ReviewSession, turns []model.ReviewTurn, feedback FinalFeedback) error {
	completionTurn := model.ReviewTurn{
		SessionID: session.ID,
		TurnIndex: nextTurnIndex(turns),
		Role:      model.ReviewTurnRoleSystem,
		TurnType:  model.ReviewTurnTypeCompletion,
		Content:   feedback.Summary,
	}
	if err := s.repo.AppendTurn(ctx, &completionTurn); err != nil {
		return fmt.Errorf("写入结束反馈失败: %w", err)
	}

	completedAt := s.now()
	session.Status = model.ReviewStatusCompleted
	session.CompletedAt = &completedAt
	session.FinalSummary = feedback.Summary
	session.Strengths = mustMarshalJSON(feedback.Strengths)
	session.Gaps = mustMarshalJSON(feedback.Gaps)
	session.NextFocus = mustMarshalJSON(feedback.NextFocus)
	session.TurnCount = completionTurn.TurnIndex
	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("更新复习会话失败: %w", err)
	}
	return nil
}

func buildSessionResponse(session model.ReviewSession, turns []model.ReviewTurn) ReviewSessionResponse {
	opening := ""
	for _, turn := range turns {
		if turn.TurnType == model.ReviewTurnTypeOpening {
			opening = turn.Content
			break
		}
	}

	return ReviewSessionResponse{
		SessionID:     session.ID,
		Status:        session.Status,
		Mode:          session.Mode,
		Title:         session.NoteTitle,
		OpeningPrompt: opening,
		InitialHints:  initialHints(session.Mode, decodeStringSlice(session.KeyPointsSnapshot)),
		NextQuestion:  nextQuestionForSession(session, turns),
		TurnIndex:     len(turns),
		Turns:         toTurnResponses(turns),
	}
}

func nextQuestionForSession(session model.ReviewSession, turns []model.ReviewTurn) string {
	if session.Mode != model.ReviewModeDetailedQA || isClosedStatus(session.Status) {
		return ""
	}

	answerCount := countUserAnswers(turns)
	if answerCount >= maxDetailedQARounds {
		return ""
	}
	return nextDetailedQuestion(answerCount)
}

func nextTurnIndex(turns []model.ReviewTurn) int {
	if len(turns) == 0 {
		return 1
	}
	return turns[len(turns)-1].TurnIndex + 1
}

func countUserAnswers(turns []model.ReviewTurn) int {
	count := 0
	for _, turn := range turns {
		if turn.Role == model.ReviewTurnRoleUser && turn.TurnType == model.ReviewTurnTypeAnswer {
			count++
		}
	}
	return count
}

func lastUserAnswer(turns []model.ReviewTurn) string {
	for idx := len(turns) - 1; idx >= 0; idx-- {
		if turns[idx].Role == model.ReviewTurnRoleUser && turns[idx].TurnType == model.ReviewTurnTypeAnswer {
			return turns[idx].Content
		}
	}
	return ""
}

func toTurnResponses(turns []model.ReviewTurn) []ReviewTurnResponse {
	items := make([]ReviewTurnResponse, 0, len(turns))
	for _, turn := range turns {
		items = append(items, toTurnResponse(turn))
	}
	return items
}

func toTurnResponse(turn model.ReviewTurn) ReviewTurnResponse {
	return ReviewTurnResponse{
		TurnIndex: turn.TurnIndex,
		Role:      turn.Role,
		TurnType:  turn.TurnType,
		Content:   turn.Content,
	}
}

func isSupportedReviewMode(mode string) bool {
	switch mode {
	case model.ReviewModeLightRecall, model.ReviewModeDetailedQA:
		return true
	default:
		return false
	}
}

func isSupportedReviewEntryType(entryType string) bool {
	switch entryType {
	case model.ReviewEntryTypeToday, model.ReviewEntryTypeManualRandom, model.ReviewEntryTypeManualSelect:
		return true
	default:
		return false
	}
}

func isClosedStatus(status string) bool {
	return status == model.ReviewStatusCompleted || status == model.ReviewStatusAbandoned
}

func decodeStringSlice(raw []byte) []string {
	var values []string
	if len(raw) == 0 {
		return []string{}
	}
	if err := json.Unmarshal(raw, &values); err != nil {
		return []string{}
	}
	return values
}

func mustMarshalJSON(value any) datatypes.JSON {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return datatypes.JSON(data)
}

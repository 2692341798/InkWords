package review

import (
	"context"
	"strings"
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

func TestService_GetHistory_ReturnsNewestSessions(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 27, 9, 0, 0, 0, time.UTC)
	svc := newTestReviewServiceWithNotes(nil)
	svc.repo = &stubReviewRepository{
		recentSessions: []model.ReviewSession{
			{
				ID:           uuid.New(),
				NotePath:     "wiki/concepts/history-a.md",
				NoteTitle:    "历史 A",
				SourceTitle:  "后端系列",
				Mode:         model.ReviewModeLightRecall,
				Status:       model.ReviewStatusCompleted,
				FinalSummary: "主线已经比较清楚",
				CompletedAt:  timePtr(now.Add(-2 * time.Hour)),
			},
			{
				ID:           uuid.New(),
				NotePath:     "wiki/concepts/history-b.md",
				NoteTitle:    "历史 B",
				SourceTitle:  "前端系列",
				Mode:         model.ReviewModeDetailedQA,
				Status:       model.ReviewStatusInProgress,
				FinalSummary: "",
				UpdatedAt:    now.Add(-1 * time.Hour),
			},
		},
	}

	resp, err := svc.GetHistory(context.Background(), uuid.New(), 1)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	require.Equal(t, "历史 A", resp.Items[0].Title)
	require.Equal(t, model.ReviewStatusCompleted, resp.Items[0].Status)
	require.Equal(t, "主线已经比较清楚", resp.Items[0].Summary)
	require.Equal(t, 1, resp.Limit)
}

func TestService_CreateSession_CapturesSnapshot(t *testing.T) {
	t.Parallel()

	svc := newTestReviewServiceWithNotes([]ReviewNote{{
		NotePath:      "wiki/concepts/并发控制与速率限制.md",
		Title:         "并发控制与速率限制",
		SourceTitle:   "InkWords 内容生成平台架构解析系列",
		Body:          strings.Repeat("正文内容", 80),
		PreferredMode: model.ReviewModeLightRecall,
	}})

	resp, err := svc.CreateSession(context.Background(), uuid.New(), CreateSessionRequest{
		NotePath:  "wiki/concepts/并发控制与速率限制.md",
		Mode:      model.ReviewModeLightRecall,
		EntryType: model.ReviewEntryTypeManualSelect,
	})
	require.NoError(t, err)
	require.Equal(t, model.ReviewStatusCreated, resp.Status)
	require.Equal(t, model.ReviewModeLightRecall, resp.Mode)
	require.Equal(t, "并发控制与速率限制", resp.Title)
	require.NotEmpty(t, resp.OpeningPrompt)
	require.NotEmpty(t, resp.InitialHints)
	require.Equal(t, 1, resp.TurnIndex)

	repo := svc.repo.(*stubReviewRepository)
	require.Len(t, repo.createCalls, 1)
	require.NotEmpty(t, repo.createCalls[0].SummarySnapshot)
	require.NotEmpty(t, repo.createCalls[0].KeyPointsSnapshot)
	require.Equal(t, model.ReviewStatusCreated, repo.createCalls[0].Status)
}

func TestService_GetSession_ReturnsPersistedTurns(t *testing.T) {
	t.Parallel()

	session := seedLightRecallSession(t)

	got, err := session.Service.GetSession(context.Background(), session.UserID, session.ID)
	require.NoError(t, err)
	require.Equal(t, session.ID, got.SessionID)
	require.Equal(t, model.ReviewStatusCreated, got.Status)
	require.Len(t, got.Turns, 1)
	require.Equal(t, model.ReviewTurnTypeOpening, got.Turns[0].TurnType)
}

func TestService_RespondDetailedQA_AdvancesThreeRounds(t *testing.T) {
	t.Parallel()

	session := seedDetailedQASession(t)

	first, err := session.Service.Respond(context.Background(), session.UserID, session.ID, RespondRequest{Answer: "这是主旨"})
	require.NoError(t, err)
	require.Equal(t, model.ReviewStatusInProgress, first.SessionStatus)
	require.Equal(t, "它的关键概念、步骤或关系是什么？", first.NextQuestion)
	require.False(t, first.Completed)

	second, err := session.Service.Respond(context.Background(), session.UserID, session.ID, RespondRequest{Answer: "这是细节"})
	require.NoError(t, err)
	require.Equal(t, "如果让你把它讲给一个新手，你会怎么解释？", second.NextQuestion)
	require.False(t, second.Completed)

	third, err := session.Service.Respond(context.Background(), session.UserID, session.ID, RespondRequest{Answer: "这是迁移解释"})
	require.NoError(t, err)
	require.True(t, third.Completed)
	require.Equal(t, model.ReviewStatusCompleted, third.SessionStatus)
	require.NotEmpty(t, third.FinalFeedback.Summary)
}

func TestService_RequestHint_StopsAtMaxCount(t *testing.T) {
	t.Parallel()

	session := seedLightRecallSession(t)

	first, err := session.Service.RequestHint(context.Background(), session.UserID, session.ID)
	require.NoError(t, err)
	require.NotEmpty(t, first.HintText)
	require.Equal(t, 1, first.RemainingHintCount)

	second, err := session.Service.RequestHint(context.Background(), session.UserID, session.ID)
	require.NoError(t, err)
	require.NotEmpty(t, second.HintText)
	require.Equal(t, 0, second.RemainingHintCount)

	_, err = session.Service.RequestHint(context.Background(), session.UserID, session.ID)
	require.ErrorContains(t, err, "提示次数已用尽")
}

func TestService_Finish_ProducesFinalFeedback(t *testing.T) {
	t.Parallel()

	session := seedLightRecallSession(t)

	_, err := session.Service.Respond(context.Background(), session.UserID, session.ID, RespondRequest{Answer: "我先讲主线"})
	require.NoError(t, err)

	resp, err := session.Service.Finish(context.Background(), session.UserID, session.ID)
	require.NoError(t, err)
	require.Equal(t, model.ReviewStatusCompleted, resp.SessionStatus)
	require.NotEmpty(t, resp.FinalFeedback.Summary)
	require.NotEmpty(t, resp.FinalFeedback.Strengths)
	require.NotEmpty(t, resp.FinalFeedback.Gaps)
	require.NotEmpty(t, resp.FinalFeedback.NextFocus)

	repo := session.Service.repo.(*stubReviewRepository)
	stored, err := repo.GetSessionByID(context.Background(), session.ID)
	require.NoError(t, err)
	require.Equal(t, model.ReviewStatusCompleted, stored.Status)
	require.NotNil(t, stored.CompletedAt)
}

type stubReviewRepository struct {
	recentSessions []model.ReviewSession
	createCalls    []*model.ReviewSession
	sessionsByID   map[uuid.UUID]*model.ReviewSession
	turnsBySession map[uuid.UUID][]model.ReviewTurn
}

func (s *stubReviewRepository) GetRecentSessions(_ context.Context, _ uuid.UUID, limit int) ([]model.ReviewSession, error) {
	sessions := append([]model.ReviewSession(nil), s.recentSessions...)
	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}
	return sessions, nil
}

func (s *stubReviewRepository) CreateSession(_ context.Context, session *model.ReviewSession) error {
	s.createCalls = append(s.createCalls, session)
	if s.sessionsByID == nil {
		s.sessionsByID = make(map[uuid.UUID]*model.ReviewSession)
	}
	clone := *session
	s.sessionsByID[session.ID] = &clone
	return nil
}

func (s *stubReviewRepository) GetSessionByID(_ context.Context, sessionID uuid.UUID) (model.ReviewSession, error) {
	if session, ok := s.sessionsByID[sessionID]; ok {
		return *session, nil
	}
	return model.ReviewSession{}, nil
}

func (s *stubReviewRepository) ListTurns(_ context.Context, sessionID uuid.UUID) ([]model.ReviewTurn, error) {
	turns := s.turnsBySession[sessionID]
	return append([]model.ReviewTurn(nil), turns...), nil
}

func (s *stubReviewRepository) AppendTurn(_ context.Context, turn *model.ReviewTurn) error {
	if s.turnsBySession == nil {
		s.turnsBySession = make(map[uuid.UUID][]model.ReviewTurn)
	}
	if turn.ID == uuid.Nil {
		turn.ID = uuid.New()
	}
	s.turnsBySession[turn.SessionID] = append(s.turnsBySession[turn.SessionID], *turn)
	return nil
}

func (s *stubReviewRepository) UpdateSession(_ context.Context, session *model.ReviewSession) error {
	if s.sessionsByID == nil {
		s.sessionsByID = make(map[uuid.UUID]*model.ReviewSession)
	}
	clone := *session
	s.sessionsByID[session.ID] = &clone
	return nil
}

type stubReviewNoteSource struct {
	notes []ReviewNote
}

func (s *stubReviewNoteSource) ListEligibleNotes(_ context.Context) ([]ReviewNote, error) {
	return append([]ReviewNote(nil), s.notes...), nil
}

func newTestReviewServiceWithNotes(notes []ReviewNote) *Service {
	return &Service{
		repo: &stubReviewRepository{
			sessionsByID:   make(map[uuid.UUID]*model.ReviewSession),
			turnsBySession: make(map[uuid.UUID][]model.ReviewTurn),
		},
		noteSource: &stubReviewNoteSource{notes: notes},
		now: func() time.Time {
			return time.Date(2026, 5, 27, 9, 0, 0, 0, time.UTC)
		},
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}

type seededSession struct {
	Service *Service
	UserID  uuid.UUID
	ID      uuid.UUID
}

func seedLightRecallSession(t *testing.T) seededSession {
	t.Helper()

	return seedSession(t, model.ReviewModeLightRecall)
}

func seedDetailedQASession(t *testing.T) seededSession {
	t.Helper()

	return seedSession(t, model.ReviewModeDetailedQA)
}

func seedSession(t *testing.T, mode string) seededSession {
	t.Helper()

	svc := newTestReviewServiceWithNotes([]ReviewNote{{
		NotePath:      "wiki/concepts/并发控制与速率限制.md",
		Title:         "并发控制与速率限制",
		SourceTitle:   "InkWords 内容生成平台架构解析系列",
		Body:          strings.Repeat("正文内容", 80),
		PreferredMode: mode,
	}})
	userID := uuid.New()

	resp, err := svc.CreateSession(context.Background(), userID, CreateSessionRequest{
		NotePath:  "wiki/concepts/并发控制与速率限制.md",
		Mode:      mode,
		EntryType: model.ReviewEntryTypeToday,
	})
	require.NoError(t, err)

	return seededSession{
		Service: svc,
		UserID:  userID,
		ID:      resp.SessionID,
	}
}

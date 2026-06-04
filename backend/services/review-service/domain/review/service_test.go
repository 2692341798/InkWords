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

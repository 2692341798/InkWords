package review

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"inkwords-backend/internal/model"
)

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

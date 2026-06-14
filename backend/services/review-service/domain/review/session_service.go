package review

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
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

	summarySnapshot, outline, sourcePreview := buildSessionSnapshot(note)
	opening := openingPrompt(req.Mode, outline)
	hints := initialHints(req.Mode, outline)
	now := s.now()

	session := ReviewSession{
		ID:                uuid.New(),
		UserID:            userID,
		NotePath:          note.NotePath,
		NoteTitle:         note.Title,
		SourceTitle:       note.SourceTitle,
		EntryType:         req.EntryType,
		Mode:              req.Mode,
		Status:            ReviewStatusCreated,
		EstimatedMinutes:  defaultReviewCardEstimatedMinutes,
		SummarySnapshot:   summarySnapshot,
		KeyPointsSnapshot: mustMarshalJSON(outline.Checkpoints),
		MetadataSnapshot: mustMarshalJSON(sessionMetadata{
			PreferredMode:  note.PreferredMode,
			SessionOutline: outline,
			SourcePreview:  sourcePreview,
		}),
		MaxHintCount: 2,
		TurnCount:    1,
		StartedAt:    now,
	}

	if err := s.repo.CreateSession(ctx, &session); err != nil {
		return ReviewSessionResponse{}, fmt.Errorf("创建复习会话失败: %w", err)
	}

	openingTurn := ReviewTurn{
		SessionID: session.ID,
		TurnIndex: 1,
		Role:      ReviewTurnRoleSystem,
		TurnType:  ReviewTurnTypeOpening,
		Content:   opening,
	}
	if err := s.repo.AppendTurn(ctx, &openingTurn); err != nil {
		return ReviewSessionResponse{}, fmt.Errorf("写入开场提示失败: %w", err)
	}

	return ReviewSessionResponse{
		SessionID:        session.ID,
		Status:           session.Status,
		Mode:             session.Mode,
		Title:            session.NoteTitle,
		SourceTitle:      session.SourceTitle,
		SourcePreview:    sourcePreview,
		ReadyToAnswer:    false,
		OpeningPrompt:    opening,
		InitialHints:     hints,
		SessionOutline:   outline,
		CurrentRoundGoal: currentRoundGoal(session.Mode, 0, outline),
		NextQuestion:     nextQuestionForSession(session, []ReviewTurn{openingTurn}, outline),
		TurnIndex:        openingTurn.TurnIndex,
		Turns:            []ReviewTurnResponse{toTurnResponse(openingTurn)},
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

	answerTurn := ReviewTurn{
		SessionID: session.ID,
		TurnIndex: nextTurnIndex(turns),
		Role:      ReviewTurnRoleUser,
		TurnType:  ReviewTurnTypeAnswer,
		Content:   answer,
	}
	if err := s.repo.AppendTurn(ctx, &answerTurn); err != nil {
		return RespondResponse{}, fmt.Errorf("写入回答失败: %w", err)
	}

	updatedTurns := append(append([]ReviewTurn(nil), turns...), answerTurn)
	session.Status = ReviewStatusInProgress
	session.TurnCount = answerTurn.TurnIndex
	outline := decodeSessionMetadata(session.MetadataSnapshot).SessionOutline
	metadata := decodeSessionMetadata(session.MetadataSnapshot)
	answerCount := countUserAnswers(updatedTurns)
	reviewFeedback := buildReviewFeedback(outline, answer)
	roundGoal := currentRoundGoal(session.Mode, answerCount, outline)
	stageFeedback := buildStageFeedback(session.Mode, reviewFeedback)
	hintText := ""
	excerptText := ""

	if indicatesMemoryGap(answer) {
		hintText = buildMemoryGapHint(outline)
		excerptText = buildMemoryGapExcerpt(metadata.SourcePreview, outline)
		stageFeedback = "这一轮先不用硬想完整答案，我先给你一个提醒，再带你回到原文里的关键位置。"
	}

	if s.aiFeedback != nil {
		result, err := s.aiFeedback.Generate(ctx, buildAIFeedbackInput(
			session.NoteTitle,
			session.Mode,
			metadata,
			toTurnResponses(updatedTurns),
			roundGoal,
			answer,
		))
		if err == nil {
			reviewFeedback = ReviewFeedback{
				Judgement:    firstNonEmpty(result.Judgement, reviewFeedback.Judgement),
				HitPoints:    ensureFeedbackItems(result.HitPoints, reviewFeedback.HitPoints[0]),
				MissedPoints: ensureFeedbackItems(result.MissedPoints, reviewFeedback.MissedPoints[0]),
				Suggestion:   firstNonEmpty(result.Suggestion, reviewFeedback.Suggestion),
			}
			stageFeedback = firstNonEmpty(result.StageFeedback, stageFeedback)
			hintText = firstNonEmpty(result.HintText, hintText)
			if result.ShouldShowQuote {
				excerptText = firstNonEmpty(result.ExcerptText, excerptText)
			}
		}
	}

	if session.Mode == ReviewModeDetailedQA {
		if answerCount >= maxDetailedQARounds {
			feedback := buildFinalFeedback(session.Mode, updatedTurns)
			if err := s.completeSession(ctx, &session, updatedTurns, feedback); err != nil {
				return RespondResponse{}, err
			}
			return RespondResponse{
				SessionID:        session.ID,
				SessionStatus:    session.Status,
				TurnIndex:        session.TurnCount,
				CurrentRoundGoal: roundGoal,
				ReviewFeedback:   reviewFeedback,
				HintText:         hintText,
				ExcerptText:      excerptText,
				Completed:        true,
				FinalFeedback:    feedback,
			}, nil
		}

		nextQuestion := nextDetailedQuestion(answerCount, outline)
		questionTurn := ReviewTurn{
			SessionID: session.ID,
			TurnIndex: answerTurn.TurnIndex + 1,
			Role:      ReviewTurnRoleSystem,
			TurnType:  ReviewTurnTypeQuestion,
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
			SessionID:        session.ID,
			SessionStatus:    session.Status,
			TurnIndex:        session.TurnCount,
			StageFeedback:    stageFeedback,
			CurrentRoundGoal: currentRoundGoal(session.Mode, answerCount, outline),
			ReviewFeedback:   reviewFeedback,
			NextQuestion:     nextQuestion,
			HintText:         hintText,
			ExcerptText:      excerptText,
			Completed:        false,
		}, nil
	}

	feedbackTurn := ReviewTurn{
		SessionID: session.ID,
		TurnIndex: answerTurn.TurnIndex + 1,
		Role:      ReviewTurnRoleSystem,
		TurnType:  ReviewTurnTypeFeedback,
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
		SessionID:        session.ID,
		SessionStatus:    session.Status,
		TurnIndex:        session.TurnCount,
		StageFeedback:    stageFeedback,
		CurrentRoundGoal: roundGoal,
		ReviewFeedback:   reviewFeedback,
		HintText:         hintText,
		ExcerptText:      excerptText,
		Completed:        false,
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

	metadata := decodeSessionMetadata(session.MetadataSnapshot)
	outline := metadata.SessionOutline
	hintText := buildHintText(session, turns, outline)
	lastAnswer := lastUserAnswer(turns)

	if indicatesMemoryGap(lastAnswer) {
		hintText = buildMemoryGapHint(outline)
		excerpt := buildMemoryGapExcerpt(metadata.SourcePreview, outline)
		if strings.TrimSpace(excerpt) != "" {
			hintText = hintText + "\n\n" + excerpt
		}
	}

	if s.aiFeedback != nil {
		result, aiErr := s.aiFeedback.Generate(ctx, buildAIFeedbackInput(
			session.NoteTitle,
			session.Mode,
			metadata,
			toTurnResponses(turns),
			currentRoundGoal(session.Mode, countUserAnswers(turns), outline),
			lastAnswer,
		))
		if aiErr == nil {
			hintText = firstNonEmpty(result.HintText, hintText)
			if result.ShouldShowQuote && strings.TrimSpace(result.ExcerptText) != "" {
				hintText = hintText + "\n\n" + result.ExcerptText
			}
		}
	}
	hintTurn := ReviewTurn{
		SessionID: session.ID,
		TurnIndex: nextTurnIndex(turns),
		Role:      ReviewTurnRoleSystem,
		TurnType:  ReviewTurnTypeHint,
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

	feedback := buildFinalFeedback(session.Mode, turns)
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

func (s *Service) loadOwnedSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (ReviewSession, []ReviewTurn, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return ReviewSession{}, nil, fmt.Errorf("查询复习会话失败: %w", err)
	}
	if session.ID == uuid.Nil {
		return ReviewSession{}, nil, errReviewSessionNotFound
	}
	if session.UserID != userID {
		return ReviewSession{}, nil, errReviewSessionDenied
	}

	turns, err := s.repo.ListTurns(ctx, session.ID)
	if err != nil {
		return ReviewSession{}, nil, fmt.Errorf("查询复习轮次失败: %w", err)
	}
	return session, turns, nil
}

func (s *Service) completeSession(ctx context.Context, session *ReviewSession, turns []ReviewTurn, feedback FinalFeedback) error {
	completionTurn := ReviewTurn{
		SessionID: session.ID,
		TurnIndex: nextTurnIndex(turns),
		Role:      ReviewTurnRoleSystem,
		TurnType:  ReviewTurnTypeCompletion,
		Content:   feedback.Summary,
	}
	if err := s.repo.AppendTurn(ctx, &completionTurn); err != nil {
		return fmt.Errorf("写入结束反馈失败: %w", err)
	}

	completedAt := s.now()
	session.Status = ReviewStatusCompleted
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

func buildSessionResponse(session ReviewSession, turns []ReviewTurn) ReviewSessionResponse {
	opening := ""
	for _, turn := range turns {
		if turn.TurnType == ReviewTurnTypeOpening {
			opening = turn.Content
			break
		}
	}
	metadata := decodeSessionMetadata(session.MetadataSnapshot)

	return ReviewSessionResponse{
		SessionID:            session.ID,
		Status:               session.Status,
		Mode:                 session.Mode,
		Title:                session.NoteTitle,
		SourceTitle:          session.SourceTitle,
		SourcePreview:        firstNonEmpty(metadata.SourcePreview, session.SummarySnapshot),
		ReadyToAnswer:        countUserAnswers(turns) > 0 || session.Status != ReviewStatusCreated,
		OpeningPrompt:        opening,
		InitialHints:         initialHints(session.Mode, metadata.SessionOutline),
		SessionOutline:       metadata.SessionOutline,
		CurrentRoundGoal:     currentRoundGoal(session.Mode, countUserAnswers(turns), metadata.SessionOutline),
		LatestReviewFeedback: latestReviewFeedback(metadata.SessionOutline, turns),
		NextQuestion:         nextQuestionForSession(session, turns, metadata.SessionOutline),
		TurnIndex:            len(turns),
		Turns:                toTurnResponses(turns),
	}
}

func nextQuestionForSession(session ReviewSession, turns []ReviewTurn, outline SessionOutline) string {
	if session.Mode != ReviewModeDetailedQA || isClosedStatus(session.Status) {
		return ""
	}

	answerCount := countUserAnswers(turns)
	if answerCount >= maxDetailedQARounds {
		return ""
	}
	return nextDetailedQuestion(answerCount, outline)
}

func latestReviewFeedback(outline SessionOutline, turns []ReviewTurn) *ReviewFeedback {
	answer := lastUserAnswer(turns)
	if strings.TrimSpace(answer) == "" {
		return nil
	}
	feedback := buildReviewFeedback(outline, answer)
	return &feedback
}

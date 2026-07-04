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
	readingContent := strings.TrimSpace(note.Body)
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
		Phase:             ReviewPhaseReading,
		EstimatedMinutes:  defaultReviewCardEstimatedMinutes,
		SummarySnapshot:   summarySnapshot,
		KeyPointsSnapshot: mustMarshalJSON(outline.Checkpoints),
		MetadataSnapshot: mustMarshalJSON(sessionMetadata{
			PreferredMode:  note.PreferredMode,
			SessionOutline: outline,
			SourcePreview:  sourcePreview,
			ReadingContent: readingContent,
		}),
		MaxHintCount: 3,
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
		Phase:            session.Phase,
		Mode:             session.Mode,
		Title:            session.NoteTitle,
		SourceTitle:      session.SourceTitle,
		SourcePreview:    sourcePreview,
		ReadingContent:   readingContent,
		ReadyToAnswer:    false,
		OpeningPrompt:    opening,
		InitialHints:     hints,
		SessionOutline:   outline,
		CurrentRoundGoal: currentRoundGoal(session.Mode, 0),
		NextQuestion:     nextQuestionForSession(session, []ReviewTurn{openingTurn}, outline),
		TurnIndex:        openingTurn.TurnIndex,
		Turns:            []ReviewTurnResponse{toTurnResponse(openingTurn)},
	}, nil
}

// CompleteReading 幂等地结束阅读阶段并允许用户开始脱离原文复述。
func (s *Service) CompleteReading(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (ReadingCompleteResponse, error) {
	session, _, err := s.loadOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return ReadingCompleteResponse{}, err
	}
	phase := resolveSessionPhase(session)
	if isClosedStatus(session.Status) {
		return ReadingCompleteResponse{SessionID: session.ID, Status: session.Status, Phase: phase}, nil
	}
	if phase != ReviewPhaseReading {
		return ReadingCompleteResponse{SessionID: session.ID, Status: session.Status, Phase: phase}, nil
	}
	now := s.now()
	session.Phase = ReviewPhaseRecalling
	session.ReadingCompletedAt = &now
	if err := s.repo.UpdateSession(ctx, &session); err != nil {
		return ReadingCompleteResponse{}, fmt.Errorf("更新阅读状态失败: %w", err)
	}
	return ReadingCompleteResponse{SessionID: session.ID, Status: session.Status, Phase: session.Phase}, nil
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
	if resolveSessionPhase(session) == ReviewPhaseReading {
		return RespondResponse{}, errReviewReadingPending
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
	session.Phase = ReviewPhaseCoaching
	session.TurnCount = answerTurn.TurnIndex
	outline := decodeSessionMetadata(session.MetadataSnapshot).SessionOutline
	metadata := decodeSessionMetadata(session.MetadataSnapshot)
	answerCount := countUserAnswers(updatedTurns)
	reviewFeedback := buildReviewFeedback(outline, answer)
	roundGoal := currentRoundGoal(session.Mode, answerCount)
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
				Phase:            session.Phase,
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
			Phase:            session.Phase,
			TurnIndex:        session.TurnCount,
			StageFeedback:    stageFeedback,
			CurrentRoundGoal: currentRoundGoal(session.Mode, answerCount),
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
		Phase:            session.Phase,
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
func (s *Service) RequestHint(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, req HintRequest) (HintResponse, error) {
	session, turns, err := s.loadOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return HintResponse{}, err
	}
	if isClosedStatus(session.Status) {
		return HintResponse{}, errReviewSessionClosed
	}
	if resolveSessionPhase(session) == ReviewPhaseReading {
		return HintResponse{}, errReviewReadingPending
	}
	if session.HintUsedCount >= session.MaxHintCount {
		return HintResponse{}, errReviewHintExhausted
	}

	metadata := decodeSessionMetadata(session.MetadataSnapshot)
	hint := buildLeveledHint(session, turns, metadata)
	if s.aiFeedback != nil {
		result, generateErr := s.aiFeedback.Generate(ctx, buildAIHintInput(session, metadata, turns, req.Answer))
		if generateErr == nil {
			hint.HintText = firstNonEmpty(result.HintText, result.Suggestion, hint.HintText)
			if len(result.MissedPoints) > 0 {
				hint.TargetGap = firstNonEmpty(result.MissedPoints[0], hint.TargetGap)
			}
			hint.NextAction = firstNonEmpty(result.Suggestion, hint.NextAction)
		}
	}
	hintTurn := ReviewTurn{
		SessionID: session.ID,
		TurnIndex: nextTurnIndex(turns),
		Role:      ReviewTurnRoleSystem,
		TurnType:  ReviewTurnTypeHint,
		Content:   hint.HintText,
	}
	if err := s.repo.AppendTurn(ctx, &hintTurn); err != nil {
		return HintResponse{}, fmt.Errorf("写入提示失败: %w", err)
	}

	session.HintUsedCount++
	// A hint requested while recalling should keep the answer surface visible.
	// Coaching begins after the user has submitted at least one recall answer.
	if resolveSessionPhase(session) == ReviewPhaseCoaching {
		session.Phase = ReviewPhaseCoaching
	}
	session.TurnCount = hintTurn.TurnIndex
	if err := s.repo.UpdateSession(ctx, &session); err != nil {
		return HintResponse{}, fmt.Errorf("更新复习会话失败: %w", err)
	}

	hint.RemainingHintCount = session.MaxHintCount - session.HintUsedCount
	return hint, nil
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
	if resolveSessionPhase(session) == ReviewPhaseReading {
		return FinishResponse{}, errReviewReadingPending
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
	session.Phase = ReviewPhaseCompleted
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
		Phase:                resolveSessionPhase(session),
		Mode:                 session.Mode,
		Title:                session.NoteTitle,
		SourceTitle:          session.SourceTitle,
		SourcePreview:        firstNonEmpty(metadata.SourcePreview, session.SummarySnapshot),
		ReadingContent:       firstNonEmpty(metadata.ReadingContent, metadata.SourcePreview, session.SummarySnapshot),
		ReadyToAnswer:        resolveSessionPhase(session) != ReviewPhaseReading,
		OpeningPrompt:        opening,
		InitialHints:         initialHints(session.Mode, metadata.SessionOutline),
		SessionOutline:       metadata.SessionOutline,
		CurrentRoundGoal:     currentRoundGoal(session.Mode, countUserAnswers(turns)),
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

package review

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"inkwords-backend/internal/model"
)

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


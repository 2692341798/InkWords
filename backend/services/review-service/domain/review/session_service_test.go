package review

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestService_CreateSession_CapturesSnapshot(t *testing.T) {
	t.Parallel()

	svc := newTestReviewServiceWithNotes([]ReviewNote{{
		NotePath:      "wiki/concepts/并发控制与速率限制.md",
		Title:         "并发控制与速率限制",
		SourceTitle:   "InkWords 内容生成平台架构解析系列",
		Body:          strings.Repeat("正文内容", 80),
		PreferredMode: ReviewModeLightRecall,
	}})

	resp, err := svc.CreateSession(context.Background(), uuid.New(), CreateSessionRequest{
		NotePath:  "wiki/concepts/并发控制与速率限制.md",
		Mode:      ReviewModeLightRecall,
		EntryType: ReviewEntryTypeManualSelect,
	})
	require.NoError(t, err)
	require.Equal(t, ReviewStatusCreated, resp.Status)
	require.Equal(t, ReviewModeLightRecall, resp.Mode)
	require.Equal(t, "并发控制与速率限制", resp.Title)
	require.NotEmpty(t, resp.OpeningPrompt)
	require.NotEmpty(t, resp.InitialHints)
	require.Equal(t, 1, resp.TurnIndex)

	repo := svc.repo.(*stubReviewRepository)
	require.Len(t, repo.createCalls, 1)
	require.NotEmpty(t, repo.createCalls[0].SummarySnapshot)
	require.NotEmpty(t, repo.createCalls[0].KeyPointsSnapshot)
	require.Equal(t, ReviewStatusCreated, repo.createCalls[0].Status)
}

func TestService_CreateSession_BuildsStructuredSnapshot(t *testing.T) {
	t.Parallel()

	svc := newTestReviewServiceWithNotes([]ReviewNote{{
		NotePath:      "wiki/concepts/并发控制与速率限制.md",
		Title:         "并发控制与速率限制",
		SourceTitle:   "InkWords 内容生成平台架构解析系列",
		Body:          "并发控制用来限制同时执行的任务数量，避免资源打满。信号量负责发放并回收执行许可。速率限制负责控制单位时间内的请求频率，避免 API 被突发流量冲垮。实际落地时，通常会把并发上限、队列等待和重试退避组合起来。",
		PreferredMode: ReviewModeDetailedQA,
	}})

	resp, err := svc.CreateSession(context.Background(), uuid.New(), CreateSessionRequest{
		NotePath:  "wiki/concepts/并发控制与速率限制.md",
		Mode:      ReviewModeDetailedQA,
		EntryType: ReviewEntryTypeManualSelect,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.SessionOutline.MainQuestion)
	require.NotEmpty(t, resp.SessionOutline.Checkpoints)
	require.NotEmpty(t, resp.CurrentRoundGoal)
	require.Contains(t, resp.NextQuestion, "并发控制")
}

func TestService_CreateSession_ReturnsPreviewContent(t *testing.T) {
	t.Parallel()

	svc := newTestReviewServiceWithNotes([]ReviewNote{{
		NotePath:      "wiki/concepts/并发控制与速率限制.md",
		Title:         "并发控制与速率限制",
		SourceTitle:   "InkWords 内容生成平台架构解析系列",
		Body:          "第一段原文。\n\n第二段原文。",
		PreferredMode: ReviewModeLightRecall,
	}})

	resp, err := svc.CreateSession(context.Background(), uuid.New(), CreateSessionRequest{
		NotePath:  "wiki/concepts/并发控制与速率限制.md",
		Mode:      ReviewModeLightRecall,
		EntryType: ReviewEntryTypeManualSelect,
	})
	require.NoError(t, err)
	require.Equal(t, "InkWords 内容生成平台架构解析系列", resp.SourceTitle)
	require.Contains(t, resp.SourcePreview, "第一段原文")
	require.False(t, resp.ReadyToAnswer)
}

func TestService_GetSession_ReturnsPersistedTurns(t *testing.T) {
	t.Parallel()

	session := seedLightRecallSession(t)

	got, err := session.Service.GetSession(context.Background(), session.UserID, session.ID)
	require.NoError(t, err)
	require.Equal(t, session.ID, got.SessionID)
	require.Equal(t, ReviewStatusCreated, got.Status)
	require.Len(t, got.Turns, 1)
	require.Equal(t, ReviewTurnTypeOpening, got.Turns[0].TurnType)
}

func TestService_RespondDetailedQA_AdvancesThreeRounds(t *testing.T) {
	t.Parallel()

	session := seedDetailedQASession(t)

	first, err := session.Service.Respond(context.Background(), session.UserID, session.ID, RespondRequest{Answer: "这是主旨"})
	require.NoError(t, err)
	require.Equal(t, ReviewStatusInProgress, first.SessionStatus)
	require.Contains(t, first.NextQuestion, "概念")
	require.False(t, first.Completed)

	second, err := session.Service.Respond(context.Background(), session.UserID, session.ID, RespondRequest{Answer: "这是细节"})
	require.NoError(t, err)
	require.Contains(t, second.NextQuestion, "新手")
	require.False(t, second.Completed)

	third, err := session.Service.Respond(context.Background(), session.UserID, session.ID, RespondRequest{Answer: "这是迁移解释"})
	require.NoError(t, err)
	require.True(t, third.Completed)
	require.Equal(t, ReviewStatusCompleted, third.SessionStatus)
	require.NotEmpty(t, third.FinalFeedback.Summary)
}

func TestService_Respond_ReturnsStructuredStageFeedback(t *testing.T) {
	t.Parallel()

	svc := newTestReviewServiceWithNotes([]ReviewNote{{
		NotePath:      "wiki/concepts/并发控制与速率限制.md",
		Title:         "并发控制与速率限制",
		SourceTitle:   "InkWords 内容生成平台架构解析系列",
		Body:          "并发控制用来限制同时执行的任务数量，避免资源打满。信号量负责发放并回收执行许可。速率限制负责控制单位时间内的请求频率，避免 API 被突发流量冲垮。",
		PreferredMode: ReviewModeDetailedQA,
	}})
	userID := uuid.New()

	created, err := svc.CreateSession(context.Background(), userID, CreateSessionRequest{
		NotePath:  "wiki/concepts/并发控制与速率限制.md",
		Mode:      ReviewModeDetailedQA,
		EntryType: ReviewEntryTypeToday,
	})
	require.NoError(t, err)

	resp, err := svc.Respond(context.Background(), userID, created.SessionID, RespondRequest{
		Answer: "这篇文章主要讲如何通过并发控制和信号量保护资源。",
	})
	require.NoError(t, err)
	require.Equal(t, "答对较多", resp.ReviewFeedback.Judgement)
	require.NotEmpty(t, resp.ReviewFeedback.HitPoints)
	require.NotEmpty(t, resp.ReviewFeedback.MissedPoints)
	require.NotEmpty(t, resp.CurrentRoundGoal)
}

func TestService_Respond_WhenUserDoesNotRemember_ReturnsHintThenExcerpt(t *testing.T) {
	t.Parallel()

	svc := newTestReviewServiceWithNotes([]ReviewNote{{
		NotePath:      "wiki/concepts/并发控制与速率限制.md",
		Title:         "并发控制与速率限制",
		SourceTitle:   "InkWords 内容生成平台架构解析系列",
		Body:          "并发控制用来限制同时执行的任务数量，避免资源打满。信号量负责发放并回收执行许可。速率限制负责控制单位时间内的请求频率。",
		PreferredMode: ReviewModeLightRecall,
	}})
	userID := uuid.New()

	created, err := svc.CreateSession(context.Background(), userID, CreateSessionRequest{
		NotePath:  "wiki/concepts/并发控制与速率限制.md",
		Mode:      ReviewModeLightRecall,
		EntryType: ReviewEntryTypeManualSelect,
	})
	require.NoError(t, err)

	resp, err := svc.Respond(context.Background(), userID, created.SessionID, RespondRequest{
		Answer: "我不记得了",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.HintText)
	require.Contains(t, resp.HintText, "先")
	require.NotEmpty(t, resp.ExcerptText)
	require.Contains(t, resp.ExcerptText, "并发控制")
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

func TestService_RequestHint_DetailedQAFollowsCurrentQuestion(t *testing.T) {
	t.Parallel()

	session := seedDetailedQASession(t)

	_, err := session.Service.Respond(context.Background(), session.UserID, session.ID, RespondRequest{Answer: "这是主旨"})
	require.NoError(t, err)

	hint, err := session.Service.RequestHint(context.Background(), session.UserID, session.ID)
	require.NoError(t, err)
	require.Contains(t, hint.HintText, "当前追问是：")
	require.Contains(t, hint.HintText, "概念")
}

func TestService_RequestHint_WhenUserIsStuck_ReturnsConcreteContext(t *testing.T) {
	t.Parallel()

	svc := newTestReviewServiceWithNotes([]ReviewNote{{
		NotePath:      "wiki/concepts/并发控制与速率限制.md",
		Title:         "并发控制与速率限制",
		SourceTitle:   "InkWords 内容生成平台架构解析系列",
		Body:          "并发控制用来限制同时执行的任务数量，避免资源打满。信号量负责发放并回收执行许可。速率限制负责控制单位时间内的请求频率。",
		PreferredMode: ReviewModeLightRecall,
	}})
	userID := uuid.New()

	created, err := svc.CreateSession(context.Background(), userID, CreateSessionRequest{
		NotePath:  "wiki/concepts/并发控制与速率限制.md",
		Mode:      ReviewModeLightRecall,
		EntryType: ReviewEntryTypeManualSelect,
	})
	require.NoError(t, err)

	_, err = svc.Respond(context.Background(), userID, created.SessionID, RespondRequest{Answer: "我不记得了"})
	require.NoError(t, err)

	hint, err := svc.RequestHint(context.Background(), userID, created.SessionID)
	require.NoError(t, err)
	require.Contains(t, hint.HintText, "并发控制")
	require.Contains(t, hint.HintText, "原文")
}

func TestService_Finish_ProducesFinalFeedback(t *testing.T) {
	t.Parallel()

	session := seedLightRecallSession(t)

	_, err := session.Service.Respond(context.Background(), session.UserID, session.ID, RespondRequest{Answer: "我先讲主线"})
	require.NoError(t, err)

	resp, err := session.Service.Finish(context.Background(), session.UserID, session.ID)
	require.NoError(t, err)
	require.Equal(t, ReviewStatusCompleted, resp.SessionStatus)
	require.NotEmpty(t, resp.FinalFeedback.Summary)
	require.NotEmpty(t, resp.FinalFeedback.Strengths)
	require.NotEmpty(t, resp.FinalFeedback.Gaps)
	require.NotEmpty(t, resp.FinalFeedback.NextFocus)

	repo := session.Service.repo.(*stubReviewRepository)
	stored, err := repo.GetSessionByID(context.Background(), session.ID)
	require.NoError(t, err)
	require.Equal(t, ReviewStatusCompleted, stored.Status)
	require.NotNil(t, stored.CompletedAt)
}

func TestService_Finish_DetailedQADoesNotOverstatePartialProgress(t *testing.T) {
	t.Parallel()

	session := seedDetailedQASession(t)

	_, err := session.Service.Respond(context.Background(), session.UserID, session.ID, RespondRequest{Answer: "我先回答第一问"})
	require.NoError(t, err)

	resp, err := session.Service.Finish(context.Background(), session.UserID, session.ID)
	require.NoError(t, err)
	require.Equal(t, ReviewStatusCompleted, resp.SessionStatus)
	require.NotContains(t, resp.FinalFeedback.Summary, "完成一轮逐步追问式复习")
	require.Contains(t, resp.FinalFeedback.Summary, "一部分逐步追问")
}

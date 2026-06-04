package review

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"inkwords-backend/internal/model"
)

func TestHandler_GetTodayCard_Returns200(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		c.Next()
	})

	h := NewHandler(&stubHandlerService{
		todayCard: ReviewCardResponse{
			NotePath:         "wiki/concepts/gin.md",
			Title:            "Gin 路由",
			SourceTitle:      "后端系列",
			ReviewReason:     "这是你最近导入但还没复习过的一篇内容。",
			EstimatedMinutes: 5,
			AvailableModes:   []string{model.ReviewModeLightRecall, model.ReviewModeDetailedQA},
		},
	})
	r.GET("/api/v1/review/today", h.GetTodayCard)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/review/today", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Code    int                `json:"code"`
		Message string             `json:"message"`
		Data    ReviewCardResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, http.StatusOK, body.Code)
	require.Equal(t, "success", body.Message)
	require.Equal(t, "wiki/concepts/gin.md", body.Data.NotePath)
}

func TestHandler_GetTodayCard_ReturnsUnauthorizedWithoutUser(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	h := NewHandler(&stubHandlerService{})
	r.GET("/api/v1/review/today", h.GetTodayCard)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/review/today", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CreateSession_Returns200(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	userID := uuid.New()
	sessionID := uuid.New()

	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})

	h := NewHandler(&stubHandlerService{
		sessionResp: ReviewSessionResponse{
			SessionID:        sessionID,
			Status:           model.ReviewStatusCreated,
			Mode:             model.ReviewModeLightRecall,
			Title:            "Gin 路由",
			OpeningPrompt:    "先别看原文，试着用自己的话讲讲这篇内容。",
			InitialHints:     []string{"这篇内容主要在解决什么问题？"},
			SessionOutline:   SessionOutline{Summary: "Gin 路由摘要", MainQuestion: "Gin 路由主要在解决什么问题？", Checkpoints: []string{"先讲清楚路由主线"}},
			CurrentRoundGoal: "先讲清楚路由主线",
			TurnIndex:        1,
		},
	})
	r.POST("/api/v1/review/sessions", h.CreateSession)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/review/sessions", strings.NewReader(`{"note_path":"wiki/concepts/gin.md","mode":"light_recall","entry_type":"today"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, CreateSessionRequest{
		NotePath:  "wiki/concepts/gin.md",
		Mode:      model.ReviewModeLightRecall,
		EntryType: model.ReviewEntryTypeToday,
	}, h.service.(*stubHandlerService).lastCreateReq)
}

func TestHandler_GetHistory_Returns200(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		c.Next()
	})

	h := NewHandler(&stubHandlerService{
		historyResp: ReviewHistoryResponse{
			Items: []ReviewHistoryItem{
				{
					SessionID:   uuid.New(),
					Title:       "Gin 路由",
					SourceTitle: "后端系列",
					Status:      model.ReviewStatusCompleted,
					Mode:        model.ReviewModeLightRecall,
					Summary:     "已经抓住了主线",
				},
			},
			Limit: 5,
		},
	})
	r.GET("/api/v1/review/history", h.GetHistory)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/review/history?limit=5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Code    int                   `json:"code"`
		Message string                `json:"message"`
		Data    ReviewHistoryResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, http.StatusOK, body.Code)
	require.Equal(t, 5, body.Data.Limit)
	require.Len(t, body.Data.Items, 1)
	require.Equal(t, "Gin 路由", body.Data.Items[0].Title)
}

type stubHandlerService struct {
	todayCard     ReviewCardResponse
	randomCard    ReviewCardResponse
	listNotesResp ListNotesResponse
	historyResp   ReviewHistoryResponse
	sessionResp   ReviewSessionResponse
	respondResp   RespondResponse
	hintResp      HintResponse
	finishResp    FinishResponse

	lastCreateReq CreateSessionRequest
}

func (s *stubHandlerService) GetTodayCard(context.Context, uuid.UUID) (ReviewCardResponse, error) {
	return s.todayCard, nil
}

func (s *stubHandlerService) PickRandomCard(context.Context, uuid.UUID) (ReviewCardResponse, error) {
	return s.randomCard, nil
}

func (s *stubHandlerService) ListNotes(context.Context, uuid.UUID, ListNotesQuery) (ListNotesResponse, error) {
	return s.listNotesResp, nil
}

func (s *stubHandlerService) GetHistory(context.Context, uuid.UUID, int) (ReviewHistoryResponse, error) {
	return s.historyResp, nil
}

func (s *stubHandlerService) CreateSession(_ context.Context, _ uuid.UUID, req CreateSessionRequest) (ReviewSessionResponse, error) {
	s.lastCreateReq = req
	return s.sessionResp, nil
}

func (s *stubHandlerService) GetSession(context.Context, uuid.UUID, uuid.UUID) (ReviewSessionResponse, error) {
	return s.sessionResp, nil
}

func (s *stubHandlerService) Respond(context.Context, uuid.UUID, uuid.UUID, RespondRequest) (RespondResponse, error) {
	return s.respondResp, nil
}

func (s *stubHandlerService) RequestHint(context.Context, uuid.UUID, uuid.UUID) (HintResponse, error) {
	return s.hintResp, nil
}

func (s *stubHandlerService) Finish(context.Context, uuid.UUID, uuid.UUID) (FinishResponse, error) {
	return s.finishResp, nil
}

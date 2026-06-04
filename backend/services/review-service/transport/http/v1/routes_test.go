package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	reviewdomain "inkwords-backend/services/review-service/domain/review"
)

func TestRegisterReviewRoutes_RegistersReviewRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	RegisterReviewRoutes(r, func(c *gin.Context) { c.Next() }, reviewdomain.NewHandler(stubReviewService{
		getTodayCard: func() reviewdomain.ReviewCardResponse {
			return reviewdomain.ReviewCardResponse{}
		},
	}))

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/review/today"},
		{method: http.MethodGet, path: "/api/v1/review/history"},
		{method: http.MethodPost, path: "/api/v1/review/pick"},
		{method: http.MethodGet, path: "/api/v1/review/notes"},
		{method: http.MethodPost, path: "/api/v1/review/sessions"},
		{method: http.MethodGet, path: "/api/v1/review/sessions/123e4567-e89b-12d3-a456-426614174000"},
		{method: http.MethodPost, path: "/api/v1/review/sessions/123e4567-e89b-12d3-a456-426614174000/respond"},
		{method: http.MethodPost, path: "/api/v1/review/sessions/123e4567-e89b-12d3-a456-426614174000/hint"},
		{method: http.MethodPost, path: "/api/v1/review/sessions/123e4567-e89b-12d3-a456-426614174000/finish"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		require.NotEqual(t, http.StatusNotFound, resp.Code)
	}

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/blogs/1/export", nil)
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusNotFound, resp.Code)
}

type stubReviewService struct {
	getTodayCard func() reviewdomain.ReviewCardResponse
}

func (s stubReviewService) GetTodayCard(_ context.Context, _ uuid.UUID) (reviewdomain.ReviewCardResponse, error) {
	if s.getTodayCard != nil {
		return s.getTodayCard(), nil
	}
	return reviewdomain.ReviewCardResponse{}, nil
}

func (stubReviewService) GetHistory(context.Context, uuid.UUID, int) (reviewdomain.ReviewHistoryResponse, error) {
	return reviewdomain.ReviewHistoryResponse{}, nil
}

func (stubReviewService) PickRandomCard(context.Context, uuid.UUID) (reviewdomain.ReviewCardResponse, error) {
	return reviewdomain.ReviewCardResponse{}, nil
}

func (stubReviewService) ListNotes(context.Context, uuid.UUID, reviewdomain.ListNotesQuery) (reviewdomain.ListNotesResponse, error) {
	return reviewdomain.ListNotesResponse{}, nil
}

func (stubReviewService) CreateSession(context.Context, uuid.UUID, reviewdomain.CreateSessionRequest) (reviewdomain.ReviewSessionResponse, error) {
	return reviewdomain.ReviewSessionResponse{}, nil
}

func (stubReviewService) GetSession(context.Context, uuid.UUID, uuid.UUID) (reviewdomain.ReviewSessionResponse, error) {
	return reviewdomain.ReviewSessionResponse{}, nil
}

func (stubReviewService) Respond(context.Context, uuid.UUID, uuid.UUID, reviewdomain.RespondRequest) (reviewdomain.RespondResponse, error) {
	return reviewdomain.RespondResponse{}, nil
}

func (stubReviewService) RequestHint(context.Context, uuid.UUID, uuid.UUID) (reviewdomain.HintResponse, error) {
	return reviewdomain.HintResponse{}, nil
}

func (stubReviewService) Finish(context.Context, uuid.UUID, uuid.UUID) (reviewdomain.FinishResponse, error) {
	return reviewdomain.FinishResponse{}, nil
}
